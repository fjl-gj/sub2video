package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/middleware"
	"github.com/sbu2video/backend/internal/response"
	"github.com/sbu2video/backend/internal/store"
)

type ClientHandler struct {
	Store *store.Store
	Cfg   config.Config
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
	mediaCtx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	refreshTaskThumbnails(mediaCtx, h.Store.LoadObjectStore(mediaCtx, h.Cfg), completed)
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
	price, available, err := h.Store.CheapestAvailableRetailPriceWithFallback(c.Request.Context(), h.Cfg.AutoDLToken)
	if err != nil || !available || price <= 0 {
		return gin.H{"available": false, "referencePricePerSecond": 0, "referenceNote": "当前无可用生成能力"}
	}
	return gin.H{"available": true, "referencePricePerSecond": price, "referenceNote": "按秒计"}
}

// LegacyQuoteTask is retained only as an explicit compatibility boundary for
// already deployed clients. New clients must use estimate followed by direct
// task submission; the historical quote flow is no longer a supported API.
func (h *ClientHandler) LegacyQuoteTask(c *gin.Context) {
	response.Fail(c, http.StatusGone, "LEGACY_QUOTE_DISABLED", "旧报价接口已停用，请使用任务预估和直接提交接口")
}

func directRouteRequestFromJSON(modeID, carrierID, modelID string, duration int, resolution, aspect string, audio bool) store.DirectRouteRequest {
	return store.DirectRouteRequest{ModeID: modeID, CarrierID: carrierID, ModelID: modelID, DurationSeconds: duration, Resolution: resolution, AspectRatio: aspect, GenerateAudio: audio}
}

func (h *ClientHandler) ensureCreationStorage(ctx context.Context, st *store.Store, route *store.ResolvedRoute, verifyBucket bool) error {
	if route == nil {
		return nil
	}
	storageCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	objectStore := st.LoadObjectStore(storageCtx, h.Cfg)
	if !objectStore.Configured() {
		return errors.New("对象存储尚未配置，当前创作暂不可用")
	}
	if !verifyBucket {
		return nil
	}
	if err := objectStore.EnsureBucket(storageCtx); err != nil {
		return errors.New("对象存储暂不可用，请稍后重试")
	}
	return nil
}

