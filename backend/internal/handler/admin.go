package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/mail"
	"github.com/sbu2video/backend/internal/response"
	"github.com/sbu2video/backend/internal/store"
)

type AdminHandler struct {
	Store *store.Store
	Cfg   config.Config
	Mail  *mail.Service
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

func (h *AdminHandler) CreateSupplier(c *gin.Context) {
	var req struct {
		Code    string `json:"code" binding:"required"`
		Name    string `json:"name" binding:"required"`
		BaseURL string `json:"baseUrl"`
		Token   string `json:"token"`
		Enabled *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	sp, err := h.Store.CreateSupplier(c.Request.Context(), req.Code, req.Name, req.BaseURL, req.Token, h.Cfg.SecretKey, enabled)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Created(c, gin.H{
		"id": sp.ID, "code": sp.Code, "name": sp.Name, "baseUrl": sp.BaseURL,
		"enabled": sp.Enabled, "hasToken": sp.TokenConfigured,
	})
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

func (h *AdminHandler) CreateRoute(c *gin.Context) {
	var req struct {
		SupplierID        string                 `json:"supplierId" binding:"required"`
		ModelID           string                 `json:"modelId" binding:"required"`
		WorkflowID        string                 `json:"workflowId" binding:"required"`
		ClientWorkflowID  string                 `json:"clientWorkflowId"`
		Name              string                 `json:"name"`
		Enabled           *bool                  `json:"enabled"`
		CostPerSecPeak    float64                `json:"costPerSecPeak"`
		CostPerSecOffpeak float64                `json:"costPerSecOffpeak"`
		Conditions        *store.RouteConditions `json:"conditions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	cond := store.RouteConditions{}
	if req.Conditions != nil {
		cond = *req.Conditions
	}
	r, err := h.Store.CreateRoute(c.Request.Context(), req.SupplierID, req.ModelID, req.WorkflowID, req.Name,
		req.CostPerSecPeak, req.CostPerSecOffpeak, enabled, cond, req.ClientWorkflowID)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Created(c, gin.H{
		"id": r.ID, "workflowId": r.WorkflowID, "name": r.Name, "enabled": r.Enabled,
		"clientWorkflowId": r.ClientWorkflowID,
		"costPerSecPeak":   r.CostPerSecPeak, "costPerSecOffpeak": r.CostPerSecOffpeak,
	})
}

func (h *AdminHandler) CreateModel(c *gin.Context) {
	var req struct {
		CarrierID        string `json:"carrierId" binding:"required"`
		Code             string `json:"code" binding:"required"`
		Name             string `json:"name" binding:"required"`
		Description      string `json:"description"`
		ClientWorkflowID string `json:"clientWorkflowId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	m, err := h.Store.CreateModel(c.Request.Context(), req.CarrierID, req.Code, req.Name, req.Description, req.ClientWorkflowID)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Created(c, gin.H{
		"id": m.ID, "code": m.Code, "name": m.Name, "carrierCode": m.CarrierCode,
		"clientWorkflowId": m.ClientWorkflowID, "enabled": m.Enabled,
	})
}

func (h *AdminHandler) UpdateRoute(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Enabled           *bool                  `json:"enabled"`
		CostPerSecPeak    *float64               `json:"costPerSecPeak"`
		CostPerSecOffpeak *float64               `json:"costPerSecOffpeak"`
		WorkflowID        *string                `json:"workflowId"`
		Name              *string                `json:"name"`
		Conditions        *store.RouteConditions `json:"conditions"`
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
		"cosConfigured":   h.Store.LoadObjectStore(c.Request.Context(), h.Cfg).Configured(),
		"storageProvider": h.Store.StorageProvider(c.Request.Context()),
		"autodlEnvToken":  h.Cfg.AutoDLToken != "",
	})
}

