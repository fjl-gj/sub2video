package handler

import (
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

	response.OK(c, gin.H{
		"workspace": gin.H{"displayName": ws},
		"billing":   gin.H{"totalSpent": totalSpent, "monthSpent": monthSpent},
		"wallet":    gin.H{"balance": bal},
		"pricing": gin.H{
			"referencePricePerSecond": 0.5,
			"referenceNote":           "768P / 1080P 按秒计",
		},
		"storage": gin.H{"usedGb": 0, "quotaGb": 5, "fileCount": 0},
		"tasks": gin.H{
			"activeCount": activeCount, "activeTask": active, "recentCompleted": recent,
		},
		"navBadges": gin.H{"tasks": activeCount, "tickets": 0, "trash": 0},
	})
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
	params := gin.H{
		"aspectRatios": []string{"16:9", "9:16", "1:1"},
		"durations": []int{5, 10, 15},
		"defaultDuration": 15,
		"resolution": "768P",
	}
	for _, w := range wfs {
		item := gin.H{
			"id": w.Code, "name": w.Name, "description": w.Description,
			"enabled": w.Status == "published",
		}
		if w.Badge != "" {
			item["badge"] = w.Badge
		}
		list = append(list, item)
		if w.Status == "published" && w.Params.DefaultDuration > 0 {
			params["aspectRatios"] = w.Params.AspectRatios
			params["durations"] = w.Params.Durations
			params["defaultDuration"] = w.Params.DefaultDuration
			if w.Params.DefaultResolution != "" {
				params["resolution"] = w.Params.DefaultResolution
			}
		}
	}
	response.OK(c, gin.H{
		"wallet": gin.H{"balance": bal},
		"draftPrompt": "",
		"currentProjectId": "prj_default",
		"workflows": list,
		"parameters": params,
		"suggestedTags": []string{"电影感", "缓慢推进", "自然光", "细节丰富"},
	})
}

func (h *ClientHandler) QuoteTask(c *gin.Context) {
	var req struct {
		WorkflowID      string `json:"workflowId"`
		DurationSeconds int    `json:"durationSeconds"`
		Resolution      string `json:"resolution"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if req.Resolution == "" {
		req.Resolution = "768P"
	}
	if req.DurationSeconds <= 0 {
		req.DurationSeconds = 15
	}
	rule, err := h.Store.FindActivePrice(c.Request.Context(), req.WorkflowID, req.Resolution)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "暂无该工作流/分辨率定价")
		return
	}
	cost := rule.PricePerSec * float64(req.DurationSeconds)
	user := middleware.CurrentUser(c)
	bal, _ := h.Store.GetBalance(c.Request.Context(), user.ID)
	response.OK(c, gin.H{
		"estimatedCost": cost,
		"billingNote":   "按输出秒数计费",
		"pricePerSecond": rule.PricePerSec,
		"walletBalance": bal,
		"priceSnapshot": gin.H{
			"ruleId": rule.ID, "version": rule.Version, "pricePerSecond": rule.PricePerSec,
			"estimatedCost": cost,
		},
	})
}

func (h *ClientHandler) CreateTask(c *gin.Context) {
	var req struct {
		WorkflowID      string `json:"workflowId" binding:"required"`
		Title           string `json:"title"`
		Prompt          string `json:"prompt"`
		AspectRatio     string `json:"aspectRatio"`
		DurationSeconds int    `json:"durationSeconds"`
		Resolution      string `json:"resolution"`
		Source          *struct {
			Type     string `json:"type"`
			Ref      string `json:"ref"`
			UploadID string `json:"uploadId"`
			AssetID  string `json:"assetId"`
		} `json:"source"`
		PriceSnapshot struct {
			RuleID         string  `json:"ruleId"`
			Version        string  `json:"version"`
			PricePerSecond float64 `json:"pricePerSecond"`
			EstimatedCost  float64 `json:"estimatedCost"`
		} `json:"priceSnapshot"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	in := store.CreateTaskInput{
		WorkflowID: req.WorkflowID, Title: req.Title, Prompt: req.Prompt,
		AspectRatio: req.AspectRatio, DurationSeconds: req.DurationSeconds, Resolution: req.Resolution,
		EstimatedCost: req.PriceSnapshot.EstimatedCost, PricePerSecond: req.PriceSnapshot.PricePerSecond,
		RuleID: req.PriceSnapshot.RuleID, Version: req.PriceSnapshot.Version,
	}
	if req.Source != nil {
		in.SourceType = req.Source.Type
		in.SourceRef = req.Source.Ref
		if in.SourceRef == "" && req.Source.UploadID != "" {
			in.SourceType = "upload"
			in.SourceRef = req.Source.UploadID
		}
		if in.SourceRef == "" && req.Source.AssetID != "" {
			in.SourceType = "asset"
			in.SourceRef = req.Source.AssetID
		}
	}
	user := middleware.CurrentUser(c)
	task, ledgerID, balance, err := h.Store.CreateTask(c.Request.Context(), user.ID, in)
	if err != nil {
		switch err.Error() {
		case "insufficient balance":
			response.Fail(c, http.StatusPaymentRequired, "INSUFFICIENT_BALANCE", "余额不足")
		case "价格快照过期":
			response.Fail(c, http.StatusConflict, "PRICE_EXPIRED", err.Error())
		default:
			response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		}
		return
	}
	response.Created(c, gin.H{
		"task": gin.H{
			"id": task.ID, "title": task.Title, "status": task.Status,
			"cost": task.Cost, "workflow": task.Workflow,
		},
		"wallet": gin.H{"balance": balance},
		"ledgerEntryId": ledgerID,
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
		"task": gin.H{"id": task.ID, "status": task.Status},
		"refundAmount": refund,
		"wallet": gin.H{"balance": balance},
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
