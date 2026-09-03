package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/auth"
	"github.com/sbu2video/backend/internal/mail"
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
	VerifyCode  string `json:"verifyCode"`
}

type loginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshReq struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

func (h *AuthHandler) PublicSettings(c *gin.Context) {
	m, _ := h.Store.GetSettings(c.Request.Context(), []string{
		"email_verify_enabled", "password_reset_enabled", "registration_enabled",
	})
	response.OK(c, gin.H{
		"emailVerifyEnabled":    m["email_verify_enabled"] == "true",
		"passwordResetEnabled":  m["password_reset_enabled"] == "true",
		"registrationEnabled":   m["registration_enabled"] != "false",
	})
}

func (h *AuthHandler) SendVerifyCode(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if !h.Store.SettingBool(c.Request.Context(), "email_verify_enabled", false) {
		response.Fail(c, http.StatusBadRequest, "VERIFY_DISABLED", "邮箱验证未开启")
		return
	}
	err := h.Auth.SendVerifyCode(c.Request.Context(), req.Email)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailExists):
			response.Fail(c, http.StatusConflict, "EMAIL_EXISTS", "邮箱已被注册")
		case errors.Is(err, auth.ErrRegistrationClosed):
			response.Fail(c, http.StatusForbidden, "REGISTRATION_CLOSED", "暂停注册")
		case errors.Is(err, mail.ErrVerifyCodeTooFrequent):
			response.Fail(c, http.StatusTooManyRequests, "TOO_FREQUENT", "请稍后再试")
		case errors.Is(err, mail.ErrEmailNotConfigured):
			response.Fail(c, http.StatusServiceUnavailable, "EMAIL_NOT_CONFIGURED", "邮件服务未配置")
		default:
			response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		}
		return
	}
	response.OK(c, gin.H{"countdown": 60})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	pair, user, err := h.Auth.Register(c.Request.Context(), req.Email, req.Password, req.DisplayName, req.VerifyCode)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailExists):
			response.Fail(c, http.StatusConflict, "EMAIL_EXISTS", "邮箱已被注册")
		case errors.Is(err, auth.ErrVerifyRequired):
			response.Fail(c, http.StatusBadRequest, "VERIFY_REQUIRED", "请填写邮箱验证码")
		case errors.Is(err, auth.ErrRegistrationClosed):
			response.Fail(c, http.StatusForbidden, "REGISTRATION_CLOSED", "暂停注册")
		case errors.Is(err, mail.ErrInvalidVerifyCode):
			response.Fail(c, http.StatusBadRequest, "INVALID_VERIFY_CODE", "验证码无效或已过期")
		case errors.Is(err, mail.ErrVerifyCodeMaxAttempts):
			response.Fail(c, http.StatusTooManyRequests, "VERIFY_MAX_ATTEMPTS", "验证码尝试过多，请重新获取")
		default:
			response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		}
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

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	_ = h.Auth.ForgotPassword(c.Request.Context(), req.Email)
	response.OK(c, gin.H{"status": "ok"})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required,email"`
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	err := h.Auth.ResetPassword(c.Request.Context(), req.Email, req.Token, req.NewPassword)
	if err != nil {
		if errors.Is(err, mail.ErrInvalidResetToken) {
			response.Fail(c, http.StatusBadRequest, "INVALID_TOKEN", "重置链接无效或已过期")
			return
		}
		response.Fail(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"status": "ok"})
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
