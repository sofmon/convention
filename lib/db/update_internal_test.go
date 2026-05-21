package db

import (
	"errors"
	"testing"
)

type fakeSQLStateErr struct{ state string }

func (e fakeSQLStateErr) Error() string    { return "fake: " + e.state }
func (e fakeSQLStateErr) SQLState() string { return e.state }

func Test_classifyContentionErr(t *testing.T) {
	t.Run("55P03_maps_to_ErrLockNotAvailable", func(t *testing.T) {
		mapped, ok := classifyContentionErr(fakeSQLStateErr{state: "55P03"})
		if !ok {
			t.Fatalf("expected ok=true for SQLSTATE 55P03")
		}
		if !errors.Is(mapped, ErrLockNotAvailable) {
			t.Fatalf("expected ErrLockNotAvailable, got %v", mapped)
		}
	})

	t.Run("other_sqlstate_does_not_map", func(t *testing.T) {
		mapped, ok := classifyContentionErr(fakeSQLStateErr{state: "23505"})
		if ok || mapped != nil {
			t.Fatalf("expected (nil, false) for non-55P03 SQLSTATE, got (%v, %v)", mapped, ok)
		}
	})

	t.Run("plain_error_does_not_map", func(t *testing.T) {
		mapped, ok := classifyContentionErr(errors.New("plain"))
		if ok || mapped != nil {
			t.Fatalf("expected (nil, false) for plain error, got (%v, %v)", mapped, ok)
		}
	})

	t.Run("nil_error_does_not_map", func(t *testing.T) {
		mapped, ok := classifyContentionErr(nil)
		if ok || mapped != nil {
			t.Fatalf("expected (nil, false) for nil error, got (%v, %v)", mapped, ok)
		}
	})
}
