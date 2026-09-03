package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/response"
	"github.com/sbu2video/backend/internal/store"
)

type AdminHandler struct {
	Store *store.Store
	Cfg   config.Config
}

func (h *AdminHandler) ListSuppliers(c *gin.Context) {
	items, err := h.Store.ListSuppliers(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, sp := range items {
		status := "未配置"
		if sp.TokenConfigured {
			status = "已配置"
		}
		if !sp.Enabled {
			status = "未启用"
		} else if sp.TokenConfigured {
			status = "就绪"
		}
		out = append(out, gin.H{
			"id": sp.ID, "code": sp.Code, "name": sp.Name, "baseUrl": sp.BaseURL,
			"enabled": sp.Enabled, "hasToken": sp.TokenConfigured, "status": status,
			"updatedAt": sp.UpdatedAt,
		})
	}
	response.OK(c, gin.H{"items": out})
}

func (h *AdminHandler) UpdateSupplier(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name    *string `json:"name"`
		BaseURL *string `json:"baseUrl"`
		Enabled *bool   `json:"enabled"`
		Token   string  `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	sp, err := h.Store.UpdateSupplier(c.Request.Context(), id, req.Name, req.BaseURL, req.Enabled, req.Token, h.Cfg.SecretKey)
	if err != nil {
		if err == store.ErrNotFound {
			response.Fail(c, http.StatusNotFound, "NOT_FOUND", "supplier not found")
			return
		}
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{
		"id": sp.ID, "code": sp.Code, "name": sp.Name, "baseUrl": sp.BaseURL,
		"enabled": sp.Enabled, "hasToken": sp.TokenConfigured,
	})
}

func (h *AdminHandler) ListRoutes(c *gin.Context) {
	items, err := h.Store.ListRoutes(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, r := range items {
		out = append(out, gin.H{
			"id": r.ID, "supplierId": r.SupplierID, "supplierCode": r.SupplierCode, "supplierName": r.SupplierName,
			"modelId": r.ModelID, "modelCode": r.ModelCode, "clientWorkflowId": r.ClientWorkflowID,
			"workflowId": r.WorkflowID, "name": r.Name, "enabled": r.Enabled,
			"costPerSecPeak": r.CostPerSecPeak, "costPerSecOffpeak": r.CostPerSecOffpeak,
			"conditions": r.Conditions, "priority": r.Priority,
		})
	}
	response.OK(c, gin.H{"items": out})
}

func (h *AdminHandler) UpdateRoute(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Enabled            *bool                   `json:"enabled"`
		CostPerSecPeak     *float64                `json:"costPerSecPeak"`
		CostPerSecOffpeak  *float64                `json:"costPerSecOffpeak"`
		WorkflowID         *string                 `json:"workflowId"`
		Name               *string                 `json:"name"`
		Conditions         *store.RouteConditions  `json:"conditions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	r, err := h.Store.UpdateRoute(c.Request.Context(), id, req.Enabled, req.CostPerSecPeak, req.CostPerSecOffpeak, req.Conditions, req.WorkflowID, req.Name)
	if err != nil {
		if err == store.ErrNotFound {
			response.Fail(c, http.StatusNotFound, "NOT_FOUND", "route not found")
			return
		}
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{
		"id": r.ID, "workflowId": r.WorkflowID, "name": r.Name, "enabled": r.Enabled,
		"costPerSecPeak": r.CostPerSecPeak, "costPerSecOffpeak": r.CostPerSecOffpeak, "conditions": r.Conditions,
	})
}

func (h *AdminHandler) ListCarriers(c *gin.Context) {
	carriers, models, err := h.Store.ListCarriersWithModels(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	cOut := make([]gin.H, 0, len(carriers))
	for _, car := range carriers {
		cOut = append(cOut, gin.H{
			"id": car.ID, "code": car.Code, "name": car.Name, "description": car.Description, "enabled": car.Enabled,
		})
	}
	mOut := make([]gin.H, 0, len(models))
	for _, m := range models {
		mOut = append(mOut, gin.H{
			"id": m.ID, "carrierId": m.CarrierID, "carrierCode": m.CarrierCode, "code": m.Code,
			"name": m.Name, "description": m.Description, "clientWorkflowId": m.ClientWorkflowID, "enabled": m.Enabled,
		})
	}
	response.OK(c, gin.H{
		"carriers": cOut, "models": mOut,
		"cosConfigured": h.Cfg.COSSecretID != "" && h.Cfg.COSBucket != "",
		"autodlEnvToken": h.Cfg.AutoDLToken != "",
	})
}
