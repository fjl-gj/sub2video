package store

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func supplierOfferingFixture(t *testing.T, env *integrationStore) SupplierOffering {
	t.Helper()
	modeID, _, _, modelID := testCatalogIDs(t, env)
	if _, err := env.pool.Exec(env.ctx, `INSERT INTO suppliers (public_id, code, name) VALUES ('sup_config_test','config_test','Isolated configuration test')`); err != nil {
		t.Fatal(err)
	}
	return SupplierOffering{
		SupplierID: "sup_config_test", ModelID: modelID, ModeID: modeID,
		Name: "Original configuration", UpstreamWorkflowID: "test_workflow", WorkflowCode: "test_workflow",
		ExecutionType: "workflow", AdapterCode: "autodl_h3_v1", HealthStatus: "unknown",
		Enabled: true, Priority: 17, MinDurationSec: 1, MaxDurationSec: 10,
		Resolutions: []string{"768P"}, AspectRatios: []string{"16:9"},
	}
}

func offeringTableCounts(t *testing.T, env *integrationStore) [3]int {
	t.Helper()
	var counts [3]int
	if err := env.pool.QueryRow(env.ctx, `SELECT
 (SELECT COUNT(*) FROM supplier_offerings),
 (SELECT COUNT(*) FROM supplier_model_bindings),
 (SELECT COUNT(*) FROM supplier_executions)`).Scan(&counts[0], &counts[1], &counts[2]); err != nil {
		t.Fatal(err)
	}
	return counts
}

func rejectOfferingWrite(t *testing.T, env *integrationStore, table string) {
	t.Helper()
	// All identifiers are test-owned and the existing helper creates a fresh
	// schema in a database explicitly named as a test database.
	if _, err := env.pool.Exec(env.ctx, `CREATE FUNCTION reject_config_write() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'injected configuration write failure' USING ERRCODE='P0001'; END $$`); err != nil {
		t.Fatal(err)
	}
	statement := "CREATE TRIGGER reject_config_write BEFORE INSERT OR UPDATE ON " + pgx.Identifier{table}.Sanitize() + " FOR EACH ROW EXECUTE FUNCTION reject_config_write()"
	if _, err := env.pool.Exec(env.ctx, statement); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := env.pool.Exec(env.ctx, "DROP FUNCTION reject_config_write() CASCADE"); err != nil {
			t.Errorf("remove isolated fault injection: %v", err)
		}
	})
}

func TestSupplierOfferingCreateRollbackIntegration(t *testing.T) {
	for _, table := range []string{"supplier_model_bindings", "supplier_executions"} {
		t.Run(table, func(t *testing.T) {
			env := openIntegrationStore(t)
			input := supplierOfferingFixture(t, env)
			before := offeringTableCounts(t, env)
			rejectOfferingWrite(t, env, table)
			_, err := env.store.CreateSupplierOffering(env.ctx, input)
			var failure *pgconn.PgError
			if !errors.As(err, &failure) || failure.Code != "P0001" {
				t.Fatalf("want injected write error, got %v", err)
			}
			if after := offeringTableCounts(t, env); after != before {
				t.Fatalf("configuration creation partially committed: before=%v after=%v", before, after)
			}
		})
	}
}

func TestSupplierOfferingCreateUpdateAndRollbackIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	input := supplierOfferingFixture(t, env)
	before := offeringTableCounts(t, env)
	created, err := env.store.CreateSupplierOffering(env.ctx, input)
	if err != nil {
		t.Fatalf("create linked offering/execution: %v", err)
	}
	if created.ExecutionID == "" || created.LegacyOfferingID == "" {
		t.Fatalf("create must return both execution and legacy identities: %+v", created)
	}
	if after := offeringTableCounts(t, env); after != [3]int{before[0] + 1, before[1] + 1, before[2] + 1} {
		t.Fatalf("create did not persist the complete aggregate: before=%v after=%v", before, after)
	}
	input.Name, input.Priority, input.MaxDurationSec = "Updated configuration", 23, 12
	updated, err := env.store.UpdateSupplierOffering(env.ctx, created.ID, input)
	if err != nil || updated.Name != input.Name || updated.Priority != 23 || updated.MaxDurationSec != 12 {
		t.Fatalf("update linked configuration: item=%+v err=%v", updated, err)
	}
	t.Run("execution_update_failure", func(t *testing.T) {
		rejectOfferingWrite(t, env, "supplier_executions")
		input.Name, input.Priority = "Must not persist", 99
		_, err := env.store.UpdateSupplierOffering(env.ctx, created.ID, input)
		var failure *pgconn.PgError
		if !errors.As(err, &failure) || failure.Code != "P0001" {
			t.Fatalf("want injected update error, got %v", err)
		}
		var name string
		var legacyPriority, executionPriority int
		if err := env.pool.QueryRow(env.ctx, `SELECT o.name, o.priority, e.priority
FROM supplier_offerings o JOIN supplier_executions e ON e.legacy_offering_id=o.id
WHERE e.public_id=$1`, created.ID).Scan(&name, &legacyPriority, &executionPriority); err != nil {
			t.Fatal(err)
		}
		if name != "Updated configuration" || legacyPriority != 23 || executionPriority != 23 {
			t.Fatalf("failed update changed configuration: name=%q legacy=%d execution=%d", name, legacyPriority, executionPriority)
		}
	})
	// A missing referenced parent must not leave an unrelated binding behind.
	before = offeringTableCounts(t, env)
	input.ModeID = "missing_mode"
	if _, err := env.store.CreateSupplierOffering(env.ctx, input); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing mode: %v", err)
	}
	if after := offeringTableCounts(t, env); after != before {
		t.Fatalf("missing parent left a partial configuration: before=%v after=%v", before, after)
	}
}

