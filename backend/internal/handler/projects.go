package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/middleware"
	"github.com/sbu2video/backend/internal/response"
	"github.com/sbu2video/backend/internal/store"
)

type ProjectHandler struct {
	Store     *store.Store
	UploadDir string
}

func projectDTO(p *store.Project) gin.H {
	trashedAt := ""
	if p.TrashedAt != nil {
		trashedAt = p.TrashedAt.Format(time.RFC3339)
	}
	return gin.H{
		"id":        p.ID,
		"name":      p.Name,
		"desc":      p.Description,
		"createdAt": p.CreatedAt.Format("2006-01-02"),
		"trashed":   p.Trashed,
		"trashedAt": trashedAt,
		"isDefault": p.IsDefault,
		"fileCount": p.FileCount,
		"sizeBytes": p.SizeBytes,
	}
}

func folderMapDTO(folders map[string]store.FsFolder, projectID string) map[string]any {
	out := make(map[string]any, len(folders))
	for name, folder := range folders {
		out[name] = folderDTO(folder, projectID)
	}
	return out
}

func folderDTO(folder store.FsFolder, projectID string) gin.H {
	items := make([]gin.H, 0, len(folder.Items))
	for _, it := range folder.Items {
		items = append(items, fileDTO(it, projectID))
	}
	return gin.H{
		"isSystem": folder.IsSystem,
		"desc":     folder.Desc,
		"folders":  folderMapDTO(folder.Folders, projectID),
		"items":    items,
	}
}

func fileDTO(it store.FsItem, projectID string) gin.H {
	h := gin.H{
		"id":   it.ID,
		"name": it.Name,
		"type": it.Type,
		"size": it.Size,
		"time": it.Time,
		"url":  fmt.Sprintf("/api/v1/client/projects/%s/files/%s", projectID, it.ID),
	}
	if it.Meta != "" {
		h["meta"] = it.Meta
	}
	if it.SourceTask != "" {
		h["sourceTask"] = it.SourceTask
	}
	if it.RemoteURL != "" {
		h["remoteUrl"] = it.RemoteURL
	}
	return h
}

func (h *ProjectHandler) StorageUsage(c *gin.Context) {
	user := middleware.CurrentUser(c)
	usedBytes, fileCount, projectCount, trashCount, err := h.Store.StorageUsage(c.Request.Context(), user.ID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	quotaStr, _ := h.Store.GetSetting(c.Request.Context(), "storage_quota_gb")
	quota := 5.0
	if v, err := strconv.ParseFloat(quotaStr, 64); err == nil && v > 0 {
		quota = v
	}
	usedGb := float64(usedBytes) / (1024 * 1024 * 1024)
	response.OK(c, gin.H{
		"usedGb":       round3(usedGb),
		"quotaGb":      quota,
		"fileCount":    fileCount,
		"projectCount": projectCount,
		"trashCount":   trashCount,
		"usedBytes":    usedBytes,
	})
}

func round3(v float64) float64 {
	return float64(int(v*1000+0.5)) / 1000
}

func (h *ProjectHandler) ListProjects(c *gin.Context) {
	user := middleware.CurrentUser(c)
	trashed := c.Query("trashed") == "true"
	items, err := h.Store.ListProjects(c.Request.Context(), user.ID, trashed)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	list := make([]gin.H, 0, len(items))
	for i := range items {
		list = append(list, projectDTO(&items[i]))
	}
	cur, _ := h.Store.ResolveCurrentProjectID(c.Request.Context(), user.ID)
	response.OK(c, gin.H{"items": list, "currentProjectId": cur})
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	user := middleware.CurrentUser(c)
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	p, err := h.Store.CreateProject(c.Request.Context(), user.ID, req.Name, "", false)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	_ = h.Store.SetCurrentProject(c.Request.Context(), user.ID, p.ID)
	response.Created(c, gin.H{"project": projectDTO(p)})
}

func (h *ProjectHandler) GetTree(c *gin.Context) {
	user := middleware.CurrentUser(c)
	p, err := h.Store.GetProject(c.Request.Context(), user.ID, c.Param("id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "NOT_FOUND", "作品不存在")
			return
		}
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	dto := projectDTO(p)
	dto["folders"] = folderMapDTO(p.Folders, p.ID)
	response.OK(c, dto)
}

func (h *ProjectHandler) GetNodes(c *gin.Context) {
	user := middleware.CurrentUser(c)
	p, err := h.Store.GetProject(c.Request.Context(), user.ID, c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusNotFound, "NOT_FOUND", "作品不存在")
		return
	}
	path := splitPath(c.Query("path"))
	children, folder, err := storeResolve(p.Folders, path)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	folders := make([]gin.H, 0)
	items := make([]gin.H, 0)
	if folder == nil {
		for name, f := range children {
			fc, _ := store.CountFolderPublic(f)
			folders = append(folders, gin.H{"name": name, "isSystem": f.IsSystem, "desc": f.Desc, "itemCount": fc})
		}
	} else {
		for name, f := range folder.Folders {
			fc, _ := store.CountFolderPublic(f)
			folders = append(folders, gin.H{"name": name, "isSystem": f.IsSystem, "desc": f.Desc, "itemCount": fc})
		}
		for _, it := range folder.Items {
			items = append(items, fileDTO(it, p.ID))
		}
	}
	response.OK(c, gin.H{"path": path, "folders": folders, "items": items})
}

func storeResolve(folders map[string]store.FsFolder, path []string) (map[string]store.FsFolder, *store.FsFolder, error) {
	if len(path) == 0 {
		return folders, nil, nil
	}
	cur := folders
	var folder *store.FsFolder
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

func splitPath(raw string) []string {
	raw = strings.Trim(strings.TrimSpace(raw), "/")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (h *ProjectHandler) Upload(c *gin.Context) {
	user := middleware.CurrentUser(c)
	projectID := c.Param("id")
	path := splitPath(c.Query("path"))
	if len(path) == 0 {
		path = []string{"资产", "素材"}
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "请选择文件")
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxUploadBytes {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "文件大小需在 12MB 以内")
		return
	}
	mime := fileHeader.Header.Get("Content-Type")
	typ := "image"
	if mime == "" || !allowedImageMIME[mime] {
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		switch ext {
		case ".jpg", ".jpeg":
			mime = "image/jpeg"
		case ".png":
			mime = "image/png"
		case ".webp":
			mime = "image/webp"
		case ".gif":
			mime = "image/gif"
		case ".mp4":
			mime = "video/mp4"
			typ = "video"
		case ".mp3", ".wav":
			mime = "audio/" + strings.TrimPrefix(ext, ".")
			typ = "audio"
		default:
			response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "暂不支持该文件类型")
			return
		}
	}
	if typ == "image" && !allowedImageMIME[mime] {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "仅支持 JPEG/PNG/WebP/GIF")
		return
	}

	assetID := store.NewAssetID()
	dest := store.AssetStoragePath(h.UploadDir, projectID, assetID, fileHeader.Filename)
	if err := store.EnsureUploadDir(filepath.Dir(dest)); err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", "无法创建上传目录")
		return
	}
	src, err := fileHeader.Open()
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "无法读取上传文件")
		return
	}
	defer src.Close()
	out, err := os.Create(dest)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", "无法保存上传文件")
		return
	}
	written, copyErr := io.Copy(out, io.LimitReader(src, maxUploadBytes+1))
	_ = out.Close()
	if copyErr != nil || written > maxUploadBytes {
		_ = os.Remove(dest)
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "文件大小需在 12MB 以内")
		return
	}

	item := store.FsItem{
		ID:          assetID,
		Name:        fileHeader.Filename,
		Type:        typ,
		Size:        store.FormatBytesPublic(written),
		Time:        time.Now().Format("2006-01-02 15:04"),
		SizeBytes:   written,
		StoragePath: dest,
		MimeType:    mime,
	}
	_, saved, err := h.Store.AddProjectFile(c.Request.Context(), user.ID, projectID, path, item)
	if err != nil {
		_ = os.Remove(dest)
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Created(c, gin.H{"item": fileDTO(*saved, projectID), "path": path})
}

