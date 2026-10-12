package migrations

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This is an upgrade rehearsal with synthetic data, not a copy of UAT data.
// Version 019 had no checksum column, so empty-database migration tests alone
// cannot prove that a release skips the historical destructive SQL.
func TestUpgradeFrom019PreservesExistingDataIntegration(t *testing.T) {
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
	name := "release_upgrade_" + hex.EncodeToString(random[:])
	schema := pgx.Identifier{name}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal("create isolated upgrade schema: ", err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanup, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error("clean isolated upgrade schema: ", err)
		}
	}()
	base.ConnConfig.RuntimeParams["search_path"] = name
	pool, err := pgxpool.NewWithConfig(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	entries, err := files.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") || entry.Name() > "019_limit_h3_creation_catalog.sql" {
			continue
		}
		body, err := files.ReadFile(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(body)); err != nil {
			t.Fatalf("prepare old migration %s: %v", entry.Name(), err)
		}
	}
	var versions, checksumColumns int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='schema_migrations' AND column_name='checksum'`).Scan(&checksumColumns); err != nil {
		t.Fatal(err)
	}
	if versions != 19 || checksumColumns != 0 {
		t.Fatalf("fixture must represent unchecksummed 019: versions=%d checksumColumns=%d", versions, checksumColumns)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO users (id,public_id,email,password_hash,display_name) VALUES (1,'release_user','release@example.test','unused','Release fixture');
INSERT INTO wallets (user_id,balance) VALUES (1,17.23);
INSERT INTO projects (public_id,user_id,name,folders_json) VALUES ('release_project',1,'Preserve work','{"资产":{"items":[],"folders":{}}}'::jsonb);
INSERT INTO generation_tasks (id,public_id,user_id,title,workflow,status,duration_sec,resolution,cost,execution_mode,upstream_task_id,upstream_status)
VALUES (1,'release_task',1,'Preserve accepted task','文生视频','running',5,'768P',0.27,'provider','accepted-fixture','RUNNING');
INSERT INTO consumption_orders (public_id,user_id,task_id,client_request_id,amount,status) VALUES ('release_order',1,1,'release_request',0.27,'paid');
INSERT INTO wallet_ledger (public_id,user_id,type,amount,balance_after,ref_type,ref_id) VALUES ('release_ledger',1,'task_charge',-0.27,17.23,'task','release_task');
INSERT INTO carriers (id,public_id,code,name) VALUES (1,'release_carrier','release','Release carrier');
INSERT INTO models (id,public_id,carrier_id,code,name) VALUES (1,'release_model',1,'fixture','Fixture model');
INSERT INTO suppliers (id,public_id,code,name) VALUES (1,'release_supplier','fixture','Fixture supplier');
INSERT INTO creation_modes (id,public_id,code,name) VALUES (1,'release_mode','text_to_video','文生视频');
INSERT INTO supplier_model_bindings (id,public_id,supplier_id,model_id) VALUES (1,'release_binding',1,1);
INSERT INTO supplier_offerings (id,public_id,supplier_id,model_id,mode_id,upstream_workflow_id,workflow_code)
VALUES (1,'release_offering',1,1,1,'fixture_keep_after_019','fixture_keep_after_019');
INSERT INTO supplier_executions (id,public_id,binding_id,mode_id,execution_type,workflow_code,legacy_offering_id)
VALUES (1,'release_execution',1,1,'workflow','fixture_keep_after_019',1);
INSERT INTO supplier_cost_rules (public_id,offering_id,execution_id,resolution,price_per_sec_peak,price_per_sec_offpeak,version,status)
VALUES ('release_cost',1,1,'768P',0.0777,0.0666,'admin-custom','disabled');
INSERT INTO retail_price_rules (public_id,mode_id,model_id,resolution,name,price_per_sec,version,status)
VALUES ('release_retail',1,NULL,'768P','Preserve admin price',0.0888,'admin-custom','disabled');`); err != nil {
		t.Fatal("prepare existing data: ", err)
	}
	snapshot := func() string {
		t.Helper()
		var value string
		if err := pool.QueryRow(ctx, `SELECT jsonb_build_object(
'wallet',(SELECT to_jsonb(x) FROM wallets x WHERE user_id=1),
'ledger',(SELECT to_jsonb(x) FROM wallet_ledger x WHERE public_id='release_ledger'),
'order',(SELECT to_jsonb(x) FROM consumption_orders x WHERE public_id='release_order'),
'project',(SELECT jsonb_build_object('name',name,'folders',folders_json,'updatedAt',updated_at) FROM projects WHERE public_id='release_project'),
'task',(SELECT jsonb_build_object('status',status,'upstream',upstream_task_id,'upstreamStatus',upstream_status,'cost',cost,'updatedAt',updated_at) FROM generation_tasks WHERE id=1),
'cost',(SELECT to_jsonb(x) FROM supplier_cost_rules x WHERE public_id='release_cost'),
'retail',(SELECT to_jsonb(x) FROM retail_price_rules x WHERE public_id='release_retail'),
'execution',(SELECT to_jsonb(x) FROM supplier_executions x WHERE public_id='release_execution'))::text`).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := snapshot()
	for run := 1; run <= 2; run++ {
		if err := Apply(ctx, pool); err != nil {
			t.Fatalf("upgrade/replay %d: %v", run, err)
		}
		if after := snapshot(); before != after {
			t.Fatalf("upgrade/replay %d changed existing task, financial or administrator data", run)
		}
	}
	var checksummed, leaseColumns, events, indexes int
	for query, target := range map[string]*int{
		`SELECT COUNT(*) FROM schema_migrations WHERE length(checksum)=64`: &checksummed,
		`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='generation_tasks' AND column_name IN ('claimed_at','claim_expires_at','claim_token','submit_started','submit_started_at','submit_uncertain','execution_deadline','poll_attempts','next_poll_at','provider_request_id')`: &leaseColumns,
		`SELECT COUNT(*) FROM provider_task_events`: &events,
		`SELECT COUNT(*) FROM pg_indexes WHERE schemaname=current_schema() AND indexname='ux_supplier_cost_execution_match'`: &indexes,
	} {
		if err := pool.QueryRow(ctx, query).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	var nullable string
	if err := pool.QueryRow(ctx, `SELECT is_nullable FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='supplier_cost_rules' AND column_name='offering_id'`).Scan(&nullable); err != nil {
		t.Fatal(err)
	}
	expectedVersions := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			expectedVersions++
		}
	}
	if checksummed != expectedVersions || leaseColumns != 10 || events != 0 || indexes != 1 || nullable != "YES" {
		t.Fatal(fmt.Sprintf("unexpected upgrade shape: checksummed=%d leaseColumns=%d events=%d indexes=%d nullable=%s", checksummed, leaseColumns, events, indexes, nullable))
	}
	var directorRevision int64
	var directorEmpty bool
	if err := pool.QueryRow(ctx, `SELECT director_revision,director_board='[]'::jsonb FROM projects WHERE public_id='release_project'`).Scan(&directorRevision, &directorEmpty); err != nil {
		t.Fatal(err)
	}
	if directorRevision != 0 || !directorEmpty {
		t.Fatal("existing works must gain an empty director board without changing their assets")
	}
	if _, err := pool.Exec(ctx, `UPDATE schema_migrations SET checksum=repeat('0',64) WHERE version='019_limit_h3_creation_catalog'`); err != nil {
		t.Fatal(err)
	}
	if err := Apply(ctx, pool); err == nil || !strings.Contains(err.Error(), "changed after it was applied") {
		t.Fatal("changed migration checksum must stop the release")
	}
}
