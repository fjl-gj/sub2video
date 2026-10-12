package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// This suite exercises actual PostgreSQL queries across every public discovery
// surface. It deliberately does not replace a real supplier acceptance test.
func TestDirectAvailabilityGatesStayConsistentIntegration(t *testing.T) {
	// This matrix shares migration/seed setup across all availability gates.
	env := openIntegrationStoreWithTimeout(t, 3*time.Minute)
	textMode, _, _, modelID := testCatalogIDs(t, env)
	assertAvailability := func(t *testing.T, fallback string, available bool) {
		t.Helper()
		catalog, err := env.store.ListClientCreationCatalogWithFallback(env.ctx, fallback)
		if err != nil {
			t.Fatalf("catalog: %v", err)
		}
		if (len(catalog.Models) > 0) != available || (len(catalog.Carriers) > 0) != available {
			t.Errorf("catalog availability=%v; models=%d carriers=%d", available, len(catalog.Models), len(catalog.Carriers))
		}
		options, err := env.store.ResolveCreationOptionsWithFallback(env.ctx, textMode, modelID, fallback)
		if (err == nil && options != nil && len(options.Resolutions) > 0) != available {
			t.Errorf("options availability=%v; options=%+v err=%v", available, options, err)
		}
		recommended, err := env.store.recommendedModelIDWithFallback(env.ctx, textMode, fallback)
		if (err == nil && recommended == modelID) != available {
			t.Errorf("recommendation availability=%v; model=%q err=%v", available, recommended, err)
		}
		request := seededDirectRequest(textMode, modelID)
		request.SupplierTokenFallback = fallback
		_, err = env.store.ResolveDirectRoute(env.ctx, request)
		if (err == nil) != available {
			t.Errorf("route availability=%v; err=%v", available, err)
		}
		_, priced, err := env.store.CheapestAvailableRetailPriceWithFallback(env.ctx, fallback)
		if err != nil || priced != available {
			t.Errorf("overview availability=%v; priced=%v err=%v", available, priced, err)
		}
	}
	assertAvailability(t, "", true)
	for _, gate := range []struct{ name, disable, restore string }{
		{"carrier", `UPDATE carriers SET enabled=FALSE`, `UPDATE carriers SET enabled=TRUE`},
		{"model", `UPDATE models SET enabled=FALSE`, `UPDATE models SET enabled=TRUE`},
		{"mode", `UPDATE creation_modes SET status='draft'`, `UPDATE creation_modes SET status='published'`},
		{"policy", `UPDATE mode_model_policies SET enabled=FALSE`, `UPDATE mode_model_policies SET enabled=TRUE`},
		{"supplier_enabled", `UPDATE suppliers SET enabled=FALSE`, `UPDATE suppliers SET enabled=TRUE`},
		{"supplier_disabled", `UPDATE suppliers SET status='disabled'`, `UPDATE suppliers SET status='healthy'`},
		{"supplier_unavailable", `UPDATE suppliers SET status='unavailable'`, `UPDATE suppliers SET status='healthy'`},
		{"binding", `UPDATE supplier_model_bindings SET enabled=FALSE`, `UPDATE supplier_model_bindings SET enabled=TRUE`},
		{"binding_status", `UPDATE supplier_model_bindings SET status='disabled'`, `UPDATE supplier_model_bindings SET status='active'`},
		{"execution", `UPDATE supplier_executions SET enabled=FALSE`, `UPDATE supplier_executions SET enabled=TRUE`},
		{"health_degraded", `UPDATE supplier_executions SET health_status='degraded'`, `UPDATE supplier_executions SET health_status='unknown'`},
		{"health_unavailable", `UPDATE supplier_executions SET health_status='unavailable'`, `UPDATE supplier_executions SET health_status='unknown'`},
		{"quota", `UPDATE supplier_executions SET quota_status='exhausted'`, `UPDATE supplier_executions SET quota_status='unknown'`},
		{"adapter", `UPDATE supplier_executions SET adapter_code='not_implemented'`, `UPDATE supplier_executions SET adapter_code='autodl_h3_v1'`},
		{"cost_status", `UPDATE supplier_cost_rules SET status='disabled'`, `UPDATE supplier_cost_rules SET status='active'`},
		{"cost_expired", `UPDATE supplier_cost_rules SET effective_to=NOW()-INTERVAL '1 second'`, `UPDATE supplier_cost_rules SET effective_to=NULL`},
		{"retail_status", `UPDATE retail_price_rules SET status='disabled'`, `UPDATE retail_price_rules SET status='active'`},
		{"retail_future", `UPDATE retail_price_rules SET effective_from=NOW()+INTERVAL '1 day'`, `UPDATE retail_price_rules SET effective_from=NULL`},
	} {
		t.Run(gate.name, func(t *testing.T) {
			if _, err := env.pool.Exec(env.ctx, gate.disable); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if _, err := env.pool.Exec(cleanupCtx, gate.restore); err != nil {
					t.Errorf("restore isolated fixture: %v", err)
				}
			})
			assertAvailability(t, "", false)
		})
	}
	t.Run("environment_credential_fallback", func(t *testing.T) {
		if _, err := env.pool.Exec(env.ctx, `UPDATE suppliers SET token_encrypted=''`); err != nil {
			t.Fatal(err)
		}
		assertAvailability(t, "", false)
		assertAvailability(t, "   ", false)
		assertAvailability(t, "integration-only-credential", true)
		request := seededDirectRequest(textMode, "")
		request.SupplierTokenFallback = "integration-only-credential"
		if route, err := env.store.ResolveDirectRoute(env.ctx, request); err != nil || route.Model.ID != modelID {
			t.Fatalf("automatic recommendation must use runtime credential: route=%+v err=%v", route, err)
		}
		user := newIntegrationUser(t, env, 1)
		if _, err := env.store.CreateTaskDirect(env.ctx, user.ID, DirectTaskInput{DirectRouteRequest: request, ClientRequestID: "fallback-submit", Prompt: "隔离测试"}); err != nil {
			t.Fatalf("catalog fallback must also support submission: %v", err)
		}
	})
}