func (h *AdminHandler) GetSettings(c *gin.Context) {
	keys := []string{
		"smtp_host", "smtp_port", "smtp_username", "smtp_from", "smtp_from_name", "smtp_use_tls",
		"email_verify_enabled", "password_reset_enabled", "registration_enabled", "frontend_url",
		"input_retention_days", "result_retention_days", "notify_task", "audit_enabled",
		"cos_secret_id", "cos_bucket", "cos_region", "cos_use_intranet",
		"storage_provider", "minio_endpoint", "minio_access_key", "minio_bucket", "minio_use_ssl", "minio_public_base",
		"initial_balance", "storage_quota_gb",
	}
	m, err := h.Store.GetSettings(c.Request.Context(), keys)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	pwd, _ := h.Store.GetSetting(c.Request.Context(), "smtp_password")
	cosKey, _ := h.Store.GetSetting(c.Request.Context(), "cos_secret_key")
	minioKey, _ := h.Store.GetSetting(c.Request.Context(), "minio_secret_key")
	provider := m["storage_provider"]
	if provider == "" {
		provider = "minio"
	}
	response.OK(c, gin.H{
		"smtpHost": m["smtp_host"], "smtpPort": m["smtp_port"], "smtpUsername": m["smtp_username"],
		"smtpFrom": m["smtp_from"], "smtpFromName": m["smtp_from_name"],
		"smtpUseTls":             m["smtp_use_tls"] != "false",
		"smtpPasswordConfigured": pwd != "",
		"emailVerifyEnabled":     m["email_verify_enabled"] == "true",
		"passwordResetEnabled":   m["password_reset_enabled"] == "true",
		"registrationEnabled":    m["registration_enabled"] != "false",
		"frontendUrl":            m["frontend_url"],
		"inputRetentionDays":     m["input_retention_days"],
		"resultRetentionDays":    m["result_retention_days"],
		"notifyTask":             m["notify_task"] != "false",
		"auditEnabled":           m["audit_enabled"] != "false",
		"storageProvider":        provider,
		"cosSecretId":            m["cos_secret_id"], "cosBucket": m["cos_bucket"], "cosRegion": m["cos_region"],
		"cosUseIntranet":           m["cos_use_intranet"] == "true",
		"cosSecretKeyConfigured":   cosKey != "",
		"minioEndpoint":            m["minio_endpoint"],
		"minioAccessKey":           m["minio_access_key"],
		"minioBucket":              m["minio_bucket"],
		"minioUseSsl":              m["minio_use_ssl"] == "true",
		"minioPublicBase":          m["minio_public_base"],
		"minioSecretKeyConfigured": minioKey != "",
		"initialBalance":           m["initial_balance"],
		"storageQuotaGb":           m["storage_quota_gb"],
	})
}

