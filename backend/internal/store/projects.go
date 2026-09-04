package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type FsItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"` // image | video | audio
	Size        string `json:"size"`
	Time        string `json:"time"`
	Meta        string `json:"meta,omitempty"`
	SourceTask  string `json:"sourceTask,omitempty"`
	SizeBytes   int64  `json:"sizeBytes,omitempty"`
	StoragePath string `json:"storagePath,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	RemoteURL   string `json:"remoteUrl,omitempty"`
}

type FsFolder struct {
	IsSystem bool                `json:"isSystem"`
	Desc     string              `json:"desc"`
	Folders  map[string]FsFolder `json:"folders"`
	Items    []FsItem            `json:"items"`
}

type Project struct {
	ID          string
	UserID      int64
	Name        string
	Description string
	IsDefault   bool
	Trashed     bool
	TrashedAt   *time.Time
	Folders     map[string]FsFolder
	CreatedAt   time.Time
	UpdatedAt   time.Time
	FileCount   int
	SizeBytes   int64
}

func NewProjectID() string { return "prj_" + shortID() }
func NewAssetID() string   { return "ast_" + shortID() }

func StandardFolders() map[string]FsFolder {
	empty := func(desc string, system bool) FsFolder {
		return FsFolder{IsSystem: system, Desc: desc, Folders: map[string]FsFolder{}, Items: []FsItem{}}
	}
	return map[string]FsFolder{
		"资产": {
			IsSystem: true,
			Desc:     "核心资产固定目录",
			Folders: map[string]FsFolder{
				"人物": empty("角色与人设图", true),
				"场景": empty("环境与场景参考", true),
				"道具": empty("装备与物品特写", true),
				"素材": empty("纹理、贴图与音效", true),
			},
			Items: []FsItem{},
		},
		"视频": {
			IsSystem: true,
			Desc:     "视频成片输出主目录",
			Folders: map[string]FsFolder{
				"正片镜头": empty("默认镜头分集", false),
			},
			Items: []FsItem{},
		},
	}
}

func FormatBytesPublic(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
}

func CountFolderPublic(folder FsFolder) (files int, bytes int64) {
	return countFolderStats(folder)
}

func countFolderStats(folder FsFolder) (files int, bytes int64) {
	files = len(folder.Items)
	for _, it := range folder.Items {
		bytes += it.SizeBytes
	}
	for _, child := range folder.Folders {
		f, b := countFolderStats(child)
		files += f
		bytes += b
	}
	return files, bytes
}

func countProjectStats(folders map[string]FsFolder) (files int, bytes int64) {
	for _, folder := range folders {
		f, b := countFolderStats(folder)
		files += f
		bytes += b
	}
	return files, bytes
}

func collectStoragePaths(folders map[string]FsFolder) []string {
	var paths []string
	var walk func(map[string]FsFolder)
	walk = func(fs map[string]FsFolder) {
		for _, folder := range fs {
			for _, it := range folder.Items {
				if it.StoragePath != "" {
					paths = append(paths, it.StoragePath)
				}
			}
			walk(folder.Folders)
		}
	}
	walk(folders)
	return paths
}

func findFileInFolders(folders map[string]FsFolder, fileID string) (*FsItem, []string, bool) {
	var walk func(map[string]FsFolder, []string) (*FsItem, []string, bool)
	walk = func(fs map[string]FsFolder, prefix []string) (*FsItem, []string, bool) {
		for name, folder := range fs {
			path := append(append([]string{}, prefix...), name)
			for i := range folder.Items {
				if folder.Items[i].ID == fileID {
					item := folder.Items[i]
					return &item, path, true
				}
			}
			if item, p, ok := walk(folder.Folders, path); ok {
				return item, p, true
			}
		}
		return nil, nil, false
	}
	return walk(folders, nil)
}

func removeFileFromFolders(folders map[string]FsFolder, fileID string) (map[string]FsFolder, *FsItem, bool) {
	var removed *FsItem
	var walk func(map[string]FsFolder) map[string]FsFolder
	walk = func(fs map[string]FsFolder) map[string]FsFolder {
		out := make(map[string]FsFolder, len(fs))
		for name, folder := range fs {
			items := make([]FsItem, 0, len(folder.Items))
			for _, it := range folder.Items {
				if it.ID == fileID {
					cp := it
					removed = &cp
					continue
				}
				items = append(items, it)
			}
			folder.Items = items
			folder.Folders = walk(folder.Folders)
			out[name] = folder
		}
		return out
	}
	next := walk(folders)
	return next, removed, removed != nil
}

func resolveFolder(folders map[string]FsFolder, path []string) (map[string]FsFolder, *FsFolder, error) {
	if len(path) == 0 {
		return folders, nil, nil
	}
	cur := folders
	var folder *FsFolder
	for _, segment := range path {
		f, ok := cur[segment]
		if !ok {
			return nil, nil, errors.New("目录不存在")
		}
		cp := f
		folder = &cp
		cur = f.Folders
	}
	return folder.Folders, folder, nil
}

func putFileInPath(folders map[string]FsFolder, path []string, item FsItem) (map[string]FsFolder, error) {
	if len(path) == 0 {
		return nil, errors.New("上传路径不能为空")
	}
	var walk func(map[string]FsFolder, []string) (map[string]FsFolder, error)
	walk = func(fs map[string]FsFolder, remaining []string) (map[string]FsFolder, error) {
		out := make(map[string]FsFolder, len(fs))
		for k, v := range fs {
			out[k] = v
		}
		name := remaining[0]
		folder, ok := out[name]
		if !ok {
			return nil, errors.New("目录不存在")
		}
		if len(remaining) == 1 {
			folder.Items = append(folder.Items, item)
			out[name] = folder
			return out, nil
		}
		if folder.Folders == nil {
			folder.Folders = map[string]FsFolder{}
		}
		next, err := walk(folder.Folders, remaining[1:])
		if err != nil {
			return nil, err
		}
		folder.Folders = next
		out[name] = folder
		return out, nil
	}
	return walk(folders, path)
}

func marshalFolders(folders map[string]FsFolder) ([]byte, error) {
	if folders == nil {
		folders = StandardFolders()
	}
	return json.Marshal(folders)
}

func unmarshalFolders(raw []byte) (map[string]FsFolder, error) {
	if len(raw) == 0 || string(raw) == "{}" || string(raw) == "null" {
		return StandardFolders(), nil
	}
	var folders map[string]FsFolder
	if err := json.Unmarshal(raw, &folders); err != nil {
		return nil, err
	}
	if folders == nil {
		return StandardFolders(), nil
	}
	return folders, nil
}

func (s *Store) scanProject(row pgx.Row) (*Project, error) {
	var p Project
	var raw []byte
	var trashedAt *time.Time
	err := row.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.IsDefault, &trashedAt, &raw, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.TrashedAt = trashedAt
	p.Trashed = trashedAt != nil
	folders, err := unmarshalFolders(raw)
	if err != nil {
		return nil, err
	}
	p.Folders = folders
	p.FileCount, p.SizeBytes = countProjectStats(folders)
	return &p, nil
}

func (s *Store) CreateProject(ctx context.Context, userID int64, name, description string, isDefault bool) (*Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("作品名称不能为空")
	}
	if description == "" {
		if isDefault {
			description = "系统默认工程"
		} else {
			description = "按标准工程模板创建"
		}
	}
	publicID := NewProjectID()
	raw, err := marshalFolders(StandardFolders())
	if err != nil {
		return nil, err
	}
	_, err = s.Pool.Exec(ctx, `