// ensureCreationSupplier verifies that the locked execution can actually
// obtain a usable credential before a client sees an estimate or is charged.
// Provider identity and credential diagnostics remain internal; callers only
// receive a stable availability error.
func (h *ClientHandler) ensureCreationSupplier(ctx context.Context, st *store.Store, route *store.ResolvedRoute) error {
	if route == nil || route.Offering == nil {
		return errors.New("供应商执行配置不可用")
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	token, err := st.ResolveSupplierToken(checkCtx, h.Cfg.SecretKey, route.Offering.SupplierCode, h.Cfg.AutoDLToken)
	if err != nil || strings.TrimSpace(token) == "" {
		return errors.New("供应商执行凭据暂不可用")
	}
	return nil
}

// clientCreationError maps internal routing/storage/provider errors to a
// stable creator-facing contract. Raw database errors and provider metadata
// must never be returned by the client API.
func clientCreationError(err error) (int, string, string) {
	if err == nil {
		return http.StatusInternalServerError, "INTERNAL", "创作暂时不可用，请稍后重试"
	}
	if errors.Is(err, store.ErrPromptRequired) {
		return http.StatusBadRequest, "PROMPT_REQUIRED", store.ErrPromptRequired.Error()
	}
	message := strings.TrimSpace(err.Error())
	switch message {
	case "insufficient balance":
		return http.StatusPaymentRequired, "INSUFFICIENT_BALANCE", "余额不足"
	case "对象存储尚未配置，当前创作暂不可用", "对象存储暂不可用，请稍后重试":
		return http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", message
	case "供应商执行凭据暂不可用", "供应商执行配置不可用":
		return http.StatusServiceUnavailable, "SUPPLY_UNAVAILABLE", "当前生成能力暂不可用，请稍后重试"
	}
	// These messages are validated input/capability failures and are safe to
	// show because they contain no supplier, cost, token, or upstream data.
	for _, safe := range []string{
		"缺少有效幂等请求 ID", "视频时长不合法", "请完整选择生成设置", "创作模式未开放",
		"模型不属于所选承运商", "该模型不支持当前创作模式", "当前模式暂无该参数售价",
		"当前参数暂无可用执行配置", "当前模型暂无可用供给", "当前参数暂无可用模型或供给",
		"提示词", "素材", "参考音频", "图片", "音频",
	} {
		if strings.Contains(message, safe) {
			return http.StatusBadRequest, "UNAVAILABLE_COMBINATION", message
		}
	}
	if strings.Contains(message, "对象存储") {
		return http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "对象存储暂不可用，请稍后重试"
	}
	if strings.Contains(message, "供应商") || strings.Contains(message, "执行配置") {
		return http.StatusServiceUnavailable, "SUPPLY_UNAVAILABLE", "当前生成能力暂不可用，请稍后重试"
	}
	return http.StatusInternalServerError, "INTERNAL", "创作暂时不可用，请稍后重试"
}

// EstimateTask calculates the current user price without creating a quote,
// order, wallet entry, or task. API and workflow executions are internal.
func (h *ClientHandler) EstimateTask(c *gin.Context) {
	var req struct {
		ModeID      string `json:"modeId" binding:"required"`
		CarrierID   string `json:"carrierId"`
		ModelID     string `json:"modelId"`
		DurationSec int    `json:"durationSec"`
		// DurationSeconds is accepted only for already deployed clients. New
		// clients must send durationSec.
		DurationSeconds int              `json:"durationSeconds,omitempty"`
		Resolution      string           `json:"resolution"`
		AspectRatio     string           `json:"aspectRatio"`
		GenerateAudio   bool             `json:"generateAudio"`
		SourceMeta      store.SourceMeta `json:"sourceMeta"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	duration := req.DurationSec
	if duration == 0 {
		duration = req.DurationSeconds
	}
	routeRequest := directRouteRequestFromJSON(req.ModeID, req.CarrierID, req.ModelID, duration, req.Resolution, req.AspectRatio, req.GenerateAudio)
	routeRequest.SupplierTokenFallback = h.Cfg.AutoDLToken
	routeRequest.ImageCount = req.SourceMeta.ImageCount()
	route, err := h.Store.ResolveDirectRoute(c.Request.Context(), routeRequest)
	if err != nil {
		status, code, message := clientCreationError(err)
		response.Fail(c, status, code, message)
		return
	}
	if err := store.ValidateCreationSourceShape(route.Mode.Code, req.SourceMeta); err != nil {
		status, code, message := clientCreationError(err)
		response.Fail(c, status, code, message)
		return
	}
	if err := h.ensureCreationSupplier(c.Request.Context(), h.Store, route); err != nil {
		status, code, message := clientCreationError(err)
		response.Fail(c, status, code, message)
		return
	}
	if err := h.ensureCreationStorage(c.Request.Context(), h.Store, route, true); err != nil {
		response.Fail(c, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", err.Error())
		return
	}
	response.OK(c, gin.H{"available": true, "selection": gin.H{"carrier": gin.H{"id": route.Model.CarrierID, "name": route.Model.CarrierName}, "model": gin.H{"id": route.Model.ID, "name": route.Model.Name}}, "billing": gin.H{"basis": "requested_duration", "durationSec": duration, "resolution": req.Resolution, "generateAudio": req.GenerateAudio, "pricePerSecond": route.Retail.PricePerSecond, "estimatedTotal": route.Total, "currency": "CNY"}})
}

func (h *ClientHandler) CreateTask(c *gin.Context) {
	var req struct {
		ModeID          string           `json:"modeId" binding:"required"`
		CarrierID       string           `json:"carrierId"`
		ModelID         string           `json:"modelId"`
		DurationSec     int              `json:"durationSec"`
		DurationSeconds int              `json:"durationSeconds,omitempty"`
		Resolution      string           `json:"resolution" binding:"required"`
		AspectRatio     string           `json:"aspectRatio" binding:"required"`
		GenerateAudio   bool             `json:"generateAudio"`
		Title           string           `json:"title"`
		Prompt          string           `json:"prompt"`
		SourceMeta      store.SourceMeta `json:"sourceMeta"`
		ClientRequestID string           `json:"clientRequestId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	duration := req.DurationSec
	if duration == 0 {
		duration = req.DurationSeconds
	}
	if duration <= 0 {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "请填写有效的视频时长")
		return
	}
	user := middleware.CurrentUser(c)
	requestID := req.ClientRequestID
	if requestID == "" {
		requestID = c.GetHeader("Idempotency-Key")
	}
	result, err := h.Store.CreateTaskDirect(c.Request.Context(), user.ID, store.DirectTaskInput{
		DirectRouteRequest: func() store.DirectRouteRequest {
			routeRequest := directRouteRequestFromJSON(req.ModeID, req.CarrierID, req.ModelID, duration, req.Resolution, req.AspectRatio, req.GenerateAudio)
			routeRequest.SupplierTokenFallback = h.Cfg.AutoDLToken
			routeRequest.ImageCount = req.SourceMeta.ImageCount()
			return routeRequest
		}(),
		ClientRequestID: requestID,
		Title:           req.Title,
		Prompt:          req.Prompt,
		SourceMeta:      req.SourceMeta,
		Preflight: func(ctx context.Context, transactionStore *store.Store, route *store.ResolvedRoute) error {
			if err := h.ensureCreationSupplier(ctx, transactionStore, route); err != nil {
				return err
			}
			return h.ensureCreationStorage(ctx, transactionStore, route, true)
		},
	})
	if err != nil {
		status, code, message := clientCreationError(err)
		response.Fail(c, status, code, message)
		return
	}
	payload := gin.H{
		"task": gin.H{
			"id": result.Task.ID, "title": result.Task.Title, "status": result.Task.Status,
			"cost": result.Task.Cost, "mode": result.Task.Workflow,
		},
		"order":            gin.H{"id": result.OrderID, "amount": result.Task.Cost, "currency": "CNY", "status": result.OrderStatus},
		"wallet":           gin.H{"balance": result.Balance},
		"ledgerEntryId":    result.LedgerID,
		"idempotentReplay": result.AlreadyDone,
	}
	if result.AlreadyDone {
		response.OK(c, payload)
		return
	}
	response.Created(c, payload)
}

func (h *ClientHandler) CreationCatalog(c *gin.Context) {
	catalog, err := h.Store.ListClientCreationCatalogWithFallback(c.Request.Context(), h.Cfg.AutoDLToken)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", "创作目录暂时不可用，请稍后重试")
		return
	}
	modes := make([]gin.H, 0, len(catalog.Modes))
	for _, mode := range catalog.Modes {
		modes = append(modes, gin.H{"id": mode.ID, "inputType": modeInputType(mode.Code), "name": mode.Name, "description": mode.Description, "enabled": mode.Status == "published"})
	}
	carriers := make([]gin.H, 0, len(catalog.Carriers))
	for _, carrier := range catalog.Carriers {
		carriers = append(carriers, gin.H{"id": carrier.ID, "name": carrier.Name})
	}
	models := make([]gin.H, 0, len(catalog.Models))
	for _, model := range catalog.Models {
		inputTypes := make([]string, 0, len(model.Modes))
		for _, code := range model.Modes {
			inputType := modeInputType(code)
			if inputType != "unsupported" {
				inputTypes = append(inputTypes, inputType)
			}
		}
		models = append(models, gin.H{"id": model.ID, "carrierId": model.CarrierID, "name": model.Name, "description": model.Description, "inputTypes": inputTypes})
	}
	response.OK(c, gin.H{"modes": modes, "carriers": carriers, "models": models})
}