func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	kv := map[string]string{}
	mapStr := func(jsonKey, dbKey string) {
		if v, ok := req[jsonKey]; ok {
			switch t := v.(type) {
			case string:
				kv[dbKey] = t
			case bool:
				if t {
					kv[dbKey] = "true"
				} else {
					kv[dbKey] = "false"
				}
			case float64:
				kv[dbKey] = strconv.FormatInt(int64(t), 10)
			}
		}
	}
	mapStr("smtpHost", "smtp_host")
	mapStr("smtpPort", "smtp_port")
	mapStr("smtpUsername", "smtp_username")
	mapStr("smtpFrom", "smtp_from")
	mapStr("smtpFromName", "smtp_from_name")
	mapStr("smtpUseTls", "smtp_use_tls")
	mapStr("emailVerifyEnabled", "email_verify_enabled")
	mapStr("passwordResetEnabled", "password_reset_enabled")
	mapStr("registrationEnabled", "registration_enabled")
	mapStr("frontendUrl", "frontend_url")
	mapStr("inputRetentionDays", "input_retention_days")
	mapStr("resultRetentionDays", "result_retention_days")
	mapStr("notifyTask", "notify_task")
	mapStr("auditEnabled", "audit_enabled")
	mapStr("cosSecretId", "cos_secret_id")
	mapStr("cosBucket", "cos_bucket")
	mapStr("cosRegion", "cos_region")
	mapStr("cosUseIntranet", "cos_use_intranet")
	mapStr("storageProvider", "storage_provider")
	mapStr("minioEndpoint", "minio_endpoint")
	mapStr("minioAccessKey", "minio_access_key")
	mapStr("minioBucket", "minio_bucket")
	mapStr("minioUseSsl", "minio_use_ssl")
	mapStr("minioPublicBase", "minio_public_base")
	mapStr("initialBalance", "initial_balance")
	mapStr("storageQuotaGb", "storage_quota_gb")
	if v, ok := req["smtpPassword"].(string); ok && strings.TrimSpace(v) != "" {
		kv["smtp_password"] = strings.TrimSpace(v)
	}
	if err := h.Store.SetSettings(c.Request.Context(), kv); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if v, ok := req["cosSecretKey"].(string); ok && strings.TrimSpace(v) != "" {
		if err := h.Store.SaveCOSSecretKey(c.Request.Context(), h.Cfg.SecretKey, strings.TrimSpace(v)); err != nil {
			response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
	}
	if v, ok := req["minioSecretKey"].(string); ok && strings.TrimSpace(v) != "" {
		if err := h.Store.SaveMinIOSecretKey(c.Request.Context(), h.Cfg.SecretKey, strings.TrimSpace(v)); err != nil {
			response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
	}
	// best-effort ensure bucket when minio is active
	if st := h.Store.LoadObjectStore(c.Request.Context(), h.Cfg); st.Configured() {
		_ = st.EnsureBucket(c.Request.Context())
	}
	h.GetSettings(c)
}

func (h *AdminHandler) TestSMTP(c *gin.Context) {
	if h.Mail == nil {
		response.Fail(c, http.StatusServiceUnavailable, "EMAIL_NOT_CONFIGURED", "mail service unavailable")
		return
	}
	if err := h.Mail.TestConnection(c.Request.Context()); err != nil {
		response.Fail(c, http.StatusBadRequest, "SMTP_TEST_FAILED", err.Error())
		return
	}
	response.OK(c, gin.H{"status": "ok"})
}

func (h *AdminHandler) ListWorkflows(c *gin.Context) {
	items, err := h.Store.ListPlatformWorkflows(c.Request.Context(), false)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, w := range items {
		statusCN := "草稿"
		switch w.Status {
		case "published":
			statusCN = "已上架"
		case "disabled":
			statusCN = "未开放"
		}
		priceLabel := "待定"
		if rule, err := h.Store.FindActivePrice(c.Request.Context(), w.Code, "768P"); err == nil {
			priceLabel = fmt.Sprintf("%.2f 元/秒", rule.PricePerSec)
		}
		carrier := w.CarrierName
		if carrier == "" {
			carrier = w.ModelCode
		}
		out = append(out, gin.H{
			"id": w.ID, "code": w.Code, "name": w.Name, "description": w.Description,
			"status": statusCN, "statusRaw": w.Status, "badge": w.Badge,
			"modelId": w.ModelID, "modelCode": w.ModelCode, "carrier": carrier, "price": priceLabel,
			"updatedAt": w.UpdatedAt,
		})
	}
	response.OK(c, gin.H{"items": out})
}

func (h *AdminHandler) CreateWorkflow(c *gin.Context) {
	var req struct {
		Code        string `json:"code" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		ModelID     string `json:"modelId" binding:"required"`
		Status      string `json:"status"`
		Badge       string `json:"badge"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	w, err := h.Store.CreatePlatformWorkflow(c.Request.Context(), req.Code, req.Name, req.Description, req.ModelID, req.Status, req.Badge)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Created(c, gin.H{"id": w.ID, "code": w.Code, "name": w.Name, "status": w.Status, "modelCode": w.ModelCode})
}

func (h *AdminHandler) UpdateWorkflow(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status      *string `json:"status"`
		Name        *string `json:"name"`
		Description *string `json:"description"`
		ModelID     *string `json:"modelId"`
		Badge       *string `json:"badge"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	// backward-compatible: status-only toggle
	if req.Name == nil && req.Description == nil && req.ModelID == nil && req.Badge == nil && req.Status != nil {
		status := *req.Status
		switch status {
		case "已上架", "published":
			status = "published"
		case "草稿", "draft":
			status = "draft"
		case "未开放", "disabled":
			status = "disabled"
		default:
			response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "invalid status")
			return
		}
		w, err := h.Store.UpdatePlatformWorkflowStatus(c.Request.Context(), id, status)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		response.OK(c, gin.H{"id": w.ID, "code": w.Code, "status": w.Status})
		return
	}
	w, err := h.Store.UpdatePlatformWorkflow(c.Request.Context(), id, req.Name, req.Description, req.ModelID, req.Status, req.Badge)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{
		"id": w.ID, "code": w.Code, "name": w.Name, "description": w.Description,
		"status": w.Status, "modelCode": w.ModelCode, "badge": w.Badge,
	})
}

func (h *AdminHandler) ListPricing(c *gin.Context) {
	items, err := h.Store.ListPricingRules(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	routes, _ := h.Store.ListRoutes(c.Request.Context())
	routeCost := map[string]float64{} // modelCode -> min peak cost among enabled
	for _, rt := range routes {
		if !rt.Enabled {
			continue
		}
		if cur, ok := routeCost[rt.ModelCode]; !ok || rt.CostPerSecPeak < cur {
			routeCost[rt.ModelCode] = rt.CostPerSecPeak
		}
	}
	// map workflow -> model via workflows list
	wfs, _ := h.Store.ListPlatformWorkflows(c.Request.Context(), false)
	wfModel := map[string]string{}
	for _, w := range wfs {
		wfModel[w.Code] = w.ModelCode
	}
	out := make([]gin.H, 0, len(items))
	for _, r := range items {
		statusCN := "草稿"
		if r.Status == "active" {
			statusCN = "生效中"
		}
		cost := r.CostHintPerSec
		if mc := wfModel[r.WorkflowCode]; mc != "" {
			if v, ok := routeCost[mc]; ok {
				cost = v
			}
		}
		out = append(out, gin.H{
			"id": r.ID, "name": r.Name, "workflowId": r.WorkflowID, "workflowCode": r.WorkflowCode,
			"resolution": r.Resolution, "cost": fmt.Sprintf("%.2f", cost),
			"price": fmt.Sprintf("%.2f", r.PricePerSec), "pricePerSec": r.PricePerSec,
			"duration": "按秒", "status": statusCN, "version": r.Version,
		})
	}
	response.OK(c, gin.H{"items": out})
}

func (h *AdminHandler) UpdatePricing(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		PricePerSec *float64 `json:"pricePerSec"`
		Price       *string  `json:"price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	var price float64
	if req.PricePerSec != nil {
		price = *req.PricePerSec
	} else if req.Price != nil {
		p, err := strconv.ParseFloat(*req.Price, 64)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "invalid price")
			return
		}
		price = p
	} else {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "price required")
		return
	}
	r, err := h.Store.UpdatePricingRulePrice(c.Request.Context(), id, price)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"id": r.ID, "pricePerSec": r.PricePerSec, "version": r.Version})
}

