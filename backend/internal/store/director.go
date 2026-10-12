package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const MaxDirectorShots = 100
const MaxDirectorCandidates = 20
const MaxDirectorBodyBytes = 1 << 20

var (
	ErrDirectorConflict       = errors.New("镜头板已更新，请重新加载后再保存")
	ErrDirectorShotMissing    = errors.New("镜头已被删除，请返回导演台选择镜头")
	ErrDirectorInvalid        = errors.New("镜头板内容不合法")
	ErrDirectorReference      = errors.New("参考素材不存在或不可用，请重新选择素材")
	ErrDirectorTask           = errors.New("候选任务不存在或不可用")
	ErrDirectorSelection      = errors.New("只有已交付的候选成片可以选用")
	directorIDPattern         = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	directorResolutionPattern = regexp.MustCompile(`^([1-9][0-9]{2,3}P|[1-9][0-9]?K)$`)
	directorAspectPattern     = regexp.MustCompile(`^[1-9][0-9]{0,2}:[1-9][0-9]{0,2}$`)
)

type DirectorShot struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Prompt           string     `json:"prompt"`
	ModeID           string     `json:"modeId"`
	CarrierID        string     `json:"carrierId"`
	ModelID          string     `json:"modelId"`
	DurationSec      int        `json:"durationSec"`
	Resolution       string     `json:"resolution"`
	AspectRatio      string     `json:"aspectRatio"`
	GenerateAudio    bool       `json:"generateAudio"`
	SourceMeta       SourceMeta `json:"sourceMeta"`
	CandidateTaskIDs []string   `json:"candidateTaskIds"`
	SelectedTaskID   string     `json:"selectedTaskId"`
}

