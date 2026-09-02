package pricing

const (
	RuleID  = "pr-1"
	Version = "v1.2"
)

var perSecond = map[string]float64{
	"720P":  0.4,
	"768P":  0.5,
	"1080P": 0.8,
}

func PricePerSecond(resolution string) (float64, bool) {
	p, ok := perSecond[resolution]
	return p, ok
}

func Quote(resolution string, durationSec int) (cost, pps float64, ok bool) {
	pps, ok = PricePerSecond(resolution)
	if !ok || durationSec <= 0 {
		return 0, 0, false
	}
	return pps * float64(durationSec), pps, true
}