func (h *AdminHandler) ListCreationModes(c *gin.Context) {
	items, err := h.Store.ListAdminCreationModes(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		policies := make([]gin.H, 0, len(item.Policies))
		for _, policy := range item.Policies {
			policies = append(policies, gin.H{"modelId": policy.ModelID, "modelName": policy.ModelName, "carrierName": policy.CarrierName,
				"enabled": policy.Enabled, "recommendationRank": policy.RecommendationRank, "displayBadge": policy.DisplayBadge,
				"defaultDuration": policy.DefaultDuration, "defaultResolution": policy.DefaultResolution})
		}
		out = append(out, gin.H{"id": item.ID, "code": item.Code, "name": item.Name, "description": item.Description,
			"status": item.Status, "sortOrder": item.SortOrder, "policies": policies})
	}
	response.OK(c, gin.H{"items": out})
}

func (h *AdminHandler) UpdateCreationMode(c *gin.Context) {
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
		SortOrder   *int    `json:"sortOrder"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	item, err := h.Store.UpdateCreationMode(c.Request.Context(), c.Param("id"), req.Name, req.Description, req.Status, req.SortOrder)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"id": item.ID, "name": item.Name, "description": item.Description, "status": item.Status, "sortOrder": item.SortOrder})
}

func (h *AdminHandler) UpsertModePolicy(c *gin.Context) {
	var req struct {
		ModelID            string `json:"modelId" binding:"required"`
		Enabled            bool   `json:"enabled"`
		RecommendationRank int    `json:"recommendationRank"`
		DisplayBadge       string `json:"displayBadge"`
		DefaultDuration    int    `json:"defaultDuration"`
		DefaultResolution  string `json:"defaultResolution"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if err := h.Store.UpsertModeModelPolicy(c.Request.Context(), c.Param("id"), req.ModelID, req.Enabled, req.RecommendationRank, req.DisplayBadge, req.DefaultDuration, req.DefaultResolution); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func offeringDTO(item store.SupplierOffering) gin.H {
	return gin.H{"id": item.ID, "supplierId": item.SupplierID, "supplierName": item.SupplierName, "modelId": item.ModelID,
		"modelName": item.ModelName, "carrierName": item.CarrierName, "modeId": item.ModeID, "modeName": item.ModeName,
		"name": item.Name, "upstreamWorkflowId": item.UpstreamWorkflowID, "adapterCode": item.AdapterCode, "enabled": item.Enabled,
		"healthStatus": item.HealthStatus, "priority": item.Priority, "minDurationSec": item.MinDurationSec, "maxDurationSec": item.MaxDurationSec,
		"resolutions": item.Resolutions, "aspectRatios": item.AspectRatios, "minImages": item.MinImages, "maxImages": item.MaxImages,
		"supportsGeneratedAudio": item.SupportsGeneratedAudio}
}

func (h *AdminHandler) ListOfferings(c *gin.Context) {
	items, err := h.Store.ListSupplierOfferings(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, offeringDTO(item))
	}
	response.OK(c, gin.H{"items": out})
}