func (h *ProjectHandler) DeleteFile(c *gin.Context) {
	user := middleware.CurrentUser(c)
	_, err := h.Store.DeleteProjectFile(c.Request.Context(), user.ID, c.Param("id"), c.Param("fileId"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "NOT_FOUND", "文件不存在")
			return
		}
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *ProjectHandler) GetFile(c *gin.Context) {
	user := middleware.CurrentUser(c)
	item, err := h.Store.GetProjectFile(c.Request.Context(), user.ID, c.Param("id"), c.Param("fileId"))
	if err != nil {
		response.Fail(c, http.StatusNotFound, "NOT_FOUND", "文件不存在")
		return
	}
	if item.StoragePath != "" {
		c.File(item.StoragePath)
		return
	}
	if item.RemoteURL != "" {
		c.Redirect(http.StatusFound, item.RemoteURL)
		return
	}
	response.Fail(c, http.StatusNotFound, "NOT_FOUND", "文件内容不可用")
}

func (h *ProjectHandler) ListAssets(c *gin.Context) {
	user := middleware.CurrentUser(c)
	projectID := c.Param("id")
	if projectID == "" || projectID == "current" {
		var err error
		projectID, err = h.Store.ResolveCurrentProjectID(c.Request.Context(), user.ID)
		if err != nil {
			response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
			return
		}
	}
	typ := c.Query("type")
	switch typ {
	case "image", "图片":
		typ = "image"
	case "video", "视频":
		typ = "video"
	case "audio", "音频":
		typ = "audio"
	default:
		typ = ""
	}
	items, err := h.Store.ListProjectAssets(c.Request.Context(), user.ID, projectID, typ)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	list := make([]gin.H, 0, len(items))
	for _, it := range items {
		list = append(list, fileDTO(it, projectID))
	}
	response.OK(c, gin.H{"items": list, "projectId": projectID})
}

func (h *ProjectHandler) Trash(c *gin.Context) {
	user := middleware.CurrentUser(c)
	p, err := h.Store.TrashProject(c.Request.Context(), user.ID, c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"project": projectDTO(p)})
}

func (h *ProjectHandler) Restore(c *gin.Context) {
	user := middleware.CurrentUser(c)
	p, err := h.Store.RestoreProject(c.Request.Context(), user.ID, c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"project": projectDTO(p)})
}

func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if c.Query("permanent") != "true" {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "请使用 permanent=true 彻底删除")
		return
	}
	if err := h.Store.PurgeProject(c.Request.Context(), user.ID, c.Param("id")); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *ProjectHandler) SetCurrentProject(c *gin.Context) {
	user := middleware.CurrentUser(c)
	var req struct {
		ProjectID string `json:"projectId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.ProjectID) == "" {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "projectId required")
		return
	}
	if err := h.Store.SetCurrentProject(c.Request.Context(), user.ID, req.ProjectID); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"currentProjectId": req.ProjectID})
}