func TestDirectOptionsOnlyAdvertiseBillableCapabilitiesIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	textMode, _, _, modelID := testCatalogIDs(t, env)
	// Keep a single execution and single matching retail/cost pair. Other
	// capabilities advertised by the upstream must not become selectable.
	if _, err := env.pool.Exec(env.ctx, `UPDATE supplier_executions SET enabled=(workflow_code='minimax_h3_lightx2v_no_pic')`); err != nil {
		t.Fatal(err)
	}
	if _, err := env.pool.Exec(env.ctx, `UPDATE supplier_cost_rules SET status=CASE WHEN resolution='768P' AND NOT generate_audio THEN 'active' ELSE 'disabled' END`); err != nil {
		t.Fatal(err)
	}
	options, err := env.store.ResolveCreationOptions(env.ctx, textMode, modelID)
	if err != nil || len(options.Resolutions) != 1 || options.Resolutions[0] != "768P" || options.SupportsGeneratedAudio {
		t.Fatalf("unpriced capability leaked into options: options=%+v err=%v", options, err)
	}
	// A retail rule and a cost rule existing independently is insufficient.
	// Recommendation must find a pair with the same resolution/audio fields.
	if _, err := env.pool.Exec(env.ctx, `UPDATE retail_price_rules SET status=CASE WHEN resolution='480P' AND NOT generate_audio THEN 'active' ELSE 'disabled' END`); err != nil {
		t.Fatal(err)
	}
	if model, err := env.store.recommendedModelID(env.ctx, textMode); err == nil {
		t.Fatalf("recommended model %s has no matching cost/retail capability pair", model)
	}
	if _, err := env.store.ResolveCreationOptions(env.ctx, textMode, modelID); err == nil {
		t.Fatal("options remained available with no matching cost/retail pair")
	}
}

