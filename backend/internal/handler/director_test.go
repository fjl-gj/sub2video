package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sbu2video/backend/internal/store"
)

func TestDirectorRequestRejectsInvalidOrPrivateFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"missing revision", `{"shots":[]}`, 400},
		{"missing shots", `{"expectedRevision":0}`, 400},
		{"null shots", `{"expectedRevision":0,"shots":null}`, 400},
		{"wrong duration type", `{"expectedRevision":0,"shots":[{"id":"shot_1","durationSec":"5"}]}`, 400},
		{"private field", `{"expectedRevision":0,"shots":[{"id":"shot_1","executionType":"workflow"}]}`, 400},
		{"source URL", `{"expectedRevision":0,"shots":[{"id":"shot_1","sourceMeta":{"first":{"type":"upload","id":"upl_1","url":"https://private.test"}}}]}`, 400},
		{"trailing JSON", `{"expectedRevision":0,"shots":[]} {}`, 400},
		{"too large", `{"expectedRevision":0,"shots":[{"id":"shot_1","prompt":"` + strings.Repeat("a", store.MaxDirectorBodyBytes) + `"}]}`, 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.PUT("/director", (&ProjectHandler{}).SaveDirectorBoard)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/director", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "private.test") || strings.Contains(w.Body.String(), "executionType") {
				t.Fatal("invalid input must not be echoed")
			}
		})
	}
}

func TestDirectorErrorsRemainSafeAcrossCreationContract(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{store.ErrDirectorConflict, 409, "DIRECTOR_BOARD_CONFLICT"},
		{store.ErrDirectorShotMissing, 409, "DIRECTOR_SHOT_MISSING"},
		{store.ErrDirectorReference, 400, "DIRECTOR_REFERENCE_INVALID"},
		{store.ErrDirectorTask, 400, "DIRECTOR_TASK_INVALID"},
		{store.ErrDirectorSelection, 400, "DIRECTOR_SELECTION_NOT_READY"},
		{store.ErrNotFound, 404, "NOT_FOUND"},
	} {
		status, code, _ := clientCreationError(fmt.Errorf("wrapped: %w", tc.err))
		if status != tc.status || code != tc.code {
			t.Fatalf("%v: status=%d code=%s", tc.err, status, code)
		}
	}
	status, _, message := directorError(errors.New("postgres password=do-not-expose"))
	if status != 500 || strings.Contains(message, "password") {
		t.Fatal("database diagnostics leaked")
	}
}
