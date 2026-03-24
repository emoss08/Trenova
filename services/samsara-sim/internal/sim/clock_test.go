package sim

import (
	"testing"
	"time"
)

func TestSimClockPauseAndStep(t *testing.T) {
	t.Parallel()

	currentWall := time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC)
	nowFn := func() time.Time {
		return currentWall
	}
	clock := newSimClockWithNowFn(
		time.Date(2026, time.March, 2, 10, 0, 0, 0, time.UTC),
		nowFn,
	)
	clock.SetPaused(true)

	currentWall = currentWall.Add(30 * time.Minute)
	before := clock.Now()
	if !before.Equal(time.Date(2026, time.March, 2, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("paused clock should not drift, got %s", before)
	}

	afterStep := clock.Step(45 * time.Minute)
	want := time.Date(2026, time.March, 2, 10, 45, 0, 0, time.UTC)
	if !afterStep.Equal(want) {
		t.Fatalf("expected stepped time %s, got %s", want, afterStep)
	}
}

func TestSimClockSpeedMultiplier(t *testing.T) {
	t.Parallel()

	currentWall := time.Date(2026, time.March, 2, 9, 0, 0, 0, time.UTC)
	nowFn := func() time.Time {
		return currentWall
	}
	clock := newSimClockWithNowFn(
		time.Date(2026, time.March, 2, 9, 0, 0, 0, time.UTC),
		nowFn,
	)
	clock.SetSpeed(2)

	currentWall = currentWall.Add(20 * time.Second)
	got := clock.Now()
	want := time.Date(2026, time.March, 2, 9, 0, 40, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("expected 2x speed advancement to %s, got %s", want, got)
	}
}

func TestSimClockSetTimeAndSnapshot(t *testing.T) {
	t.Parallel()

	currentWall := time.Date(2026, time.March, 2, 11, 30, 0, 0, time.UTC)
	nowFn := func() time.Time {
		return currentWall
	}
	clock := newSimClockWithNowFn(
		time.Date(2026, time.March, 2, 11, 0, 0, 0, time.UTC),
		nowFn,
	)
	clock.SetPaused(true)
	assigned := clock.SetTime(time.Date(2026, time.March, 2, 14, 15, 0, 0, time.UTC))
	if !assigned.Equal(time.Date(2026, time.March, 2, 14, 15, 0, 0, time.UTC)) {
		t.Fatalf("expected assigned set time, got %s", assigned)
	}

	snapshot := clock.Snapshot()
	if !snapshot.Now.Equal(assigned) {
		t.Fatalf("snapshot now mismatch: want %s, got %s", assigned, snapshot.Now)
	}
	if !snapshot.Paused {
		t.Fatal("expected paused snapshot")
	}
}
