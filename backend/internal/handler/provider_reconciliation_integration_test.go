package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sbu2video/backend/internal/auth"
	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/server"
	"github.com/sbu2video/backend/internal/store"
)

// Real database/router/auth, with a local provider used only for read-only
// verification of the manual recovery protocol. No paid generation occurs.
func TestTaskReconciliationHandlerIntegration(t *testing.T) {
	st := deliveryIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := st.SeedBasics(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.SeedCreationCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	var queries atomic.Int32
	var mismatched atomic.Bool
	mismatched.Store(true)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries.Add(1)
		if r.Method != http.MethodGet {
			t.Error("reconciliation must not submit")
			w.WriteHeader(500)
			return
		}
		id := "verified-provider-task"
		if mismatched.Load() {
			id = "different-task"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": "Success", "data": map[string]any{"task_id": id, "status": "RUNNING"}})
	}))
	defer provider.Close()
	if _, err := st.Pool.Exec(ctx, `UPDATE suppliers SET token_encrypted='',base_url=$1 WHERE code='autodl'`, provider.URL); err != nil {
		t.Fatal(err)
	}
	admin, err := st.CreateUserWithWallet(ctx, "operator@reconcile.test", "unused", "Operator", "Operator", 100)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Pool.Exec(ctx, `UPDATE users SET role='admin' WHERE id=$1`, admin.ID); err != nil {
		t.Fatal(err)
	}
	user, err := st.CreateUserWithWallet(ctx, "user@reconcile.test", "unused", "User", "User", 100)
	if err != nil {
		t.Fatal(err)
	}
	result, err := st.CreateTaskDirect(ctx, user.ID, store.DirectTaskInput{DirectRouteRequest: store.DirectRouteRequest{ModeID: "mode_text_to_video", ModelID: "mdl_h3", Resolution: "768P", AspectRatio: "16:9", DurationSeconds: 5, SupplierTokenFallback: "isolated-test-token"}, ClientRequestID: "handler-reconcile", Prompt: "测试", Title: "核对"})
	if err != nil {
		t.Fatal(err)
	}
	var taskID int64
	if err := st.Pool.QueryRow(ctx, `SELECT id FROM generation_tasks WHERE public_id=$1`, result.Task.ID).Scan(&taskID); err != nil {
		t.Fatal(err)
	}
	claimed, claim, err := st.ClaimTaskForDispatchWithToken(ctx, taskID)
	if err != nil || !claimed {
		t.Fatalf("claim: %v %v", claimed, err)
	}
	if err := st.MarkTaskSubmitStarted(ctx, taskID, claim); err != nil {
		t.Fatal(err)
	}
	if err := st.MarkTaskSubmissionUnknown(ctx, taskID, "lost response", claim); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{JWTSecret: "reconcile-test-only", AutoDLToken: "isolated-test-token", AutoDLBaseURL: provider.URL}
	router := server.NewRouter(cfg, st, auth.NewService(st, cfg), nil)
	adminToken, userToken := deliveryTestToken(t, admin, cfg.JWTSecret), deliveryTestToken(t, user, cfg.JWTSecret)
	post := func(token string) *httptest.ResponseRecorder {
		body := `{"action":"resume","upstreamTaskId":"verified-provider-task","confirmed":true,"evidence":"已核对供应商后台记录的任务输入和创建时间"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tasks/"+result.Task.ID+"/reconcile", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	if res := post(userToken); res.Code != http.StatusForbidden || queries.Load() != 0 {
		t.Fatalf("user recovery allowed: %d %s", res.Code, res.Body)
	}
	if res := post(adminToken); res.Code != http.StatusConflict {
		t.Fatalf("mismatched provider ID accepted: %d %s", res.Code, res.Body)
	}
	mismatched.Store(false)
	if res := post(adminToken); res.Code != http.StatusOK {
		t.Fatalf("verified resume failed: %d %s", res.Code, res.Body)
	}
	provider.Close()
	before := queries.Load()
	if res := post(adminToken); res.Code != http.StatusOK || queries.Load() != before || !strings.Contains(res.Body.String(), `"alreadyResolved":true`) {
		t.Fatalf("replay should avoid provider: %d %s", res.Code, res.Body)
	}
	var audits, refunds int
	if err := st.Pool.QueryRow(ctx, `SELECT (SELECT COUNT(*) FROM admin_audit_logs WHERE target_id=$1 AND action='task.reconcile'),(SELECT COUNT(*) FROM wallet_ledger WHERE ref_id=$1 AND type='task_refund')`, result.Task.ID).Scan(&audits, &refunds); err != nil {
		t.Fatal(err)
	}
	if audits != 1 || refunds != 0 {
		t.Fatalf("audit=%d refund=%d", audits, refunds)
	}
}
