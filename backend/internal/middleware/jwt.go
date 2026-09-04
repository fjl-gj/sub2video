package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/auth"
	"github.com/sbu2video/backend/internal/response"
	"github.com/sbu2video/backend/internal/store"
)

const ContextUserID = "userID"
const ContextUser = "user"

func JWTAuth(authSvc *auth.Service, st *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		header := c.GetHeader("Authorization")
		if header != "" {
			parts := strings.SplitN(header, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				token = strings.TrimSpace(parts[1])
			}
		}
		if token == "" {
			token = strings.TrimSpace(c.Query("access_token"))
		}
		if token == "" {
			response.Fail(c, 401, "UNAUTHORIZED", "Authorization header is required")
			return
		}
		claims, err := authSvc.ValidateAccessToken(token)
		if err != nil {
			if errors.Is(err, auth.ErrTokenExpired) {
				response.Fail(c, 401, "TOKEN_EXPIRED", "Token has expired")
				return
			}
			response.Fail(c, 401, "INVALID_TOKEN", "Invalid token")
			return
		}
		user, err := st.GetUserByID(c.Request.Context(), claims.UserID)
		if err != nil || !user.IsActive() {
			response.Fail(c, 401, "USER_INACTIVE", "User not found or inactive")
			return
		}
		if claims.TokenVersion != user.TokenVersion {
			response.Fail(c, 401, "TOKEN_REVOKED", "Token has been revoked")
			return
		}
		c.Set(ContextUserID, user.ID)
		c.Set(ContextUser, user)
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) int64 {
	v, _ := c.Get(ContextUserID)
	id, _ := v.(int64)
	return id
}

func CurrentUser(c *gin.Context) *store.User {
	v, _ := c.Get(ContextUser)
	u, _ := v.(*store.User)
	return u
}
