package store

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func testCatalogIDs(t *testing.T, env *integrationStore) (textMode, firstLastMode, multiMode, modelID string) {
	t.Helper()
	var err error
	rows, err := env.pool.Query(env.ctx, `SELECT public_id, code FROM creation_modes ORDER BY sort_order`)
	if err != nil {
		t.Fatalf("list creation modes: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, code string
		if err := rows.Scan(&id, &code); err != nil {
			t.Fatalf("scan creation mode: %v", err)
		}
		switch code {
		case "text_to_video":
			textMode = id
		case "first_last_frame_to_video":
			firstLastMode = id
		case "multi_image_to_video":
			multiMode = id
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate creation modes: %v", err)
	}
	if err := env.pool.QueryRow(env.ctx, `SELECT public_id FROM models WHERE code='h3' ORDER BY id LIMIT 1`).Scan(&modelID); err != nil {
		t.Fatalf("find H3 model: %v", err)
	}
	if textMode == "" || firstLastMode == "" || multiMode == "" || modelID == "" {
		t.Fatalf("seed catalog missing expected IDs: text=%q firstLast=%q multi=%q model=%q", textMode, firstLastMode, multiMode, modelID)
	}
	return
}

func TestDirectCatalogMigrationAndSeedIsolationIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	textMode, firstLastMode, _, modelID := testCatalogIDs(t, env)

	var migrationCount, migrationVersionCount int
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatalf("count applied migrations: %v", err)
	}
	if migrationCount < 21 {
		t.Fatalf("expected all current migrations in isolated schema, got %d", migrationCount)
	}
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version='021_provider_execution_reliability'`).Scan(&migrationVersionCount); err != nil {
		t.Fatalf("check migration version: %v", err)
	}
	if migrationVersionCount != 1 {
		t.Fatalf("migration replay must not duplicate version rows: %d", migrationVersionCount)
	}

	var retailID string
	if err := env.pool.QueryRow(env.ctx, `
SELECT retail.public_id FROM retail_price_rules retail
JOIN creation_modes mode ON mode.id=retail.mode_id
WHERE mode.public_id=$1 AND retail.model_id IS NULL AND retail.resolution='480P' AND retail.generate_audio=FALSE
ORDER BY retail.public_id LIMIT 1`, firstLastMode).Scan(&retailID); err != nil {
		t.Fatalf("find mode-level retail price: %v", err)
	}
	if _, err := env.pool.Exec(env.ctx, `UPDATE retail_price_rules SET price_per_sec=0.0777 WHERE public_id=$1`, retailID); err != nil {
		t.Fatalf("customize retail price: %v", err)
	}
	var costID string
	if err := env.pool.QueryRow(env.ctx, `SELECT public_id FROM supplier_cost_rules ORDER BY public_id LIMIT 1`).Scan(&costID); err != nil {
		t.Fatalf("find supplier cost rule: %v", err)
	}
	if _, err := env.pool.Exec(env.ctx, `UPDATE supplier_cost_rules SET price_per_sec_peak=0.0777, price_per_sec_offpeak=0.0666, status='disabled', version='admin-custom', effective_to='2001-01-01T00:00:00Z' WHERE public_id=$1`, costID); err != nil {
		t.Fatalf("customize supplier cost: %v", err)
	}
	var offeringsBefore, executionBefore, costsBefore, retailBefore int
	for query, destination := range map[string]*int{
		`SELECT COUNT(*) FROM supplier_offerings`:  &offeringsBefore,
		`SELECT COUNT(*) FROM supplier_executions`: &executionBefore,
		`SELECT COUNT(*) FROM supplier_cost_rules`: &costsBefore,
		`SELECT COUNT(*) FROM retail_price_rules`:  &retailBefore,
	} {
		if err := env.pool.QueryRow(env.ctx, query).Scan(destination); err != nil {
			t.Fatalf("count seeded rows %q: %v", query, err)
		}
	}
	if err := env.store.SeedCreationCatalog(env.ctx); err != nil {
		t.Fatalf("repeat seed catalog: %v", err)
	}
	var retailPrice, peakPrice, offpeakPrice float64
	var costStatus, costVersion string
	var effectiveTo time.Time
	if err := env.pool.QueryRow(env.ctx, `SELECT price_per_sec::float8 FROM retail_price_rules WHERE public_id=$1`, retailID).Scan(&retailPrice); err != nil {
		t.Fatalf("read preserved retail price: %v", err)
	}
	if err := env.pool.QueryRow(env.ctx, `SELECT price_per_sec_peak::float8, price_per_sec_offpeak::float8, status, version, effective_to FROM supplier_cost_rules WHERE public_id=$1`, costID).Scan(&peakPrice, &offpeakPrice, &costStatus, &costVersion, &effectiveTo); err != nil {
		t.Fatalf("read preserved supplier cost: %v", err)
	}
	if retailPrice != 0.0777 || peakPrice != 0.0777 || offpeakPrice != 0.0666 {
		t.Fatalf("repeat seed reset configured price: retail=%v peak=%v offpeak=%v", retailPrice, peakPrice, offpeakPrice)
	}
	if costStatus != "disabled" || costVersion != "admin-custom" || !effectiveTo.Equal(time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("repeat seed changed administrator cost lifecycle: status=%s version=%s effectiveTo=%s", costStatus, costVersion, effectiveTo)
	}
	var got int
	for query, want := range map[string]int{
		`SELECT COUNT(*) FROM supplier_offerings`:  offeringsBefore,
		`SELECT COUNT(*) FROM supplier_executions`: executionBefore,
		`SELECT COUNT(*) FROM supplier_cost_rules`: costsBefore,
		`SELECT COUNT(*) FROM retail_price_rules`:  retailBefore,
	} {
		if err := env.pool.QueryRow(env.ctx, query).Scan(&got); err != nil {
			t.Fatalf("count repeat seeded rows %q: %v", query, err)
		}
		if got != want {
			t.Fatalf("repeat seed changed row count %q: got=%d want=%d", query, got, want)
		}
	}

	// Add a model-specific row with a deliberately different amount. Direct
	// routing must use the mode-level retail rule and never fall back to this
	// historical/model-specific path.
	if _, err := env.pool.Exec(env.ctx, `
INSERT INTO retail_price_rules (public_id, mode_id, model_id, resolution, generate_audio, name, price_per_sec, version, status)
SELECT 'it_model_price', mode.id, model.id, '768P', FALSE, 'test model-specific legacy row', 9.99, 'it-v1', 'active'
FROM creation_modes mode, models model WHERE mode.public_id=$1 AND model.public_id=$2
ON CONFLICT (public_id) DO NOTHING`, textMode, modelID); err != nil {
		t.Fatalf("insert historical model price row: %v", err)
	}
	route, err := env.store.ResolveDirectRoute(env.ctx, seededDirectRequest(textMode, modelID))
	if err != nil {
		t.Fatalf("resolve mode-level retail route: %v", err)
	}
	if route.Retail.ModelID != "" || route.Retail.PricePerSecond != 0.03 || route.Total != 0.15 {
		t.Fatalf("direct route used model-specific or unexpected retail row: retail=%+v total=%v", route.Retail, route.Total)
	}
}

func addTestAPIExecution(t *testing.T, env *integrationStore, modeID, modelID, publicID, apiCode string, priority int, cost float64) {
	t.Helper()
	if _, err := env.pool.Exec(env.ctx, `
INSERT INTO supplier_executions (
 public_id, binding_id, mode_id, execution_type, api_model_code, api_version, workflow_code,
 adapter_code, enabled, health_status, quota_status, priority, min_duration_sec, max_duration_sec,
 resolutions, aspect_ratios, min_images, max_images, supports_generated_audio, parameter_mapping, result_mapping
)
SELECT $1, binding.id, mode.id, 'api', $4, 'v1', '', 'autodl_h3_v1', TRUE, 'unknown', 'unknown', $5, 1, 15,
 '["480P","768P"]'::jsonb, '["16:9","9:16","1:1"]'::jsonb, 0, 0, FALSE, '{}'::jsonb, '{}'::jsonb
FROM supplier_model_bindings binding
JOIN creation_modes mode ON mode.public_id=$2
WHERE binding.model_id=(SELECT id FROM models WHERE public_id=$3)
  AND binding.supplier_id=(SELECT id FROM suppliers WHERE public_id='sup_autodl')`, publicID, modeID, modelID, apiCode, priority); err != nil {
		t.Fatalf("insert API test execution %s: %v", publicID, err)
	}
	if _, err := env.pool.Exec(env.ctx, `
INSERT INTO supplier_cost_rules (public_id, offering_id, execution_id, resolution, generate_audio, price_per_sec_peak, price_per_sec_offpeak, currency, version, status)
SELECT $1, NULL, execution.id, '768P', FALSE, $3, $3, 'CNY', 'it-v1', 'active'
FROM supplier_executions execution WHERE execution.public_id=$2`, "cost_"+publicID, publicID, cost); err != nil {
		t.Fatalf("insert API test cost %s: %v", publicID, err)
	}
}

func addTestWorkflowExecution(t *testing.T, env *integrationStore, modeID, modelID, publicID string, priority int, cost float64) {
	t.Helper()
	addTestAPIExecution(t, env, modeID, modelID, publicID, "unused-api-code", priority, cost)
	if _, err := env.pool.Exec(env.ctx, `UPDATE supplier_executions SET execution_type='workflow', api_model_code='', workflow_code='minimax_h3_lightx2v_no_pic' WHERE public_id=$1`, publicID); err != nil {
		t.Fatalf("configure supported workflow %s: %v", publicID, err)
	}
}

func TestDirectLowestCostSupportedExecutionAndCapabilityFilteringIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	textMode, _, multiMode, modelID := testCatalogIDs(t, env)
	// API is a valid configuration type, but no real API adapter is currently
	// supported. Its lower price must never cause an unexecutable charge.
	addTestAPIExecution(t, env, textMode, modelID, "it_api_cheap", "it-api-cheap", 1, 0.0001)
	addTestWorkflowExecution(t, env, textMode, modelID, "it_unknown_adapter", 1, 0.0001)
	if _, err := env.pool.Exec(env.ctx, `UPDATE supplier_executions SET adapter_code='not_implemented' WHERE public_id='it_unknown_adapter'`); err != nil {
		t.Fatal(err)
	}
	addTestWorkflowExecution(t, env, textMode, modelID, "it_workflow_expensive", 1, 0.20)
	addTestWorkflowExecution(t, env, textMode, modelID, "it_workflow_cheap", 50, 0.001)

	route, err := env.store.ResolveDirectRoute(env.ctx, seededDirectRequest(textMode, modelID))
	if err != nil {
		t.Fatalf("resolve mixed API/workflow route: %v", err)
	}
	if route.Offering.ExecutionType != "workflow" || route.Offering.ID != "it_workflow_cheap" || route.CostPerSecond != 0.001 {
		t.Fatalf("lowest-cost supported execution mismatch: %+v", route.Offering)
	}
	// Same cost and priority must be stable by public execution ID, regardless
	// of insertion order or PostgreSQL plan choices.
	addTestWorkflowExecution(t, env, textMode, modelID, "it_workflow_tie_b", 1, 0.001)
	addTestWorkflowExecution(t, env, textMode, modelID, "it_workflow_tie_a", 1, 0.001)
	// Public IDs tie-break after priority. The two tie IDs are lower than the
	// original cheap execution's priority, so the stable winner is tie_a.
	route, err = env.store.ResolveDirectRoute(env.ctx, seededDirectRequest(textMode, modelID))
	if err != nil {
		t.Fatalf("resolve tied cost routes: %v", err)
	}
	if route.Offering.ID != "it_workflow_tie_a" {
		t.Fatalf("stable priority/ID route mismatch: selected=%+v", route.Offering)
	}

	// Multi-image executions advertise one to nine images. Zero and ten must
	// fail capability filtering while a valid count must route normally.
	zeroImages := seededDirectRequest(multiMode, modelID)
	zeroImages.ImageCount = 0
	if _, err := env.store.ResolveDirectRoute(env.ctx, zeroImages); err == nil {
		t.Fatal("multi-image route accepted zero image references")
	}
	validImages := zeroImages
	validImages.ImageCount = 1
	if _, err := env.store.ResolveDirectRoute(env.ctx, validImages); err != nil {
		t.Fatalf("multi-image route rejected one image reference: %v", err)
	}
	tooManyImages := zeroImages
	tooManyImages.ImageCount = 10
	if _, err := env.store.ResolveDirectRoute(env.ctx, tooManyImages); err == nil {
		t.Fatal("multi-image route accepted more than execution max image count")
	}

	// All execution health and credential gates are server-side filters.
	if _, err := env.pool.Exec(env.ctx, `UPDATE suppliers SET token_encrypted='' WHERE public_id='sup_autodl'`); err != nil {
		t.Fatal(err)
	}
	if _, err := env.store.ResolveDirectRoute(env.ctx, seededDirectRequest(textMode, modelID)); err == nil {
		t.Fatal("route remained available with missing supplier credentials")
	}
	if _, err := env.pool.Exec(env.ctx, `UPDATE suppliers SET token_encrypted='integration-configured-token' WHERE public_id='sup_autodl'`); err != nil {
		t.Fatal(err)
	}
	for _, update := range []string{
		`UPDATE supplier_executions SET health_status='unavailable' WHERE mode_id=(SELECT id FROM creation_modes WHERE public_id=$1)`,
		`UPDATE supplier_executions SET health_status='degraded' WHERE mode_id=(SELECT id FROM creation_modes WHERE public_id=$1)`,
		`UPDATE supplier_executions SET health_status='unknown', quota_status='exhausted' WHERE mode_id=(SELECT id FROM creation_modes WHERE public_id=$1)`,
		`UPDATE supplier_executions SET quota_status='unknown', enabled=FALSE WHERE mode_id=(SELECT id FROM creation_modes WHERE public_id=$1)`,
	} {
		if _, err := env.pool.Exec(env.ctx, update, textMode); err != nil {
			t.Fatalf("disable execution gate: %v", err)
		}
		if _, err := env.store.ResolveDirectRoute(env.ctx, seededDirectRequest(textMode, modelID)); err == nil {
			t.Fatalf("route remained available after execution gate %q", update)
		}
		if _, err := env.pool.Exec(env.ctx, `UPDATE supplier_executions SET enabled=TRUE, health_status='unknown', quota_status='unknown' WHERE mode_id=(SELECT id FROM creation_modes WHERE public_id=$1)`, textMode); err != nil {
			t.Fatalf("restore execution gate: %v", err)
		}
	}
}

func TestDirectSubmissionBillingRefundAndSnapshotIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	textMode, _, _, modelID := testCatalogIDs(t, env)
	user := newIntegrationUser(t, env, 100)
	in := DirectTaskInput{
		DirectRouteRequest: seededDirectRequest(textMode, modelID),
		ClientRequestID:    "direct-recovery-" + fmt.Sprint(time.Now().UnixNano()),
		Title:              "直接提交测试",
		Prompt:             "夜晚街道",
	}
	first, err := env.store.CreateTaskDirect(env.ctx, user.ID, in)
	if err != nil {
		t.Fatalf("direct submission: %v", err)
	}
	if first.AlreadyDone || first.OrderID == "" || first.Task.ID == "" || first.Balance >= 100 {
		t.Fatalf("direct submission did not charge atomically: %+v", first)
	}
	var orderCount, taskCount, ledgerCount int
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM consumption_orders WHERE user_id=$1 AND client_request_id=$2`, user.ID, in.ClientRequestID).Scan(&orderCount); err != nil {
		t.Fatal(err)
	}
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM generation_tasks WHERE user_id=$1 AND client_request_id=$2`, user.ID, in.ClientRequestID).Scan(&taskCount); err != nil {
		t.Fatal(err)
	}
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM wallet_ledger WHERE user_id=$1 AND ref_type='order' AND ref_id=$2 AND type='task_charge'`, user.ID, first.OrderID).Scan(&ledgerCount); err != nil {
		t.Fatal(err)
	}
	if orderCount != 1 || taskCount != 1 || ledgerCount != 1 {
		t.Fatalf("atomic charge counts mismatch: orders=%d tasks=%d ledgers=%d", orderCount, taskCount, ledgerCount)
	}

	pending, err := env.store.ListPendingProviderTasks(env.ctx, 20)
	if err != nil {
		t.Fatal(err)
	}
	var providerTask *ProviderTask
	for i := range pending {
		if pending[i].PublicID == first.Task.ID {
			providerTask = &pending[i]
			break
		}
	}
	if providerTask == nil || providerTask.WorkflowCode != "minimax_h3_lightx2v_no_pic" || providerTask.AdapterCode != "autodl_h3_v1" {
		t.Fatalf("worker did not read immutable task execution snapshot: %+v", providerTask)
	}
	if _, err := env.pool.Exec(env.ctx, `UPDATE supplier_executions SET workflow_code='tampered', adapter_code='tampered' WHERE public_id=$1`, providerTask.SupplierRouteID); err != nil {
		t.Fatal(err)
	}
	pending, err = env.store.ListPendingProviderTasks(env.ctx, 20)
	if err != nil {
		t.Fatal(err)
	}
	for i := range pending {
		if pending[i].PublicID == first.Task.ID && (pending[i].WorkflowCode != "minimax_h3_lightx2v_no_pic" || pending[i].AdapterCode != "autodl_h3_v1") {
			t.Fatalf("worker reread mutable execution data instead of snapshot: %+v", pending[i])
		}
	}

	// A later retry must return the original order/task even after its supply is
	// disabled, without attempting routing or charging again.
	if _, err := env.pool.Exec(env.ctx, `UPDATE supplier_executions SET enabled=FALSE WHERE public_id=$1`, providerTask.SupplierRouteID); err != nil {
		t.Fatal(err)
	}
	replayed, err := env.store.CreateTaskDirect(env.ctx, user.ID, in)
	if err != nil || !replayed.AlreadyDone || replayed.OrderID != first.OrderID || replayed.Task.ID != first.Task.ID || replayed.Balance != first.Balance {
		t.Fatalf("idempotent replay after supply change failed: result=%+v err=%v", replayed, err)
	}

	var taskInternalID int64
	if err := env.pool.QueryRow(env.ctx, `SELECT id FROM generation_tasks WHERE public_id=$1`, first.Task.ID).Scan(&taskInternalID); err != nil {
		t.Fatal(err)
	}
	if err := env.store.FailTaskUpstream(env.ctx, taskInternalID, "integration failure"); err != nil {
		t.Fatal(err)
	}
	if err := env.store.FailTaskUpstream(env.ctx, taskInternalID, "duplicate integration failure"); err != nil {
		t.Fatal(err)
	}
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM wallet_ledger WHERE user_id=$1 AND ref_type='task' AND ref_id=$2 AND type='task_refund'`, user.ID, first.Task.ID).Scan(&ledgerCount); err != nil {
		t.Fatal(err)
	}
	if ledgerCount != 1 {
		t.Fatalf("refund must be idempotent, got %d entries", ledgerCount)
	}
	var orderStatus string
	if err := env.pool.QueryRow(env.ctx, `SELECT status FROM consumption_orders WHERE public_id=$1`, first.OrderID).Scan(&orderStatus); err != nil {
		t.Fatal(err)
	}
	if orderStatus != "refunded" {
		t.Fatalf("failed task must refund consumption order, status=%s", orderStatus)
	}
}

func TestDirectSubmissionRollbackAndConcurrentIdempotencyIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	textMode, _, _, modelID := testCatalogIDs(t, env)
	user := newIntegrationUser(t, env, 100)
	requestID := "direct-concurrent-" + fmt.Sprint(time.Now().UnixNano())
	in := DirectTaskInput{DirectRouteRequest: seededDirectRequest(textMode, modelID), ClientRequestID: requestID, Title: "并发提交", Prompt: "测试"}
	var wg sync.WaitGroup
	results := make(chan *DirectTaskResult, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := env.store.CreateTaskDirect(env.ctx, user.ID, in)
			results <- result
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	var created, replayed int
	var taskID, orderID string
	for result := range results {
		if result == nil {
			continue
		}
		if result.AlreadyDone {
			replayed++
		} else {
			created++
		}
		if taskID == "" {
			taskID, orderID = result.Task.ID, result.OrderID
		} else if result.Task.ID != taskID || result.OrderID != orderID {
			t.Fatalf("concurrent same-key requests returned different identities: first=%s/%s got=%s/%s", taskID, orderID, result.Task.ID, result.OrderID)
		}
	}
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent idempotent submission failed: %v", err)
		}
	}
	if created != 1 || replayed != 1 {
		t.Fatalf("expected one creation and one replay, created=%d replayed=%d", created, replayed)
	}
	var chargedCount int
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM wallet_ledger WHERE user_id=$1 AND type='task_charge'`, user.ID).Scan(&chargedCount); err != nil || chargedCount != 1 {
		t.Fatalf("concurrent idempotency charged more than once: charges=%d err=%v", chargedCount, err)
	}

	// Fail the order insert after all preceding work has run. The trigger lives
	// only in this disposable schema and proves the transaction rolls back task,
	// ledger, and wallet changes together.
	if _, err := env.pool.Exec(env.ctx, `
CREATE FUNCTION it_fail_order_insert() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'injected consumption order failure'; END; $$;
CREATE TRIGGER it_fail_order_insert BEFORE INSERT ON consumption_orders
FOR EACH ROW EXECUTE FUNCTION it_fail_order_insert();`); err != nil {
		t.Fatalf("install rollback trigger: %v", err)
	}
	failingID := "direct-rollback-" + fmt.Sprint(time.Now().UnixNano())
	failing := in
	failing.ClientRequestID = failingID
	before, err := env.store.GetBalance(env.ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.store.CreateTaskDirect(env.ctx, user.ID, failing); err == nil {
		t.Fatal("injected order failure unexpectedly succeeded")
	}
	var count int
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM generation_tasks WHERE user_id=$1 AND client_request_id=$2`, user.ID, failingID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("failed transaction left a generation task: %d", count)
	}
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM consumption_orders WHERE user_id=$1 AND client_request_id=$2`, user.ID, failingID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("failed transaction left a consumption order: %d", count)
	}
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM wallet_ledger WHERE user_id=$1 AND type='task_charge'`, user.ID).Scan(&count); err != nil || count != chargedCount {
		t.Fatalf("failed transaction left a charge ledger: before=%d after=%d err=%v", chargedCount, count, err)
	}
	after, err := env.store.GetBalance(env.ctx, user.ID)
	if err != nil || after != before {
		t.Fatalf("failed transaction changed wallet balance: before=%v after=%v err=%v", before, after, err)
	}
	if _, err := env.pool.Exec(env.ctx, `DROP TRIGGER it_fail_order_insert ON consumption_orders; DROP FUNCTION it_fail_order_insert();`); err != nil {
		t.Fatalf("remove rollback trigger: %v", err)
	}
	if retry, err := env.store.CreateTaskDirect(env.ctx, user.ID, failing); err != nil || retry.AlreadyDone {
		t.Fatalf("rolled-back request should be retryable: result=%+v err=%v", retry, err)
	}
}

func TestSaveCompletedTaskToProjectIsIdempotentIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	textMode, _, _, modelID := testCatalogIDs(t, env)
	user := newIntegrationUser(t, env, 100)
	input := DirectTaskInput{DirectRouteRequest: seededDirectRequest(textMode, modelID), ClientRequestID: "direct-save-" + fmt.Sprint(time.Now().UnixNano()), Title: "保存测试", Prompt: "成片"}
	result, err := env.store.CreateTaskDirect(env.ctx, user.ID, input)
	if err != nil {
		t.Fatalf("create save task: %v", err)
	}
	var internalID int64
	if err := env.pool.QueryRow(env.ctx, `SELECT id FROM generation_tasks WHERE public_id=$1`, result.Task.ID).Scan(&internalID); err != nil {
		t.Fatal(err)
	}
	if err := env.store.CompleteTaskUpstream(env.ctx, internalID, "provider-result-id", "https://storage.test/results/save.mp4", 5); err != nil {
		t.Fatalf("complete save task: %v", err)
	}
	projectID, err := env.store.ResolveCurrentProjectID(env.ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	type saveResult struct {
		item *FsItem
		err  error
	}
	results := make(chan saveResult, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			item, _, err := env.store.SaveTaskToProject(env.ctx, user.ID, result.Task.ID, projectID)
			results <- saveResult{item: item, err: err}
		}()
	}
	wg.Wait()
	close(results)
	var saved int
	var savedID string
	for item := range results {
		if item.err != nil {
			t.Fatalf("concurrent save failed: %v", item.err)
		}
		if item.item != nil {
			saved++
			if savedID == "" {
				savedID = item.item.ID
			} else if savedID != item.item.ID {
				t.Fatalf("concurrent save returned different assets: %s vs %s", savedID, item.item.ID)
			}
		}
	}
	if saved != 2 {
		t.Fatalf("both save calls should return the same saved asset, count=%d", saved)
	}
	var assetCount int
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM projects, jsonb_path_query(folders_json, '$.**.items[*]') item WHERE projects.user_id=$1 AND item->>'sourceTask'=$2`, user.ID, result.Task.ID).Scan(&assetCount); err != nil {
		// PostgreSQL versions without jsonpath support should fail the test
		// rather than silently accepting a duplicate asset.
		t.Fatalf("count saved project assets: %v", err)
	}
	if assetCount != 1 {
		t.Fatalf("save-to-project must be idempotent, assets=%d", assetCount)
	}
}

func TestProviderLeaseAndRefundStateIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	textMode, _, _, modelID := testCatalogIDs(t, env)
	user := newIntegrationUser(t, env, 100)
	input := DirectTaskInput{DirectRouteRequest: seededDirectRequest(textMode, modelID), ClientRequestID: "direct-lease-" + fmt.Sprint(time.Now().UnixNano()), Title: "租约", Prompt: "测试"}
	result, err := env.store.CreateTaskDirect(env.ctx, user.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	var taskID int64
	if err := env.pool.QueryRow(env.ctx, `SELECT id FROM generation_tasks WHERE public_id=$1`, result.Task.ID).Scan(&taskID); err != nil {
		t.Fatal(err)
	}
	claimed, err := env.store.ClaimTaskForDispatch(env.ctx, taskID)
	if err != nil || !claimed {
		t.Fatalf("first dispatch claim failed: claimed=%v err=%v", claimed, err)
	}
	claimed, err = env.store.ClaimTaskForDispatch(env.ctx, taskID)
	if err != nil || claimed {
		t.Fatalf("second dispatch claim must be rejected: claimed=%v err=%v", claimed, err)
	}
	if err := env.store.MarkTaskSubmitted(env.ctx, taskID, "provider-request-1", "RUNNING"); err != nil {
		t.Fatal(err)
	}
	var providerRequestID string
	var eventCount int
	if err := env.pool.QueryRow(env.ctx, `SELECT provider_request_id FROM generation_tasks WHERE id=$1`, taskID).Scan(&providerRequestID); err != nil {
		t.Fatal(err)
	}
	if providerRequestID != "provider-request-1" {
		t.Fatalf("provider request ID was not persisted: %q", providerRequestID)
	}
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM provider_task_events WHERE task_id=$1 AND event_type='submitted'`, taskID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 1 {
		t.Fatalf("submit event count=%d", eventCount)
	}
	if err := env.store.RecordProviderPollFailure(env.ctx, taskID, "temporary poll failure", 2*time.Second); err != nil {
		t.Fatal(err)
	}
	var polls int
	if err := env.pool.QueryRow(env.ctx, `SELECT poll_attempts FROM generation_tasks WHERE id=$1`, taskID).Scan(&polls); err != nil {
		t.Fatal(err)
	}
	if polls != 1 {
		t.Fatalf("poll attempt not incremented: %d", polls)
	}
	// Poll failure is recoverable and does not refund. A terminal failure then
	// refunds exactly once, which is checked by the billing test as well.
	var status string
	if err := env.pool.QueryRow(env.ctx, `SELECT status FROM generation_tasks WHERE id=$1`, taskID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "running" {
		t.Fatalf("poll failure unexpectedly terminated task: %s", status)
	}
	if err := env.store.FailTaskUpstream(env.ctx, taskID, "terminal provider failure"); err != nil {
		t.Fatal(err)
	}
	if err := env.store.FailTaskUpstream(env.ctx, taskID, "duplicate terminal provider failure"); err != nil {
		t.Fatal(err)
	}
	if err := env.pool.QueryRow(env.ctx, `SELECT COUNT(*) FROM wallet_ledger WHERE ref_type='task' AND ref_id=$1 AND type='task_refund'`, result.Task.ID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 1 {
		t.Fatalf("terminal refund count=%d", eventCount)
	}
}