type DirectorBoard struct {
	ProjectID string         `json:"projectId"`
	Revision  int64          `json:"revision"`
	Shots     []DirectorShot `json:"shots"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// DirectorContext binds a normal paid task to a saved shot in the very same
// transaction. It never replaces the existing task idempotency contract.
type DirectorContext struct {
	ProjectID        string `json:"projectId"`
	ShotID           string `json:"shotId"`
	ExpectedRevision int64  `json:"expectedRevision"`
}

func directorInvalid(reason string) error { return fmt.Errorf("%w：%s", ErrDirectorInvalid, reason) }

// ValidateDirectorShots accepts incomplete plans without consulting current
// supply/capabilities. Generating still uses the authoritative creation checks.
func ValidateDirectorShots(shots []DirectorShot) error {
	if len(shots) > MaxDirectorShots {
		return directorInvalid("最多保存 100 个镜头")
	}
	seen := make(map[string]bool, len(shots))
	for i := range shots {
		shot := &shots[i]
		if !directorIDPattern.MatchString(shot.ID) || seen[shot.ID] {
			return directorInvalid("镜头 ID 无效或重复")
		}
		seen[shot.ID] = true
		if utf8.RuneCountInString(shot.Title) > 128 || utf8.RuneCountInString(shot.Prompt) > 10000 {
			return directorInvalid("镜头名称或描述过长")
		}
		for _, id := range []string{shot.ModeID, shot.CarrierID, shot.ModelID} {
			if id != "" && !directorIDPattern.MatchString(id) {
				return directorInvalid("生成参数 ID 格式不正确")
			}
		}
		if shot.DurationSec < 0 || shot.DurationSec > 3600 {
			return directorInvalid("镜头时长格式不正确")
		}
		if shot.Resolution != "" && !directorResolutionPattern.MatchString(shot.Resolution) {
			return directorInvalid("分辨率格式不正确")
		}
		if shot.AspectRatio != "" && !directorAspectPattern.MatchString(shot.AspectRatio) {
			return directorInvalid("画幅比例格式不正确")
		}
		if shot.SourceMeta.Audio != nil {
			return directorInvalid("导演台暂不支持参考音频")
		}
		refs := directorReferences(shot.SourceMeta)
		if len(refs) > 20 {
			return directorInvalid("每个镜头最多保存 20 个参考素材")
		}
		for _, ref := range refs {
			if (ref.Type != "asset" && ref.Type != "upload") || !directorIDPattern.MatchString(ref.ID) {
				return directorInvalid("参考素材必须使用素材 ID")
			}
		}
		if len(shot.CandidateTaskIDs) > MaxDirectorCandidates {
			return directorInvalid("每个镜头最多保存 20 个候选任务")
		}
		candidates := make(map[string]bool, len(shot.CandidateTaskIDs))
		for _, id := range shot.CandidateTaskIDs {
			if !directorIDPattern.MatchString(id) || candidates[id] {
				return directorInvalid("候选任务 ID 无效或重复")
			}
			candidates[id] = true
		}
		if shot.SelectedTaskID != "" && !candidates[shot.SelectedTaskID] {
			return directorInvalid("选用成片必须属于当前镜头候选")
		}
		if shot.CandidateTaskIDs == nil {
			shot.CandidateTaskIDs = []string{}
		}
	}
	return nil
}

func directorReferences(meta SourceMeta) []SourceRefItem {
	refs := make([]SourceRefItem, 0, meta.ImageCount())
	if meta.First != nil {
		refs = append(refs, *meta.First)
	}
	if meta.Last != nil {
		refs = append(refs, *meta.Last)
	}
	refs = append(refs, meta.Extras...)
	return append(refs, meta.Refs...)
}

func readDirectorBoard(ctx context.Context, db routeQuerier, userID int64, projectID string, lock bool) (*DirectorBoard, error) {
	query := `SELECT public_id,director_revision,director_board,updated_at FROM projects WHERE public_id=$1 AND user_id=$2 AND trashed_at IS NULL`
	if lock {
		query += ` FOR UPDATE`
	}
	board := &DirectorBoard{Shots: []DirectorShot{}}
	var raw []byte
	err := db.QueryRow(ctx, query, projectID, userID).Scan(&board.ProjectID, &board.Revision, &raw, &board.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, &board.Shots); err != nil {
		return nil, err
	}
	if board.Shots == nil {
		board.Shots = []DirectorShot{}
	}
	return board, nil
}

func (s *Store) GetDirectorBoard(ctx context.Context, userID int64, projectID string) (*DirectorBoard, error) {
	return readDirectorBoard(ctx, s.Pool, userID, projectID, false)
}

func writeDirectorBoard(ctx context.Context, tx pgx.Tx, userID int64, board *DirectorBoard) error {
	if board.Shots == nil {
		board.Shots = []DirectorShot{}
	}
	raw, err := json.Marshal(board.Shots)
	if err != nil {
		return err
	}
	if len(raw) > MaxDirectorBodyBytes {
		return directorInvalid("镜头板内容过大")
	}
	return tx.QueryRow(ctx, `UPDATE projects SET director_board=$1::jsonb,director_revision=director_revision+1,updated_at=NOW()
WHERE public_id=$2 AND user_id=$3 AND trashed_at IS NULL RETURNING director_revision,updated_at`, string(raw), board.ProjectID, userID).Scan(&board.Revision, &board.UpdatedAt)
}

// Resolve only owned image IDs and task IDs. Nothing from provider snapshots,
// object URLs, wallet data or credentials is stored in the board response.
func validateDirectorOwnership(ctx context.Context, tx pgx.Tx, userID int64, shots []DirectorShot) error {
	refs := map[SourceRefItem]bool{}
	taskIDs := map[string]bool{}
	for _, shot := range shots {
		for _, ref := range directorReferences(shot.SourceMeta) {
			refs[ref] = true
		}
		for _, id := range shot.CandidateTaskIDs {
			taskIDs[id] = true
		}
	}
	if err := validateDirectorReferences(ctx, tx, userID, refs); err != nil {
		return err
	}
	ids := make([]string, 0, len(taskIDs))
	for id := range taskIDs {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := tx.Query(ctx, `SELECT public_id,status='completed' AND COALESCE(result_oss_url,'')<>'' FROM generation_tasks WHERE user_id=$1 AND public_id=ANY($2)`, userID, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	ready := map[string]bool{}
	for rows.Next() {
		var id string
		var delivered bool
		if err := rows.Scan(&id, &delivered); err != nil {
			return err
		}
		ready[id] = delivered
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(ready) != len(taskIDs) {
		return ErrDirectorTask
	}
	for _, shot := range shots {
		if shot.SelectedTaskID != "" && !ready[shot.SelectedTaskID] {
			return ErrDirectorSelection
		}
	}
	return nil
}

// A board can contain many references. Check uploads in one query and scan
// owned project folders once, rather than repeating a project scan per shot.
func validateDirectorReferences(ctx context.Context, tx pgx.Tx, userID int64, refs map[SourceRefItem]bool) error {
	uploadIDs := []string{}
	assetIDs := map[string]bool{}
	for ref := range refs {
		if ref.Type == "upload" {
			uploadIDs = append(uploadIDs, ref.ID)
		} else {
			assetIDs[ref.ID] = true
		}
	}
	if len(uploadIDs) > 0 {
		rows, err := tx.Query(ctx, `SELECT public_id,mime_type FROM uploads WHERE user_id=$1 AND public_id=ANY($2)`, userID, uploadIDs)
		if err != nil {
			return err
		}
		found := 0
		for rows.Next() {
			var id, mime string
			if err := rows.Scan(&id, &mime); err != nil {
				rows.Close()
				return err
			}
			switch strings.ToLower(strings.TrimSpace(mime)) {
			case "image/jpeg", "image/png", "image/webp", "image/gif":
				found++
			default:
				rows.Close()
				return ErrDirectorReference
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if found != len(uploadIDs) {
			return ErrDirectorReference
		}
	}
	if len(assetIDs) > 0 {
		rows, err := tx.Query(ctx, `SELECT folders_json FROM projects WHERE user_id=$1 AND trashed_at IS NULL`, userID)
		if err != nil {
			return err
		}
		defer rows.Close()
		var visit func(map[string]FsFolder)
		visit = func(folders map[string]FsFolder) {
			for _, folder := range folders {
				for _, item := range folder.Items {
					if item.Type == "image" {
						delete(assetIDs, item.ID)
					}
				}
				visit(folder.Folders)
			}
		}
		for rows.Next() {
			var raw []byte
			if err := rows.Scan(&raw); err != nil {
				return err
			}
			folders, err := unmarshalFolders(raw)
			if err != nil {
				return err
			}
			visit(folders)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if len(assetIDs) > 0 {
			return ErrDirectorReference
		}
	}
	return nil
}

func (s *Store) SaveDirectorBoard(ctx context.Context, userID int64, projectID string, revision int64, shots []DirectorShot) (*DirectorBoard, error) {
	if revision < 0 {
		return nil, directorInvalid("修订号不合法")
	}
	if err := ValidateDirectorShots(shots); err != nil {
		return nil, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	board, err := readDirectorBoard(ctx, tx, userID, projectID, true)
	if err != nil {
		return nil, err
	}
	if board.Revision != revision {
		return nil, ErrDirectorConflict
	}
	if err = validateDirectorOwnership(ctx, tx, userID, shots); err != nil {
		return nil, err
	}
	board.Shots = shots
	if err = writeDirectorBoard(ctx, tx, userID, board); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return board, nil
}

func directorShotIndex(board *DirectorBoard, id string) int {
	for i := range board.Shots {
		if board.Shots[i].ID == id {
			return i
		}
	}
	return -1
}

func (s *Store) AddDirectorCandidate(ctx context.Context, userID int64, projectID, shotID, taskID string) (*DirectorBoard, error) {
	if !directorIDPattern.MatchString(taskID) {
		return nil, directorInvalid("候选任务 ID 格式不正确")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	board, err := readDirectorBoard(ctx, tx, userID, projectID, true)
	if err != nil {
		return nil, err
	}
	i := directorShotIndex(board, shotID)
	if i < 0 {
		return nil, ErrDirectorShotMissing
	}
	for _, id := range board.Shots[i].CandidateTaskIDs {
		if id == taskID {
			return board, nil
		}
	}
	if len(board.Shots[i].CandidateTaskIDs) >= MaxDirectorCandidates {
		return nil, directorInvalid("每个镜头最多保存 20 个候选任务")
	}
	// Validate the new link only; unrelated old material may have been removed
	// and must not prevent attaching a task that has already been paid for.
	if err = validateDirectorOwnership(ctx, tx, userID, []DirectorShot{{CandidateTaskIDs: []string{taskID}}}); err != nil {
		return nil, err
	}
	board.Shots[i].CandidateTaskIDs = append(board.Shots[i].CandidateTaskIDs, taskID)
	if err = writeDirectorBoard(ctx, tx, userID, board); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return board, nil
}

func lockDirectorTask(ctx context.Context, tx pgx.Tx, userID int64, origin *DirectorContext) (*DirectorBoard, int, error) {
	if !directorIDPattern.MatchString(origin.ProjectID) || !directorIDPattern.MatchString(origin.ShotID) || origin.ExpectedRevision < 0 {
		return nil, 0, directorInvalid("导演台提交上下文不合法")
	}
	board, err := readDirectorBoard(ctx, tx, userID, origin.ProjectID, true)
	if err != nil {
		return nil, 0, err
	}
	index := directorShotIndex(board, origin.ShotID)
	if index < 0 {
		return nil, 0, ErrDirectorShotMissing
	}
	if board.Revision != origin.ExpectedRevision {
		return nil, 0, ErrDirectorConflict
	}
	if len(board.Shots[index].CandidateTaskIDs) >= MaxDirectorCandidates {
		return nil, 0, directorInvalid("每个镜头最多保存 20 个候选任务")
	}
	return board, index, nil
}

func bindDirectorTask(ctx context.Context, tx pgx.Tx, userID int64, board *DirectorBoard, index int, taskID string, in DirectTaskInput, route *ResolvedRoute) error {
	shot := &board.Shots[index]
	shot.Prompt, shot.ModeID, shot.CarrierID, shot.ModelID = in.Prompt, route.Mode.ID, route.Model.CarrierID, route.Model.ID
	shot.DurationSec, shot.Resolution, shot.AspectRatio, shot.GenerateAudio = in.DurationSeconds, in.Resolution, in.AspectRatio, in.GenerateAudio
	shot.SourceMeta = in.SourceMeta
	shot.CandidateTaskIDs = append(shot.CandidateTaskIDs, taskID)
	if err := ValidateDirectorShots([]DirectorShot{*shot}); err != nil {
		return err
	}
	return writeDirectorBoard(ctx, tx, userID, board)
}
