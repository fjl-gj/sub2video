package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/response"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := CurrentUser(c)
		if user == nil || user.Role != "admin" {
			response.Fail(c, http.StatusForbidden, "FORBIDDEN", "admin only")
			return
		}
		c.Next()
	}
}