func TestDirectCapabilityCombinationsAndModelPreferenceIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	textMode, firstLastMode, multiMode, modelID := testCatalogIDs(t, env)
	for _, scenario := range []struct {
		name string
		mode string
		edit func(*DirectRouteRequest)
		want bool
	}{
		{"text_baseline", textMode, func(*DirectRouteRequest) {}, true},
		{"text_rejects_images", textMode, func(in *DirectRouteRequest) { in.ImageCount = 1 }, false},
		{"too_short", textMode, func(in *DirectRouteRequest) { in.DurationSeconds = 0 }, false},
		{"too_long", textMode, func(in *DirectRouteRequest) { in.DurationSeconds = 16 }, false},
		{"unsupported_resolution", textMode, func(in *DirectRouteRequest) { in.Resolution = "1080P" }, false},
		{"unsupported_aspect", textMode, func(in *DirectRouteRequest) { in.AspectRatio = "4:3" }, false},
		{"unsupported_audio", textMode, func(in *DirectRouteRequest) { in.GenerateAudio = true }, false},
		{"first_last_pair", firstLastMode, func(in *DirectRouteRequest) { in.ImageCount = 2 }, true},
		{"first_last_same_image", firstLastMode, func(in *DirectRouteRequest) {
			image := SourceRefItem{Type: "pending", ID: "same-local-image"}
			in.ImageCount = (SourceMeta{First: &image, Last: &image}).ImageCount()
		}, true},
		{"first_last_missing_end", firstLastMode, func(in *DirectRouteRequest) { in.ImageCount = 1 }, false},
		{"multi_image", multiMode, func(in *DirectRouteRequest) { in.ImageCount = 2 }, true},
		{"multi_missing_image", multiMode, func(in *DirectRouteRequest) { in.ImageCount = 0 }, false},
		{"multi_excess_images", multiMode, func(in *DirectRouteRequest) { in.ImageCount = 10 }, false},
		{"unknown_model", textMode, func(in *DirectRouteRequest) { in.ModelID = "requested-unavailable-model" }, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			request := seededDirectRequest(scenario.mode, modelID)
			scenario.edit(&request)
			route, err := env.store.ResolveDirectRoute(env.ctx, request)
			if (err == nil) != scenario.want {
				t.Fatalf("capability availability=%v; result=%+v err=%v", scenario.want, route, err)
			}
			if route != nil && route.Model.ID != request.ModelID {
				t.Fatalf("explicit model silently changed from %q to %q", request.ModelID, route.Model.ID)
			}
		})
	}
}

