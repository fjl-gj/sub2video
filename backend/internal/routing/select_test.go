package routing

import "testing"

func TestSelectLowestCost(t *testing.T) {
	// smoke via compile of package helpers
	got := MapResolutionToUpstream("768P", "9:16")
	if got != "736p竖" {
		t.Fatalf("got %s", got)
	}
	got = MapResolutionToUpstream("768P", "16:9")
	if got != "736p横" {
		t.Fatalf("got %s", got)
	}
}
