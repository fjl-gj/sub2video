package store

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sbu2video/backend/internal/db"
)

// This test exercises the real PostgreSQL transaction and idempotency path.
// It is skipped unless an explicitly named test database is provided.
func TestDirectTaskSubmissionIsIdempotentAfterSupplyChanges(t *testing.T) {
	if os.Getenv("SUB2VIDEO_RUN_INTEGRATION") != "1" {
		t.Skip("set SUB2VIDEO_RUN_INTEGRATION=1 with SUB2VIDEO_TEST_DATABASE_URL to run direct routing integration tests")
	}
	databaseURL := os.Getenv("SUB2VIDEO_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SUB2VIDEO_TEST_DATABASE_URL is not configured; skipping direct routing integration test")
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil || !strings.Contains(strings.ToLower(parsed.Path), "test") {
		t.Fatal("SUB2VIDEO_TEST_DATABASE_URL must target a database whose name contains 'test'")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	base, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	base.MaxConns = 3
	admin, err := pgxpool.NewWithConfig(ctx, base)
	if err != nil {
		t.Skipf("PostgreSQL integration infrastructure is unavailable: %v", err)
	}
	defer admin.Close()
	if err := admin.Ping(ctx); err != nil {
		t.Skipf("PostgreSQL integration infrastructure is unavailable: %v", err)
	}

	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	schema := pgx.Identifier{fmt.Sprintf("direct_it_%x", suffix)}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		_, _ = admin.Exec(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE")
	}()

	isolated := base.Copy()
	isolated.ConnConfig.RuntimeParams["search_path"] = schema
	p, err := pgxpool.NewWithConfig(ctx, isolated)
	if err != nil {
		t.Skipf("isolated PostgreSQL integration pool is unavailable: %v", err)
	}
	defer p.Close()
	if err := p.Ping(ctx); err != nil {
		t.Skipf("isolated PostgreSQL integration pool is unavailable: %v", err)
	}
	if err := db.Migrate(ctx, p); err != nil {
		t.Fatal(err)
	}
	s := New(p)
	if err := s.SeedBasics(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.SeedCreationCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Exec(ctx, `UPDATE suppliers SET token_encrypted='integration-configured-token' WHERE public_id='sup_autodl'`); err != nil {
		t.Fatal(err)
	}
	user, err := s.CreateUserWithWallet(ctx, "direct-it@example.test", "unused", "Direct", "Direct", 100)
	if err != nil {
		t.Fatal(err)
	}

	input := DirectTaskInput{
		DirectRouteRequest: DirectRouteRequest{
			ModeID: "mode_text_to_video", ModelID: "mdl_h3", DurationSeconds: 5,
			Resolution: "768P", AspectRatio: "16:9",
		},
		ClientRequestID: "direct-idempotency-test",
		Title:           "幂等测试",
		Prompt:          "测试镜头",
	}
	first, err := s.CreateTaskDirect(ctx, user.ID, input)
	if err != nil {
		t.Fatalf("first direct submission: %v", err)
	}
	if first.AlreadyDone || first.OrderID == "" || first.Task.ID == "" {
		t.Fatalf("first submission must create a task and order: %+v", first)
	}
	firstBalance := first.Balance

	if _, err := p.Exec(ctx, `UPDATE supplier_executions SET enabled=FALSE WHERE enabled`); err != nil {
		t.Fatal(err)
	}
	second, err := s.CreateTaskDirect(ctx, user.ID, input)
	if err != nil {
		t.Fatalf("idempotent replay after supply disable: %v", err)
	}
	if !second.AlreadyDone || second.Task.ID != first.Task.ID || second.OrderID != first.OrderID || second.Balance != firstBalance {
		t.Fatalf("replay must return the original result without routing or charging: first=%+v second=%+v", first, second)
	}

	var orderCount, taskCount int
	if err := p.QueryRow(ctx, `SELECT COUNT(*) FROM consumption_orders WHERE user_id=$1 AND client_request_id=$2`, user.ID, input.ClientRequestID).Scan(&orderCount); err != nil {
		t.Fatal(err)
	}
	if err := p.QueryRow(ctx, `SELECT COUNT(*) FROM generation_tasks WHERE user_id=$1 AND client_request_id=$2`, user.ID, input.ClientRequestID).Scan(&taskCount); err != nil {
		t.Fatal(err)
	}
	if orderCount != 1 || taskCount != 1 {
		t.Fatalf("idempotency must create one order and task: orders=%d tasks=%d", orderCount, taskCount)
	}
}
