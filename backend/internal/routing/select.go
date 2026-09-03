package routing

import (
	"time"

	"github.com/sbu2video/backend/internal/store"
)

type Request struct {
	ClientWorkflowID string
	SourceType       string // text | upload | asset
	DurationSec      int
	Resolution       string
	AspectRatio      string
	Now              time.Time
}

// SelectLowestCost picks an enabled route that satisfies conditions, preferring lowest cost.
func SelectLowestCost(routes []store.SupplierRoute, req Request) (*store.SupplierRoute, float64) {
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	peak := isPeak(req.Now)
	var best *store.SupplierRoute
	bestCost := 0.0
	for i := range routes {
		r := &routes[i]
		if !r.Enabled {
			continue
		}
		if req.ClientWorkflowID != "" && r.ClientWorkflowID != req.ClientWorkflowID {
			continue
		}
		if !matchConditions(r.Conditions, req) {
			continue
		}
		cost := r.CostPerSecOffpeak
		if peak {
			cost = r.CostPerSecPeak
		}
		if best == nil || cost < bestCost || (cost == bestCost && r.Priority < best.Priority) {
			best = r
			bestCost = cost
		}
	}
	return best, bestCost
}

func isPeak(t time.Time) bool {
	h := t.Hour()
	return h >= 8 && h < 24
}

func matchConditions(c store.RouteConditions, req Request) bool {
	if c.MaxDuration > 0 && req.DurationSec > c.MaxDuration {
		return false
	}
	if c.MinDuration > 0 && req.DurationSec < c.MinDuration {
		return false
	}
	if len(c.Resolutions) > 0 && req.Resolution != "" && !containsFold(c.Resolutions, req.Resolution) {
		return false
	}
	if len(c.RequireSource) > 0 && req.SourceType != "" && !containsFold(c.RequireSource, req.SourceType) {
		return false
	}
	if len(c.AspectRatios) > 0 && req.AspectRatio != "" && !containsFold(c.AspectRatios, req.AspectRatio) {
		return false
	}
	return true
}

func containsFold(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// MapResolutionToUpstream converts UI resolution + aspect to AutoDL string.
// AutoDL H3 workflows currently expect 736p / 480p style values.
func MapResolutionToUpstream(resolution, aspect string) string {
	base := "736p"
	switch resolution {
	case "480P", "720P":
		base = "480p"
	case "1080P":
		base = "1080p"
	case "768P", "736P":
		base = "736p"
	}
	switch aspect {
	case "16:9":
		return base + "横"
	case "1:1":
		return base + "(1:1)"
	default:
		return base + "竖"
	}
}
