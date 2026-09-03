package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/auth"
	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/handler"
	"github.com/sbu2video/backend/internal/middleware"
	"github.com/sbu2video/backend/internal/store"
)

func NewRouter(cfg config.Config, st *store.Store, authSvc *auth.Service) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(cors(cfg.CORSOrigin))
	r.MaxMultipartMemory = 16 << 20

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC()})
	})

	authH := &handler.AuthHandler{Auth: authSvc, Store: st}
	clientH := &handler.ClientHandler{Store: st}
	uploadH := &handler.UploadHandler{Store: st, UploadDir: cfg.UploadDir}
	adminH := &handler.AdminHandler{Store: st, Cfg: cfg}

	v1 := r.Group("/api/v1")
	{
		a := v1.Group("/auth")
		{
			a.POST("/register", authH.Register)
			a.POST("/login", authH.Login)
			a.POST("/refresh", authH.Refresh)
			a.POST("/logout", authH.Logout)
		}

		authed := v1.Group("")
		authed.Use(middleware.JWTAuth(authSvc, st))
		{
			authed.GET("/auth/me", authH.Me)

			client := authed.Group("/client")
			{
				client.GET("/overview", clientH.Overview)
				client.GET("/create/form", clientH.CreateForm)
				client.POST("/uploads", uploadH.Create)
				client.GET("/uploads/:id", uploadH.Get)
				client.POST("/tasks/quote", clientH.QuoteTask)
				client.POST("/tasks", clientH.CreateTask)
				client.GET("/tasks", clientH.ListTasks)
				client.POST("/tasks/:id/cancel", clientH.CancelTask)
				client.POST("/tasks/:id/retry", clientH.RetryTask)
				client.GET("/wallet", clientH.Wallet)
				client.GET("/wallet/transactions", clientH.WalletTransactions)
				client.POST("/wallet/redeem", clientH.Redeem)
			}

			admin := authed.Group("/admin")
			admin.Use(middleware.RequireAdmin())
			{
				admin.GET("/carriers", adminH.ListCarriers)
				admin.GET("/suppliers", adminH.ListSuppliers)
				admin.PATCH("/suppliers/:id", adminH.UpdateSupplier)
				admin.GET("/routes", adminH.ListRoutes)
				admin.PATCH("/routes/:id", adminH.UpdateRoute)
			}
		}
	}

	return r
}

func cors(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,Accept")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
