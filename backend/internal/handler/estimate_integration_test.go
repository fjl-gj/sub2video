package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sbu2video/backend/internal/auth"
	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/server"
)

// Exercises the exact creator-facing HTTP durationSec contract against real
// PostgreSQL. Storage availability is a local read-only fixture; an estimate
// must neither contact a generation supplier nor leave financial records.
func TestEstimateDurationSecHTTPContractHasNoFinancialSideEffectsIntegration(t *testing.T) {
	st := deliveryIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := st.SeedBasics(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.SeedCreationCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	var storageReads, supplierRequests atomic.Int32
	objects := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/estimate-test/" || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
			t.Errorf("estimate performed an unexpected storage operation: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		storageReads.Add(1)
		if r.Method == http.MethodGet && r.URL.Query().Has("location") {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-east-1</LocationConstraint>`))
		}
	}))
	defer objects.Close()
	supplier := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		supplierRequests.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer supplier.Close()
	if _, err := st.Pool.Exec(ctx, `UPDATE suppliers SET token_encrypted='',base_url=$1,enabled=TRUE,status='healthy' WHERE code='autodl'`, supplier.URL); err != nil {
		t.Fatal(err)
	}
	if err := st.SetSettings(ctx, map[string]string{
		"storage_provider": "minio", "minio_endpoint": objects.URL,
		"minio_access_key": "estimate-test-access", "minio_secret_key": "estimate-test-secret",
		"minio_bucket": "estimate-test", "minio_use_ssl": "false", "minio_public_base": "",
	}); err != nil {
		t.Fatal(err)
	}
	user, err := st.CreateUserWithWallet(ctx, "creator@estimate.test", "unused", "Creator", "Estimate fixture", 100)
	if err != nil {
		t.Fatal(err)
	}
	type financialState struct {
		Balance                              float64
		Tasks, Orders, Quotes, LedgerEntries int
	}
	snapshot := func() financialState {
		t.Helper()
		var state financialState
		if err := st.Pool.QueryRow(ctx, `SELECT
(SELECT balance::float8 FROM wallets WHERE user_id=$1),
(SELECT COUNT(*) FROM generation_tasks WHERE user_id=$1),
(SELECT COUNT(*) FROM consumption_orders WHERE user_id=$1),
(SELECT COUNT(*) FROM task_quotes WHERE user_id=$1),
(SELECT COUNT(*) FROM wallet_ledger WHERE user_id=$1)`, user.ID).Scan(
			&state.Balance, &state.Tasks, &state.Orders, &state.Quotes, &state.LedgerEntries); err != nil {
			t.Fatal(err)
		}
		return state
	}
	before := snapshot()
	cfg := config.Config{JWTSecret: "estimate-test-only", SecretKey: "estimate-settings-test", AutoDLToken: "estimate-only-token", AutoDLBaseURL: supplier.URL}
	router := server.NewRouter(cfg, st, auth.NewService(st, cfg), nil)
	// Keep this payload identical to the deployed creator request: optional
	// carrier/model selectors are intentionally absent, and there is no quoteId.
	payload := `{"modeId":"mode_text_to_video","durationSec":5,"resolution":"768P","aspectRatio":"16:9","generateAudio":false}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/client/tasks/estimate", strings.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+deliveryTestToken(t, user, cfg.JWTSecret))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("durationSec=5 must reach the route as five seconds: status=%d body=%s", response.Code, response.Body.String())
	}
	var result struct {
		Available bool `json:"available"`
		Billing   struct {
			DurationSec    int     `json:"durationSec"`
			PricePerSecond float64 `json:"pricePerSecond"`
			EstimatedTotal float64 `json:"estimatedTotal"`
		} `json:"billing"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Available || result.Billing.DurationSec != 5 || result.Billing.PricePerSecond != 0.03 || result.Billing.EstimatedTotal != 0.15 {
		t.Fatalf("unexpected five-second estimate: %+v", result)
	}
	for _, forbidden := range []string{"quoteId", "supplier", "workflow", "executionType", "costSnapshot", "token", "upstream", "apiModelCode"} {
		if strings.Contains(strings.ToLower(response.Body.String()), strings.ToLower(forbidden)) {
			t.Fatalf("estimate leaked an internal or legacy field: %s", forbidden)
		}
	}
	if after := snapshot(); after != before || after.Balance != 100 || after.Tasks != 0 || after.Orders != 0 || after.Quotes != 0 || after.LedgerEntries != 0 {
		t.Fatalf("estimate changed financial state: before=%+v after=%+v", before, after)
	}
	if storageReads.Load() == 0 || supplierRequests.Load() != 0 {
		t.Fatalf("estimate must check storage without invoking the supplier: storageReads=%d supplierRequests=%d", storageReads.Load(), supplierRequests.Load())
	}
	// A reset/unconfigured environment has no recommendable model. This is a
	// public availability failure, not an internal server error, and still must
	// not create financial records or contact a generation supplier.
	if _, err := st.Pool.Exec(ctx, `UPDATE suppliers SET enabled=FALSE`); err != nil {
		t.Fatal(err)
	}
	unavailableRequest := httptest.NewRequest(http.MethodPost, "/api/v1/client/tasks/estimate", strings.NewReader(payload))
	unavailableRequest.Header.Set("Content-Type", "application/json")
	unavailableRequest.Header.Set("Authorization", "Bearer "+deliveryTestToken(t, user, cfg.JWTSecret))
	unavailableResponse := httptest.NewRecorder()
	router.ServeHTTP(unavailableResponse, unavailableRequest)
	var unavailable struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(unavailableResponse.Body.Bytes(), &unavailable); err != nil {
		t.Fatal(err)
	}
	if unavailableResponse.Code != http.StatusServiceUnavailable || unavailable.Code != "SUPPLY_UNAVAILABLE" || unavailable.Message != "当前生成能力暂不可用，请稍后重试" {
		t.Fatalf("empty recommendation must expose a stable availability error: status=%d result=%+v", unavailableResponse.Code, unavailable)
	}
	if after := snapshot(); after != before || supplierRequests.Load() != 0 {
		t.Fatalf("unavailable estimate must have no side effects: before=%+v after=%+v supplierRequests=%d", before, after, supplierRequests.Load())
	}
}