INSERT INTO projects (public_id, user_id, name, description, is_default, folders_json)
VALUES ($1,$2,$3,$4,$5,$6::jsonb)`, publicID, userID, name, description, isDefault, string(raw))
	if err != nil {
		return nil, err
	}
	return s.GetProject(ctx, userID, publicID)
}

func (s *Store) EnsureDefaultProject(ctx context.Context, userID int64) (*Project, error) {
	var publicID string
	err := s.Pool.QueryRow(ctx, `
SELECT public_id FROM projects
WHERE user_id=$1 AND is_default=TRUE AND trashed_at IS NULL
ORDER BY created_at ASC LIMIT 1`, userID).Scan(&publicID)
	if err == nil {
		return s.GetProject(ctx, userID, publicID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	p, err := s.CreateProject(ctx, userID, "default作品", "系统默认工程", true)
	if err != nil {
		return nil, err
	}
	_ = s.SetCurrentProject(ctx, userID, p.ID)
	return p, nil
}

func (s *Store) GetProject(ctx context.Context, userID int64, publicID string) (*Project, error) {
	row := s.Pool.QueryRow(ctx, `
SELECT public_id, user_id, name, description, is_default, trashed_at, folders_json, created_at, updated_at
FROM projects WHERE public_id=$1 AND user_id=$2`, publicID, userID)
	return s.scanProject(row)
}

func (s *Store) ListProjects(ctx context.Context, userID int64, trashed bool) ([]Project, error) {
	if _, err := s.EnsureDefaultProject(ctx, userID); err != nil {
		return nil, err
	}
	query := `