func modeInputType(code string) string {
	switch code {
	case "text_to_video":
		return "text"
	case "first_last_frame_to_video":
		return "first_last"
	case "multi_image_to_video":
		return "multi_image"
	default:
		return "unsupported"
	}
}

func (h *ClientHandler) CreationOptions(c *gin.Context) {
	options, err := h.Store.ResolveCreationOptionsWithFallback(c.Request.Context(), c.Query("modeId"), c.Query("modelId"), h.Cfg.AutoDLToken)
	if err != nil {
		status, code, message := clientCreationError(err)
		response.Fail(c, status, code, message)
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
	mediaCtx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	refreshTaskThumbnails(mediaCtx, h.Store.LoadObjectStore(mediaCtx, h.Cfg), items)
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

func (h *ClientHandler) GetTask(c *gin.Context) {
	user := middleware.CurrentUser(c)
	task, err := h.Store.GetClientTaskDetail(c.Request.Context(), user.ID, c.Param("id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "NOT_FOUND", "任务不存在")
			return
		}
		response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	mediaCtx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	objects := h.Store.LoadObjectStore(mediaCtx, h.Cfg)
	if err := refreshTaskDelivery(mediaCtx, objects, task); err != nil {
		response.Fail(c, http.StatusServiceUnavailable, "DELIVERY_UNAVAILABLE", "成片访问地址暂不可用，请稍后刷新或联系支持")
		return
	}
	if c.Query("download") == "true" {
		if err := refreshTaskDownload(mediaCtx, objects, task); err != nil {
			response.Fail(c, http.StatusServiceUnavailable, "DELIVERY_UNAVAILABLE", "成片下载地址暂不可用，请稍后重试或联系支持")
			return
		}
	}
	c.Header("Cache-Control", "private, no-store")
	response.OK(c, clientTaskDetailDTO(task))
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
		description := e.Desc
		// Provider and order identifiers are internal diagnostics. Keep the
		// creator-facing ledger useful without exposing upstream terminology.
		if e.Type == "任务退款" {
			description = "任务退款"
		}
		list = append(list, gin.H{
			"id": e.ID, "type": e.Type, "amount": e.Amount, "desc": description,
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
		list = append(list, clientTicketDTO(&t))
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
	response.Created(c, gin.H{"ticket": clientTicketDTO(t)})
}

func clientTicketDTO(t *store.Ticket) gin.H {
	return gin.H{
		"id": t.ID, "title": t.Title, "category": t.Category, "status": t.Status,
		"task": t.TaskRef, "detail": t.Detail, "updated": t.UpdatedAt.Local().Format("01-02 15:04"),
	}
}

func taskDTO(t *store.Task) gin.H {
	return gin.H{
		"id": t.ID, "title": t.Title, "mode": t.Workflow, "status": t.Status,
		"progress": t.Progress, "duration": fmt.Sprintf("%d 秒", t.DurationSec),
		"resolution": t.Resolution, "cost": t.Cost, "thumbUrl": t.ThumbURL,
	}
}

func taskListDTO(t *store.Task) gin.H {
	return gin.H{
		"id": t.ID, "title": t.Title, "mode": t.Workflow, "status": t.Status,
		"progress": t.Progress, "duration": fmt.Sprintf("%d 秒", t.DurationSec),
		"resolution": t.Resolution, "cost": t.Cost, "time": t.UpdatedAt.Local().Format("01-02 15:04"),
		"thumbUrl": t.ThumbURL, "savedToProject": t.SavedToProject,
		"canCancel": t.CanCancel, "reviewRequired": t.ReviewRequired,
	}
}

func clientTaskDetailDTO(task *store.ClientTaskDetail) gin.H {
	failureReason := clientSafeFailureReason(task.Status, task.FailureReason, task.RefundedAt != nil || task.RefundAmount > 0)
	if task.ReviewRequired {
		failureReason = "提交结果正在核对，请勿重复提交；如需帮助请联系客服"
	}
	out := gin.H{
		"id": task.ID, "title": task.Title, "mode": task.Workflow, "status": task.Status,
		"progress": task.Progress, "duration": fmt.Sprintf("%d 秒", task.DurationSec),
		"modeId": task.ModeID, "carrierId": task.CarrierID, "modelId": task.ModelID,
		"prompt": task.Prompt, "sourceMeta": task.SourceMeta,
		"durationSeconds": task.DurationSec, "resolution": task.Resolution, "aspectRatio": task.AspectRatio,
		"generateAudio": task.GenerateAudio, "cost": task.Cost, "thumbUrl": task.ThumbURL,
		"savedToProject": task.SavedToProject, "failureReason": failureReason,
		"canCancel": task.CanCancel, "reviewRequired": task.ReviewRequired,
		"refundAmount": task.RefundAmount, "refundedAt": task.RefundedAt,
		"executionWaiting": task.ExecutionWaiting, "deliveryStatus": task.DeliveryStatus,
		"downloadUrl": task.DownloadURL, "createdAt": task.CreatedAt, "updatedAt": task.UpdatedAt,
	}
	if task.SavedDestination != nil {
		out["savedDestination"] = task.SavedDestination
	}
	return out
}

// clientSafeFailureReason keeps provider diagnostics out of the creator
// contract. Detailed reasons remain available to operators in the admin task
// view and provider event log.
func clientSafeFailureReason(status, reason string, refunded bool) string {
	reason = strings.TrimSpace(reason)
	if status == "失败" && !refunded {
		return "任务已结束，费用状态待核对，请联系客服"
	}
	if reason == "" {
		return ""
	}
	if status != "失败" && !refunded {
		lower := strings.ToLower(reason)
		switch {
		case strings.Contains(lower, "config") || strings.Contains(reason, "配置"):
			return "当前生成能力暂不可用，任务等待恢复"
		case strings.Contains(lower, "storage") || strings.Contains(reason, "对象存储") || strings.Contains(reason, "交付"):
			return "成片交付暂不可用，任务等待重试"
		default:
			return "任务暂时无法推进，正在重试"
		}
	}
	lower := strings.ToLower(reason)
	switch {
	case strings.Contains(lower, "cancel") || strings.Contains(reason, "用户终止") || strings.Contains(reason, "管理员终止"):
		return "任务已终止，费用已退回"
	case strings.Contains(lower, "timeout") || strings.Contains(reason, "超时"):
		return "生成超时，费用已退回"
	case strings.Contains(lower, "storage") || strings.Contains(reason, "对象存储") || strings.Contains(reason, "交付"):
		return "成片交付暂时失败，费用已退回"
	case strings.Contains(lower, "config") || strings.Contains(reason, "配置"):
		return "当前生成能力暂不可用，费用已退回"
	default:
		return "生成失败，费用已退回"
	}
}