func TestDirectEstimateThenRecalculateWithoutQuoteIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	textMode, _, _, modelID := testCatalogIDs(t, env)
	user := newIntegrationUser(t, env, 10)
	request := seededDirectRequest(textMode, modelID)
	estimate, err := env.store.ResolveDirectRoute(env.ctx, request)
	if err != nil || estimate.Total != 0.15 {
		t.Fatalf("initial estimate: result=%+v err=%v", estimate, err)
	}
	for _, table := range []string{"task_quotes", "generation_tasks", "consumption_orders", "wallet_ledger"} {
		var count int
		if err := env.pool.QueryRow(env.ctx, "SELECT COUNT(*) FROM "+table+" WHERE user_id=$1", user.ID).Scan(&count); err != nil || count != 0 {
			t.Fatalf("estimate created %s records: count=%d err=%v", table, count, err)
		}
	}
	if balance, err := env.store.GetBalance(env.ctx, user.ID); err != nil || balance != 10 {
		t.Fatalf("estimate changed balance: balance=%v err=%v", balance, err)
	}
	if _, err := env.pool.Exec(env.ctx, `UPDATE retail_price_rules SET price_per_sec=0.071 WHERE public_id=$1`, estimate.Retail.ID); err != nil {
		t.Fatal(err)
	}
	addTestWorkflowExecution(t, env, textMode, modelID, "after_estimate_execution", 1, 0.0001)
	created, err := env.store.CreateTaskDirect(env.ctx, user.ID, DirectTaskInput{DirectRouteRequest: request, ClientRequestID: "estimate-recalculate", Prompt: "提交采用最新设置"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Task.Cost != 0.36 || created.Balance != 9.64 {
		t.Fatalf("submit must recalculate and round to cents: %+v", created)
	}
	var selected, quoteID string
	var amount float64
	if err := env.pool.QueryRow(env.ctx, `SELECT task.supplier_route_id,task.quote_public_id,orders.amount::float8 FROM generation_tasks task JOIN consumption_orders orders ON orders.task_id=task.id WHERE task.public_id=$1`, created.Task.ID).Scan(&selected, &quoteID, &amount); err != nil {
		t.Fatal(err)
	}
	if selected != "after_estimate_execution" || quoteID != "" || amount != created.Task.Cost {
		t.Fatalf("submission retained estimate or quote: execution=%q quote=%q amount=%v", selected, quoteID, amount)
	}
	if err := env.store.FailTaskUpstream(env.ctx, taskInternalID(t, env, created.Task.ID), "对象存储失败测试"); err != nil {
		t.Fatal(err)
	}
	if balance, err := env.store.GetBalance(env.ctx, user.ID); err != nil || balance != 10 {
		t.Fatalf("rounded refund failed to restore balance: balance=%v err=%v", balance, err)
	}
}

func taskInternalID(t *testing.T, env *integrationStore, publicID string) int64 {
	t.Helper()
	var id int64
	if err := env.pool.QueryRow(env.ctx, `SELECT id FROM generation_tasks WHERE public_id=$1`, publicID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestDirectSubmissionRejectionsLeaveNoFinancialRecordsIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	textMode, _, _, modelID := testCatalogIDs(t, env)
	for _, scenario := range []struct {
		name    string
		balance float64
		prepare func(*DirectTaskInput)
	}{
		{"insufficient_balance", 0.01, func(*DirectTaskInput) {}},
		{"explicit_model_no_switch", 10, func(in *DirectTaskInput) { in.ModelID = "does_not_exist" }},
		{"wrong_carrier_no_switch", 10, func(in *DirectTaskInput) { in.CarrierID = "another_carrier" }},
		{"storage_unavailable", 10, func(in *DirectTaskInput) {
			in.Preflight = func(context.Context, *Store, *ResolvedRoute) error { return errors.New("object storage unavailable") }
		}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			user := newIntegrationUser(t, env, scenario.balance)
			input := DirectTaskInput{DirectRouteRequest: seededDirectRequest(textMode, modelID), ClientRequestID: scenario.name, Prompt: "测试"}
			scenario.prepare(&input)
			if _, err := env.store.CreateTaskDirect(env.ctx, user.ID, input); err == nil {
				t.Fatal("rejected submission unexpectedly succeeded")
			}
			for _, table := range []string{"generation_tasks", "consumption_orders", "wallet_ledger", "task_quotes"} {
				var count int
				if err := env.pool.QueryRow(env.ctx, "SELECT COUNT(*) FROM "+table+" WHERE user_id=$1", user.ID).Scan(&count); err != nil || count != 0 {
					t.Fatalf("rejection left %s records: count=%d err=%v", table, count, err)
				}
			}
			if balance, err := env.store.GetBalance(env.ctx, user.ID); err != nil || balance != scenario.balance {
				t.Fatalf("rejection changed wallet: balance=%v err=%v", balance, err)
			}
		})
	}
}

func TestDirectSubmissionLeaseCrashRecoveryAndFencingIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	textMode, _, _, modelID := testCatalogIDs(t, env)
	user := newIntegrationUser(t, env, 10)
	created, err := env.store.CreateTaskDirect(env.ctx, user.ID, DirectTaskInput{DirectRouteRequest: seededDirectRequest(textMode, modelID), ClientRequestID: "crash-recovery", Prompt: "测试"})
	if err != nil {
		t.Fatal(err)
	}
	id := taskInternalID(t, env, created.Task.ID)
	claimed, oldToken, err := env.store.ClaimTaskForDispatchWithToken(env.ctx, id)
	if err != nil || !claimed || oldToken == "" {
		t.Fatalf("initial claim: claimed=%v tokenSet=%v err=%v", claimed, oldToken != "", err)
	}
	// Crash before calling upstream: the stale lease is safe to reclaim.
	if _, err := env.pool.Exec(env.ctx, `UPDATE generation_tasks SET claim_expires_at=NOW()-INTERVAL '1 second',next_poll_at=NOW()-INTERVAL '1 second' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	claimed, newToken, err := env.store.ClaimTaskForDispatchWithToken(env.ctx, id)
	if err != nil || !claimed || newToken == oldToken {
		t.Fatalf("stale safe claim not recovered: claimed=%v err=%v", claimed, err)
	}
	if err := env.store.MarkTaskSubmitStarted(env.ctx, id, oldToken); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stale worker crossed submit boundary: %v", err)
	}
	if err := env.store.MarkTaskSubmitStarted(env.ctx, id, newToken); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := env.store.CancelTask(env.ctx, user.ID, created.Task.ID); err == nil {
		t.Fatal("client cancellation refunded a submission already crossing the provider boundary")
	}
	// Crash after request could have left the process: never submit again.
	if _, err := env.pool.Exec(env.ctx, `UPDATE generation_tasks SET claim_expires_at=NOW()-INTERVAL '1 second',next_poll_at=NOW()-INTERVAL '1 second' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if claimed, _, err := env.store.ClaimTaskForDispatchWithToken(env.ctx, id); err != nil || claimed {
		t.Fatalf("uncertain submission was reclaimed: claimed=%v err=%v", claimed, err)
	}
	if err := env.store.MarkTaskSubmissionUnknown(env.ctx, id, "provider response lost", newToken); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := env.store.CancelTask(env.ctx, user.ID, created.Task.ID); err == nil {
		t.Fatal("client cancellation refunded an uncertain provider submission")
	}
	if err := env.store.FailTaskUpstreamWithToken(env.ctx, id, "stale worker failure", oldToken); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stale worker refunded task: %v", err)
	}
	var status, upstream string
	var uncertain bool
	if err := env.pool.QueryRow(env.ctx, `SELECT status,upstream_status,submit_uncertain FROM generation_tasks WHERE id=$1`, id).Scan(&status, &upstream, &uncertain); err != nil {
		t.Fatal(err)
	}
	if status != "running" || upstream != "SUBMIT_UNKNOWN" || !uncertain {
		t.Fatalf("uncertain state lost: status=%s upstream=%s uncertain=%v", status, upstream, uncertain)
	}
	if balance, err := env.store.GetBalance(env.ctx, user.ID); err != nil || balance != created.Balance {
		t.Fatalf("uncertain submission changed balance: balance=%v err=%v", balance, err)
	}
}

