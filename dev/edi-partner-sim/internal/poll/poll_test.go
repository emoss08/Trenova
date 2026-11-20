package poll

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestUntil(t *testing.T) {
	t.Parallel()
	calls := 0
	err := Until(context.Background(), 5*time.Second, func(context.Context) (bool, error) {
		calls++
		return calls == 3, nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("until done: %v after %d calls", err, calls)
	}
	stop := errors.New("stop")
	err = Until(context.Background(), 5*time.Second, func(context.Context) (bool, error) {
		return false, Stop(stop)
	})
	if !errors.Is(err, stop) {
		t.Fatalf("a stop: %v", err)
	}
	last := errors.New("still failing")
	err = Until(context.Background(), 600*time.Millisecond, func(context.Context) (bool, error) {
		return false, last
	})
	if !errors.Is(err, ErrTimedOut) || !errors.Is(err, last) {
		t.Fatalf("a timeout keeps the last error: %v", err)
	}
}
