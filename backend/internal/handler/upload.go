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

const maxUploadBytes = 12 << 20 // 12 MiB

var allowedImageMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

type UploadHandler struct {
	Store     *store.Store
	UploadDir string
}

func (h *UploadHandler) Create(c *gin.Context) {
	user := middleware.CurrentUser(c)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "请选择图片文件")
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxUploadBytes {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "图片大小需在 12MB 以内")
		return
	}

	mime := fileHeader.Header.Get("Content-Type")
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
		default:
			response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "仅支持 JPEG/PNG/WebP/GIF")
			return
		}
	}
	if !allowedImageMIME[mime] {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "仅支持 JPEG/PNG/WebP/GIF")
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
	written, copyErr := io.Copy(out, io.LimitReader(src, maxUploadBytes+1))
	_ = out.Close()
	if copyErr != nil || written > maxUploadBytes {
		_ = os.Remove(dest)
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "图片大小需在 12MB 以内")
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
