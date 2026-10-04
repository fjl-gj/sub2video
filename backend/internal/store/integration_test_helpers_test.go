package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/sbu2video/backend/internal/db"
	"github.com/sbu2video/backend/internal/redisx"
)

// integrationStore creates a disposable schema inside an explicitly named
// test database. Every test gets a fresh schema, so migration and seed tests do
// not depend on a shared database's history or production rows.
type integrationStore struct {
	ctx   context.Context
	pool  *pgxpool.Pool
	admin *pgxpool.Pool
	rdb   *redis.Client
	store *Store
	name  string
}

func openIntegrationStore(t *testing.T) *integrationStore {
	t.Helper()
	if os.Getenv("SUB2VIDEO_RUN_INTEGRATION") != "1" {
		t.Skip("set SUB2VIDEO_RUN_INTEGRATION=1 with SUB2VIDEO_TEST_DATABASE_URL and SUB2VIDEO_TEST_REDIS_URL to run PostgreSQL/Redis integration tests")
	}
	databaseURL := strings.TrimSpace(os.Getenv("SUB2VIDEO_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("SUB2VIDEO_TEST_DATABASE_URL is not configured; skipping PostgreSQL/Redis integration tests")
	}
	redisURL := strings.TrimSpace(os.Getenv("SUB2VIDEO_TEST_REDIS_URL"))
	if redisURL == "" {
		t.Skip("SUB2VIDEO_TEST_REDIS_URL is not configured; skipping PostgreSQL/Redis integration tests")
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil || !strings.Contains(strings.ToLower(parsed.Path), "test") {
		t.Fatal("SUB2VIDEO_TEST_DATABASE_URL must target a database whose name contains 'test'")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	base, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse test postgres URL: %v", err)
	}
	base.MaxConns = 8
	base.MinConns = 1
	admin, err := pgxpool.NewWithConfig(ctx, base)
	if err != nil {
		t.Skipf("PostgreSQL integration infrastructure is unavailable: %v", err)
	}
	if err := admin.Ping(ctx); err != nil {
		admin.Close()
		t.Skipf("PostgreSQL integration infrastructure is unavailable: %v", err)
	}

	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		admin.Close()
		t.Fatalf("generate integration schema: %v", err)
	}
	name := "s2v_test_" + hex.EncodeToString(suffix[:])
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{name}.Sanitize()); err != nil {
		admin.Close()
		t.Fatalf("create isolated test schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		_, _ = admin.Exec(cleanupCtx, "DROP SCHEMA "+pgx.Identifier{name}.Sanitize()+" CASCADE")
		admin.Close()
	})

	isolated := base.Copy()
	isolated.ConnConfig.RuntimeParams["search_path"] = name + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, isolated)
	if err != nil {
		t.Skipf("isolated PostgreSQL integration pool is unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("isolated PostgreSQL integration pool is unavailable: %v", err)
	}

	rdb, err := redisx.Connect(ctx, redisURL)
	if err != nil {
		t.Skipf("Redis integration infrastructure is unavailable: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	probeKey := "sub2video:integration:" + name
	if err := rdb.Set(ctx, probeKey, "ready", time.Minute).Err(); err != nil {
		t.Fatalf("write redis integration probe: %v", err)
	}
	if got, err := rdb.Get(ctx, probeKey).Result(); err != nil || got != "ready" {
		t.Fatalf("read redis integration probe: got=%q err=%v", got, err)
	}
	t.Cleanup(func() { _ = rdb.Del(context.Background(), probeKey).Err() })

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
		t.Fatalf("seed creation catalog: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE suppliers SET token_encrypted='integration-configured-token', enabled=TRUE, status='healthy' WHERE public_id='sup_autodl'`); err != nil {
		t.Fatalf("configure test supplier credential: %v", err)
	}
	return &integrationStore{ctx: ctx, pool: pool, admin: admin, rdb: rdb, store: s, name: name}
}

func newIntegrationUser(t *testing.T, env *integrationStore, balance float64) *User {
	t.Helper()
	email := fmt.Sprintf("%s-%d@sub2video.test", t.Name(), time.Now().UnixNano())
	user, err := env.store.CreateUserWithWallet(env.ctx, email, "unused-test-password", "Integration", "Integration", balance)
	if err != nil {
		t.Fatalf("create integration user: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		// The order mirrors FK dependencies and is intentionally scoped to this
		// test user. The schema itself is dropped after the test as a final guard.
		for _, statement := range []string{
			`DELETE FROM admin_audit_logs WHERE actor_user_id=$1`,
			`DELETE FROM task_quotes WHERE user_id=$1`,
			`DELETE FROM consumption_orders WHERE user_id=$1`,
			`DELETE FROM generation_tasks WHERE user_id=$1`,
			`DELETE FROM wallet_ledger WHERE user_id=$1`,
			`DELETE FROM projects WHERE user_id=$1`,
			`DELETE FROM user_drafts WHERE user_id=$1`,
			`DELETE FROM workspaces WHERE user_id=$1`,
			`DELETE FROM wallets WHERE user_id=$1`,
			`DELETE FROM users WHERE id=$1`,
		} {
			_, _ = env.pool.Exec(cleanupCtx, statement, user.ID)
		}
	})
	return user
}

func seededDirectRequest(modeID, modelID string) DirectRouteRequest {
	return DirectRouteRequest{
		ModeID: modeID, CarrierID: "car_minimax", ModelID: modelID,
		DurationSeconds: 5, Resolution: "768P", AspectRatio: "16:9",
	}
}