SELECT public_id, user_id, name, description, is_default, trashed_at, folders_json, created_at, updated_at
FROM projects WHERE user_id=$1`
	if trashed {
		query += ` AND trashed_at IS NOT NULL`
	} else {
		query += ` AND trashed_at IS NULL`
	}
	query += ` ORDER BY is_default DESC, updated_at DESC`
	rows, err := s.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Project
	for rows.Next() {
		p, err := s.scanProject(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *p)
	}
	return items, rows.Err()
}

func (s *Store) TrashProject(ctx context.Context, userID int64, publicID string) (*Project, error) {
	p, err := s.GetProject(ctx, userID, publicID)
	if err != nil {
		return nil, err
	}
	if p.IsDefault {
		return nil, errors.New("默认作品不能移入回收站")
	}
	if p.Trashed {
		return nil, errors.New("作品已在回收站")
	}
	_, err = s.Pool.Exec(ctx, `UPDATE projects SET trashed_at=NOW(), updated_at=NOW() WHERE public_id=$1 AND user_id=$2`, publicID, userID)
	if err != nil {
		return nil, err
	}
	cur, _ := s.GetCurrentProjectID(ctx, userID)
	if cur == publicID {
		if def, err := s.EnsureDefaultProject(ctx, userID); err == nil {
			_ = s.SetCurrentProject(ctx, userID, def.ID)
		}
	}
	return s.GetProject(ctx, userID, publicID)
}

func (s *Store) RestoreProject(ctx context.Context, userID int64, publicID string) (*Project, error) {
	p, err := s.GetProject(ctx, userID, publicID)
	if err != nil {
		return nil, err
	}
	if !p.Trashed {
		return nil, errors.New("作品不在回收站")
	}
	_, err = s.Pool.Exec(ctx, `UPDATE projects SET trashed_at=NULL, updated_at=NOW() WHERE public_id=$1 AND user_id=$2`, publicID, userID)
	if err != nil {
		return nil, err
	}
	cur, _ := s.GetCurrentProjectID(ctx, userID)
	if cur == "" {
		_ = s.SetCurrentProject(ctx, userID, publicID)
	}
	return s.GetProject(ctx, userID, publicID)
}

func (s *Store) PurgeProject(ctx context.Context, userID int64, publicID string) error {
	p, err := s.GetProject(ctx, userID, publicID)
	if err != nil {
		return err
	}
	if !p.Trashed {
		return errors.New("只能从回收站彻底删除作品")
	}
	if p.IsDefault {
		return errors.New("默认作品不能彻底删除")
	}
	paths := collectStoragePaths(p.Folders)
	_, err = s.Pool.Exec(ctx, `DELETE FROM projects WHERE public_id=$1 AND user_id=$2`, publicID, userID)
	if err != nil {
		return err
	}
	for _, path := range paths {
		_ = os.Remove(path)
	}
	return nil
}

func (s *Store) SaveProjectFolders(ctx context.Context, userID int64, publicID string, folders map[string]FsFolder) error {
	raw, err := marshalFolders(folders)
	if err != nil {
		return err
	}
	tag, err := s.Pool.Exec(ctx, `
