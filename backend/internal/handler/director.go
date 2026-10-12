package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/middleware"
	"github.com/sbu2video/backend/internal/response"
	"github.com/sbu2video/backend/internal/store"
)

func directorError(err error) (int, string, string) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND", "作品不存在或已移入回收站"
	case errors.Is(err, store.ErrDirectorConflict):
		return http.StatusConflict, "DIRECTOR_BOARD_CONFLICT", store.ErrDirectorConflict.Error()
	case errors.Is(err, store.ErrDirectorShotMissing):
		return http.StatusConflict, "DIRECTOR_SHOT_MISSING", store.ErrDirectorShotMissing.Error()
	case errors.Is(err, store.ErrDirectorInvalid):
		return http.StatusBadRequest, "DIRECTOR_BOARD_INVALID", err.Error()
	case errors.Is(err, store.ErrDirectorReference):
		return http.StatusBadRequest, "DIRECTOR_REFERENCE_INVALID", store.ErrDirectorReference.Error()
	case errors.Is(err, store.ErrDirectorTask):
		return http.StatusBadRequest, "DIRECTOR_TASK_INVALID", store.ErrDirectorTask.Error()
	case errors.Is(err, store.ErrDirectorSelection):
		return http.StatusBadRequest, "DIRECTOR_SELECTION_NOT_READY", store.ErrDirectorSelection.Error()
	default:
		return http.StatusInternalServerError, "INTERNAL", "镜头板暂时不可用，请稍后重试"
	}
}

func respondDirectorError(c *gin.Context, err error) {
	status, code, message := directorError(err)
	response.Fail(c, status, code, message)
}

func decodeDirectorRequest(c *gin.Context, target any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, store.MaxDirectorBodyBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(target)
	if err == nil {
		var extra any
		err = decoder.Decode(&extra)
		if errors.Is(err, io.EOF) {
			return true
		}
	}
	var oversized *http.MaxBytesError
	if errors.As(err, &oversized) {
		response.Fail(c, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "镜头板内容不能超过 1MB")
	} else {
		response.Fail(c, http.StatusBadRequest, "DIRECTOR_BOARD_INVALID", "镜头板请求格式不正确")
	}
	return false
}

func (h *ProjectHandler) GetDirectorBoard(c *gin.Context) {
	board, err := h.Store.GetDirectorBoard(c.Request.Context(), middleware.CurrentUser(c).ID, c.Param("id"))
	if err != nil {
		respondDirectorError(c, err)
		return
	}
	response.OK(c, board)
}

func (h *ProjectHandler) SaveDirectorBoard(c *gin.Context) {
	var req struct {
		ExpectedRevision *int64                `json:"expectedRevision"`
		Shots            *[]store.DirectorShot `json:"shots"`
	}
	if !decodeDirectorRequest(c, &req) {
		return
	}
	if req.ExpectedRevision == nil || req.Shots == nil {
		response.Fail(c, http.StatusBadRequest, "DIRECTOR_BOARD_INVALID", "请提交修订号和镜头列表")
		return
	}
	board, err := h.Store.SaveDirectorBoard(c.Request.Context(), middleware.CurrentUser(c).ID, c.Param("id"), *req.ExpectedRevision, *req.Shots)
	if err != nil {
		respondDirectorError(c, err)
		return
	}
	response.OK(c, board)
}

func (h *ProjectHandler) AddDirectorCandidate(c *gin.Context) {
	var req struct {
		TaskID string `json:"taskId"`
	}
	if !decodeDirectorRequest(c, &req) {
		return
	}
	board, err := h.Store.AddDirectorCandidate(c.Request.Context(), middleware.CurrentUser(c).ID, c.Param("id"), c.Param("shotId"), req.TaskID)
	if err != nil {
		respondDirectorError(c, err)
		return
	}
	response.OK(c, board)
}
