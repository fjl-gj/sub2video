package store

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sbu2video/backend/internal/config"
)

// One available connection makes any nested pool borrow deterministic. This
// represents production saturation when every concurrent request holds a
// transaction and then tries to acquire a second connection for validation.
func TestDirectImageSubmissionUsesOneTransactionConnectionIntegration(t *testing.T) {
	env := openIntegrationStore(t)
	_, firstLastMode, _, modelID := testCatalogIDs(t, env)
	owner := newIntegrationUser(t, env, 10)
	other := newIntegrationUser(t, env, 10)
	var picture bytes.Buffer
	if err := png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(t.TempDir(), "reference.png")
	if err := os.WriteFile(imagePath, picture.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	upload, err := env.store.CreateUpload(env.ctx, owner.ID, "", "reference.png", "image/png", imagePath, int64(picture.Len()))
	if err != nil {
		t.Fatal(err)
	}
	projectID, err := env.store.ResolveCurrentProjectID(env.ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	assetID := NewAssetID()
	if _, _, err := env.store.AddProjectFile(env.ctx, owner.ID, projectID, []string{"资产", "素材"}, FsItem{
		ID: assetID, Name: "reference.png", Type: "image", MimeType: "image/png", StoragePath: imagePath,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.pool.Exec(env.ctx, `UPDATE suppliers SET token_encrypted='' WHERE code='autodl'`); err != nil {
		t.Fatal(err)
	}
	// Exercise the real settings and credential reads used by the handler's
	// preflight. The endpoint is never contacted; this test proves DB access,
	// not a real MinIO or supplier delivery.
	if err := env.store.SetSettings(env.ctx, map[string]string{
		"storage_provider": "minio", "minio_endpoint": "127.0.0.1:9000",
		"minio_access_key": "fixture", "minio_secret_key": "fixture-secret", "minio_bucket": "fixture",
	}); err != nil {
		t.Fatal(err)
	}
	poolConfig := env.pool.Config()
	poolConfig.MaxConns, poolConfig.MinConns = 1, 1
	pool, err := pgxpool.NewWithConfig(env.ctx, poolConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	constrained := New(pool)
	preflightCalls := 0
	for _, source := range []SourceRefItem{{Type: "upload", ID: upload.ID}, {Type: "asset", ID: assetID}} {
		t.Run(source.Type, func(t *testing.T) {
			request := seededDirectRequest(firstLastMode, modelID)
			request.SupplierTokenFallback = "integration-only-token"
			input := DirectTaskInput{
				DirectRouteRequest: request, ClientRequestID: "one-connection-" + source.Type, Prompt: "同图首尾帧",
				SourceMeta: SourceMeta{First: &source, Last: &source},
				Preflight: func(ctx context.Context, transactionStore *Store, route *ResolvedRoute) error {
					preflightCalls++
					if transactionStore == constrained || transactionStore.routeTx == nil {
						return errors.New("preflight did not receive an isolated transaction store")
					}
					token, err := transactionStore.ResolveSupplierToken(ctx, "fixture-secret", route.Offering.SupplierCode, request.SupplierTokenFallback)
					if err != nil || token != request.SupplierTokenFallback {
						return errors.New("transaction-scoped credential lookup failed")
					}
					objects := transactionStore.LoadObjectStore(ctx, config.Config{SecretKey: "fixture-secret"})
					if !objects.Configured() || objects.Provider() != "minio" {
						return errors.New("transaction-scoped storage configuration lookup failed")
					}
					return nil
				},
			}
			ctx, cancel := context.WithTimeout(env.ctx, 10*time.Second)
			defer cancel()
			result, err := constrained.CreateTaskDirect(ctx, owner.ID, input)
			if err != nil {
				t.Fatalf("one-connection submission must not borrow from pool again: %v", err)
			}
			if result.Task.Cost != 0.15 || constrained.routeTx != nil {
				t.Fatalf("unexpected charge or shared transaction state: result=%+v routeTxSet=%v", result, constrained.routeTx != nil)
			}
			beforeReplay := preflightCalls
			replay, err := constrained.CreateTaskDirect(ctx, owner.ID, input)
			if err != nil || !replay.AlreadyDone || replay.OrderID != result.OrderID || preflightCalls != beforeReplay {
				t.Fatalf("retry reran preflight or charged again: replay=%+v err=%v", replay, err)
			}
			input.ClientRequestID += "-foreign"
			if _, err := constrained.CreateTaskDirect(ctx, other.ID, input); err == nil {
				t.Fatal("foreign image source was accepted")
			}
			if preflightCalls != beforeReplay {
				t.Fatal("foreign source passed ownership validation")
			}
		})
	}
	if balance, err := env.store.GetBalance(env.ctx, owner.ID); err != nil || balance != 9.70 {
		t.Fatalf("owner balance mismatch: balance=%v err=%v", balance, err)
	}
	if balance, err := env.store.GetBalance(env.ctx, other.ID); err != nil || balance != 10 {
		t.Fatalf("foreign source attempt charged other user: balance=%v err=%v", balance, err)
	}
}
