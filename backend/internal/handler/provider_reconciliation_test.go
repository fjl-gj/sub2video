package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/middleware"
	"github.com/sbu2video/backend/internal/store"
)

func TestTaskReconciliationRequiresAdminAndValidatedEvidence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		role, body string
		status     int
	}{
		{role: "user", body: `{"action":"refund","confirmed":true,"evidence":"供应商确认没有创建任务且无任何扣费"}`, status: http.StatusForbidden},
		{role: "admin", body: `{"action":"retry","confirmed":true,"evidence":"供应商确认没有创建任务且无任何扣费"}`, status: http.StatusBadRequest},
		{role: "admin", body: `{"action":"refund","confirmed":false,"evidence":"供应商确认没有创建任务且无任何扣费"}`, status: http.StatusBadRequest},
	} {
		router := gin.New()
		router.POST("/tasks/:id/reconcile", func(c *gin.Context) {
			c.Set(middleware.ContextUser, &store.User{ID: 1, Role: tt.role})
			(&AdminHandler{}).ReconcileTask(c)
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/tasks/task-1/reconcile", strings.NewReader(tt.body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		if w.Code != tt.status {
			t.Fatalf("role=%s got=%d body=%s", tt.role, w.Code, w.Body)
		}
	}
}
