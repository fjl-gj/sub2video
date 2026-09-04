package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/middleware"
	"github.com/sbu2video/backend/internal/response"
	"github.com/sbu2video/backend/internal/store"
)

type ClientHandler struct {
	Store *store.Store
}

func (h *ClientHandler) Overview(c *gin.Context) {
	user := middleware.CurrentUser(c)
	ws, _ := h.Store.GetWorkspaceName(c.Request.Context(), user.ID)
	bal, _ := h.Store.GetBalance(c.Request.Context(), user.ID)
	totalSpent, monthSpent, _ := h.Store.SpentTotals(c.Request.Context(), user.ID)

	activeItems, _, counts, _ := h.Store.ListTasks(c.Request.Context(), user.ID, "生成中", 1, 1)
	if len(activeItems) == 0 {
		activeItems, _, _, _ = h.Store.ListTasks(c.Request.Context(), user.ID, "排队中", 1, 1)
	}
	activeCount := counts["排队中"] + counts["生成中"]
	var active any
	if len(activeItems) > 0 {
		active = taskDTO(&activeItems[0])
	}

	completed, _, _, _ := h.Store.ListTasks(c.Request.Context(), user.ID, "已完成", 1, 2)
	recent := make([]gin.H, 0, len(completed))
	for i := range completed {
		t := &completed[i]
		recent = append(recent, gin.H{
			"id": t.ID, "title": t.Title, "status": t.Status, "resolution": t.Resolution,
			"duration": fmt.Sprintf("%d 秒", t.DurationSec), "cost": t.Cost, "thumbUrl": t.ThumbURL,
		})
	}

	usedBytes, fileCount, _, trashCount, err := h.Store.StorageUsage(c.Request.Context(), user.ID)
	if err != nil {
		usedBytes, fileCount, trashCount = 0, 0, 0
	}
	quotaStr, _ := h.Store.GetSetting(c.Request.Context(), "storage_quota_gb")
	quota := 5.0
	if v, err := strconv.ParseFloat(quotaStr, 64); err == nil && v > 0 {
		quota = v
	}
	usedGb := float64(usedBytes) / (1024 * 1024 * 1024)
	usedGb = float64(int(usedGb*1000+0.5)) / 1000

	response.OK(c, gin.H{
		"workspace": gin.H{"displayName": ws},
		"billing":   gin.H{"totalSpent": totalSpent, "monthSpent": monthSpent},
		"wallet":    gin.H{"balance": bal},
		"pricing":   h.referencePricing(c),
		"storage": gin.H{
			"usedGb": usedGb, "quotaGb": quota, "fileCount": fileCount,
		},
		"tasks": gin.H{
			"activeCount": activeCount, "activeTask": active, "recentCompleted": recent,
		},
		"navBadges": gin.H{"tasks": activeCount, "tickets": 0, "trash": trashCount},
	})
}

func (h *ClientHandler) referencePricing(c *gin.Context) gin.H {
	price, err := h.Store.CheapestActivePrice(c.Request.Context())
	if err != nil || price <= 0 {
		price = 0.5
	}
	return gin.H{"referencePricePerSecond": price, "referenceNote": "768P / 1080P 按秒计"}
}

func (h *ClientHandler) CreateForm(c *gin.Context) {
	user := middleware.CurrentUser(c)
	bal, _ := h.Store.GetBalance(c.Request.Context(), user.ID)
	wfs, err := h.Store.ListPlatformWorkflows(c.Request.Context(), false)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	list := make([]gin.H, 0, len(wfs))
	var defaultParams gin.H
	for _, w := range wfs {
		item := gin.H{
			"id": w.Code, "name": w.Name, "description": w.Description,
			"enabled":   w.Status == "published",
			"modelCode": w.ModelCode, "carrierCode": w.CarrierCode,
			"params": gin.H{
				"aspectRatios":      w.Params.AspectRatios,
				"durations":         w.Params.Durations,
				"defaultDuration":   w.Params.DefaultDuration,
				"resolutions":       w.Params.Resolutions,
				"defaultResolution": w.Params.DefaultResolution,
			},
		}
		if w.Badge != "" {
			item["badge"] = w.Badge
		}
		list = append(list, item)
		if w.Status == "published" && defaultParams == nil {
			defaultParams = item["params"].(gin.H)
		}
	}
	if defaultParams == nil {
		defaultParams = gin.H{
			"aspectRatios": []string{"16:9", "9:16", "1:1"},
			"durations":    []int{5, 10, 15}, "defaultDuration": 5,
			"resolutions": []string{"768P"}, "defaultResolution": "768P",
		}
	}
	cur, _ := h.Store.ResolveCurrentProjectID(c.Request.Context(), user.ID)
	response.OK(c, gin.H{
		"wallet":           gin.H{"balance": bal},
		"draftPrompt":      "",
		"currentProjectId": cur,
		"workflows":        list,
		"parameters":       defaultParams,
		"suggestedTags":    []string{"电影感", "缓慢推进", "自然光", "细节丰富"},
	})
}

