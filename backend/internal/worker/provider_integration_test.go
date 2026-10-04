package worker

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/db"
	"github.com/sbu2video/backend/internal/store"
)

// Uses real PostgreSQL transactions and an explicitly fake HTTP provider to
// inject acceptance/persistence failures. This is not supplier UAT evidence.
func TestWorkerSubmissionUncertaintyIntegration(t *testing.T) {
	if os.Getenv("SUB2VIDEO_RUN_INTEGRATION") != "1" {
		t.Skip("set SUB2VIDEO_RUN_INTEGRATION=1 and SUB2VIDEO_TEST_DATABASE_URL")
	}
	dsn := strings.TrimSpace(os.Getenv("SUB2VIDEO_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("SUB2VIDEO_TEST_DATABASE_URL is not configured")
	}
	u, err := url.Parse(dsn)
	if err != nil || !strings.Contains(strings.ToLower(u.Path), "test") {
		t.Fatal("test database name must contain test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if err := admin.Ping(ctx); err != nil {
		t.Skipf("PostgreSQL unavailable: %v", err)
	}
	schemaName := "worker_" + strings.ReplaceAll(store.NewPublicID(), "-", "")
	schema := pgx.Identifier{schemaName}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		_, _ = admin.Exec(cleanup, "DROP SCHEMA "+schema+" CASCADE")
	}()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schemaName
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	st := store.New(pool)
	if err := st.SeedBasics(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.SeedCreationCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	var submitCount atomic.Int32
	var queryCount atomic.Int32
	var responseStatus atomic.Int32
	var lastSubmitPath atomic.Value
	var pollStatus atomic.Value
	pollStatus.Store("RUNNING")
	responseStatus.Store(http.StatusOK)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			queryCount.Add(1)
			_, _ = fmt.Fprintf(w, `{"code":"Success","data":{"task_id":"accepted-provider-1","status":%q,"results":[]}}`, pollStatus.Load().(string))
			return
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected provider request %s", r.Method)
			w.WriteHeader(500)
			return
		}
		submitCount.Add(1)
		lastSubmitPath.Store(r.URL.Path)
		w.WriteHeader(int(responseStatus.Load()))
		_, _ = w.Write([]byte(`{"code":"Success","data":{"task_id":"accepted-provider-1","status":"QUEUED"}}`))
	}))
	defer provider.Close()
	if _, err := pool.Exec(ctx, `UPDATE suppliers SET token_encrypted='',base_url=$1 WHERE code='autodl'`, provider.URL); err != nil {
		t.Fatal(err)
	}
	runner := &Runner{Store: st, Cfg: config.Config{AutoDLToken: "isolated-fake-provider", AutoDLBaseURL: provider.URL}}
	for i, scenario := range []string{"accepted response but database write fails", "HTTP 408 does not prove rejection"} {
		t.Run(scenario, func(t *testing.T) {
			user, err := st.CreateUserWithWallet(ctx, fmt.Sprintf("worker-%d@example.test", i), "unused", "worker", "worker", 100)
			if err != nil {
				t.Fatal(err)
			}
			out, err := st.CreateTaskDirect(ctx, user.ID, store.DirectTaskInput{DirectRouteRequest: store.DirectRouteRequest{ModeID: "mode_text_to_video", ModelID: "mdl_h3", DurationSeconds: 5, Resolution: "768P", AspectRatio: "16:9", SupplierTokenFallback: "isolated-fake-provider"}, ClientRequestID: "worker-" + store.NewPublicID(), Prompt: "测试镜头", Title: "故障注入"})
			if err != nil {
				t.Fatal(err)
			}
			if i == 0 {
				if _, err := pool.Exec(ctx, `CREATE FUNCTION reject_provider_id() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.upstream_task_id<>'' THEN RAISE EXCEPTION 'injected persistence failure'; END IF; RETURN NEW; END $$;
CREATE TRIGGER reject_provider_id BEFORE UPDATE ON generation_tasks FOR EACH ROW EXECUTE FUNCTION reject_provider_id()`); err != nil {
					t.Fatal(err)
				}
			} else {
				responseStatus.Store(http.StatusRequestTimeout)
			}
			before := submitCount.Load()
			if err := runner.Tick(ctx); err != nil {
				t.Fatal(err)
			}
			if err := runner.Tick(ctx); err != nil {
				t.Fatal(err)
			}
			var status, upstream, providerID string
			var uncertain bool
			var balance float64
			var refunds int
			if err := pool.QueryRow(ctx, `SELECT t.status,t.upstream_status,t.submit_uncertain,t.provider_request_id,w.balance::float8,(SELECT COUNT(*) FROM wallet_ledger WHERE ref_id=t.public_id AND type='task_refund') FROM generation_tasks t JOIN wallets w ON w.user_id=t.user_id WHERE t.public_id=$1`, out.Task.ID).Scan(&status, &upstream, &uncertain, &providerID, &balance, &refunds); err != nil {
				t.Fatal(err)
			}
			if submitCount.Load() != before+1 || status != "running" || upstream != "SUBMIT_UNKNOWN" || !uncertain || balance != out.Balance || refunds != 0 {
				t.Fatalf("unsafe result: submits=%d status=%s upstream=%s unknown=%v balance=%v refund=%d", submitCount.Load()-before, status, upstream, uncertain, balance, refunds)
			}
			if providerID != "accepted-provider-1" {
				t.Fatalf("lost provider ID received with acceptance/error: %s", providerID)
			}
			if i == 0 {
				if _, err := pool.Exec(ctx, `DROP TRIGGER reject_provider_id ON generation_tasks`); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	// Reuse this isolated schema to exercise the actual Runner transitions,
	// instead of only calling store state setters and claiming worker coverage.
	responseStatus.Store(http.StatusOK)
	newTask := func(t *testing.T, label string) *store.DirectTaskResult {
		t.Helper()
		user, err := st.CreateUserWithWallet(ctx, label+"@example.test", "unused", "worker", "worker", 100)
		if err != nil {
			t.Fatal(err)
		}
		out, err := st.CreateTaskDirect(ctx, user.ID, store.DirectTaskInput{DirectRouteRequest: store.DirectRouteRequest{ModeID: "mode_text_to_video", ModelID: "mdl_h3", DurationSeconds: 5, Resolution: "768P", AspectRatio: "16:9", SupplierTokenFallback: "isolated-fake-provider"}, ClientRequestID: label, Prompt: "测试镜头", Title: label})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	assertRefund := func(t *testing.T, task *store.DirectTaskResult) {
		t.Helper()
		var status, orderStatus string
		var refunds int
		var balance, refunded, charge float64
		if err := pool.QueryRow(ctx, `SELECT t.status,o.status,w.balance::float8,t.cost::float8,
 (SELECT COUNT(*) FROM wallet_ledger WHERE ref_id=t.public_id AND type='task_refund'),
 (SELECT COALESCE(SUM(amount),0)::float8 FROM wallet_ledger WHERE ref_id=t.public_id AND type='task_refund')
 FROM generation_tasks t JOIN wallets w ON w.user_id=t.user_id JOIN consumption_orders o ON o.task_id=t.id WHERE t.public_id=$1`, task.Task.ID).Scan(&status, &orderStatus, &balance, &charge, &refunds, &refunded); err != nil {
			t.Fatal(err)
		}
		if status != "failed" || orderStatus != "refunded" || balance != 100 || refunds != 1 || refunded != charge {
			t.Fatalf("refund mismatch status=%s order=%s balance=%v refunds=%d refunded=%v charge=%v", status, orderStatus, balance, refunds, refunded, charge)
		}
		maskID := func(id string) string {
			if len(id) <= 8 {
				return "redacted"
			}
			return id[:4] + "..." + id[len(id)-4:]
		}
		t.Logf("billing evidence task=%s order=%s amount=%.2f balanceBefore=100.00 balanceAfterDebit=%.2f balanceAfterRefund=%.2f refundCount=%d orderStatus=%s", maskID(task.Task.ID), maskID(task.OrderID), charge, task.Balance, balance, refunds, orderStatus)
	}
	for _, upstream := range []string{"RUNNING", "WAITING_STORAGE"} {
		t.Run("runner timeout "+upstream, func(t *testing.T) {
			out := newTask(t, "timeout-"+strings.ToLower(upstream))
			var id int64
			if err := pool.QueryRow(ctx, `SELECT id FROM generation_tasks WHERE public_id=$1`, out.Task.ID).Scan(&id); err != nil {
				t.Fatal(err)
			}
			claimed, token, err := st.ClaimTaskForDispatchWithToken(ctx, id)
			if err != nil || !claimed {
				t.Fatalf("claim=%v err=%v", claimed, err)
			}
			if err := st.MarkTaskSubmitStarted(ctx, id, token); err != nil {
				t.Fatal(err)
			}
			if err := st.MarkTaskSubmittedWithToken(ctx, id, "timeout-provider-"+upstream, "RUNNING", token); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `UPDATE generation_tasks SET execution_deadline=NOW()-INTERVAL '1 second',next_poll_at=NOW()-INTERVAL '1 second',upstream_status=$2::text,result_url=CASE WHEN $2::text='WAITING_STORAGE' THEN 'https://provider.test/video.mp4' ELSE '' END WHERE id=$1`, id, upstream); err != nil {
				t.Fatal(err)
			}
			beforeSubmit, beforeQuery := submitCount.Load(), queryCount.Load()
			for i := 0; i < 2; i++ {
				if err := runner.Tick(ctx); err != nil {
					t.Fatal(err)
				}
			}
			assertRefund(t, out)
			var failure string
			if err := pool.QueryRow(ctx, `SELECT fail_reason FROM generation_tasks WHERE id=$1`, id).Scan(&failure); err != nil {
				t.Fatal(err)
			}
			if upstream == "WAITING_STORAGE" && failure != "对象存储交付等待超时" {
				t.Fatalf("storage timeout misreported: %s", failure)
			}
			if submitCount.Load() != beforeSubmit || queryCount.Load() != beforeQuery {
				t.Fatal("expired task should not make a provider request")
			}
		})
	}
	t.Run("runner submits snapshot and refunds failed poll once", func(t *testing.T) {
		out := newTask(t, "snapshot-route")
		var executionID, workflowCode, costSnapshot string
		if err := pool.QueryRow(ctx, `SELECT supplier_route_id,execution_snapshot->>'workflowCode',cost_snapshot::text FROM generation_tasks WHERE public_id=$1`, out.Task.ID).Scan(&executionID, &workflowCode, &costSnapshot); err != nil {
			t.Fatal(err)
		}
		if workflowCode == "" {
			t.Fatal("expected locked workflow")
		}
		if _, err := pool.Exec(ctx, `UPDATE supplier_executions SET workflow_code='changed-after-task-created' WHERE public_id=$1`, executionID); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE supplier_cost_rules SET price_per_sec_peak=price_per_sec_peak+100,price_per_sec_offpeak=price_per_sec_offpeak+100 WHERE execution_id=(SELECT id FROM supplier_executions WHERE public_id=$1)`, executionID); err != nil {
			t.Fatal(err)
		}
		before := submitCount.Load()
		if err := runner.Tick(ctx); err != nil {
			t.Fatal(err)
		}
		if submitCount.Load() != before+1 || lastSubmitPath.Load().(string) != "/api/v1/comfyui/comfyui_workflow/"+workflowCode {
			t.Fatalf("worker used changed execution instead of snapshot: %v", lastSubmitPath.Load())
		}
		var afterCost string
		if err := pool.QueryRow(ctx, `SELECT cost_snapshot::text FROM generation_tasks WHERE public_id=$1`, out.Task.ID).Scan(&afterCost); err != nil || afterCost != costSnapshot {
			t.Fatalf("cost snapshot mutated: %v", err)
		}
		pollStatus.Store("FAILED")
		if _, err := pool.Exec(ctx, `UPDATE generation_tasks SET next_poll_at=NOW()-INTERVAL '1 second' WHERE public_id=$1`, out.Task.ID); err != nil {
			t.Fatal(err)
		}
		beforeQueries := queryCount.Load()
		for i := 0; i < 2; i++ {
			if err := runner.Tick(ctx); err != nil {
				t.Fatal(err)
			}
		}
		assertRefund(t, out)
		if queryCount.Load() != beforeQueries+1 || submitCount.Load() != before+1 {
			t.Fatal("terminal provider failure was polled/submitted again")
		}
	})
}
