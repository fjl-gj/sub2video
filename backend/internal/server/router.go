package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/auth"
	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/handler"
	"github.com/sbu2video/backend/internal/mail"
	"github.com/sbu2video/backend/internal/middleware"
	"github.com/sbu2video/backend/internal/store"
)

func NewRouter(cfg config.Config, st *store.Store, authSvc *auth.Service, mailSvc *mail.Service) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(cors(cfg.CORSOrigins()))
	r.MaxMultipartMemory = 16 << 20

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC()})
	})

	authH := &handler.AuthHandler{Auth: authSvc, Store: st}
	clientH := &handler.ClientHandler{Store: st, Cfg: cfg}
	uploadH := &handler.UploadHandler{Store: st, UploadDir: cfg.UploadDir}
	projectH := &handler.ProjectHandler{Store: st, UploadDir: cfg.UploadDir, Cfg: cfg}
	adminH := &handler.AdminHandler{Store: st, Cfg: cfg, Mail: mailSvc}

	v1 := r.Group("/api/v1")
	{
		a := v1.Group("/auth")
		{
			a.POST("/register", authH.Register)
			a.POST("/login", authH.Login)
			a.POST("/refresh", authH.Refresh)
			a.POST("/logout", authH.Logout)
			a.POST("/send-verify-code", authH.SendVerifyCode)
			a.POST("/forgot-password", authH.ForgotPassword)
			a.POST("/reset-password", authH.ResetPassword)
		}
		v1.GET("/settings/public", authH.PublicSettings)

		authed := v1.Group("")
		authed.Use(middleware.JWTAuth(authSvc, st))
		{
			authed.GET("/auth/me", authH.Me)

			client := authed.Group("/client")
			{
				client.GET("/overview", clientH.Overview)
				client.GET("/create/catalog", clientH.CreationCatalog)
				client.GET("/create/options", clientH.CreationOptions)
				client.POST("/uploads", uploadH.Create)
				client.POST("/uploads/reference-audio", uploadH.CreateReferenceAudio)
				client.GET("/uploads/:id", uploadH.Get)
				client.GET("/storage/usage", projectH.StorageUsage)
				client.GET("/projects", projectH.ListProjects)
				client.POST("/projects", projectH.CreateProject)
				client.GET("/projects/:id/tree", projectH.GetTree)
				client.GET("/projects/:id/nodes", projectH.GetNodes)
				client.GET("/projects/:id/assets", projectH.ListAssets)
				client.POST("/projects/:id/uploads", projectH.Upload)
				client.DELETE("/projects/:id/files/:fileId", projectH.DeleteFile)
				client.GET("/projects/:id/files/:fileId", projectH.GetFile)
				client.POST("/projects/:id/trash", projectH.Trash)
				client.POST("/projects/:id/restore", projectH.Restore)
				client.DELETE("/projects/:id", projectH.DeleteProject)
				client.PUT("/session/current-project", projectH.SetCurrentProject)
				// The quote endpoint belongs to the retired, quote-backed contract.
				// Keep the path for deployed clients, but fail explicitly so new
				// clients cannot accidentally re-enter the legacy billing flow.
				client.POST("/tasks/quote", clientH.LegacyQuoteTask)
				client.POST("/tasks/estimate", clientH.EstimateTask)
				client.POST("/tasks", clientH.CreateTask)
				client.GET("/tasks", clientH.ListTasks)
				client.GET("/tasks/:id", clientH.GetTask)
				client.POST("/tasks/:id/cancel", clientH.CancelTask)
				client.POST("/tasks/:id/save-to-project", clientH.SaveTaskToProject)
				client.GET("/wallet", clientH.Wallet)
				client.GET("/wallet/transactions", clientH.WalletTransactions)
				client.POST("/wallet/redeem", clientH.Redeem)
				client.GET("/tickets", clientH.ListTickets)
				client.POST("/tickets", clientH.CreateTicket)
			}

			admin := authed.Group("/admin")
			admin.Use(middleware.RequireAdmin())
			{
				admin.GET("/carriers", adminH.ListCarriers)
				admin.POST("/carriers", adminH.CreateCarrier)
				admin.PATCH("/carriers/:id", adminH.UpdateCarrier)
				admin.GET("/suppliers", adminH.ListSuppliers)
				admin.POST("/suppliers", adminH.CreateSupplier)
				admin.PATCH("/suppliers/:id", adminH.UpdateSupplier)
				admin.POST("/models", adminH.CreateModel)
				admin.PATCH("/models/:id", adminH.UpdateModel)
				admin.GET("/creation-modes", adminH.ListCreationModes)
				admin.PATCH("/creation-modes/:id", adminH.UpdateCreationMode)
				admin.GET("/model-configurations", adminH.ListModelConfigurations)
				admin.PUT("/model-configurations", adminH.UpsertModelConfiguration)
				admin.GET("/offerings", adminH.ListOfferings)
				admin.POST("/offerings", adminH.CreateOffering)
				admin.PATCH("/offerings/:id", adminH.UpdateOffering)
				admin.GET("/supplier-costs", adminH.ListSupplierCosts)
				admin.PUT("/supplier-costs", adminH.UpsertSupplierCost)
				admin.GET("/retail-pricing", adminH.ListRetailPricing)
				admin.PUT("/retail-pricing", adminH.UpsertRetailPricing)
				admin.GET("/tasks", adminH.ListTasks)
				admin.GET("/tasks/:id", adminH.GetTask)
				admin.POST("/tasks/:id/cancel", adminH.CancelTask)
				admin.POST("/tasks/:id/reconcile", adminH.ReconcileTask)
				admin.GET("/tickets", adminH.ListTickets)
				admin.PATCH("/tickets/:id", adminH.UpdateTicket)
				admin.GET("/audit-logs", adminH.ListAuditLogs)
				admin.GET("/settings", adminH.GetSettings)
				admin.PUT("/settings", adminH.UpdateSettings)
				admin.POST("/settings/test-smtp", adminH.TestSMTP)
			}
		}
	}

	return r
}

func cors(allowed []string) gin.HandlerFunc {
	wildcard := false
	exact := make(map[string]bool, len(allowed))
	for _, o := range allowed {
		if o == "*" {
			wildcard = true
			continue
		}
		exact[o] = true
	}
	return func(c *gin.Context) {
		c.Writer.Header().Add("Vary", "Origin")
		reqOrigin := c.GetHeader("Origin")
		switch {
		case wildcard:
			c.Header("Access-Control-Allow-Origin", "*")
		case reqOrigin != "" && exact[reqOrigin]:
			c.Header("Access-Control-Allow-Origin", reqOrigin)
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,Accept,Idempotency-Key,X-Request-ID")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
