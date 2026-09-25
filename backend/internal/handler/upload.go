package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/middleware"
	"github.com/sbu2video/backend/internal/response"
	"github.com/sbu2video/backend/internal/store"
)

const maxUploadBytes = 12 << 20               // 12 MiB
const maxReferenceAudioUploadBytes = 50 << 20 // 50 MiB

var allowedImageMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

var allowedReferenceAudioMIME = map[string]bool{
	"audio/mpeg":  true,
	"audio/wav":   true,
	"audio/x-wav": true,
	"audio/flac":  true,
	"audio/mp4":   true,
	"video/mp4":   true,
}

type UploadHandler struct {
	Store     *store.Store
	UploadDir string
}

func (h *UploadHandler) Create(c *gin.Context) {
	h.create(c, uploadPolicy{
		label: "图片", maxBytes: maxUploadBytes, allowedMIME: allowedImageMIME,
		extensions: map[string]string{".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".webp": "image/webp", ".gif": "image/gif"},
		formats:    "JPEG/PNG/WebP/GIF",
	})
}

// CreateReferenceAudio stores an optional creator-provided audio source. It is
// separate from image uploads so a video/image task cannot silently accept an
// arbitrary file type.
func (h *UploadHandler) CreateReferenceAudio(c *gin.Context) {
	h.create(c, uploadPolicy{
		label: "参考音频", maxBytes: maxReferenceAudioUploadBytes, allowedMIME: allowedReferenceAudioMIME,
		extensions: map[string]string{".mp3": "audio/mpeg", ".wav": "audio/wav", ".flac": "audio/flac", ".m4a": "audio/mp4", ".mp4": "audio/mp4"},
		formats:    "MP3/WAV/FLAC/M4A/MP4",
	})
}

type uploadPolicy struct {
	label       string
	maxBytes    int64
	allowedMIME map[string]bool
	extensions  map[string]string
	formats     string
}

func (h *UploadHandler) create(c *gin.Context, policy uploadPolicy) {
	user := middleware.CurrentUser(c)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "请选择"+policy.label+"文件")
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > policy.maxBytes {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", fmt.Sprintf("%s大小需在 %dMB 以内", policy.label, policy.maxBytes>>20))
		return
	}

	mime := fileHeader.Header.Get("Content-Type")
	if mime == "" || !policy.allowedMIME[mime] {
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		mime = policy.extensions[ext]
	}
	if !policy.allowedMIME[mime] {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "仅支持 "+policy.formats)
		return
	}

	publicID := store.NewUploadID()
	dest := store.UploadPath(h.UploadDir, publicID, fileHeader.Filename)
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
	written, copyErr := io.Copy(out, io.LimitReader(src, policy.maxBytes+1))
	_ = out.Close()
	if copyErr != nil || written > policy.maxBytes {
		_ = os.Remove(dest)
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", fmt.Sprintf("%s大小需在 %dMB 以内", policy.label, policy.maxBytes>>20))
		return
	}

	upload, err := h.Store.CreateUpload(c.Request.Context(), user.ID, publicID, fileHeader.Filename, mime, dest, written)
	if err != nil {
		_ = os.Remove(dest)
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", "保存上传记录失败")
		return
	}

	response.Created(c, gin.H{
		"id":        upload.ID,
		"fileName":  upload.FileName,
		"mimeType":  upload.MimeType,
		"sizeBytes": upload.SizeBytes,
		"url":       fmt.Sprintf("/api/v1/client/uploads/%s", upload.ID),
	})
}

func (h *UploadHandler) Get(c *gin.Context) {
	user := middleware.CurrentUser(c)
	id := c.Param("id")
	upload, err := h.Store.GetUpload(c.Request.Context(), user.ID, id)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "NOT_FOUND", "上传不存在")
		return
	}
	c.File(upload.StoragePath)
}