UPDATE projects SET folders_json=$1::jsonb, updated_at=NOW()
WHERE public_id=$2 AND user_id=$3 AND trashed_at IS NULL`, string(raw), publicID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) AddProjectFile(ctx context.Context, userID int64, publicID string, path []string, item FsItem) (*Project, *FsItem, error) {
	p, err := s.GetProject(ctx, userID, publicID)
	if err != nil {
		return nil, nil, err
	}
	if p.Trashed {
		return nil, nil, errors.New("回收站作品不可上传")
	}
	next, err := putFileInPath(p.Folders, path, item)
	if err != nil {
		return nil, nil, err
	}
	if err := s.SaveProjectFolders(ctx, userID, publicID, next); err != nil {
		return nil, nil, err
	}
	p, err = s.GetProject(ctx, userID, publicID)
	return p, &item, err
}

func (s *Store) DeleteProjectFile(ctx context.Context, userID int64, publicID, fileID string) (*FsItem, error) {
	p, err := s.GetProject(ctx, userID, publicID)
	if err != nil {
		return nil, err
	}
	if p.Trashed {
		return nil, errors.New("回收站作品不可删除文件")
	}
	next, removed, ok := removeFileFromFolders(p.Folders, fileID)
	if !ok {
		return nil, ErrNotFound
	}
	if err := s.SaveProjectFolders(ctx, userID, publicID, next); err != nil {
		return nil, err
	}
	if removed.StoragePath != "" {
		_ = os.Remove(removed.StoragePath)
	}
	return removed, nil
}

func (s *Store) GetProjectFile(ctx context.Context, userID int64, publicID, fileID string) (*FsItem, error) {
	p, err := s.GetProject(ctx, userID, publicID)
	if err != nil {
		return nil, err
	}
	item, _, ok := findFileInFolders(p.Folders, fileID)
	if !ok {
		return nil, ErrNotFound
	}
	return item, nil
}

func (s *Store) FindUserAsset(ctx context.Context, userID int64, assetID string) (*FsItem, string, error) {
	items, err := s.ListProjects(ctx, userID, false)
	if err != nil {
		return nil, "", err
	}
	for i := range items {
		item, _, ok := findFileInFolders(items[i].Folders, assetID)
		if ok {
			return item, items[i].ID, nil
		}
	}
	return nil, "", ErrNotFound
}

func (s *Store) ListProjectAssets(ctx context.Context, userID int64, projectID, typ string) ([]FsItem, error) {
	p, err := s.GetProject(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if p.Trashed {
		return nil, errors.New("作品在回收站")
	}
	var out []FsItem
	var walk func(map[string]FsFolder)
	walk = func(fs map[string]FsFolder) {
		for _, folder := range fs {
			for _, it := range folder.Items {
				if typ == "" || it.Type == typ {
					out = append(out, it)
				}
			}
			walk(folder.Folders)
		}
	}
	walk(p.Folders)
	return out, nil
}

func (s *Store) StorageUsage(ctx context.Context, userID int64) (usedBytes int64, fileCount, projectCount, trashCount int, err error) {
	if _, err = s.EnsureDefaultProject(ctx, userID); err != nil {
		return 0, 0, 0, 0, err
	}
	active, err := s.ListProjects(ctx, userID, false)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	trashed, err := s.ListProjects(ctx, userID, true)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	projectCount = len(active)
	trashCount = len(trashed)
	for _, p := range active {
		fileCount += p.FileCount
		usedBytes += p.SizeBytes
	}
	// 回收站仍计入配额（与原型一致）
	for _, p := range trashed {
		fileCount += p.FileCount
		usedBytes += p.SizeBytes
	}
	return usedBytes, fileCount, projectCount, trashCount, nil
}

func (s *Store) GetCurrentProjectID(ctx context.Context, userID int64) (string, error) {
	var id *string
	err := s.Pool.QueryRow(ctx, `SELECT current_project_public_id FROM user_drafts WHERE user_id=$1`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", nil
	}
	return *id, nil
}

func (s *Store) SetCurrentProject(ctx context.Context, userID int64, projectID string) error {
	p, err := s.GetProject(ctx, userID, projectID)
	if err != nil {
		return err
	}
	if p.Trashed {
		return errors.New("不能将回收站作品设为当前作品")
	}
	_, err = s.Pool.Exec(ctx, `
