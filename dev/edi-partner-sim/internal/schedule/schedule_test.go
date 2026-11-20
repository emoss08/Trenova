package schedule

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/schedule/scheduletest"
)

func TestJobsRunInDueOrderWhenTheClockReachesThem(t *testing.T) {
	t.Parallel()

	clock := scheduletest.NewClock(time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))
	scheduler := New(clock, 1)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go scheduler.Run(ctx)

	var mu sync.Mutex
	var ran []string
	done := make(chan string, 4)
	record := func(name string) func(context.Context) {
		return func(context.Context) {
			mu.Lock()
			ran = append(ran, name)
			mu.Unlock()
			done <- name
		}
	}
	scheduler.After(2*time.Minute, record("late"))
	scheduler.After(45*time.Second, record("first"))
	scheduler.After(45*time.Second, record("second"))
	scheduler.After(0, record("now"))

	if got := wait(t, done); got != "now" {
		t.Fatalf("ran %q first, want now", got)
	}
	clock.Advance(44 * time.Second)
	select {
	case name := <-done:
		t.Fatalf("%s ran before it was due", name)
	case <-time.After(50 * time.Millisecond):
	}
	clock.Advance(time.Second)
	wait(t, done)
	wait(t, done)
	clock.Advance(2 * time.Minute)
	wait(t, done)

	mu.Lock()
	defer mu.Unlock()
	want := []string{"now", "first", "second", "late"}
	for index := range want {
		if ran[index] != want[index] {
			t.Fatalf("ran %v, want %v", ran, want)
		}
	}
	if scheduler.Pending() != 0 {
		t.Fatalf("%d jobs left", scheduler.Pending())
	}
}

func wait(t *testing.T, done <-chan string) string {
	t.Helper()
	select {
	case name := <-done:
		return name
	case <-time.After(5 * time.Second):
		t.Fatal("no job ran")
		return ""
	}
}