func (h *AdminHandler) CreateOffering(c *gin.Context) {
	var req store.SupplierOffering
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	item, err := h.Store.CreateSupplierOffering(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Created(c, offeringDTO(*item))
}

func (h *AdminHandler) UpdateOffering(c *gin.Context) {
	var req store.SupplierOffering
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	item, err := h.Store.UpdateSupplierOffering(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, offeringDTO(*item))
}

func (h *AdminHandler) ListSupplierCosts(c *gin.Context) {
	items, err := h.Store.ListSupplierCostRules(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, gin.H{"id": item.ID, "offeringId": item.OfferingID, "resolution": item.Resolution,
			"generateAudio": item.GenerateAudio, "peakPerSecond": item.PeakPerSecond, "offpeakPerSecond": item.OffpeakPerSecond,
			"currency": item.Currency, "version": item.Version, "status": item.Status})
	}
	response.OK(c, gin.H{"items": out})
}

func (h *AdminHandler) UpsertSupplierCost(c *gin.Context) {
	var req store.SupplierCostRule
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	item, err := h.Store.UpsertSupplierCostRule(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"id": item.ID, "offeringId": item.OfferingID, "resolution": item.Resolution, "generateAudio": item.GenerateAudio,
		"peakPerSecond": item.PeakPerSecond, "offpeakPerSecond": item.OffpeakPerSecond, "version": item.Version, "status": item.Status})
}

func (h *AdminHandler) ListRetailPricing(c *gin.Context) {
	items, err := h.Store.ListRetailPriceRules(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, gin.H{"id": item.ID, "modeId": item.ModeID, "modeName": item.ModeName,
			"modelId": item.ModelID, "modelName": item.ModelName, "carrierName": item.CarrierName, "resolution": item.Resolution,
			"generateAudio": item.GenerateAudio, "name": item.Name, "pricePerSecond": item.PricePerSecond, "version": item.Version, "status": item.Status})
	}
	response.OK(c, gin.H{"items": out})
}

