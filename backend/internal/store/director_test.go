package store

import (
	"errors"
	"strings"
	"testing"
)

func TestDirectorDraftValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		shots []DirectorShot
		valid bool
	}{
		{"empty board", nil, true},
		{"incomplete draft", []DirectorShot{{ID: "shot_1"}}, true},
		{"stale capability remains a plan", []DirectorShot{{ID: "shot_1", ModeID: "mode_retired", ModelID: "model_retired", Resolution: "1080P", DurationSec: 15}}, true},
		{"duplicate identity", []DirectorShot{{ID: "shot_1"}, {ID: "shot_1"}}, false},
		{"missing identity", []DirectorShot{{Title: "镜头"}}, false},
		{"oversized prompt", []DirectorShot{{ID: "shot_1", Prompt: strings.Repeat("景", 10001)}}, false},
		{"negative duration", []DirectorShot{{ID: "shot_1", DurationSec: -1}}, false},
		{"bad resolution", []DirectorShot{{ID: "shot_1", Resolution: "auto"}}, false},
		{"bad aspect", []DirectorShot{{ID: "shot_1", AspectRatio: "16:0"}}, false},
		{"source URL", []DirectorShot{{ID: "shot_1", SourceMeta: SourceMeta{Refs: []SourceRefItem{{Type: "upload", ID: "https://private.example/file"}}}}}, false},
		{"reference audio out of scope", []DirectorShot{{ID: "shot_1", SourceMeta: SourceMeta{Audio: &SourceRefItem{Type: "upload", ID: "upl_audio"}}}}, false},
		{"selected outside candidates", []DirectorShot{{ID: "shot_1", SelectedTaskID: "tsk_one"}}, false},
		{"duplicate candidates", []DirectorShot{{ID: "shot_1", CandidateTaskIDs: []string{"tsk_one", "tsk_one"}}}, false},
		{"candidate limit", []DirectorShot{{ID: "shot_1", CandidateTaskIDs: make([]string, 21)}}, false},
		{"shot limit", make([]DirectorShot, 101), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDirectorShots(tc.shots)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
			if err != nil && !errors.Is(err, ErrDirectorInvalid) {
				t.Fatalf("unexpected error type: %v", err)
			}
			if tc.valid {
				for _, shot := range tc.shots {
					if shot.CandidateTaskIDs == nil {
						t.Fatal("candidate collection must serialize as []")
					}
				}
			}
		})
	}
}
