package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sbu2video/backend/internal/auth"
	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/store"
)

func TestCORSAllowListEchoesOnlyConfiguredOrigins(t *testing.T) {
	cfg := config.Config{JWTSecret: "cors-test", CORSOrigin: "https://app.example.com, https://admin.example.com"}
	router := NewRouter(cfg, &store.Store{}, auth.NewService(&store.Store{}, cfg), nil)

	allowed := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://app.example.com")
	router.ServeHTTP(allowed, req)
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("allowed origin not echoed, got %q", got)
	}
	if got := allowed.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary must include Origin for cache correctness, got %q", got)
	}

	denied := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://evil.example.net")
	router.ServeHTTP(denied, req)
	if got := denied.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("foreign origin must not receive ACAO, got %q", got)
	}
}

func TestCORSWildcardStillAvailableForDevelopment(t *testing.T) {
	cfg := config.Config{JWTSecret: "cors-test", CORSOrigin: "*"}
	router := NewRouter(cfg, &store.Store{}, auth.NewService(&store.Store{}, cfg), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	router.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("development wildcard changed, got %q", got)
	}
}

func TestCORSPreflightReturnsNoContent(t *testing.T) {
	cfg := config.Config{JWTSecret: "cors-test", CORSOrigin: "https://app.example.com"}
	router := NewRouter(cfg, &store.Store{}, auth.NewService(&store.Store{}, cfg), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/me", nil)
	req.Header.Set("Origin", "https://app.example.com")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
}
