package handler

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/sbu2video/backend/internal/store"
)

func TestQuoteDTOExcludesInternalSupplyAndCostFields(t *testing.T) {
	quote := &store.TaskQuote{
		ID: "qt_safe", ExpiresAt: time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC),
		CarrierID: "car_minimax", CarrierName: "MINMAX", ModelID: "mdl_h3", ModelName: "H3",
		SupplierCode: "autodl-secret", SupplierBaseURL: "https://supplier.internal", SupplierAuth: "X-Token",
		OfferingID: "off_internal", CostRuleID: "cost_internal", UpstreamWorkflow: "upstream-internal",
		DurationSeconds: 10, PricePerSecond: 0.5, Total: 5,
		CostSnapshot: map[string]any{"costPerSecond": 0.1}, OfferingSnapshot: map[string]any{"token": "secret"},
	}

	raw, err := json.Marshal(quoteDTO(quote))
	if err != nil {
		t.Fatalf("marshal quote dto: %v", err)
	}
	for _, forbidden := range []string{"supplier", "offering", "cost", "token", "upstream", "workflow"} {
		if strings.Contains(strings.ToLower(string(raw)), forbidden) {
			t.Fatalf("quote response leaked %q: %s", forbidden, raw)
		}
	}
	var response map[string]any
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatalf("decode quote dto: %v", err)
	}
	if got := response["quoteId"]; got != quote.ID {
		t.Fatalf("quoteId: got %v want %s", got, quote.ID)
	}
	selection, ok := response["selection"].(map[string]any)
	if !ok || len(selection) != 2 || selection["carrier"] == nil || selection["model"] == nil {
		t.Fatalf("expected only carrier and model in selection: %#v", response["selection"])
	}
}

func TestClientTaskDTOUsesModeWithoutLegacyOrInternalFields(t *testing.T) {
	task := &store.ClientTaskDetail{
		Task: store.Task{
			ID: "tsk_safe", Title: "安全任务", Workflow: "文生视频", Status: "已完成",
			DurationSec: 5, Resolution: "1080P", Cost: 2.5,
		},
		AspectRatio: "16:9",
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
