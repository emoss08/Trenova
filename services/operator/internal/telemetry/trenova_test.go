package telemetry_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/telemetry"
	"github.com/emoss08/trenova/operator/internal/tms"
)

type logCounter struct {
	mu      sync.Mutex
	records []slog.Record
}

func (l *logCounter) Enabled(context.Context, slog.Level) bool { return true }

func (l *logCounter) Handle(_ context.Context, r slog.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, r)
	return nil
}

func (l *logCounter) WithAttrs([]slog.Attr) slog.Handler { return l }

func (l *logCounter) WithGroup(string) slog.Handler { return l }

func (l *logCounter) count(level slog.Level, contains string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, r := range l.records {
		if r.Level == level && strings.Contains(r.Message, contains) {
			n++
		}
	}
	return n
}

func TestTrenovaIsAskedAtMostEveryFiveMinutes(t *testing.T) {
	t.Parallel()
	w := newWorld(t, "", standardObjects()...)

	for m := range 5 {
		sample := w.sampler.Sample(t.Context(), minute.Add(time.Duration(m)*time.Minute), params())
		require.Equal(t, 5*gib, *sample.Data.DatabaseBytes, "minute %d carries the last answer", m)
	}
	require.Equal(t, 1, w.data.count())

	w.data.set(tms.DataUsage{DatabaseBytes: ptr(6 * gib)}, nil)
	sample := w.sampler.Sample(t.Context(), minute.Add(telemetry.DataEvery), params())
	require.Equal(t, 2, w.data.count())
	require.Equal(t, 6*gib, *sample.Data.DatabaseBytes)
	require.Nil(t, sample.Data.DocumentsBytes, "a figure Trenova could not measure is left out")
}

func TestAReleaseWithoutTheEndpointLeavesTheDataOutAndSaysSoOnceAnHour(t *testing.T) {
	t.Parallel()
	w := newWorld(t, "", standardObjects()...)
	w.data.set(tms.DataUsage{}, tms.ErrNoDataUsage)

	for m := 0; m < 120; m += 5 {
		sample := w.sampler.Sample(t.Context(), minute.Add(time.Duration(m)*time.Minute), params())
		require.Nil(t, sample.Data.DatabaseBytes)
		require.Nil(t, sample.Data.DocumentsBytes)
		require.Nil(t, sample.Data.SearchBytes)
		require.Equal(t, 2*gib, *sample.Data.LogsBytes, "the logs come from Loki's volume")
	}

	require.Equal(t, 24, w.data.count())
	require.Equal(t, 2, w.logs.count(slog.LevelInfo, "does not measure its data"))
}

func TestTheLastAnswerIsKeptForThirtyMinutesThenLeftOut(t *testing.T) {
	t.Parallel()
	w := newWorld(t, "", standardObjects()...)

	w.sampler.Sample(t.Context(), minute, params())
	w.data.set(tms.DataUsage{}, errors.New("the TMS answered 503"))

	for m := 5; m <= 30; m += 5 {
		sample := w.sampler.Sample(t.Context(), minute.Add(time.Duration(m)*time.Minute), params())
		require.Equal(t, 5*gib, *sample.Data.DatabaseBytes, "minute %d keeps the last good answer", m)
	}
	sample := w.sampler.Sample(t.Context(), minute.Add(31*time.Minute), params())
	require.Nil(t, sample.Data.DatabaseBytes)
	require.Nil(t, sample.Data.SearchBytes)
	require.Equal(t, 7, w.data.count())
}
