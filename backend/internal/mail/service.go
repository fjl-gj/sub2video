package mail

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"math/big"
	"net"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sbu2video/backend/internal/store"
)

var (
	ErrEmailNotConfigured    = errors.New("email not configured")
	ErrInvalidVerifyCode     = errors.New("invalid or expired verification code")
	ErrVerifyCodeTooFrequent = errors.New("please wait before requesting a new code")
	ErrVerifyCodeMaxAttempts = errors.New("too many failed attempts")
	ErrInvalidResetToken     = errors.New("invalid or expired reset token")
)

const (
	verifyCodeTTL         = 15 * time.Minute
	verifyCodeCooldown    = 1 * time.Minute
	maxVerifyCodeAttempts = 5
	passwordResetTokenTTL = 30 * time.Minute
	passwordResetCooldown = 30 * time.Second
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	UseTLS   bool
}

type CodeData struct {
	Code      string    `json:"code"`
	Attempts  int       `json:"attempts"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ResetTokenData struct {
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
}

type Service struct {
	Store *store.Store
	Redis *redis.Client
}

func (s *Service) GetSMTPConfig(ctx context.Context) (*SMTPConfig, error) {
	m, err := s.Store.GetSettings(ctx, []string{
		"smtp_host", "smtp_port", "smtp_username", "smtp_password",
		"smtp_from", "smtp_from_name", "smtp_use_tls",
	})
	if err != nil {
		return nil, err
	}
	host := strings.TrimSpace(m["smtp_host"])
	if host == "" {
		return nil, ErrEmailNotConfigured
	}
	port := 587
	if p, err := strconv.Atoi(m["smtp_port"]); err == nil && p > 0 {
		port = p
	}
	return &SMTPConfig{
		Host: host, Port: port,
		Username: strings.TrimSpace(m["smtp_username"]),
		Password: strings.TrimSpace(m["smtp_password"]),
		From:     strings.TrimSpace(m["smtp_from"]),
		FromName: strings.TrimSpace(m["smtp_from_name"]),
		UseTLS:   m["smtp_use_tls"] == "true",
	}, nil
}

func (s *Service) Configured(ctx context.Context) bool {
	_, err := s.GetSMTPConfig(ctx)
	return err == nil
}

func (s *Service) SendVerifyCode(ctx context.Context, email, siteName string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	key := "verify_code:" + email
	if raw, err := s.Redis.Get(ctx, key).Result(); err == nil {
		var existing CodeData
		if json.Unmarshal([]byte(raw), &existing) == nil && time.Since(existing.CreatedAt) < verifyCodeCooldown {
			return ErrVerifyCodeTooFrequent
		}
	}
	code, err := generateDigits(6)
	if err != nil {
		return err
	}
	data := CodeData{Code: code, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(verifyCodeTTL)}
	b, _ := json.Marshal(data)
	if err := s.Redis.Set(ctx, key, b, verifyCodeTTL).Err(); err != nil {
		return err
	}
	subject := fmt.Sprintf("[%s] 邮箱验证码", siteName)
	body := verifyEmailHTML(siteName, code)
	if err := s.Send(ctx, email, subject, body); err != nil {
		if errors.Is(err, ErrEmailNotConfigured) {
			DevLogCode(email, code)
			return nil
		}
		return err
	}
	return nil
}

func (s *Service) VerifyCode(ctx context.Context, email, code string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	key := "verify_code:" + email
	raw, err := s.Redis.Get(ctx, key).Result()
	if err != nil {
		return ErrInvalidVerifyCode
	}
	var data CodeData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ErrInvalidVerifyCode
	}
	if data.Attempts >= maxVerifyCodeAttempts {
		return ErrVerifyCodeMaxAttempts
	}
	if subtle.ConstantTimeCompare([]byte(data.Code), []byte(strings.TrimSpace(code))) != 1 {
		data.Attempts++
		remaining := time.Until(data.ExpiresAt)
		if remaining > 0 {
			b, _ := json.Marshal(data)
			_ = s.Redis.Set(ctx, key, b, remaining).Err()
		}
		if data.Attempts >= maxVerifyCodeAttempts {
			return ErrVerifyCodeMaxAttempts
		}
		return ErrInvalidVerifyCode
	}
	_ = s.Redis.Del(ctx, key).Err()
	return nil
}

func (s *Service) SendPasswordReset(ctx context.Context, email, siteName, resetBaseURL string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	coolKey := "password_reset_sent:" + email
	if n, _ := s.Redis.Exists(ctx, coolKey).Result(); n > 0 {
		return nil
	}
	tokenKey := "password_reset:" + email
	token := ""
	if raw, err := s.Redis.Get(ctx, tokenKey).Result(); err == nil {
		var existing ResetTokenData
		if json.Unmarshal([]byte(raw), &existing) == nil {
			token = existing.Token
		}
	}
	if token == "" {
		t, err := randomHex(32)
		if err != nil {
			return err
		}
		token = t
		b, _ := json.Marshal(ResetTokenData{Token: token, CreatedAt: time.Now()})
		if err := s.Redis.Set(ctx, tokenKey, b, passwordResetTokenTTL).Err(); err != nil {
			return err
		}
	}
	full := fmt.Sprintf("%s?email=%s&token=%s", strings.TrimRight(resetBaseURL, "/"), url.QueryEscape(email), url.QueryEscape(token))
	subject := fmt.Sprintf("[%s] 密码重置", siteName)
	body := resetEmailHTML(siteName, full)
	if err := s.Send(ctx, email, subject, body); err != nil {
		return err
	}
	_ = s.Redis.Set(ctx, coolKey, "1", passwordResetCooldown).Err()
	return nil
}

func (s *Service) ConsumeResetToken(ctx context.Context, email, token string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	key := "password_reset:" + email
	raw, err := s.Redis.Get(ctx, key).Result()
	if err != nil {
		return ErrInvalidResetToken
	}
	var data ResetTokenData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ErrInvalidResetToken
	}
	if subtle.ConstantTimeCompare([]byte(data.Token), []byte(token)) != 1 {
		return ErrInvalidResetToken
	}
	_ = s.Redis.Del(ctx, key).Err()
	return nil
}

func (s *Service) Send(ctx context.Context, to, subject, body string) error {
	cfg, err := s.GetSMTPConfig(ctx)
	if err != nil {
		return err
	}
	return SendWithConfig(cfg, to, subject, body)
}

func (s *Service) TestConnection(ctx context.Context) error {
	cfg, err := s.GetSMTPConfig(ctx)
	if err != nil {
		return err
	}
	client, err := connectSMTP(cfg)
	if err != nil {
		return err
	}
	defer client.Close()
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	_ = client.Quit()
	return nil
}

func SendWithConfig(cfg *SMTPConfig, to, subject, body string) error {
	from := cfg.From
	if from == "" {
		from = cfg.Username
	}
	headers := []string{
		fmt.Sprintf("From: %s <%s>", cfg.FromName, from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		body,
	}
	msg := []byte(strings.Join(headers, "\r\n"))
	client, err := connectSMTP(cfg)
	if err != nil {
		return err
	}
	defer client.Close()
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	_ = client.Quit()
	return nil
}

func connectSMTP(cfg *SMTPConfig) (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	tlsConfig := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
	if cfg.UseTLS {
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		if err == nil {
			return smtp.NewClient(conn, cfg.Host)
		}
		var recordErr tls.RecordHeaderError
		if !errors.As(err, &recordErr) {
			return nil, err
		}
		return connectStartTLS(dialer, addr, cfg.Host, tlsConfig, true)
	}
	return connectStartTLS(dialer, addr, cfg.Host, tlsConfig, false)
}

func connectStartTLS(dialer *net.Dialer, addr, host string, tlsConfig *tls.Config, mandatory bool) (*smtp.Client, error) {
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(tlsConfig); err != nil {
			_ = client.Close()
			return nil, err
		}
	} else if mandatory {
		_ = client.Close()
		return nil, errors.New("smtp server does not support STARTTLS")
	}
	return client, nil
}

func generateDigits(n int) (string, error) {
	const digits = "0123456789"
	out := make([]byte, n)
	for i := range out {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		out[i] = digits[num.Int64()]
	}
	return string(out), nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func verifyEmailHTML(site, code string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><body style="font-family:sans-serif">
<h2>%s</h2><p>你的验证码是：</p>
<p style="font-size:28px;letter-spacing:6px;font-weight:bold">%s</p>
<p>15 分钟内有效。如非本人操作请忽略。</p></body></html>`, html.EscapeString(site), code)
}

func resetEmailHTML(site, link string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><body style="font-family:sans-serif">
<h2>%s</h2><p>点击下方链接重置密码（30 分钟内有效）：</p>
<p><a href="%s">%s</a></p>
<p>如非本人操作请忽略。</p></body></html>`, html.EscapeString(site), html.EscapeString(link), html.EscapeString(link))
}

// DevLogCode logs verify code when SMTP missing (UAT only helper).
func DevLogCode(email, code string) {
	log.Printf("DEV verify code for %s: %s", email, code)
}
