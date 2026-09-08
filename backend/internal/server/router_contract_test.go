package server

import (
	"testing"

	"github.com/sbu2video/backend/internal/auth"
	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/store"
)

func TestRouterDoesNotExposeLegacyWorkflowSupplyOrPricingEndpoints(t *testing.T) {
	cfg := config.Config{JWTSecret: "router-test", CORSOrigin: "*"}
	router := NewRouter(cfg, &store.Store{}, auth.NewService(&store.Store{}, cfg), nil)
	legacy := map[string]bool{
		"/api/v1/client/create":                     true,
		"/api/v1/admin/routes":                      true,
		"/api/v1/admin/workflows":                   true,
		"/api/v1/admin/pricing":                     true,
		"/api/v1/admin/creation-modes/:id/policies": true,
		"/api/v1/admin/tasks/:id/retry":             true,
	}
	for _, route := range router.Routes() {
		if legacy[route.Path] {
			t.Fatalf("legacy route is still registered: %s %s", route.Method, route.Path)
		}
	}
}