func TestDirectPollLeaseAndStorageRecoveryIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	textMode, _, _, modelID := testCatalogIDs(t, env)
	user := newIntegrationUser(t, env, 10)
	created, err := env.store.CreateTaskDirect(env.ctx, user.ID, DirectTaskInput{DirectRouteRequest: seededDirectRequest(textMode, modelID), ClientRequestID: fmt.Sprintf("storage-recovery-%d", time.Now().UnixNano()), Prompt: "测试"})
	if err != nil {
		t.Fatal(err)
	}
	id := taskInternalID(t, env, created.Task.ID)
	_, submitToken, err := env.store.ClaimTaskForDispatchWithToken(env.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.store.MarkTaskSubmitStarted(env.ctx, id, submitToken); err != nil {
		t.Fatal(err)
	}
	if err := env.store.MarkTaskSubmittedWithToken(env.ctx, id, "integration-provider-task", "RUNNING", submitToken); err != nil {
		t.Fatal(err)
	}
	claimed, oldPollToken, err := env.store.ClaimTaskForPollWithToken(env.ctx, id)
	if err != nil || !claimed {
		t.Fatalf("first poll claim: claimed=%v err=%v", claimed, err)
	}
	if claimed, _, err := env.store.ClaimTaskForPollWithToken(env.ctx, id); err != nil || claimed {
		t.Fatalf("concurrent poll claimed active lease: claimed=%v err=%v", claimed, err)
	}
	if _, err := env.pool.Exec(env.ctx, `UPDATE generation_tasks SET claim_expires_at=NOW()-INTERVAL '1 second' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	claimed, pollToken, err := env.store.ClaimTaskForPollWithToken(env.ctx, id)
	if err != nil || !claimed || pollToken == oldPollToken {
		t.Fatalf("poll crash recovery failed: claimed=%v err=%v", claimed, err)
	}
	if err := env.store.CompleteTaskUpstreamWithToken(env.ctx, id, "upstream", "https://storage.test/stale.mp4", 5, oldPollToken); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stale poll worker overwrote task: %v", err)
	}
	if err := env.store.FailTaskUpstreamWithToken(env.ctx, id, "stale failure", oldPollToken); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stale poll worker refunded task: %v", err)
	}
	if err := env.store.MarkTaskDeliveryPendingWithToken(env.ctx, id, "https://provider.test/result.mp4", "对象存储暂不可用", pollToken); err != nil {
		t.Fatal(err)
	}
	var status, upstream, resultURL, ossURL string
	if err := env.pool.QueryRow(env.ctx, `SELECT status,upstream_status,result_url,result_oss_url FROM generation_tasks WHERE id=$1`, id).Scan(&status, &upstream, &resultURL, &ossURL); err != nil {
		t.Fatal(err)
	}
	if status != "running" || upstream != "WAITING_STORAGE" || !strings.Contains(resultURL, "provider.test") || ossURL != "" {
		t.Fatalf("storage failure appeared delivered: status=%s upstream=%s result=%q oss=%q", status, upstream, resultURL, ossURL)
	}
	claimed, recoveryToken, err := env.store.ClaimTaskForPollWithToken(env.ctx, id)
	if err != nil || !claimed {
		t.Fatalf("storage retry claim: claimed=%v err=%v", claimed, err)
	}
	if err := env.store.CompleteTaskUpstreamWithToken(env.ctx, id, resultURL, "https://storage.test/recovered.mp4", 5, recoveryToken); err != nil {
		t.Fatal(err)
	}
	if err := env.store.FailTaskUpstreamWithToken(env.ctx, id, "late duplicate failure", recoveryToken); err != nil {
		t.Fatal(err)
	}
	var refundCount, eventCount int
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM wallet_ledger WHERE user_id=$1 AND type='task_refund'`, user.ID).Scan(&refundCount); err != nil {
		t.Fatal(err)
	}
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM provider_task_events WHERE task_id=$1 AND event_type IN ('submit_started','submitted','delivery_wait','completed')`, id).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if refundCount != 0 || eventCount != 4 {
		t.Fatalf("storage recovery audit/billing mismatch: refunds=%d events=%d", refundCount, eventCount)
	}
}