func (h *AdminHandler) UpsertRetailPricing(c *gin.Context) {
	var req store.RetailPriceRule
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	item, err := h.Store.UpsertRetailPriceRule(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"id": item.ID, "modeId": item.ModeID, "modelId": item.ModelID, "pricePerSecond": item.PricePerSecond, "version": item.Version, "status": item.Status})
}

func (h *AdminHandler) CreateCarrier(c *gin.Context) {
	var req struct {
		Code        string `json:"code" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	item, err := h.Store.CreateCarrier(c.Request.Context(), req.Code, req.Name, req.Description)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Created(c, gin.H{"id": item.ID, "code": item.Code, "name": item.Name, "description": item.Description, "enabled": item.Enabled})
}

func (h *AdminHandler) UpdateCarrier(c *gin.Context) {
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Enabled     *bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	item, err := h.Store.UpdateCarrier(c.Request.Context(), c.Param("id"), req.Name, req.Description, req.Enabled)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"id": item.ID, "name": item.Name, "description": item.Description, "enabled": item.Enabled})
}

func (h *AdminHandler) UpdateModel(c *gin.Context) {
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Enabled     *bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	item, err := h.Store.UpdateModel(c.Request.Context(), c.Param("id"), req.Name, req.Description, req.Enabled)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"id": item.ID, "name": item.Name, "description": item.Description, "enabled": item.Enabled})
}

func (h *AdminHandler) ListTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	items, total, err := h.Store.ListAdminTasks(c.Request.Context(), c.Query("status"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, t := range items {
		provider := t.SupplierCode
		if provider == "" {
			provider = "-"
		}
		out = append(out, gin.H{
			"id": t.ID, "title": t.Title, "workflow": t.Workflow, "workflowId": t.WorkflowPublicID,
			"status": t.Status, "progress": t.Progress, "resolution": t.Resolution,
			"durationSec": t.DurationSec, "cost": t.Cost,
			"user": t.UserName, "userEmail": t.UserEmail,
			"provider": provider, "upstreamTaskId": t.UpstreamTaskID, "upstreamStatus": t.UpstreamStatus,
			"failReason": t.FailReason,
			"time":       t.CreatedAt.Local().Format("01-02 15:04"),
			"updatedAt":  t.UpdatedAt,
		})
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	response.OK(c, gin.H{"items": out, "total": total, "page": page, "pageSize": pageSize})
}

func (h *AdminHandler) GetTask(c *gin.Context) {
	detail, err := h.Store.GetAdminTaskDetail(c.Request.Context(), c.Param("id"))
	if err != nil {
		if err == store.ErrNotFound {
			response.Fail(c, http.StatusNotFound, "NOT_FOUND", "任务不存在")
			return
		}
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	response.OK(c, detail)
}

func (h *AdminHandler) CancelTask(c *gin.Context) {
	task, refund, err := h.Store.AdminCancelTask(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"task": gin.H{"id": task.ID, "status": task.Status}, "refund": refund})
}

func (h *AdminHandler) RetryTask(c *gin.Context) {
	task, err := h.Store.AdminRetryTask(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"task": gin.H{"id": task.ID, "status": task.Status}})
}

func (h *AdminHandler) ListTickets(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	items, total, err := h.Store.ListAdminTickets(c.Request.Context(), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, t := range items {
		out = append(out, gin.H{
			"id": t.ID, "title": t.Title, "category": t.Category, "status": t.Status,
			"task": t.TaskRef, "user": t.UserName, "userEmail": t.UserEmail,
			"updated": t.UpdatedAt.Local().Format("01-02 15:04"),
		})
	}
	response.OK(c, gin.H{"items": out, "total": total})
}

func (h *AdminHandler) UpdateTicket(c *gin.Context) {
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if err := h.Store.UpdateTicketStatus(c.Request.Context(), c.Param("id"), req.Status); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"id": c.Param("id"), "status": req.Status})
}
