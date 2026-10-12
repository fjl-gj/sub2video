package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sbu2video/backend/internal/auth"
	"github.com/sbu2video/backend/internal/config"
	"github.com/sbu2video/backend/internal/server"
	"github.com/sbu2video/backend/internal/store"
)

// This exercises the real router, JWT/user lookup, handler and PostgreSQL.
// Candidate tasks are explicit database fixtures; no supplier, worker, object
// storage, HTTP listener or paid generation is started by this test.
func TestDirectorBoardHandlerIntegration(t *testing.T) {
	st := deliveryIntegrationStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	owner, err := st.CreateUserWithWallet(ctx, "owner@director.test", "unused", "Owner", "Director", 10)
	if err != nil {
		t.Fatal(err)
	}
	other, err := st.CreateUserWithWallet(ctx, "other@director.test", "unused", "Other", "Other", 10)
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(ctx, owner.ID, "导演台 HTTP 验收", "", false)
	if err != nil {
		t.Fatal(err)
	}
	const readyID, waitingID, foreignID = "tsk_director_ready", "tsk_director_waiting", "tsk_director_foreign"
	for _, fixture := range []struct {
		id             string
		user           int64
		status, result string
	}{
		{readyID, owner.ID, "completed", "https://private-storage.invalid/director.mp4"},
		{waitingID, owner.ID, "completed", ""},
		{foreignID, other.ID, "completed", "https://private-storage.invalid/foreign.mp4"},
	} {
		if _, err := st.Pool.Exec(ctx, `INSERT INTO generation_tasks
(public_id,user_id,title,workflow,status,duration_sec,resolution,cost,result_oss_url,supplier_code,upstream_task_id,cost_snapshot,execution_snapshot)
VALUES ($1,$2,'数据库合成候选','文生视频',$3,5,'768P',0,$4,'private-supplier','private-upstream',
'{"costPerSecond":0.007}'::jsonb,'{"executionType":"workflow","workflowCode":"private-workflow","token":"private-credential"}'::jsonb)`, fixture.id, fixture.user, fixture.status, fixture.result); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Config{JWTSecret: "director-http-integration-only", CORSOrigin: "https://client.example.test"}
	router := server.NewRouter(cfg, st, auth.NewService(st, cfg), nil)
	ownerToken, otherToken := deliveryTestToken(t, owner, cfg.JWTSecret), deliveryTestToken(t, other, cfg.JWTSecret)
	path := "/api/v1/client/projects/" + project.ID + "/director-board"
	candidatePath := path + "/shots/shot_one/candidates"
	request := func(method, url, token string, body any, status int, code string) *httptest.ResponseRecorder {
		t.Helper()
		var raw []byte
		if body != nil {
			var err error
			raw, err = json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
		}
		req := httptest.NewRequest(method, url, strings.NewReader(string(raw)))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != status {
			t.Fatalf("%s %s: status=%d want=%d body=%s", method, url, res.Code, status, res.Body.String())
		}
		if code != "" {
			var payload struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil || payload.Code != code {
				t.Fatalf("error code: got=%s want=%s err=%v", payload.Code, code, err)
			}
		}
		assertDirectorResponsePrivacy(t, res.Body.Bytes())
		return res
	}
	readBoard := func(res *httptest.ResponseRecorder) store.DirectorBoard {
		t.Helper()
		var board store.DirectorBoard
		if err := json.Unmarshal(res.Body.Bytes(), &board); err != nil {
			t.Fatal(err)
		}
		if board.ProjectID != project.ID {
			t.Fatalf("wrong project response: %q", board.ProjectID)
		}
		return board
	}
	put := func(revision int64, shots []store.DirectorShot) map[string]any {
		return map[string]any{"expectedRevision": revision, "shots": shots}
	}
	initial := readBoard(request(http.MethodGet, path, ownerToken, nil, http.StatusOK, ""))
	if initial.Revision != 0 || initial.Shots == nil || len(initial.Shots) != 0 {
		t.Fatalf("new board must be empty: %+v", initial)
	}
	shots := []store.DirectorShot{{ID: "shot_one", Title: "海边首镜", Prompt: "A token of friendship on the shore", CandidateTaskIDs: []string{}}}
	for _, denied := range []struct {
		token  string
		status int
		code   string
	}{{"", 401, "UNAUTHORIZED"}, {otherToken, 404, "NOT_FOUND"}} {
		request(http.MethodGet, path, denied.token, nil, denied.status, denied.code)
		request(http.MethodPut, path, denied.token, put(0, shots), denied.status, denied.code)
		request(http.MethodPost, candidatePath, denied.token, map[string]string{"taskId": readyID}, denied.status, denied.code)
	}
	saved := readBoard(request(http.MethodPut, path, ownerToken, put(0, shots), 200, ""))
	if saved.Revision != 1 || len(saved.Shots) != 1 || saved.Shots[0].Title != shots[0].Title {
		t.Fatalf("save response: %+v", saved)
	}
	fresh := readBoard(request(http.MethodGet, path, ownerToken, nil, 200, ""))
	if fresh.Revision != saved.Revision || fresh.Shots[0].Prompt != shots[0].Prompt {
		t.Fatal("fresh HTTP read lost saved data")
	}
	request(http.MethodPut, path, ownerToken, put(0, []store.DirectorShot{}), 409, "DIRECTOR_BOARD_CONFLICT")
	added := readBoard(request(http.MethodPost, candidatePath, ownerToken, map[string]string{"taskId": readyID}, 200, ""))
	if added.Revision != 2 || len(added.Shots[0].CandidateTaskIDs) != 1 {
		t.Fatalf("candidate append: %+v", added)
	}
	replayed := readBoard(request(http.MethodPost, candidatePath, ownerToken, map[string]string{"taskId": readyID}, 200, ""))
	if replayed.Revision != 2 || len(replayed.Shots[0].CandidateTaskIDs) != 1 {
		t.Fatal("candidate retry changed revision or duplicated link")
	}
	added = readBoard(request(http.MethodPost, candidatePath, ownerToken, map[string]string{"taskId": waitingID}, 200, ""))
	if added.Revision != 3 {
		t.Fatalf("second candidate revision=%d", added.Revision)
	}
	added.Shots[0].SelectedTaskID = waitingID
	request(http.MethodPut, path, ownerToken, put(3, added.Shots), 400, "DIRECTOR_SELECTION_NOT_READY")
	added.Shots[0].SelectedTaskID = readyID
	saved = readBoard(request(http.MethodPut, path, ownerToken, put(3, added.Shots), 200, ""))
	if saved.Revision != 4 {
		t.Fatalf("selection revision=%d", saved.Revision)
	}
	fresh = readBoard(request(http.MethodGet, path, ownerToken, nil, 200, ""))
	if fresh.Revision != 4 || fresh.Shots[0].SelectedTaskID != readyID {
		t.Fatal("selected result did not persist through a fresh request")
	}
	request(http.MethodPost, candidatePath, ownerToken, map[string]string{"taskId": foreignID}, 400, "DIRECTOR_TASK_INVALID")
	request(http.MethodPost, path+"/shots/deleted_shot/candidates", ownerToken, map[string]string{"taskId": readyID}, 409, "DIRECTOR_SHOT_MISSING")
	request(http.MethodPut, path, ownerToken, map[string]any{"expectedRevision": 4, "shots": []map[string]any{{"id": "shot_one", "executionType": "workflow"}}}, 400, "DIRECTOR_BOARD_INVALID")
	request(http.MethodPost, candidatePath, ownerToken, map[string]string{"taskId": readyID, "token": "private-credential"}, 400, "DIRECTOR_BOARD_INVALID")
	fresh = readBoard(request(http.MethodGet, path, ownerToken, nil, 200, ""))
	if fresh.Revision != 4 || fresh.Shots[0].SelectedTaskID != readyID || len(fresh.Shots[0].CandidateTaskIDs) != 2 {
		t.Fatal("rejected requests mutated the board")
	}
	if _, err := st.TrashProject(ctx, owner.ID, project.ID); err != nil {
		t.Fatal(err)
	}
	request(http.MethodGet, path, ownerToken, nil, 404, "NOT_FOUND")
	request(http.MethodPut, path, ownerToken, put(4, fresh.Shots), 404, "NOT_FOUND")
	request(http.MethodPost, candidatePath, ownerToken, map[string]string{"taskId": readyID}, 404, "NOT_FOUND")
}

func assertDirectorResponsePrivacy(t *testing.T, raw []byte) {
	t.Helper()
	var body any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	forbidden := map[string]bool{"supplier": true, "suppliercode": true, "cost": true, "costsnapshot": true, "executiontype": true, "executionsnapshot": true, "workflowcode": true, "apimodelcode": true, "token": true, "upstreamtaskid": true, "result_oss_url": true}
	var walk func(any)
	walk = func(value any) {
		switch node := value.(type) {
		case map[string]any:
			for key, value := range node {
				if forbidden[strings.ToLower(key)] {
					t.Fatalf("director response leaked internal field %s", key)
				}
				walk(value)
			}
		case []any:
			for _, value := range node {
				walk(value)
			}
		case string:
			for _, secret := range []string{"private-storage.invalid", "private-supplier", "private-upstream", "private-workflow", "private-credential"} {
				if strings.Contains(node, secret) {
					t.Fatal("director response leaked an internal fixture value")
				}
			}
		}
	}
	walk(body)
}
