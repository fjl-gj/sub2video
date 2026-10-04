package store

import "testing"

func TestRoundCurrencyMatchesWalletPrecision(t *testing.T) {
	for value, want := range map[float64]float64{
		0.333: 0.33,
		1.236: 1.24,
		2.5:   2.5,
	} {
		if got := roundCurrency(value); got != want {
			t.Fatalf("roundCurrency(%v)=%v, want %v", value, got, want)
		}
	}
}
