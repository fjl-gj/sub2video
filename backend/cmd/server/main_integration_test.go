package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrateOnlyCLIHasNoServiceSideEffectsIntegration(t *testing.T) {
	if os.Getenv("SUB2VIDEO_RUN_INTEGRATION") != "1" || os.Getenv("SUB2VIDEO_TEST_DATABASE_URL") == "" {
		t.Skip("requires SUB2VIDEO_RUN_INTEGRATION=1 and a dedicated SUB2VIDEO_TEST_DATABASE_URL")
	}
	base, err := pgxpool.ParseConfig(os.Getenv("SUB2VIDEO_TEST_DATABASE_URL"))
	if err != nil || !strings.Contains(strings.ToLower(base.ConnConfig.Database), "test") {
		t.Fatal("requires a valid dedicated database whose name contains test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	admin, err := pgxpool.NewWithConfig(ctx, base)
	if err != nil {
		t.Fatal("connect dedicated test database")
	}
	defer admin.Close()
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	name := "release_cli_" + hex.EncodeToString(random[:])
	schema := pgx.Identifier{name}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal("create isolated command schema: ", err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanup, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error("clean isolated command schema: ", err)
		}
	}()
	base.ConnConfig.RuntimeParams["search_path"] = name
	commandDSN, err := url.Parse(os.Getenv("SUB2VIDEO_TEST_DATABASE_URL"))
	if err != nil || (commandDSN.Scheme != "postgres" && commandDSN.Scheme != "postgresql") {
		t.Fatal("CLI integration requires a PostgreSQL URL")
	}
	query := commandDSN.Query()
	query.Set("search_path", name)
	commandDSN.RawQuery = query.Encode()
	pool, err := pgxpool.NewWithConfig(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tmp := t.TempDir()
	binary := filepath.Join(tmp, "sub2video-server")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build command fixture: %v\n%s", err, output)
	}
	uploads := filepath.Join(tmp, "must-not-be-created")
	env := make([]string, 0, len(os.Environ())+5)
	for _, item := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(item), "SUB2VIDEO_") {
			env = append(env, item)
		}
	}
	env = append(env,
		"SUB2VIDEO_ENV=development",
		"SUB2VIDEO_DATABASE_URL="+commandDSN.String(),
		"SUB2VIDEO_SKIP_BOOTSTRAP=true",
		"SUB2VIDEO_REDIS_URL=invalid-redis-url",
		"SUB2VIDEO_ADDR=not-a-listenable-address",
		"SUB2VIDEO_UPLOAD_DIR="+uploads,
	)
	for run := 1; run <= 2; run++ {
		command := exec.CommandContext(ctx, binary, "--migrate-only")
		command.Env = env
		output, err := command.CombinedOutput()
		if err != nil || !strings.Contains(string(output), "no services or seeds started") {
			t.Fatalf("explicit migrate-only run %d failed: %v\n%s", run, err, output)
		}
	}
	if _, err := os.Stat(uploads); !os.IsNotExist(err) {
		t.Fatal("migrate-only created or accessed the upload directory")
	}
	migrationPaths, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.sql"))
	if err != nil || len(migrationPaths) == 0 {
		t.Fatalf("read expected migration files: %v (found %d)", err, len(migrationPaths))
	}
	for _, path := range migrationPaths {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		version := strings.TrimSuffix(filepath.Base(path), ".sql")
		digest := sha256.Sum256(body)
		var checksum string
		if err := pool.QueryRow(ctx, `SELECT checksum FROM schema_migrations WHERE version=$1`, version).Scan(&checksum); err != nil {
			t.Fatalf("expected migration %s was not recorded: %v", version, err)
		}
		if checksum != hex.EncodeToString(digest[:]) {
			t.Fatalf("migration %s checksum does not match its source", version)
		}
	}
	var versions, users, products, codes, runtimeSeeds int
	for query, target := range map[string]*int{
		`SELECT COUNT(*) FROM schema_migrations`:                                            &versions,
		`SELECT COUNT(*) FROM users`:                                                        &users,
		`SELECT COUNT(*) FROM recharge_products`:                                            &products,
		`SELECT COUNT(*) FROM redeem_codes`:                                                 &codes,
		`SELECT COUNT(*) FROM settings WHERE key IN ('initial_balance','storage_quota_gb')`: &runtimeSeeds,
	} {
		if err := pool.QueryRow(ctx, query).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	if versions != len(migrationPaths) || users != 0 || products != 0 || codes != 0 || runtimeSeeds != 0 {
		t.Fatalf("migration did not remain schema-only: versions=%d want=%d users=%d products=%d codes=%d runtimeSeeds=%d", versions, len(migrationPaths), users, products, codes, runtimeSeeds)
	}
	for _, args := range [][]string{{"--unknown-release-option"}, {"--migrate-only", "unexpected-positional"}} {
		command := exec.CommandContext(ctx, binary, args...)
		command.Env = append(env, "SUB2VIDEO_DATABASE_URL=invalid-database-url")
		output, err := command.CombinedOutput()
		if err == nil || strings.Contains(string(output), "postgres:") {
			t.Fatal("invalid command arguments must fail before connecting to PostgreSQL")
		}
	}
}
