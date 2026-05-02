package shutdown

import (
	"context"
	"errors"
	"testing"
)

func TestRunInLIFOOrder(t *testing.T) {
	r := NewRegistry()
	var order []string
	r.Register("first", func(_ context.Context) error {
		order = append(order, "first")
		return nil
	})
	r.Register("second", func(_ context.Context) error {
		order = append(order, "second")
		return nil
	})

	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(order) != 2 || order[0] != "second" || order[1] != "first" {
		t.Fatalf("expected LIFO, got %v", order)
	}
}

func TestRunAggregatesErrors(t *testing.T) {
	r := NewRegistry()
	r.Register("a", func(_ context.Context) error { return errors.New("boom-a") })
	r.Register("b", func(_ context.Context) error { return errors.New("boom-b") })
	r.Register("ok", func(_ context.Context) error { return nil })

	err := r.Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errorContains(err, "a:") || !errorContains(err, "b:") {
		t.Fatalf("expected joined errors, got %v", err)
	}
}

func errorContains(err error, sub string) bool {
	return err != nil && (sub == err.Error() || (len(err.Error()) >= len(sub) && containsSubstring(err.Error(), sub)))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
