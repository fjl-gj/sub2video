package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/auth"
	"github.com/sbu2video/backend/internal/middleware"
	"github.com/sbu2video/backend/internal/response"
	"github.com/sbu2video/backend/internal/store"
)

type AuthHandler struct {
	Auth  *auth.Service
	Store *store.Store
}

type registerReq struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"displayName"`
}

type loginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshReq struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	pair, user, err := h.Auth.Register(c.Request.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		if errors.Is(err, auth.ErrEmailExists) {
			response.Fail(c, http.StatusConflict, "EMAIL_EXISTS", "邮箱已被注册")
			return
		}
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	ws, _ := h.Store.GetWorkspaceName(c.Request.Context(), user.ID)
	bal, _ := h.Store.GetBalance(c.Request.Context(), user.ID)
	response.Created(c, gin.H{
		"token": pair,
		"user": gin.H{
			"id": user.PublicID, "email": user.Email, "displayName": user.DisplayName,
			"role": user.Role, "workspaceName": ws, "balance": bal,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	pair, user, err := h.Auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			response.Fail(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "邮箱或密码错误")
		case errors.Is(err, auth.ErrUserInactive):
			response.Fail(c, http.StatusForbidden, "USER_INACTIVE", "账号已冻结")
		default:
			response.Fail(c, http.StatusInternalServerError, "INTERNAL", err.Error())
		}
		return
	}
	ws, _ := h.Store.GetWorkspaceName(c.Request.Context(), user.ID)
	bal, _ := h.Store.GetBalance(c.Request.Context(), user.ID)
	response.OK(c, gin.H{
		"token": pair,
		"user": gin.H{
			"id": user.PublicID, "email": user.Email, "displayName": user.DisplayName,
			"role": user.Role, "workspaceName": ws, "balance": bal,
		},
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	pair, err := h.Auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Fail(c, http.StatusUnauthorized, "INVALID_REFRESH", "刷新令牌无效")
		return
	}
	response.OK(c, gin.H{"token": pair})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req refreshReq
	_ = c.ShouldBindJSON(&req)
	_ = h.Auth.Logout(c.Request.Context(), req.RefreshToken)
	response.OK(c, gin.H{"status": "ok"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	user := middleware.CurrentUser(c)
	ws, _ := h.Store.GetWorkspaceName(c.Request.Context(), user.ID)
	bal, _ := h.Store.GetBalance(c.Request.Context(), user.ID)
	response.OK(c, gin.H{
		"id": user.PublicID, "email": user.Email, "displayName": user.DisplayName,
		"role": user.Role, "workspaceName": ws, "balance": bal,
	})
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
