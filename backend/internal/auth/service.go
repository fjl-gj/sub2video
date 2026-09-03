package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/store"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailExists        = errors.New("email already exists")
	ErrUserInactive       = errors.New("user inactive")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenInvalid       = errors.New("token invalid")
	ErrRefreshInvalid     = errors.New("refresh token invalid")
	ErrVerifyRequired     = errors.New("verification code required")
	ErrRegistrationClosed = errors.New("registration closed")
)

type Service struct {
	store  *store.Store
	cfg    config.Config
	secret []byte
	Mail   Mailer
}

type Mailer interface {
	SendVerifyCode(ctx context.Context, email, siteName string) error
	VerifyCode(ctx context.Context, email, code string) error
	SendPasswordReset(ctx context.Context, email, siteName, resetBaseURL string) error
	ConsumeResetToken(ctx context.Context, email, token string) error
	Configured(ctx context.Context) bool
}

type Claims struct {
	UserID       int64  `json:"uid"`
	PublicID     string `json:"pid"`
	Role         string `json:"role"`
	TokenVersion int    `json:"tv"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
	TokenType    string `json:"tokenType"`
}

func NewService(st *store.Store, cfg config.Config) *Service {
	return &Service{store: st, cfg: cfg, secret: []byte(cfg.JWTSecret)}
}

func (s *Service) HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func (s *Service) CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (s *Service) Register(ctx context.Context, email, password, displayName, verifyCode string) (*TokenPair, *store.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || len(password) < 8 {
		return nil, nil, fmt.Errorf("invalid email or password")
	}
	if !s.store.SettingBool(ctx, "registration_enabled", true) {
		return nil, nil, ErrRegistrationClosed
	}
	verifyEnabled := s.store.SettingBool(ctx, "email_verify_enabled", false)
	if verifyEnabled {
		if strings.TrimSpace(verifyCode) == "" {
			return nil, nil, ErrVerifyRequired
		}
		if s.Mail == nil {
			return nil, nil, fmt.Errorf("mail service unavailable")
		}
		if err := s.Mail.VerifyCode(ctx, email, verifyCode); err != nil {
			return nil, nil, err
		}
	}
	exists, err := s.store.EmailExists(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if exists {
		return nil, nil, ErrEmailExists
	}
	hash, err := s.HashPassword(password)
	if err != nil {
		return nil, nil, err
	}
	if displayName == "" {
		displayName = strings.Split(email, "@")[0]
	}
	user, err := s.store.CreateUserWithWallet(ctx, email, hash, displayName, displayName+"工作室", 0)
	if err != nil {
		return nil, nil, err
	}
	if verifyEnabled {
		_ = s.store.MarkEmailVerified(ctx, user.ID)
	}
	pair, err := s.issuePair(ctx, user)
	return pair, user, err
}

func (s *Service) SendVerifyCode(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return fmt.Errorf("email required")
	}
	if !s.store.SettingBool(ctx, "registration_enabled", true) {
		return ErrRegistrationClosed
	}
	if s.Mail == nil {
		return fmt.Errorf("mail service unavailable")
	}
	exists, err := s.store.EmailExists(ctx, email)
	if err != nil {
		return err
	}
	if exists {
		return ErrEmailExists
	}
	return s.Mail.SendVerifyCode(ctx, email, "Sub2Video")
}

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if !s.store.SettingBool(ctx, "password_reset_enabled", false) {
		return nil // silent
	}
	if s.Mail == nil {
		return nil
	}
	_, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil // silent — don't reveal
	}
	front := "http://192.168.0.107:12000"
	if v, _ := s.store.GetSetting(ctx, "frontend_url"); v != "" {
		front = strings.TrimRight(v, "/")
	}
	resetURL := front + "/auth/reset-password"
	return s.Mail.SendPasswordReset(ctx, email, "Sub2Video", resetURL)
}

func (s *Service) ResetPassword(ctx context.Context, email, token, newPassword string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(newPassword) < 8 {
		return fmt.Errorf("password too short")
	}
	if s.Mail == nil {
		return fmt.Errorf("mail service unavailable")
	}
	if err := s.Mail.ConsumeResetToken(ctx, email, token); err != nil {
		return err
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return ErrInvalidCredentials
	}
	hash, err := s.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.store.UpdatePasswordHash(ctx, user.ID, hash)
}

func (s *Service) Login(ctx context.Context, email, password string) (*TokenPair, *store.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}
	if !user.IsActive() {
		return nil, nil, ErrUserInactive
	}
	if !s.CheckPassword(user.PasswordHash, password) {
		return nil, nil, ErrInvalidCredentials
	}
	pair, err := s.issuePair(ctx, user)
	return pair, user, err
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	hash := hashToken(refreshToken)
	userID, err := s.store.FindValidRefreshToken(ctx, hash)
	if err != nil {
		return nil, ErrRefreshInvalid
	}
	_ = s.store.RevokeRefreshToken(ctx, hash)
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive() {
		return nil, ErrUserInactive
	}
	return s.issuePair(ctx, user)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.store.RevokeRefreshToken(ctx, hashToken(refreshToken))
}

func (s *Service) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrTokenInvalid
		}
		return s.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

func (s *Service) issuePair(ctx context.Context, user *store.User) (*TokenPair, error) {
	now := time.Now()
	expires := now.Add(s.cfg.AccessTokenTTL)
	claims := Claims{
		UserID:       user.ID,
		PublicID:     user.PublicID,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.PublicID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
			Issuer:    "sub2video",
			Audience:  []string{"client"},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	access, err := token.SignedString(s.secret)
	if err != nil {
		return nil, err
	}

	refreshRaw, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	refreshExp := now.Add(s.cfg.RefreshTokenTTL)
	if err := s.store.SaveRefreshToken(ctx, user.ID, hashToken(refreshRaw), refreshExp); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refreshRaw,
		ExpiresIn:    int64(s.cfg.AccessTokenTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
