package autodl

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientRejectsUnverifiedAPIExecutionAndPollsWorkflow(t *testing.T) {
	seen := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"Success","data":{"task_id":"task-1","status":"RUNNING","results":[]}}`))
	}))
	defer server.Close()

	client := New(server.URL, "token", "Authorization")
	ctx := context.Background()
	if _, err := client.QueryAPI(ctx, "task-api"); !errors.Is(err, ErrAPIExecutionUnsupported) {
		t.Fatalf("query api should be explicitly unsupported, got %v", err)
	}
	if _, err := client.Query(ctx, "task-workflow"); err != nil {
		t.Fatalf("query workflow: %v", err)
	}

	if got := <-seen; got != "/api/v1/comfyui/comfyui_workflow/result/task-workflow" {
		t.Fatalf("unexpected workflow poll endpoint: %s", got)
	}
}

func TestSubmitClassifiesDefiniteAndUncertainFailures(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		uncertain bool
	}{
		{name: "unverified client rejection", status: http.StatusBadRequest, body: `{"code":"InvalidParameter","msg":"bad prompt"}`, uncertain: true},
		{name: "upstream timeout after acceptance", status: http.StatusRequestTimeout, body: `{"code":"Timeout"}`, uncertain: true},
		{name: "conflict could be accepted duplicate", status: http.StatusConflict, body: `{"code":"Conflict","data":{"task_id":"accepted-1"}}`, uncertain: true},
		{name: "unverified business rejection", status: http.StatusOK, body: `{"code":"Error","msg":"error"}`, uncertain: true},
		{name: "provider outage", status: http.StatusBadGateway, body: `{"code":"Unavailable","msg":"retry"}`, uncertain: true},
		{name: "malformed success", status: http.StatusOK, body: `not-json`, uncertain: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			_, err := New(server.URL, "token", "Authorization").Submit(context.Background(), "workflow", map[string]any{"prompt": "test"})
			if err == nil || IsSubmitUncertain(err) != tt.uncertain {
				t.Fatalf("uncertainty=%v want %v, err=%v", IsSubmitUncertain(err), tt.uncertain, err)
			}
		})
	}
}

func TestSubmitLocalFailuresAreDefiniteAndDoNotSend(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer server.Close()
	for _, body := range []map[string]any{{"unsupported": make(chan int)}} {
		_, err := New(server.URL, "token", "Authorization").Submit(context.Background(), "workflow", body)
		if err == nil || IsSubmitUncertain(err) {
			t.Fatalf("local encode failure must be definite: %v", err)
		}
	}
	_, err := New(server.URL, "", "Authorization").Submit(context.Background(), "workflow", nil)
	if err == nil || IsSubmitUncertain(err) || called {
		t.Fatalf("local validation sent request or uncertain: called=%v err=%v", called, err)
	}
}
