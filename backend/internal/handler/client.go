package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/middleware"
	"github.com/sbu2video/backend/internal/pricing"
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
	response.OK(c, gin.H{
		"wallet": gin.H{"balance": bal},
		"draftPrompt": "",
		"currentProjectId": "prj_default",
		"workflows": []gin.H{
			{"id": "wf-h3-image", "name": "图生视频", "description": "一张图片 + 文字描述", "enabled": true},
			{"id": "wf-h3-text", "name": "文生视频", "description": "纯文字描述生成视频", "enabled": true},
			{"id": "wf-h3-multi", "name": "多图生视频", "description": "多张参考图", "enabled": false, "badge": "即将开放"},
		},
		"parameters": gin.H{
			"aspectRatios": []string{"16:9", "9:16", "1:1"},
			"durations": []int{5, 10, 15},
			"defaultDuration": 15,
			"resolution": "768P",
		},
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
	cost, pps, ok := pricing.Quote(req.Resolution, req.DurationSeconds)
	if !ok {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", "unsupported quote params")
		return
	}
	user := middleware.CurrentUser(c)
	bal, _ := h.Store.GetBalance(c.Request.Context(), user.ID)
	response.OK(c, gin.H{
		"estimatedCost": cost,
		"billingNote":   "按输出秒数计费",
		"pricePerSecond": pps,
		"walletBalance": bal,
		"priceSnapshot": gin.H{
			"ruleId": pricing.RuleID, "version": pricing.Version, "pricePerSecond": pps,
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
