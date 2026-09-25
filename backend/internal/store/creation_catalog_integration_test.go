package store

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sbu2video/backend/internal/db"
	"github.com/sbu2video/backend/internal/redisx"
)

// This test intentionally requires an explicitly named test database. It
// exercises the real PostgreSQL transaction path rather than a mock, and is
// skipped by default on developer machines and production deployments.
func TestCreationQuoteChargeAndRefundIntegration(t *testing.T) {
	if os.Getenv("SUB2VIDEO_RUN_INTEGRATION") != "1" {
		t.Skip("set SUB2VIDEO_RUN_INTEGRATION=1 with SUB2VIDEO_TEST_DATABASE_URL and SUB2VIDEO_TEST_REDIS_URL to run integration tests")
	}
	databaseURL := os.Getenv("SUB2VIDEO_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("SUB2VIDEO_TEST_DATABASE_URL is required when integration tests are enabled")
	}
	redisURL := os.Getenv("SUB2VIDEO_TEST_REDIS_URL")
	if redisURL == "" {
		t.Fatal("SUB2VIDEO_TEST_REDIS_URL is required when integration tests are enabled")
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil || !strings.Contains(strings.ToLower(parsed.Path), "test") {
		t.Fatal("SUB2VIDEO_TEST_DATABASE_URL must target a database whose name contains 'test'")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect test postgres: %v", err)
	}
	defer pool.Close()
	rdb, err := redisx.Connect(ctx, redisURL)
	if err != nil {
		t.Fatalf("connect test redis: %v", err)
	}
	defer rdb.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("second migrate must be idempotent: %v", err)
	}

	s := New(pool)
	if err := s.SeedBasics(ctx); err != nil {
		t.Fatalf("seed basics: %v", err)
	}
	if err := s.SeedCreationCatalog(ctx); err != nil {
		t.Fatalf("seed catalog: %v", err)
	}
	// Platform retail prices are mode-level and therefore have a NULL model
	// reference. The admin listing must normalize those nullable joins instead
	// of failing the whole management page during its initial load.
	if retailPrices, err := s.ListRetailPriceRules(ctx); err != nil {
		t.Fatalf("list mode-level retail prices: %v", err)
	} else if len(retailPrices) == 0 {
		t.Fatal("seed catalog must expose at least one retail price rule")
	}
	legacySuffix := fmt.Sprintf("%d", time.Now().UnixNano())
	legacyCarrierID := "car_legacy_" + legacySuffix
	legacyModelID := "mdl_legacy_" + legacySuffix
	legacyRouteID := "rte_legacy_" + legacySuffix
	legacyCode := "legacy-workflow-" + legacySuffix
	if _, err := pool.Exec(ctx, `
INSERT INTO carriers (public_id, code, name, enabled, sort_order)
VALUES ($1, $2, 'Legacy workflow carrier', TRUE, 9999)`, legacyCarrierID, legacyCode); err != nil {
		t.Fatalf("seed legacy carrier: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO models (public_id, carrier_id, code, name, client_workflow_id, enabled, sort_order)
SELECT $1, id, 'legacy-mode', 'Legacy workflow model', 'wf_legacy', TRUE, 1
FROM carriers WHERE public_id=$2`, legacyModelID, legacyCarrierID); err != nil {
		t.Fatalf("seed legacy workflow-bound model: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO supplier_routes (public_id, supplier_id, model_id, workflow_id, enabled)
SELECT $1, supplier.id, model.id, $3, TRUE
FROM suppliers supplier, models model
WHERE supplier.public_id='sup_autodl' AND model.public_id=$2`, legacyRouteID, legacyModelID, "minimax_h3_legacy_"+legacySuffix); err != nil {
		t.Fatalf("seed legacy supplier route: %v", err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("legacy retirement migration: %v", err)
	}
	var legacyCarrierEnabled bool
	if err := pool.QueryRow(ctx, `SELECT enabled FROM carriers WHERE public_id=$1`, legacyCarrierID).Scan(&legacyCarrierEnabled); err != nil || legacyCarrierEnabled {
		t.Fatalf("legacy carrier must be retired: enabled=%t err=%v", legacyCarrierEnabled, err)
	}
	var legacyRouteCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM supplier_routes WHERE public_id=$1`, legacyRouteID).Scan(&legacyRouteCount); err != nil || legacyRouteCount != 0 {
		t.Fatalf("legacy H3 supplier route must be deleted: count=%d err=%v", legacyRouteCount, err)
	}
	managedCarriers, managedModels, err := s.ListCarriersWithModels(ctx)
	if err != nil {
		t.Fatalf("list managed catalog: %v", err)
	}
	for _, item := range managedCarriers {
		if item.Code == legacyCode {
			t.Fatalf("legacy workflow carrier leaked into management catalog: %#v", item)
		}
	}
	for _, item := range managedModels {
		if item.CarrierCode == legacyCode {
			t.Fatalf("legacy workflow model leaked into management catalog: %#v", item)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM supplier_routes WHERE public_id=$1`, legacyRouteID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM models WHERE public_id=$1`, legacyModelID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM carriers WHERE public_id=$1`, legacyCarrierID)
	})
	if _, err := pool.Exec(ctx, `UPDATE suppliers SET token_encrypted='integration-configured-token' WHERE public_id='sup_autodl'`); err != nil {
		t.Fatalf("configure integration supplier: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE supplier_offerings SET failure_count=0, last_failure_at=NULL, health_status='unknown', enabled=TRUE WHERE public_id='off_h3_text_lightx'`); err != nil {
		t.Fatalf("reset integration offering: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE supplier_executions SET failure_count=0, last_failure_at=NULL, health_status='unknown', enabled=TRUE WHERE public_id='exec_off_h3_text_lightx'`); err != nil {
		t.Fatalf("reset integration execution: %v", err)
	}

	email := fmt.Sprintf("catalog-it-%d@sub2video.test", time.Now().UnixNano())
	user, err := s.CreateUserWithWallet(ctx, email, "not-used-by-store-test", "Integration", "Integration", 100)
	if err != nil {
		t.Fatalf("create integration user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM admin_audit_logs WHERE actor_user_id=$1`, user.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM generation_tasks WHERE user_id=$1`, user.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM task_quotes WHERE user_id=$1`, user.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM wallet_ledger WHERE user_id=$1`, user.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM projects WHERE user_id=$1`, user.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM user_drafts WHERE user_id=$1`, user.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM workspaces WHERE user_id=$1`, user.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM wallets WHERE user_id=$1`, user.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, user.ID)
		_, _ = pool.Exec(context.Background(), `UPDATE supplier_offerings SET failure_count=0, last_failure_at=NULL, health_status='unknown', enabled=TRUE WHERE public_id='off_h3_text_lightx'`)
		_, _ = pool.Exec(context.Background(), `UPDATE supplier_executions SET failure_count=0, last_failure_at=NULL, health_status='unknown', enabled=TRUE WHERE public_id='exec_off_h3_text_lightx'`)
	})
	if err := s.AppendAdminAudit(ctx, user.ID, "integration.audit", "test", "catalog", map[string]any{"verified": true}); err != nil {
		t.Fatalf("append admin audit: %v", err)
	}
	auditRows, err := s.ListAdminAuditLogs(ctx, 10)
	if err != nil || len(auditRows) == 0 || auditRows[0].Action != "integration.audit" {
		t.Fatalf("list admin audit: rows=%d err=%v", len(auditRows), err)
	}

	catalog, err := s.ListClientCreationCatalog(ctx)
	if err != nil {
		t.Fatalf("list client catalog: %v", err)
	}
	rows, err := pool.Query(ctx, `
SELECT workflow_code
FROM supplier_executions
WHERE enabled
ORDER BY workflow_code`)
	if err != nil {
		t.Fatalf("list enabled H3 executions: %v", err)
	}
	var enabledWorkflows []string
	for rows.Next() {
		var workflowCode string
		if err := rows.Scan(&workflowCode); err != nil {
			rows.Close()
			t.Fatalf("scan enabled H3 execution: %v", err)
		}
		enabledWorkflows = append(enabledWorkflows, workflowCode)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatalf("iterate enabled H3 executions: %v", err)
	}
	rows.Close()
	wantEnabledWorkflows := []string{
		"minimax_h3_lightx2v",
		"minimax_h3_lightx2v_no_pic",
		"minimax_h3_zm_u08",
		"minimax_h3_zm_u24",
	}
	if len(enabledWorkflows) != len(wantEnabledWorkflows) {
		t.Fatalf("enabled workflow count: got=%v want=%v", enabledWorkflows, wantEnabledWorkflows)
	}
	for i := range wantEnabledWorkflows {
		if enabledWorkflows[i] != wantEnabledWorkflows[i] {
			t.Fatalf("enabled workflow %d: got=%s want=%s", i, enabledWorkflows[i], wantEnabledWorkflows[i])
		}
	}
	offeringRows, err := pool.Query(ctx, `
SELECT workflow_code
FROM supplier_offerings
ORDER BY workflow_code`)
	if err != nil {
		t.Fatalf("list retained H3 offerings: %v", err)
	}
	var retainedOfferings []string
	for offeringRows.Next() {
		var workflowCode string
		if err := offeringRows.Scan(&workflowCode); err != nil {
			offeringRows.Close()
			t.Fatalf("scan retained H3 offering: %v", err)
		}
		retainedOfferings = append(retainedOfferings, workflowCode)
	}
	if err := offeringRows.Err(); err != nil {
		offeringRows.Close()
		t.Fatalf("iterate retained H3 offerings: %v", err)
	}
	offeringRows.Close()
	if len(retainedOfferings) != len(wantEnabledWorkflows) {
		t.Fatalf("retained offering count: got=%v want=%v", retainedOfferings, wantEnabledWorkflows)
	}
	for i := range wantEnabledWorkflows {
		if retainedOfferings[i] != wantEnabledWorkflows[i] {
			t.Fatalf("retained offering %d: got=%s want=%s", i, retainedOfferings[i], wantEnabledWorkflows[i])
		}
	}
	if price, available, err := s.CheapestAvailableRetailPrice(ctx); err != nil || !available || price <= 0 {
		t.Fatalf("overview reference price must come from an executable catalog combination: price=%f available=%t err=%v", price, available, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE suppliers SET token_encrypted='' WHERE public_id='sup_autodl'`); err != nil {
		t.Fatalf("clear supplier token for capability filtering: %v", err)
	}
	hiddenCatalog, err := s.ListClientCreationCatalog(ctx)
	if err != nil || len(hiddenCatalog.Carriers) != 0 || len(hiddenCatalog.Models) != 0 {
		t.Fatalf("client catalog must hide unconfigured supplier capacity: carriers=%d models=%d err=%v", len(hiddenCatalog.Carriers), len(hiddenCatalog.Models), err)
	}
	if _, err := pool.Exec(ctx, `UPDATE suppliers SET token_encrypted='integration-configured-token' WHERE public_id='sup_autodl'`); err != nil {
		t.Fatalf("restore supplier token after capability filtering: %v", err)
	}
	var textMode, multiImageMode, h3Model string
	for _, mode := range catalog.Modes {
		if mode.Code == "text_to_video" {
			textMode = mode.ID
		}
		if mode.Code == "multi_image_to_video" {
			multiImageMode = mode.ID
		}
	}
	for _, model := range catalog.Models {
		if model.Code == "h3" {
			h3Model = model.ID
		}
	}
	if textMode == "" || multiImageMode == "" || h3Model == "" {
		t.Fatalf("expected configured text mode and H3 model, got modes=%d models=%d", len(catalog.Modes), len(catalog.Models))
	}
	// Audio pricing is independent from the no-audio row. A valid multi-image
	// audio request must route and price without exposing its chosen execution.
	directRoute, err := s.ResolveDirectRoute(ctx, DirectRouteRequest{
		ModeID: multiImageMode, ModelID: h3Model, DurationSeconds: 12,
		Resolution: "768P", AspectRatio: "16:9", GenerateAudio: true,
	})
	if err != nil {
		t.Fatalf("resolve multi-image audio route: %v", err)
	}
	if directRoute.Retail.PricePerSecond != 0.03 || directRoute.Total != 0.36 {
		t.Fatalf("multi-image audio retail price mismatch: rate=%f total=%f", directRoute.Retail.PricePerSecond, directRoute.Total)
	}
	if directRoute.Offering.WorkflowCode != "minimax_h3_zm_u08" && directRoute.Offering.WorkflowCode != "minimax_h3_zm_u24" {
		t.Fatalf("multi-image route selected retired workflow: %s", directRoute.Offering.WorkflowCode)
	}

	options, err := s.ResolveCreationOptions(ctx, textMode, h3Model)
	if err != nil || len(options.Durations) == 0 || len(options.Resolutions) == 0 || len(options.AspectRatios) == 0 {
		t.Fatalf("resolve text options: options=%+v err=%v", options, err)
	}
	textRoute, err := s.ResolveDirectRoute(ctx, DirectRouteRequest{
		ModeID: textMode, ModelID: h3Model, DurationSeconds: 5,
		Resolution: "768P", AspectRatio: "16:9", GenerateAudio: false,
	})
	if err != nil || textRoute.Offering.WorkflowCode != "minimax_h3_lightx2v_no_pic" {
		t.Fatalf("text route must use lightx no-pic workflow: route=%+v err=%v", textRoute, err)
	}
	request := QuoteInput{
		ModeID: textMode, ModelID: h3Model, DurationSeconds: options.Durations[0],
		Resolution: options.Resolutions[0], AspectRatio: options.AspectRatios[0], GenerateAudio: false,
	}
	recommended, err := s.CreateQuote(ctx, user.ID, QuoteInput{
		ModeID: textMode, DurationSeconds: options.Durations[0], Resolution: options.Resolutions[0],
		AspectRatio: options.AspectRatios[0], GenerateAudio: false,
	})
	if err != nil {
		t.Fatalf("create recommended quote: %v", err)
	}
	var recommendationAllowed bool
	if err := pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1
  FROM mode_model_policies p
  JOIN creation_modes mode ON mode.id=p.mode_id
  JOIN models model ON model.id=p.model_id
  WHERE mode.public_id=$1 AND model.public_id=$2 AND p.enabled
)`, textMode, recommended.ModelID).Scan(&recommendationAllowed); err != nil || !recommendationAllowed {
		t.Fatalf("smart recommendation must select only a mode policy model: model=%s allowed=%t err=%v", recommended.ModelID, recommendationAllowed, err)
	}

	expired, err := s.CreateQuote(ctx, user.ID, request)
	if err != nil {
		t.Fatalf("create quote for expiry check: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE task_quotes SET expires_at=NOW()-INTERVAL '1 second' WHERE public_id=$1`, expired.ID); err != nil {
		t.Fatalf("expire quote: %v", err)
	}
	if _, _, _, err := s.CreateTaskFromQuote(ctx, user.ID, expired.ID, "expired", "neon street", SourceMeta{}); err == nil || !strings.Contains(err.Error(), "报价已过期") {
		t.Fatalf("expired quote must not create a task: %v", err)
	}

	offline, err := s.CreateQuote(ctx, user.ID, request)
	if err != nil {
		t.Fatalf("create quote for supply-offline check: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE supplier_offerings SET enabled=FALSE WHERE public_id=$1`, offline.OfferingID); err != nil {
		t.Fatalf("disable offering: %v", err)
	}
	if _, _, _, err := s.CreateTaskFromQuote(ctx, user.ID, offline.ID, "offline", "neon street", SourceMeta{}); err == nil || !strings.Contains(err.Error(), "报价已过期或不可用") {
		t.Fatalf("offline offering must invalidate the quote: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE supplier_offerings SET enabled=TRUE WHERE public_id=$1`, offline.OfferingID); err != nil {
		t.Fatalf("restore offering after supply-offline check: %v", err)
	}

	insufficient, err := s.CreateQuote(ctx, user.ID, request)
	if err != nil {
		t.Fatalf("create quote for insufficient-balance check: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE wallets SET balance=0 WHERE user_id=$1`, user.ID); err != nil {
		t.Fatalf("empty wallet: %v", err)
	}
	if _, _, _, err := s.CreateTaskFromQuote(ctx, user.ID, insufficient.ID, "insufficient", "neon street", SourceMeta{}); err == nil || !strings.Contains(err.Error(), "insufficient balance") {
		t.Fatalf("insufficient balance must not consume the quote or create a task: %v", err)
	}
	var insufficientConsumedAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT consumed_at FROM task_quotes WHERE public_id=$1`, insufficient.ID).Scan(&insufficientConsumedAt); err != nil || insufficientConsumedAt != nil {
		t.Fatalf("insufficient-balance quote must remain unconsumed: consumed_at=%v err=%v", insufficientConsumedAt, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE wallets SET balance=100 WHERE user_id=$1`, user.ID); err != nil {
		t.Fatalf("restore wallet after insufficient-balance check: %v", err)
	}

	quote, err := s.CreateQuote(ctx, user.ID, request)
	if err != nil {
		t.Fatalf("create explicit-model quote: %v", err)
	}
	if quote.ModelID != h3Model {
		t.Fatalf("explicit model was substituted: got %s want %s", quote.ModelID, h3Model)
	}
	if quote.SupplierCode == "" || quote.OfferingID == "" || quote.CostRuleID == "" {
		t.Fatal("server quote must lock an executable offering and procurement snapshot")
	}
	if _, err := s.CreateSupplierOffering(ctx, SupplierOffering{
		SupplierID: "sup_autodl", ModelID: h3Model, ModeID: textMode, Name: "unsupported adapter",
		UpstreamWorkflowID: "unsupported_adapter_test", AdapterCode: "not_deployed_v1", Enabled: true,
		MinDurationSec: 1, MaxDurationSec: 5, Resolutions: []string{"768P"}, AspectRatios: []string{"16:9"},
	}); err == nil {
		t.Fatal("an unsupported provider adapter must not be configured or exposed to client quotes")
	}

	task, ledgerID, balance, err := s.CreateTaskFromQuote(ctx, user.ID, quote.ID, "integration text", "neon street", SourceMeta{})
	if err != nil {
		t.Fatalf("create task from quote: %v", err)
	}
	if ledgerID == "" || balance >= 100 {
		t.Fatalf("task charge was not applied atomically: ledger=%q balance=%f", ledgerID, balance)
	}
	if _, _, _, err := s.CreateTaskFromQuote(ctx, user.ID, quote.ID, "duplicate", "neon street", SourceMeta{}); err == nil || !strings.Contains(err.Error(), "报价已使用") {
		t.Fatalf("reusing a quote must fail with quote-used error, got %v", err)
	}

	var internalID int64
	var quoteID, executionMode string
	if err := pool.QueryRow(ctx, `SELECT id, quote_public_id, execution_mode FROM generation_tasks WHERE public_id=$1`, task.ID).Scan(&internalID, &quoteID, &executionMode); err != nil {
		t.Fatalf("read task snapshot: %v", err)
	}
	if quoteID != quote.ID || executionMode != "provider" {
		t.Fatalf("new task must retain quote and provider execution mode: quote=%s mode=%s", quoteID, executionMode)
	}
	pending, err := s.ListPendingProviderTasks(ctx, 100)
	if err != nil {
		t.Fatalf("list pending provider tasks: %v", err)
	}
	var workerTask *ProviderTask
	for i := range pending {
		if pending[i].PublicID == task.ID {
			workerTask = &pending[i]
			break
		}
	}
	if workerTask == nil {
		t.Fatal("quote-backed task must be visible to the provider worker")
	}
	if workerTask.QuotePublicID != quote.ID || workerTask.SupplierRouteID != quote.OfferingID ||
		workerTask.UpstreamWorkflowID != quote.UpstreamWorkflow || workerTask.AdapterCode != quote.AdapterCode {
		t.Fatalf("worker must consume the immutable quote snapshot: task=%+v quote=%+v", workerTask, quote)
	}
	claimed, err := s.ClaimTaskForDispatch(ctx, internalID)
	if err != nil || !claimed {
		t.Fatalf("first worker must claim quote-backed task: claimed=%t err=%v", claimed, err)
	}
	claimed, err = s.ClaimTaskForDispatch(ctx, internalID)
	if err != nil || claimed {
		t.Fatalf("a second worker must not claim the same submission: claimed=%t err=%v", claimed, err)
	}
	if err := s.MarkTaskWaitingForConfiguration(ctx, internalID, "integration configuration wait"); err != nil {
		t.Fatalf("release claim while configuration is unavailable: %v", err)
	}
	claimed, err = s.ClaimTaskForDispatch(ctx, internalID)
	if err != nil || !claimed {
		t.Fatalf("released configuration wait must become dispatchable: claimed=%t err=%v", claimed, err)
	}
	if err := s.FailTaskUpstream(ctx, internalID, "integration upstream failure"); err != nil {
		t.Fatalf("first failure refund: %v", err)
	}
	if err := s.FailTaskUpstream(ctx, internalID, "duplicate failure"); err != nil {
		t.Fatalf("duplicate failure must remain idempotent: %v", err)
	}
	var refundCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM wallet_ledger WHERE user_id=$1 AND ref_type='task' AND ref_id=$2 AND type='task_refund'`, user.ID, task.ID).Scan(&refundCount); err != nil {
		t.Fatalf("count refunds: %v", err)
	}
	if refundCount != 1 {
		t.Fatalf("expected exactly one refund ledger entry, got %d", refundCount)
	}

	// A client cancellation is also a one-time refund. A late upstream write
	// must never revive that terminal task after the refund has been issued.
	cancelQuote, err := s.CreateQuote(ctx, user.ID, request)
	if err != nil {
		t.Fatalf("create quote for cancellation check: %v", err)
	}
	cancelTask, _, _, err := s.CreateTaskFromQuote(ctx, user.ID, cancelQuote.ID, "cancelled", "neon street", SourceMeta{})
	if err != nil {
		t.Fatalf("create cancellation task: %v", err)
	}
	var cancelTaskID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM generation_tasks WHERE public_id=$1`, cancelTask.ID).Scan(&cancelTaskID); err != nil {
		t.Fatalf("read cancellation task: %v", err)
	}
	_, cancelRefund, cancelBalance, err := s.CancelTask(ctx, user.ID, cancelTask.ID)
	if err != nil || cancelRefund != cancelTask.Cost || cancelBalance != 100 {
		t.Fatalf("cancel must refund once: refund=%f balance=%f err=%v", cancelRefund, cancelBalance, err)
	}
	if _, _, _, err := s.CancelTask(ctx, user.ID, cancelTask.ID); err == nil {
		t.Fatal("a cancelled task must not be refunded twice")
	}
	if err := s.MarkTaskSubmitted(ctx, cancelTaskID, "late-upstream-id", "RUNNING"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("late upstream submission must not revive cancelled task: %v", err)
	}
	if err := s.CompleteTaskUpstream(ctx, cancelTaskID, "https://upstream.invalid/result.mp4", "https://storage.invalid/result.mp4", 5); !errors.Is(err, ErrNotFound) {
		t.Fatalf("late upstream completion must not revive cancelled task: %v", err)
	}
	var cancelStatus, cancelUpstreamStatus, cancelReason string
	if err := pool.QueryRow(ctx, `SELECT status, upstream_status, fail_reason FROM generation_tasks WHERE id=$1`, cancelTaskID).Scan(&cancelStatus, &cancelUpstreamStatus, &cancelReason); err != nil || cancelStatus != "failed" || cancelUpstreamStatus != "CANCELLED_BY_USER" || cancelReason != "用户终止并退款" {
		t.Fatalf("cancelled task state changed after late upstream updates: status=%s upstream=%s reason=%s err=%v", cancelStatus, cancelUpstreamStatus, cancelReason, err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM wallet_ledger WHERE user_id=$1 AND ref_type='task' AND ref_id=$2 AND type='task_refund'`, user.ID, cancelTask.ID).Scan(&refundCount); err != nil || refundCount != 1 {
		t.Fatalf("cancel refund must remain idempotent: count=%d err=%v", refundCount, err)
	}
	for i := 0; i < 2; i++ {
		healthQuote, err := s.CreateQuote(ctx, user.ID, request)
		if err != nil {
			t.Fatalf("create health-check quote %d: %v", i, err)
		}
		healthTask, _, _, err := s.CreateTaskFromQuote(ctx, user.ID, healthQuote.ID, fmt.Sprintf("health-%d", i), "neon street", SourceMeta{})
		if err != nil {
			t.Fatalf("create health-check task %d: %v", i, err)
		}
		var healthTaskID int64
		if err := pool.QueryRow(ctx, `SELECT id FROM generation_tasks WHERE public_id=$1`, healthTask.ID).Scan(&healthTaskID); err != nil {
			t.Fatalf("read health-check task %d: %v", i, err)
		}
		if err := s.FailTaskUpstream(ctx, healthTaskID, "health threshold check"); err != nil {
			t.Fatalf("fail health-check task %d: %v", i, err)
		}
	}
	var offeringEnabled bool
	var offeringHealth string
	var failureCount int
	if err := pool.QueryRow(ctx, `SELECT enabled, health_status, failure_count FROM supplier_offerings WHERE public_id=$1`, quote.OfferingID).Scan(&offeringEnabled, &offeringHealth, &failureCount); err != nil {
		t.Fatalf("read offering health: %v", err)
	}
	if offeringEnabled || offeringHealth != "unavailable" || failureCount != 3 {
		t.Fatalf("three upstream failures must disable the offering: enabled=%t health=%s failures=%d", offeringEnabled, offeringHealth, failureCount)
	}
	refundedBalance, err := s.GetBalance(ctx, user.ID)
	if err != nil || refundedBalance != 100 {
		t.Fatalf("refund balance: got %f err=%v", refundedBalance, err)
	}
}
