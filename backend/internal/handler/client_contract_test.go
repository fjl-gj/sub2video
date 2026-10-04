package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/response"
	"github.com/sbu2video/backend/internal/store"
)

func TestLegacyQuoteEndpointIsExplicitlyDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/client/tasks/quote", (&ClientHandler{}).LegacyQuoteTask)
	req := httptest.NewRequest(http.MethodPost, "/client/tasks/quote", strings.NewReader(`{"modeId":"mode_text"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusGone {
		t.Fatalf("legacy quote status: got %d want %d", recorder.Code, http.StatusGone)
	}
	var payload response.Body
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode legacy quote response: %v", err)
	}
	if payload.Code != "LEGACY_QUOTE_DISABLED" {
		t.Fatalf("legacy quote error: got %q", payload.Code)
	}
}

func TestClientTaskDTOUsesModeWithoutLegacyOrInternalFields(t *testing.T) {
	task := &store.ClientTaskDetail{
		Task: store.Task{
			ID: "tsk_safe", Title: "安全任务", Workflow: "文生视频", Status: "已完成",
			DurationSec: 5, Resolution: "1080P", Cost: 2.5,
		},
		AspectRatio:    "16:9",
		DeliveryStatus: "ready", DownloadURL: "https://storage.example.test/tsk_safe.mp4",
	}
	raw, err := json.Marshal(clientTaskDetailDTO(task))
	if err != nil {
		t.Fatalf("marshal client task dto: %v", err)
	}
	for _, forbidden := range []string{"workflow", "supplier", "offering", "costsnapshot", "upstream", "token", "adapter"} {
		if strings.Contains(strings.ToLower(string(raw)), forbidden) {
			t.Fatalf("client task response leaked %q: %s", forbidden, raw)
		}
	}
	if !strings.Contains(string(raw), `"mode":"文生视频"`) {
		t.Fatalf("client task response must expose mode: %s", raw)
	}
}

func TestClientTaskDTOCarriesSafeActionsAndPersistedDestination(t *testing.T) {
	task := &store.ClientTaskDetail{
		Task:             store.Task{ID: "tsk_saved", Status: "已完成", SavedToProject: true},
		SavedDestination: &store.TaskSavedDestination{ProjectID: "prj_target", FolderPath: []string{"视频", "正片镜头"}},
	}
	for name, dto := range map[string]map[string]any{"list": taskListDTO(&task.Task), "detail": clientTaskDetailDTO(task)} {
		if dto["canCancel"] != false || dto["reviewRequired"] != false {
			t.Errorf("%s must explicitly deny unavailable actions: %v", name, dto)
		}
	}
	raw, err := json.Marshal(clientTaskDetailDTO(task))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"savedDestination":{"projectId":"prj_target","folderPath":["视频","正片镜头"]}`) {
		t.Fatalf("saved destination lost creator navigation fields: %s", raw)
	}
	task.SavedDestination = nil
	if _, present := clientTaskDetailDTO(task)["savedDestination"]; present {
		t.Fatal("unavailable or removed saved asset must not invent a destination")
	}
}

func TestClientTicketDTOIncludesOriginalQuestionForDetails(t *testing.T) {
	ticket := &store.Ticket{ID: "tkt_owned", Title: "生成问题", TaskRef: "tsk_owned", Detail: "我想补充说明原问题", Status: "待处理"}
	dto := clientTicketDTO(ticket)
	if dto["detail"] != ticket.Detail || dto["task"] != ticket.TaskRef || dto["status"] != ticket.Status {
		t.Fatalf("ticket detail must preserve submitted question and task context: %v", dto)
	}
	for _, field := range []string{"reply", "resolvedMessage", "userId"} {
		if _, present := dto[field]; present {
			t.Fatalf("ticket details must not invent replies or expose internal owner fields: %v", dto)
		}
	}
}
