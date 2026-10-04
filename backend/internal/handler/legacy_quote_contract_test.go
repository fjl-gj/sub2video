package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLegacyQuoteTaskReturnsGone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	(&ClientHandler{}).LegacyQuoteTask(ctx)

	if recorder.Code != http.StatusGone {
		t.Fatalf("legacy quote status: got %d want %d", recorder.Code, http.StatusGone)
	}
	if recorder.Body.Len() == 0 {
		t.Fatal("legacy quote response must explain the retired contract")
	}
}
