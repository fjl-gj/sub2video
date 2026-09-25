package worker

import (
	"testing"

	"github.com/sbu2video/backend/internal/store"
)

func TestBuildProviderRequestBodyFiltersWorkflowParameters(t *testing.T) {
	task := &store.ProviderTask{
		Prompt: "portrait speaking to camera", DurationSec: 12, Resolution: "768P", AspectRatio: "16:9",
		GenerateAudio: true, ExecutionType: "workflow",
		ParameterMapping: map[string]any{"omit": []any{"prompt", "duration"}},
	}

	body := buildProviderRequestBody(task)
	if _, ok := body["prompt"]; ok {
		t.Fatalf("workflow body must omit prompt: %#v", body)
	}
	if _, ok := body["duration"]; ok {
		t.Fatalf("workflow body must omit duration: %#v", body)
	}
	if _, ok := body["generate_audio"]; ok {
		t.Fatalf("workflow body must not receive generic generate_audio: %#v", body)
	}
	if body["resolution"] == "" {
		t.Fatalf("workflow body must retain resolution: %#v", body)
	}
}

func TestBuildProviderRequestBodyKeepsAPIGeneratedAudioFlag(t *testing.T) {
	task := &store.ProviderTask{
		Prompt: "cinematic city", DurationSec: 5, Resolution: "480P", AspectRatio: "9:16",
		GenerateAudio: true, ExecutionType: "api",
	}

	body := buildProviderRequestBody(task)
	if body["generate_audio"] != true {
		t.Fatalf("API body must retain generated audio flag: %#v", body)
	}
	if body["prompt"] != task.Prompt || body["duration"] != task.DurationSec {
		t.Fatalf("API body lost generic fields: %#v", body)
	}
}