func (h *ClientHandler) QuoteTask(c *gin.Context) {
	var req struct {
		ModeID          string `json:"modeId" binding:"required"`
		ModelID         string `json:"modelId"`
		DurationSeconds int    `json:"durationSeconds"`
		Resolution      string `json:"resolution"`
		AspectRatio     string `json:"aspectRatio"`
		GenerateAudio   bool   `json:"generateAudio"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	user := middleware.CurrentUser(c)
	quote, err := h.Store.CreateQuote(c.Request.Context(), user.ID, store.QuoteInput{
		ModeID: req.ModeID, ModelID: req.ModelID, DurationSeconds: req.DurationSeconds,
		Resolution: req.Resolution, AspectRatio: req.AspectRatio, GenerateAudio: req.GenerateAudio,
	})
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "UNAVAILABLE_COMBINATION", err.Error())
		return
	}
	response.OK(c, gin.H{
		"quoteId": quote.ID, "expiresAt": quote.ExpiresAt,
		"selection": gin.H{
			"carrier": gin.H{"id": quote.CarrierID, "name": quote.CarrierName},
			"model":   gin.H{"id": quote.ModelID, "name": quote.ModelName},
		},
		"price": gin.H{
			"currency": "CNY", "pricePerSecond": quote.PricePerSecond,
			"durationSeconds": quote.DurationSeconds, "total": quote.Total,
		},
	})
}

func (h *ClientHandler) CreateTask(c *gin.Context) {
	var req struct {
		QuoteID    string           `json:"quoteId" binding:"required"`
		Title      string           `json:"title"`
		Prompt     string           `json:"prompt"`
		SourceMeta store.SourceMeta `json:"sourceMeta"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	user := middleware.CurrentUser(c)
	task, ledgerID, balance, err := h.Store.CreateTaskFromQuote(c.Request.Context(), user.ID, req.QuoteID, req.Title, req.Prompt, req.SourceMeta)
	if err != nil {
		switch err.Error() {
		case "insufficient balance":
			response.Fail(c, http.StatusPaymentRequired, "INSUFFICIENT_BALANCE", "余额不足")
		case "报价已过期", "报价已使用", "报价已过期或不可用":
			response.Fail(c, http.StatusConflict, "QUOTE_EXPIRED_OR_UNAVAILABLE", err.Error())
		default:
			response.Fail(c, http.StatusBadRequest, "INVALID_TASK_INPUT", err.Error())
		}
		return
	}
	response.Created(c, gin.H{
		"task": gin.H{
			"id": task.ID, "title": task.Title, "status": task.Status,
			"cost": task.Cost, "workflow": task.Workflow,
		},
		"wallet":        gin.H{"balance": balance},
		"ledgerEntryId": ledgerID,
	})
}

func (h *ClientHandler) CreationCatalog(c *gin.Context) {
	catalog, err := h.Store.ListClientCreationCatalog(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	modes := make([]gin.H, 0, len(catalog.Modes))
	for _, mode := range catalog.Modes {
		modes = append(modes, gin.H{"id": mode.ID, "code": mode.Code, "name": mode.Name, "description": mode.Description, "enabled": mode.Status == "published"})
	}
	carriers := make([]gin.H, 0, len(catalog.Carriers))
	for _, carrier := range catalog.Carriers {
		carriers = append(carriers, gin.H{"id": carrier.ID, "name": carrier.Name})
	}
	models := make([]gin.H, 0, len(catalog.Models))
	for _, model := range catalog.Models {
		models = append(models, gin.H{"id": model.ID, "carrierId": model.CarrierID, "name": model.Name, "description": model.Description, "modes": model.Modes})
	}
	response.OK(c, gin.H{"modes": modes, "carriers": carriers, "models": models})
}

func (h *ClientHandler) CreationOptions(c *gin.Context) {
	options, err := h.Store.ResolveCreationOptions(c.Request.Context(), c.Query("modeId"), c.Query("modelId"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "UNAVAILABLE_COMBINATION", err.Error())
		return
	}
	response.OK(c, gin.H{
		"durations": options.Durations, "resolutions": options.Resolutions, "aspectRatios": options.AspectRatios,
		"supportsGeneratedAudio": options.SupportsGeneratedAudio,
		"minImages":              options.MinImages, "maxImages": options.MaxImages,
		"defaultDuration": options.DefaultDuration, "defaultResolution": options.DefaultResolution,
	})
}

func (h *ClientHandler) ListTasks(c *gin.Context) {
	user := middleware.CurrentUser(c)
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	items, total, counts, err := h.Store.ListTasks(c.Request.Context(), user.ID, c.Query("status"), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	list := make([]gin.H, 0, len(items))
	for i := range items {
		list = append(list, taskListDTO(&items[i]))
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	response.OK(c, gin.H{"items": list, "total": total, "page": page, "pageSize": pageSize, "counts": counts})
}

func (h *ClientHandler) CancelTask(c *gin.Context) {
	user := middleware.CurrentUser(c)
	task, refund, balance, err := h.Store.CancelTask(c.Request.Context(), user.ID, c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{
		"task":         gin.H{"id": task.ID, "status": task.Status},
		"refundAmount": refund,
		"wallet":       gin.H{"balance": balance},
	})
}

func (h *ClientHandler) RetryTask(c *gin.Context) {
	user := middleware.CurrentUser(c)
	task, err := h.Store.RetryTask(c.Request.Context(), user.ID, c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"task": gin.H{"id": task.ID, "status": task.Status}})
}

func (h *ClientHandler) Wallet(c *gin.Context) {
	user := middleware.CurrentUser(c)
	bal, _ := h.Store.GetBalance(c.Request.Context(), user.ID)
	products, err := h.Store.ListRechargeProducts(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	response.OK(c, gin.H{
		"balance": bal, "products": products, "paymentMethods": []string{"wechat_pay"},
	})
}

func (h *ClientHandler) WalletTransactions(c *gin.Context) {
	user := middleware.CurrentUser(c)
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	items, total, err := h.Store.ListLedger(c.Request.Context(), user.ID, page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	list := make([]gin.H, 0, len(items))
	for _, e := range items {
		list = append(list, gin.H{
			"id": e.ID, "type": e.Type, "amount": e.Amount, "desc": e.Desc,
			"time": e.Time.Local().Format("01-02 15:04"),
		})
	}
	response.OK(c, gin.H{"items": list, "total": total, "page": page, "pageSize": pageSize})
}

func (h *ClientHandler) Redeem(c *gin.Context) {
	var body struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	user := middleware.CurrentUser(c)
	amount, balance, err := h.Store.Redeem(c.Request.Context(), user.ID, body.Code)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"credited": amount, "wallet": gin.H{"balance": balance}})
}

func (h *ClientHandler) ListTickets(c *gin.Context) {
	user := middleware.CurrentUser(c)
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	items, total, err := h.Store.ListTickets(c.Request.Context(), user.ID, page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	list := make([]gin.H, 0, len(items))
	for _, t := range items {
		list = append(list, gin.H{
			"id": t.ID, "title": t.Title, "category": t.Category, "status": t.Status,
			"task": t.TaskRef, "updated": t.UpdatedAt.Local().Format("01-02 15:04"),
		})
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	response.OK(c, gin.H{"items": list, "total": total, "page": page, "pageSize": pageSize})
}

func (h *ClientHandler) SaveTaskToProject(c *gin.Context) {
	user := middleware.CurrentUser(c)
	var req struct {
		ProjectID string `json:"projectId"`
	}
	_ = c.ShouldBindJSON(&req)
	item, path, err := h.Store.SaveTaskToProject(c.Request.Context(), user.ID, c.Param("id"), req.ProjectID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "NOT_FOUND", "任务不存在")
			return
		}
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	projectID := req.ProjectID
	if projectID == "" {
		projectID, _ = h.Store.ResolveCurrentProjectID(c.Request.Context(), user.ID)
	}
	response.OK(c, gin.H{
		"asset": gin.H{
			"id": item.ID, "name": item.Name, "type": item.Type, "sourceTask": item.SourceTask,
			"url": fmt.Sprintf("/api/v1/client/projects/%s/files/%s", projectID, item.ID),
		},
		"path": path,
	})
}

func (h *ClientHandler) CreateTicket(c *gin.Context) {
	var req struct {
		Title    string `json:"title" binding:"required"`
		Category string `json:"category"`
		TaskRef  string `json:"taskId"`
		Detail   string `json:"detail"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	user := middleware.CurrentUser(c)
	t, err := h.Store.CreateTicket(c.Request.Context(), user.ID, req.Title, req.Category, req.TaskRef, req.Detail)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Created(c, gin.H{"ticket": gin.H{"id": t.ID, "title": t.Title, "status": t.Status}})
}

func taskDTO(t *store.Task) gin.H {
	return gin.H{
		"id": t.ID, "title": t.Title, "workflow": t.Workflow, "status": t.Status,
		"progress": t.Progress, "duration": fmt.Sprintf("%d 秒", t.DurationSec),
		"resolution": t.Resolution, "cost": t.Cost, "thumbUrl": t.ThumbURL,
	}
}

func taskListDTO(t *store.Task) gin.H {
	return gin.H{
		"id": t.ID, "title": t.Title, "workflow": t.Workflow, "status": t.Status,
		"progress": t.Progress, "duration": fmt.Sprintf("%d 秒", t.DurationSec),
		"resolution": t.Resolution, "cost": t.Cost, "time": t.UpdatedAt.Local().Format("01-02 15:04"),
		"thumbUrl": t.ThumbURL, "savedToProject": t.SavedToProject,
	}
}
