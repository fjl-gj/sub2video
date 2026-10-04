package handler_test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sbu2video/backend/internal/auth"
	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/db"
	"github.com/sbu2video/backend/internal/server"
	"github.com/sbu2video/backend/internal/store"
)

// Real PostgreSQL exercises ownership queries, persisted URLs and settings.
// Only MinIO's bucket-location lookup is simulated; signing uses its real SDK
// with fake credentials. No supplier, object download, Redis or worker runs.
//
// SUB2VIDEO_RUN_INTEGRATION=1 SUB2VIDEO_TEST_DATABASE_URL='postgres://.../sub2video_test?sslmode=disable' \
// go test ./internal/handler -run '^TestDeliveryURLHandlerIntegration$' -count=1 -v
func TestDeliveryURLHandlerIntegration(t *testing.T) {
	st := deliveryIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var storageRequests, objectRequests atomic.Int64
	var signingUnavailable atomic.Bool
	objects := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		storageRequests.Add(1)
		w.Header().Set("Content-Type", "application/xml")
		if r.Method != http.MethodGet || r.URL.Path != "/delivery-test/" || !r.URL.Query().Has("location") {
			objectRequests.Add(1)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`<Error><Code>InvalidRequest</Code></Error>`))
			return
		}
		if signingUnavailable.Load() {
			w.WriteHeader(http.StatusForbidden)
			// MinIO intentionally accepts a location AccessDenied by assuming
			// us-east-1. InvalidAccessKeyId exercises an actual signing failure.
			_, _ = w.Write([]byte(`<Error><Code>InvalidAccessKeyId</Code><Message>fixture signing denied</Message></Error>`))
			return
		}
		_, _ = w.Write([]byte(`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-east-1</LocationConstraint>`))
	}))
	defer objects.Close()
	if err := st.SetSettings(ctx, map[string]string{
		"storage_provider": "minio", "minio_endpoint": objects.URL,
		"minio_access_key": "delivery-test-access", "minio_secret_key": "delivery-test-secret",
		"minio_bucket": "delivery-test", "minio_use_ssl": "false", "minio_public_base": "",
	}); err != nil {
		t.Fatal(err)
	}
	owner, err := st.CreateUserWithWallet(ctx, "owner@delivery.test", "unused", "Owner", "Owner workspace", 10)
	if err != nil {
		t.Fatal(err)
	}
	other, err := st.CreateUserWithWallet(ctx, "other@delivery.test", "unused", "Other", "Other workspace", 10)
	if err != nil {
		t.Fatal(err)
	}
	const taskID = "tsk_delivery_test"
	oldURL := objects.URL + "/delivery-test/results/scene.mp4?X-Amz-Date=20000101T000000Z&X-Amz-Signature=expired&response-content-type=text/plain"
	if _, err := st.Pool.Exec(ctx, `
INSERT INTO generation_tasks
  (public_id,user_id,title,workflow,status,progress,duration_sec,resolution,cost,result_oss_url,thumb_url,completed_at)
VALUES ($1,$2,'Delivery fixture','文生视频','completed',100,5,'768P',1,$3,$3,NOW())`, taskID, owner.ID, oldURL); err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(ctx, owner.ID, "Delivery project", "Integration fixture", false)
	if err != nil {
		t.Fatal(err)
	}
	const remoteFileID, localFileID = "ast_remote_delivery", "ast_local_delivery"
	if _, _, err := st.AddProjectFile(ctx, owner.ID, project.ID, []string{"视频", "正片镜头"}, store.FsItem{
		ID: remoteFileID, Name: "scene.mp4", Type: "video", SourceTask: taskID, RemoteURL: oldURL,
	}); err != nil {
		t.Fatal(err)
	}
	localPath := filepath.Join(t.TempDir(), "local-source.txt")
	const localContent = "local upload remains available"
	if err := os.WriteFile(localPath, []byte(localContent), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.AddProjectFile(ctx, owner.ID, project.ID, []string{"视频", "正片镜头"}, store.FsItem{
		ID: localFileID, Name: "local-source.txt", Type: "video", StoragePath: localPath, RemoteURL: oldURL,
	}); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{JWTSecret: "delivery-integration-only", SecretKey: "delivery-settings-test", CORSOrigin: "https://client.example.test"}
	router := server.NewRouter(cfg, st, auth.NewService(st, cfg), nil)
	ownerToken := deliveryTestToken(t, owner, cfg.JWTSecret)
	otherToken := deliveryTestToken(t, other, cfg.JWTSecret)
	taskPath := "/api/v1/client/tasks/" + taskID
	filePath := "/api/v1/client/projects/" + project.ID + "/files/" + remoteFileID
	get := func(path, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res
	}

	t.Run("owner task renews result and thumbnail without overwriting storage", func(t *testing.T) {
		before := storageRequests.Load()
		res := get(taskPath, ownerToken)
		if res.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
		}
		var result struct {
			DeliveryStatus string `json:"deliveryStatus"`
			DownloadURL    string `json:"downloadUrl"`
			ThumbURL       string `json:"thumbUrl"`
		}
		if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		assertFreshDeliveryURL(t, result.DownloadURL, objects.URL)
		if result.DeliveryStatus != "ready" || result.ThumbURL != result.DownloadURL || storageRequests.Load() <= before {
			t.Fatal("ready delivery and thumbnail must share a freshly signed object URL")
		}
		if res.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("signed task detail must not be cached")
		}
		var persisted string
		if err := st.Pool.QueryRow(ctx, `SELECT result_oss_url FROM generation_tasks WHERE public_id=$1`, taskID).Scan(&persisted); err != nil || persisted != oldURL {
			t.Fatalf("read-time renewal must not overwrite persisted URL: %v", err)
		}
	})

	t.Run("other user receives not found before storage is consulted", func(t *testing.T) {
		for _, path := range []string{taskPath, taskPath + "?download=true", filePath} {
			before := storageRequests.Load()
			res := get(path, otherToken)
			if res.Code != http.StatusNotFound || storageRequests.Load() != before {
				t.Fatalf("foreign resource must be 404 without signing: status=%d", res.Code)
			}
			assertNoStoredDeliveryURL(t, res, oldURL)
		}
	})

	t.Run("anonymous request cannot renew a task or asset", func(t *testing.T) {
		for _, path := range []string{taskPath, taskPath + "?download=true", filePath} {
			before := storageRequests.Load()
			res := get(path, "")
			if res.Code != http.StatusUnauthorized || storageRequests.Load() != before {
				t.Fatalf("anonymous resource must be 401 without signing: status=%d", res.Code)
			}
			assertNoStoredDeliveryURL(t, res, oldURL)
		}
	})

	t.Run("explicit download signs attachment while normal preview stays inline", func(t *testing.T) {
		for _, download := range []bool{true, false} {
			path := taskPath
			if download {
				path += "?download=true"
			}
			res := get(path, ownerToken)
			if res.Code != http.StatusOK {
				t.Fatalf("download=%v status=%d", download, res.Code)
			}
			var result struct {
				DownloadURL string `json:"downloadUrl"`
				ThumbURL    string `json:"thumbUrl"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			u, err := url.Parse(result.DownloadURL)
			if err != nil {
				t.Fatal(err)
			}
			if download {
				kind, params, err := mime.ParseMediaType(u.Query().Get("response-content-disposition"))
				if err != nil || kind != "attachment" || params["filename"] != "Delivery fixture.mp4" {
					t.Fatal("explicit download did not use signed attachment disposition")
				}
			} else if u.Query().Has("response-content-disposition") {
				t.Fatal("ordinary task details must remain usable for video preview")
			}
			thumb, _ := url.Parse(result.ThumbURL)
			if thumb.Query().Has("response-content-disposition") {
				t.Fatal("thumbnail must not inherit attachment response override")
			}
			assertFreshDeliveryURL(t, result.DownloadURL, objects.URL)
		}
	})

	t.Run("asset redirects to a fresh signature while preserving stored data", func(t *testing.T) {
		res := get(filePath, ownerToken)
		if res.Code != http.StatusFound {
			t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
		}
		assertFreshDeliveryURL(t, res.Header().Get("Location"), objects.URL)
		if res.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("signed asset redirect must not be cached")
		}
		item, err := st.GetProjectFile(ctx, owner.ID, project.ID, remoteFileID)
		if err != nil || item.RemoteURL != oldURL {
			t.Fatalf("read-time asset renewal must not overwrite stored URL: %v", err)
		}
	})

	t.Run("asset listings expose authenticated routes instead of stale object URLs", func(t *testing.T) {
		for _, suffix := range []string{"/assets", "/tree"} {
			res := get("/api/v1/client/projects/"+project.ID+suffix, ownerToken)
			if res.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
			}
			assertNoStoredDeliveryURL(t, res, oldURL)
			if !strings.Contains(res.Body.String(), filePath) {
				t.Fatal("asset DTO must retain its protected download route")
			}
		}
	})

	t.Run("signing failure returns unavailable without the stored signature", func(t *testing.T) {
		signingUnavailable.Store(true)
		defer signingUnavailable.Store(false)
		for _, path := range []string{taskPath, taskPath + "?download=true", filePath} {
			before := storageRequests.Load()
			res := get(path, ownerToken)
			if res.Code != http.StatusServiceUnavailable || storageRequests.Load() <= before {
				t.Fatalf("failed signing must be 503: status=%d body=%s", res.Code, res.Body.String())
			}
			if !strings.Contains(res.Body.String(), `"code":"DELIVERY_UNAVAILABLE"`) {
				t.Fatal("expected stable public delivery error")
			}
			assertNoStoredDeliveryURL(t, res, oldURL)
			if res.Header().Get("Location") != "" || strings.Contains(res.Body.String(), "fixture signing denied") {
				t.Fatal("signing failure must not redirect or expose storage diagnostics")
			}
		}
	})

	t.Run("local file remains readable when object signing is unavailable", func(t *testing.T) {
		signingUnavailable.Store(true)
		defer signingUnavailable.Store(false)
		before := storageRequests.Load()
		res := get("/api/v1/client/projects/"+project.ID+"/files/"+localFileID, ownerToken)
		if res.Code != http.StatusOK || res.Body.String() != localContent || res.Header().Get("Location") != "" || storageRequests.Load() != before {
			t.Fatalf("local file should bypass object storage: status=%d body=%s", res.Code, res.Body.String())
		}
	})
	if objectRequests.Load() != 0 {
		t.Fatal("handler tests must never download or modify object content")
	}
}

func deliveryIntegrationStore(t *testing.T) *store.Store {
	t.Helper()
	databaseURL := os.Getenv("SUB2VIDEO_TEST_DATABASE_URL")
	if os.Getenv("SUB2VIDEO_RUN_INTEGRATION") != "1" || databaseURL == "" {
		t.Skip("set SUB2VIDEO_RUN_INTEGRATION=1 and SUB2VIDEO_TEST_DATABASE_URL to run real PostgreSQL delivery tests")
	}
	base, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal("invalid SUB2VIDEO_TEST_DATABASE_URL")
	}
	if !strings.Contains(strings.ToLower(base.ConnConfig.Database), "test") {
		t.Fatal("delivery integration requires a database whose name contains 'test'; never use the UAT application database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	base.MaxConns = 2
	admin, err := pgxpool.NewWithConfig(ctx, base)
	if err != nil {
		t.Fatal("connect delivery test PostgreSQL: ", err)
	}
	var scoped *pgxpool.Pool
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	schema := pgx.Identifier{fmt.Sprintf("delivery_it_%x", suffix)}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal("create isolated delivery schema: ", err)
	}
	t.Cleanup(func() {
		if scoped != nil {
			scoped.Close()
		}
		cleanupCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("clean isolated delivery schema %s: %v", schema, err)
		}
		admin.Close()
	})
	isolated := base.Copy()
	// Omitting public prevents table lookups or migration writes from reaching
	// another suite's fixtures or any pre-existing application schema.
	isolated.ConnConfig.RuntimeParams["search_path"] = schema
	scoped, err = pgxpool.NewWithConfig(ctx, isolated)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(ctx, scoped); err != nil {
		t.Fatal("migrate isolated delivery schema: ", err)
	}
	return store.New(scoped)
}

func deliveryTestToken(t *testing.T, user *store.User, secret string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
		UserID: user.ID, PublicID: user.PublicID, Role: user.Role, TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func assertFreshDeliveryURL(t *testing.T, raw, origin string) {
	t.Helper()
	fresh, err := url.Parse(raw)
	if err != nil || fresh.Scheme+"://"+fresh.Host != origin || fresh.Path != "/delivery-test/results/scene.mp4" {
		t.Fatal("renewal must preserve the configured storage origin and object key")
	}
	q := fresh.Query()
	if q.Get("X-Amz-Signature") == "" || q.Get("X-Amz-Signature") == "expired" || q.Get("X-Amz-Expires") != "3600" || q.Has("response-content-type") {
		t.Fatal("renewal must replace the old signature and discard stored query overrides")
	}
	signedAt, err := time.Parse("20060102T150405Z", q.Get("X-Amz-Date"))
	if err != nil || time.Since(signedAt) > time.Minute || signedAt.After(time.Now().Add(time.Minute)) {
		t.Fatal("renewal must use the current signing time")
	}
}

func assertNoStoredDeliveryURL(t *testing.T, res *httptest.ResponseRecorder, stored string) {
	t.Helper()
	for _, text := range []string{res.Body.String(), res.Header().Get("Location")} {
		if strings.Contains(text, stored) || strings.Contains(text, "X-Amz-Signature=expired") || strings.Contains(text, "20000101T000000Z") {
			t.Fatal("response exposed an expired stored object URL")
		}
	}
}
