package store

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// Cancellation is a user-visible money movement. This integration test keeps
// the retry guarantee tied to PostgreSQL row locking and the wallet ledger,
// rather than only checking an in-memory mock.
func TestCancelTaskIsIdempotentAfterResponseRetry(t *testing.T) {
	if os.Getenv("SUB2VIDEO_RUN_INTEGRATION") != "1" {
		t.Skip("set SUB2VIDEO_RUN_INTEGRATION=1 with SUB2VIDEO_TEST_DATABASE_URL to run cancellation integration tests")
	}
	env := openIntegrationStore(t)
	textMode, _, _, modelID := testCatalogIDs(t, env)
	user := newIntegrationUser(t, env, 100)
	input := DirectTaskInput{
		DirectRouteRequest: seededDirectRequest(textMode, modelID),
		ClientRequestID:    fmt.Sprintf("cancel-idempotency-%d", time.Now().UnixNano()),
		Title:              "取消重试",
		Prompt:             "测试取消",
	}
	created, err := env.store.CreateTaskDirect(env.ctx, user.ID, input)
	if err != nil {
		t.Fatal(err)
	}

	first, firstRefund, firstBalance, err := env.store.CancelTask(env.ctx, user.ID, created.Task.ID)
	if err != nil {
		t.Fatalf("first cancellation: %v", err)
	}
	second, secondRefund, secondBalance, err := env.store.CancelTask(env.ctx, user.ID, created.Task.ID)
	if err != nil {
		t.Fatalf("retry cancellation: %v", err)
	}
	if first.ID != second.ID || firstRefund != secondRefund || firstBalance != secondBalance {
		t.Fatalf("retry must return the original cancellation result: first=%+v refund=%v balance=%v second=%+v refund=%v balance=%v", first, firstRefund, firstBalance, second, secondRefund, secondBalance)
	}

	var refunds int
	if err := env.pool.QueryRow(env.ctx, `
SELECT COUNT(*) FROM wallet_ledger
WHERE user_id=$1 AND ref_type='task' AND ref_id=$2 AND type='task_refund'`, user.ID, created.Task.ID).Scan(&refunds); err != nil {
		t.Fatal(err)
	}
	if refunds != 1 {
		t.Fatalf("cancellation retry must create one refund ledger entry, got %d", refunds)
	}
}