INSERT INTO user_drafts (user_id, draft_prompt, current_project_public_id)
VALUES ($1, '', $2)
ON CONFLICT (user_id) DO UPDATE SET current_project_public_id = EXCLUDED.current_project_public_id`, userID, projectID)
	return err
}

func (s *Store) ResolveCurrentProjectID(ctx context.Context, userID int64) (string, error) {
	id, err := s.GetCurrentProjectID(ctx, userID)
	if err != nil {
		return "", err
	}
	if id != "" {
		if p, err := s.GetProject(ctx, userID, id); err == nil && !p.Trashed {
			return p.ID, nil
		}
	}
	def, err := s.EnsureDefaultProject(ctx, userID)
	if err != nil {
		return "", err
	}
	_ = s.SetCurrentProject(ctx, userID, def.ID)
	return def.ID, nil
}

func AssetStoragePath(root, projectID, assetID, fileName string) string {
	ext := filepath.Ext(fileName)
	if ext == "" {
		ext = ".bin"
	}
	return filepath.Join(root, "projects", projectID, fmt.Sprintf("%s%s", assetID, ext))
}

func (s *Store) SaveTaskToProject(ctx context.Context, userID int64, taskID, projectID string) (*FsItem, []string, error) {
	var status, title, resultURL, resultOSS, thumb string
	var saved bool
	err := s.Pool.QueryRow(ctx, `
SELECT status, title, COALESCE(result_url,''), COALESCE(result_oss_url,''), COALESCE(thumb_url,''), saved_to_project
FROM generation_tasks WHERE public_id=$1 AND user_id=$2`, taskID, userID).Scan(
		&status, &title, &resultURL, &resultOSS, &thumb, &saved)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if statusCN(status) != "已完成" {
		return nil, nil, errors.New("仅已完成任务可保存到作品")
	}
	if saved {
		return nil, nil, errors.New("任务已保存到作品")
	}
	if projectID == "" {
		projectID, err = s.ResolveCurrentProjectID(ctx, userID)
		if err != nil {
			return nil, nil, err
		}
	}
	name := strings.TrimSpace(title)
	if name == "" {
		name = taskID
	}
	if !strings.HasSuffix(strings.ToLower(name), ".mp4") {
		name += ".mp4"
	}
	remote := resultOSS
	if remote == "" {
		remote = resultURL
	}
	if remote == "" {
		remote = thumb
	}
	item := FsItem{
		ID:         NewAssetID(),
		Name:       name,
		Type:       "video",
		Size:       "—",
		Time:       time.Now().Format("2006-01-02 15:04"),
		Meta:       "成片",
		SourceTask: taskID,
		RemoteURL:  remote,
	}
	path := []string{"视频", "正片镜头"}
	if _, _, err := s.AddProjectFile(ctx, userID, projectID, path, item); err != nil {
		return nil, nil, err
	}
	_, err = s.Pool.Exec(ctx, `
UPDATE generation_tasks
SET saved_to_project=TRUE, project_public_id=$1, updated_at=NOW()
WHERE public_id=$2 AND user_id=$3`, projectID, taskID, userID)
	if err != nil {
		return nil, nil, err
	}
	return &item, path, nil
}
