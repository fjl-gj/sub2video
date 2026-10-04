package store

import (
	"errors"
	"testing"
)

func reconciliationFixture(t *testing.T, env *integrationStore) (*User, *DirectTaskResult, int64, string) {
	t.Helper()
	user := newIntegrationUser(t, env, 100)
	if _, err := env.pool.Exec(env.ctx, `UPDATE users SET role='admin' WHERE id=$1`, user.ID); err != nil {
		t.Fatal(err)
	}
	result, err := env.store.CreateTaskDirect(env.ctx, user.ID, DirectTaskInput{DirectRouteRequest: seededDirectRequest("mode_text_to_video", "mdl_h3"), ClientRequestID: "reconcile-" + NewPublicID(), Prompt: "测试镜头", Title: "核对测试"})
	if err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := env.pool.QueryRow(env.ctx, `SELECT id FROM generation_tasks WHERE public_id=$1`, result.Task.ID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	claimed, token, err := env.store.ClaimTaskForDispatchWithToken(env.ctx, id)
	if err != nil || !claimed {
		t.Fatalf("claim=%v err=%v", claimed, err)
	}
	if err := env.store.MarkTaskSubmitStarted(env.ctx, id, token); err != nil {
		t.Fatal(err)
	}
	return user, result, id, token
}

func TestProviderCrashRecoveryAndResumeIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	admin, result, id, token := reconciliationFixture(t, env)
	if _, err := env.pool.Exec(env.ctx, `UPDATE generation_tasks SET claim_expires_at=NOW()-INTERVAL '1 second' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if err := env.store.RecoverExpiredProviderSubmissions(env.ctx); err != nil {
		t.Fatal(err)
	}
	if err := env.store.RecoverExpiredProviderSubmissions(env.ctx); err != nil {
		t.Fatal(err)
	}
	claimed, _, err := env.store.ClaimTaskForDispatchWithToken(env.ctx, id)
	if err != nil || claimed {
		t.Fatalf("uncertain submission reclaimed: %v %v", claimed, err)
	}
	if err := env.store.MarkTaskSubmittedWithToken(env.ctx, id, "stale-response", "RUNNING", token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stale submit not fenced: %v", err)
	}
	if err := env.store.FailTaskUpstreamWithToken(env.ctx, id, "stale failure", token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stale refund not fenced: %v", err)
	}
	input := TaskReconciliationInput{Action: "resume", UpstreamTaskID: "provider-verified-1", Evidence: "已核对供应商记录中的输入参数与提交时间", Confirmed: true}
	out, err := env.store.ReconcileTaskSubmission(env.ctx, admin.ID, result.Task.ID, input)
	if err != nil || out.AlreadyResolved || out.Refund != 0 {
		t.Fatalf("resume=%+v err=%v", out, err)
	}
	claimed, pollToken, err := env.store.ClaimTaskForPollWithToken(env.ctx, id)
	if err != nil || !claimed {
		t.Fatalf("reconciled task not pollable: %v %v", claimed, err)
	}
	if err := env.store.UpdateTaskUpstreamWithRetryAndToken(env.ctx, id, "RUNNING", 20, 0, token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stale poll not fenced: %v", err)
	}
	if err := env.store.CompleteTaskUpstreamWithToken(env.ctx, id, "https://upstream.test/result.mp4", "https://storage.test/result.mp4", 5, pollToken); err != nil {
		t.Fatal(err)
	}
	out, err = env.store.ReconcileTaskSubmission(env.ctx, admin.ID, result.Task.ID, input)
	if err != nil || !out.AlreadyResolved {
		t.Fatalf("resume replay=%+v err=%v", out, err)
	}
	var auditCount, eventCount, refunds int
	if err := env.pool.QueryRow(env.ctx, `SELECT (SELECT COUNT(*) FROM admin_audit_logs WHERE target_id=$1 AND action='task.reconcile'),(SELECT COUNT(*) FROM provider_task_events WHERE task_id=$2 AND event_type='submit_unknown'),(SELECT COUNT(*) FROM wallet_ledger WHERE ref_id=$1 AND type='task_refund')`, result.Task.ID, id).Scan(&auditCount, &eventCount, &refunds); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 || eventCount != 1 || refunds != 0 {
		t.Fatalf("audit=%d unknown=%d refunds=%d", auditCount, eventCount, refunds)
	}
}

func TestProviderReconciliationRefundAtomicAndIdempotentIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	admin, result, id, token := reconciliationFixture(t, env)
	if err := env.store.MarkTaskSubmissionUnknown(env.ctx, id, "response lost", token); err != nil {
		t.Fatal(err)
	}
	input := TaskReconciliationInput{Action: "refund", Evidence: "供应商后台和客服工单确认该请求未创建任务", Confirmed: true}
	// A failing audit insert must roll back the refund, order and task together.
	if _, err := env.pool.Exec(env.ctx, `CREATE FUNCTION reject_reconciliation_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected audit failure'; END $$;
CREATE TRIGGER reject_reconciliation_audit BEFORE INSERT ON admin_audit_logs FOR EACH ROW EXECUTE FUNCTION reject_reconciliation_audit()`); err != nil {
		t.Fatal(err)
	}
	if _, err := env.store.ReconcileTaskSubmission(env.ctx, admin.ID, result.Task.ID, input); err == nil {
		t.Fatal("injected audit failure did not abort")
	}
	var balance float64
	var status, upstream string
	if err := env.pool.QueryRow(env.ctx, `SELECT w.balance::float8,t.status,t.upstream_status FROM wallets w JOIN generation_tasks t ON t.user_id=w.user_id WHERE t.id=$1`, id).Scan(&balance, &status, &upstream); err != nil {
		t.Fatal(err)
	}
	if balance != result.Balance || status != "running" || upstream != "SUBMIT_UNKNOWN" {
		t.Fatalf("partial refund committed: %v %s %s", balance, status, upstream)
	}
	if _, err := env.pool.Exec(env.ctx, `DROP TRIGGER reject_reconciliation_audit ON admin_audit_logs`); err != nil {
		t.Fatal(err)
	}
	out, err := env.store.ReconcileTaskSubmission(env.ctx, admin.ID, result.Task.ID, input)
	if err != nil || out.Refund <= 0 {
		t.Fatalf("refund=%+v err=%v", out, err)
	}
	out, err = env.store.ReconcileTaskSubmission(env.ctx, admin.ID, result.Task.ID, input)
	if err != nil || !out.AlreadyResolved {
		t.Fatalf("replay=%+v err=%v", out, err)
	}
	input.Action, input.UpstreamTaskID = "resume", "different-task"
	if _, err := env.store.ReconcileTaskSubmission(env.ctx, admin.ID, result.Task.ID, input); !errors.Is(err, ErrReconciliationConflict) {
		t.Fatalf("conflicting reconciliation allowed: %v", err)
	}
	var refunds, audits int
	var orderStatus string
	if err := env.pool.QueryRow(env.ctx, `SELECT (SELECT balance::float8 FROM wallets WHERE user_id=$1),(SELECT COUNT(*) FROM wallet_ledger WHERE ref_id=$2 AND type='task_refund'),(SELECT COUNT(*) FROM admin_audit_logs WHERE target_id=$2 AND action='task.reconcile'),(SELECT status FROM consumption_orders WHERE task_id=$3)`, admin.ID, result.Task.ID, id).Scan(&balance, &refunds, &audits, &orderStatus); err != nil {
		t.Fatal(err)
	}
	if balance != 100 || refunds != 1 || audits != 1 || orderStatus != "refunded" {
		t.Fatalf("balance=%v refunds=%d audits=%d order=%s", balance, refunds, audits, orderStatus)
	}
}

func TestProviderKnownAcceptedIDCannotBeRefundedAsUnacceptedIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	admin, result, id, token := reconciliationFixture(t, env)
	if err := env.store.MarkTaskSubmissionUnknownWithRequestID(env.ctx, id, "persistence failed", token, "accepted-task"); err != nil {
		t.Fatal(err)
	}
	input := TaskReconciliationInput{Action: "refund", Evidence: "声称供应商后台没有任务但已有已受理的任务编号", Confirmed: true}
	if _, err := env.store.ReconcileTaskSubmission(env.ctx, admin.ID, result.Task.ID, input); !errors.Is(err, ErrReconciliationConflict) {
		t.Fatalf("accepted task incorrectly refunded: %v", err)
	}
	input.Action, input.UpstreamTaskID = "resume", "different-task"
	if _, err := env.store.ReconcileTaskSubmission(env.ctx, admin.ID, result.Task.ID, input); !errors.Is(err, ErrReconciliationConflict) {
		t.Fatalf("accepted task reassigned: %v", err)
	}
	input.UpstreamTaskID = "accepted-task"
	if _, err := env.store.ReconcileTaskSubmission(env.ctx, admin.ID, result.Task.ID, input); err != nil {
		t.Fatal(err)
	}
}