func TestSupplierOfferingLegacyOnlyReadAndUpdateIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	input := supplierOfferingFixture(t, env)
	const legacyID = "off_old_only_test"
	if _, err := env.pool.Exec(env.ctx, `INSERT INTO supplier_offerings
 (public_id, supplier_id, model_id, mode_id, name, upstream_workflow_id, execution_type,
  api_model_code, api_version, workflow_code, quota_status, resolutions, aspect_ratios)
SELECT $1, s.id, m.id, mode.id, 'Legacy only', 'legacy_workflow', 'workflow',
 'legacy_model', 'v-test', 'legacy_workflow', 'exhausted', '["768P"]', '["16:9"]'
FROM suppliers s, models m, creation_modes mode WHERE s.public_id=$2 AND m.public_id=$3 AND mode.public_id=$4`,
		legacyID, input.SupplierID, input.ModelID, input.ModeID); err != nil {
		t.Fatal(err)
	}
	legacy, err := env.store.GetSupplierOfferingExecution(env.ctx, legacyID)
	if err != nil {
		t.Fatalf("read old-only offering: %v", err)
	}
	if legacy.ID != legacyID || legacy.LegacyOfferingID != legacyID || legacy.ExecutionID != "" ||
		legacy.ExecutionType != "workflow" || legacy.APIModelCode != "legacy_model" || legacy.APIVersion != "v-test" ||
		legacy.WorkflowCode != "legacy_workflow" || legacy.QuotaStatus != "exhausted" ||
		!reflect.DeepEqual(legacy.Resolutions, []string{"768P"}) || !reflect.DeepEqual(legacy.AspectRatios, []string{"16:9"}) {
		t.Fatalf("legacy field mapping mismatch: %+v", legacy)
	}
	legacy.Name, legacy.Priority = "Edited legacy", 51
	updated, err := env.store.UpdateSupplierOffering(env.ctx, legacyID, *legacy)
	if err != nil || updated.Name != "Edited legacy" || updated.Priority != 51 || updated.ExecutionID != "" {
		t.Fatalf("edit legacy-only offering: item=%+v err=%v", updated, err)
	}
}

func TestDirectRouteDatabaseFailuresAreNotMissingSupplyIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	modeID, _, _, modelID := testCatalogIDs(t, env)
	request := seededDirectRequest(modeID, modelID)
	if _, err := env.store.ResolveDirectRoute(env.ctx, request); err != nil {
		t.Fatalf("baseline route: %v", err)
	}
	for _, table := range []string{"mode_model_policies", "supplier_cost_rules"} {
		t.Run(table, func(t *testing.T) {
			tx, err := env.pool.Begin(env.ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(env.ctx)
			if _, err := tx.Exec(env.ctx, fmt.Sprintf("ALTER TABLE %s RENAME TO hidden_for_test", pgx.Identifier{table}.Sanitize())); err != nil {
				t.Fatal(err)
			}
			isolated := *env.store
			isolated.routeTx = tx
			_, err = isolated.ResolveDirectRoute(env.ctx, request)
			var failure *pgconn.PgError
			if !errors.As(err, &failure) || failure.Code != "42P01" {
				t.Fatalf("database failure must not become missing supply: %v", err)
			}
		})
	}
}
