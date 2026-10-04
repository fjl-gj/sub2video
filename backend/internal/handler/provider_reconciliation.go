package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/middleware"
	"github.com/sbu2video/backend/internal/provider/autodl"
	"github.com/sbu2video/backend/internal/response"
	"github.com/sbu2video/backend/internal/store"
)

// ReconcileTask is administrator-only and never submits new provider work.
func (h *AdminHandler) ReconcileTask(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil || user.Role != "admin" {
		response.Fail(c, http.StatusForbidden, "FORBIDDEN", "仅管理员可核对任务")
		return
	}
	var in store.TaskReconciliationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, "INVALID_RECONCILIATION", "核对参数无效")
		return
	}
	if err := in.Validate(); err != nil {
		response.Fail(c, http.StatusBadRequest, "INVALID_RECONCILIATION", err.Error())
		return
	}
	// Validate the existence of an accepted upstream task with the supplier
	// captured on this task. The operator must still confirm its ownership using
	// provider-side evidence; an existing random ID is not sufficient evidence.
	if in.Action == "resume" {
		detail, err := h.Store.GetAdminTaskDetail(c.Request.Context(), c.Param("id"))
		if err != nil {
			response.Fail(c, http.StatusNotFound, "NOT_FOUND", "任务不存在")
			return
		}
		// An already reconciled replay is handled by the store without making a
		// supplier request, even if credentials/supply later become unavailable.
		if detail["upstreamStatus"] == "SUBMIT_UNKNOWN" {
			code, _ := detail["supplierCode"].(string)
			supplier, err := h.Store.GetSupplierByCode(c.Request.Context(), code)
			if err != nil {
				response.Fail(c, http.StatusConflict, "SUPPLIER_UNAVAILABLE", "无法读取锁定供应商，任务保持待核对")
				return
			}
			token, err := h.Store.ResolveSupplierToken(c.Request.Context(), h.Cfg.SecretKey, code, h.Cfg.AutoDLToken)
			if err != nil || token == "" {
				response.Fail(c, http.StatusConflict, "SUPPLIER_UNAVAILABLE", "锁定供应商凭据不可用，任务保持待核对")
				return
			}
			var snapshot struct {
				AdapterCode string `json:"adapterCode"`
			}
			raw, _ := json.Marshal(detail["executionSnapshot"])
			_ = json.Unmarshal(raw, &snapshot)
			if detail["executionType"] != "workflow" || snapshot.AdapterCode != "autodl_h3_v1" {
				response.Fail(c, http.StatusConflict, "ADAPTER_UNAVAILABLE", "当前执行适配器不支持核对，任务保持待核对")
				return
			}
			baseURL := supplier.BaseURL
			if baseURL == "" {
				baseURL = h.Cfg.AutoDLBaseURL
			}
			ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
			defer cancel()
			res, err := autodl.New(baseURL, token, supplier.AuthHeader).Query(ctx, in.UpstreamTaskID)
			if err != nil || res == nil || res.TaskID != in.UpstreamTaskID || !knownReconciliationStatus(res.Status) {
				response.Fail(c, http.StatusConflict, "UPSTREAM_UNVERIFIED", "上游任务无法核实，任务保持待核对，请核对任务 ID 和供应商记录")
				return
			}
		}
	}
	out, err := h.Store.ReconcileTaskSubmission(c.Request.Context(), user.ID, c.Param("id"), in)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			response.Fail(c, http.StatusNotFound, "NOT_FOUND", "任务不存在")
		case errors.Is(err, store.ErrReconciliationInput):
			response.Fail(c, http.StatusBadRequest, "INVALID_RECONCILIATION", err.Error())
		case errors.Is(err, store.ErrReconciliationConflict):
			response.Fail(c, http.StatusConflict, "RECONCILIATION_CONFLICT", err.Error())
		case errors.Is(err, store.ErrReconciliationForbidden):
			response.Fail(c, http.StatusForbidden, "FORBIDDEN", err.Error())
		default:
			_ = c.Error(err)
			response.Fail(c, http.StatusInternalServerError, "INTERNAL", "核对结果未保存，请刷新后重试")
		}
		return
	}
	response.OK(c, out)
}

func knownReconciliationStatus(status string) bool {
	switch strings.ToUpper(status) {
	case "QUEUED", "PENDING", "RUNNING", "PROCESSING", "SUCCESS", "COMPLETED", "DONE", "FAILED", "ERROR", "CANCELLED", "CANCELED":
		return true
	default:
		return false
	}
}
