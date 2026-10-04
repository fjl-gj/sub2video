package migrations

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.sql
var files embed.FS

func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	// The first migration creates schema_migrations. Creating the minimal
	// table here makes the runner safe for an empty database and lets every
	// migration execute in its own transaction.
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil {
		return fmt.Errorf("prepare schema migrations: %w", err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS checksum TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("prepare schema migration checksum: %w", err)
	}
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		body, err := files.ReadFile(name)
		if err != nil {
			return err
		}
		version := strings.TrimSuffix(name, ".sql")
		digest := fmt.Sprintf("%x", sha256.Sum256(body))

		var appliedChecksum string
		err = pool.QueryRow(ctx, `SELECT checksum FROM schema_migrations WHERE version=$1`, version).Scan(&appliedChecksum)
		if err == nil {
			if appliedChecksum == "" {
				if _, err := pool.Exec(ctx, `UPDATE schema_migrations SET checksum=$1 WHERE version=$2`, digest, version); err != nil {
					return fmt.Errorf("backfill migration checksum %s: %w", version, err)
				}
			}
			if appliedChecksum != "" && appliedChecksum != digest {
				return fmt.Errorf("migration %s changed after it was applied", version)
			}
			continue
		}
		if err != pgx.ErrNoRows {
			return fmt.Errorf("check migration %s: %w", version, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", version, err)
		}
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('sub2video:migrations'))`); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("lock migration %s: %w", version, err)
		}
		// Another process may have applied this version while we waited for the
		// advisory lock. Re-check inside the transaction before running SQL.
		var lockedChecksum string
		err = tx.QueryRow(ctx, `SELECT checksum FROM schema_migrations WHERE version=$1`, version).Scan(&lockedChecksum)
		if err == nil {
			if lockedChecksum != "" && lockedChecksum != digest {
				_ = tx.Rollback(ctx)
				return fmt.Errorf("migration %s changed after it was applied", version)
			}
			if lockedChecksum == "" {
				if _, err = tx.Exec(ctx, `UPDATE schema_migrations SET checksum=$1 WHERE version=$2`, digest, version); err != nil {
					_ = tx.Rollback(ctx)
					return fmt.Errorf("backfill migration checksum %s: %w", version, err)
				}
			}
			if err = tx.Commit(ctx); err != nil {
				return fmt.Errorf("commit migration %s: %w", version, err)
			}
			continue
		}
		if err != pgx.ErrNoRows {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("check locked migration %s: %w", version, err)
		}
		if _, err = tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrate %s: %w", version, err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations (version, checksum) VALUES ($1,$2) ON CONFLICT (version) DO UPDATE SET checksum=EXCLUDED.checksum`, version, digest); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", version, err)
		}
		if err = tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", version, err)
		}
	}
	return nil
}
