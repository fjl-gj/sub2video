package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type offeringErrorRow struct{ err error }

func (r offeringErrorRow) Scan(...any) error { return r.err }

type offeringErrorQuerier struct {
	pgx.Tx
	err   error
	calls int
}

func (q *offeringErrorQuerier) QueryRow(context.Context, string, ...any) pgx.Row {
	q.calls++
	return offeringErrorRow{q.err}
}

func (q *offeringErrorQuerier) Query(context.Context, string, ...any) (pgx.Rows, error) {
	q.calls++
	return nil, q.err
}

func TestSupplierOfferingFallbackOnlyOnMissingExecution(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		err   error
		want  error
		calls int
	}{
		{"database_failure", errors.New("database unavailable"), nil, 1},
		{"cancelled", context.Canceled, context.Canceled, 1},
		{"not_found", pgx.ErrNoRows, ErrNotFound, 2},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			q := &offeringErrorQuerier{err: scenario.err}
			_, err := getSupplierOfferingExecution(context.Background(), q, "offering")
			want := scenario.want
			if want == nil {
				want = scenario.err
			}
			if !errors.Is(err, want) || q.calls != scenario.calls {
				t.Fatalf("lookup err=%v calls=%d; want err=%v calls=%d", err, q.calls, want, scenario.calls)
			}
		})
	}
}

func TestPromptRequiredDoesNotDependOnMessageMatching(t *testing.T) {
	for _, prompt := range []string{"", " ", "\n\t"} {
		err := validateModeInput(context.Background(), nil, 1, creationSourceConstraints{ModeCode: "text_to_video"}, prompt, nil)
		if !errors.Is(err, ErrPromptRequired) {
			t.Fatalf("prompt %q: want ErrPromptRequired, got %v", prompt, err)
		}
	}
}

func TestRoutingHelpersPreserveDatabaseFailures(t *testing.T) {
	broken := errors.New("database read failed")
	q := &offeringErrorQuerier{err: broken}
	s := &Store{routeTx: q}
	for name, operation := range map[string]func() error{
		"mode_route": func() error {
			_, err := s.ResolveDirectRoute(context.Background(), DirectRouteRequest{DurationSeconds: 5, Resolution: "768P", AspectRatio: "16:9"})
			return err
		},
		"model_policy":   func() error { _, err := s.modelSupportsMode(context.Background(), "mode", "model"); return err },
		"recommendation": func() error { _, err := s.recommendedModelID(context.Background(), "mode"); return err },
		"cost": func() error {
			_, _, _, err := s.findSupplierCost(context.Background(), "execution", "768P", false)
			return err
		},
		"billable_pair": func() error {
			_, err := s.hasBillableOfferingPair(context.Background(), "mode", "model", "execution", "768P", false)
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := operation(); !errors.Is(err, broken) {
				t.Fatalf("infrastructure failure was replaced by availability result: %v", err)
			}
		})
	}
	q.err = pgx.ErrNoRows
	if _, _, _, err := s.findSupplierCost(context.Background(), "execution", "768P", false); !errors.Is(err, errSupplierCostUnavailable) {
		t.Fatalf("missing cost must remain an unavailable candidate: %v", err)
	}
}
