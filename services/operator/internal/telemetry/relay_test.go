package telemetry_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/telemetry"
)

type source struct {
	mu     sync.Mutex
	params []*health.CheckParams
}

func (s *source) Sample(_ context.Context, at time.Time, p *health.CheckParams) instanceapi.TelemetrySample {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.params = append(s.params, p)
	return instanceapi.TelemetrySample{
		At:    at,
		Parts: []instanceapi.PartUsage{{Name: "api", Ready: true}},
	}
}

type relayWorld struct {
	cp    *controlplanetest.Server
	relay *telemetry.Relay
	src   *source
	logs  *logCounter
	now   time.Time
}

func newRelay(t *testing.T) *relayWorld {
	t.Helper()
	cp := controlplanetest.New(t, controlplanetest.Spec(uuid.New()))
	client := clientOf(t, cp)
	w := &relayWorld{
		cp:   cp,
		src:  &source{},
		logs: &logCounter{},
		now:  time.Now().UTC().Truncate(time.Minute).Add(-time.Hour),
	}
	var err error
	w.relay, err = telemetry.New(&telemetry.Params{
		ControlPlane: client,
		Source:       w.src,
		Logger:       slog.New(w.logs),
		Now:          func() time.Time { return w.now },
	})
	require.NoError(t, err)
	return w
}

func (w *relayWorld) tick(t *testing.T) {
	t.Helper()
	w.now = w.now.Add(time.Minute)
	w.relay.Tick(t.Context(), w.now)
}

func ascending(t *testing.T, samples []instanceapi.TelemetrySample) {
	t.Helper()
	for i := 1; i < len(samples); i++ {
		require.True(t, samples[i].At.After(samples[i-1].At), "oldest first, no minute twice")
	}
}

func TestAnAcceptedSampleIsDropped(t *testing.T) {
	t.Parallel()
	w := newRelay(t)

	w.tick(t)

	require.Empty(t, w.relay.Kept())
	reports := w.cp.TelemetryReports()
	require.Len(t, reports, 1)
	require.Len(t, reports[0].Samples, 1)
	require.Equal(t, w.now, reports[0].Samples[0].At)
}

func TestAFailureKeepsTheSamplesForTheNextTick(t *testing.T) {
	t.Parallel()
	w := newRelay(t)
	w.cp.FailTelemetryNext(1)

	w.tick(t)
	require.Len(t, w.relay.Kept(), 1)
	w.tick(t)

	require.Empty(t, w.relay.Kept())
	reports := w.cp.TelemetryReports()
	require.Len(t, reports, 1)
	require.Len(t, reports[0].Samples, 2)
	ascending(t, reports[0].Samples)
}

func TestACaughtUpBacklogGoesOutThirtyAtATimeOldestFirst(t *testing.T) {
	t.Parallel()
	w := newRelay(t)
	w.cp.FailTelemetryNext(44)
	first := w.now.Add(time.Minute)

	for range 45 {
		w.tick(t)
	}

	reports := w.cp.TelemetryReports()
	require.Len(t, reports, 1)
	require.Len(t, reports[0].Samples, telemetry.MaxBatch)
	require.Equal(t, first, reports[0].Samples[0].At)
	ascending(t, reports[0].Samples)
	require.Len(t, w.relay.Kept(), 15)

	w.tick(t)
	reports = w.cp.TelemetryReports()
	require.Len(t, reports, 2)
	require.Len(t, reports[1].Samples, 16)
	require.Equal(t, first.Add(30*time.Minute), reports[1].Samples[0].At)
	ascending(t, reports[1].Samples)
	require.Empty(t, w.relay.Kept())
	require.Len(t, w.cp.TelemetrySamples(), 46)
}

func TestTheRingDropsTheOldestPastOneHundredTwenty(t *testing.T) {
	t.Parallel()
	w := newRelay(t)
	w.cp.FailTelemetryNext(1000)
	first := w.now.Add(time.Minute)

	for range telemetry.MaxKept + 5 {
		w.tick(t)
	}

	kept := w.relay.Kept()
	require.Len(t, kept, telemetry.MaxKept)
	require.Equal(t, first.Add(5*time.Minute), kept[0].At)
	require.Equal(t, w.now, kept[len(kept)-1].At)
	ascending(t, kept)
}

func TestTheSameMinuteIsNeverSentTwiceInABatch(t *testing.T) {
	t.Parallel()
	w := newRelay(t)
	w.cp.FailTelemetryNext(3)

	w.tick(t)
	w.relay.Tick(t.Context(), w.now.Add(30*time.Second))
	require.Len(t, w.relay.Kept(), 1, "a second sample for a minute replaces the first")
	w.relay.Tick(t.Context(), w.now.Add(-time.Minute))
	require.Len(t, w.relay.Kept(), 1, "a sample older than the last kept is dropped")

	w.tick(t)
	reports := w.cp.TelemetryReports()
	require.Len(t, reports, 1)
	require.Len(t, reports[0].Samples, 2)
	ascending(t, reports[0].Samples)
}

func TestARateLimitWaitsOutRetryAfter(t *testing.T) {
	t.Parallel()
	w := newRelay(t)
	w.cp.RateLimitNext(1)

	w.tick(t)
	calls := w.cp.TelemetryCalls()
	require.Len(t, w.relay.Kept(), 1)
	w.relay.Tick(t.Context(), w.now)
	require.Equal(t, calls, w.cp.TelemetryCalls(), "nothing is sent before Retry-After")

	w.tick(t)
	require.Empty(t, w.relay.Kept())
	require.Equal(t, calls+1, w.cp.TelemetryCalls())
}

func TestARefusedBatchIsDroppedAndNeverBlocksTheNext(t *testing.T) {
	t.Parallel()
	w := newRelay(t)
	w.cp.FailTelemetryNext(1)
	w.cp.RefuseTelemetryNext(1)

	w.tick(t)
	require.Len(t, w.relay.Kept(), 1, "a failure keeps the sample")
	w.tick(t)
	require.Empty(t, w.relay.Kept(), "a refusal can never succeed, so the batch is dropped")
	require.Equal(t, 1, w.logs.count(slog.LevelError, "refused the samples"))

	w.tick(t)
	reports := w.cp.TelemetryReports()
	require.Len(t, reports, 1)
	require.Len(t, reports[0].Samples, 1)
	require.Equal(t, w.now, reports[0].Samples[0].At)
	require.Equal(t, 1, w.logs.count(slog.LevelError, "refused the samples"))
}

func TestTheSampleCarriesWhatTheLoopObserved(t *testing.T) {
	t.Parallel()
	w := newRelay(t)

	w.tick(t)
	w.relay.Observe(health.CheckParams{ReleaseJob: "trenova-release-3", Generation: "2"})
	w.tick(t)

	require.Nil(t, w.src.params[0], "nothing is observed before the loop has read the release")
	require.Equal(t, &health.CheckParams{ReleaseJob: "trenova-release-3", Generation: "2"}, w.src.params[1])
}

func TestRunSamplesOnTheMinuteAndStopsCleanly(t *testing.T) {
	t.Parallel()
	w := newRelay(t)
	start := time.Now()
	base := start.UTC().Truncate(time.Minute).Add(time.Minute - 50*time.Millisecond)
	relay, err := telemetry.New(&telemetry.Params{
		ControlPlane: clientOf(t, w.cp),
		Source:       w.src,
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:          func() time.Time { return base.Add(time.Since(start)) },
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- relay.Run(ctx) }()
	require.Eventually(t, func() bool { return len(w.cp.TelemetryReports()) == 1 },
		5*time.Second, 10*time.Millisecond)
	cancel()
	select {
	case err = <-done:
		require.True(t, errors.Is(err, context.Canceled))
	case <-time.After(5 * time.Second):
		t.Fatal("the relay did not stop")
	}

	at := w.cp.TelemetryReports()[0].Samples[0].At
	require.Equal(t, base.Add(50*time.Millisecond), at)
	require.Zero(t, at.Second())
	require.Zero(t, at.Nanosecond())
}

func clientOf(t *testing.T, cp *controlplanetest.Server) *controlplane.Client {
	t.Helper()
	bundle := cp.Bundle(time.Hour)
	ident, err := bundle.Identity()
	require.NoError(t, err)
	tr := &credential.Transport{}
	require.NoError(t, tr.Use(&ident))
	client, err := controlplane.New(cp.URL, &http.Client{Transport: tr})
	require.NoError(t, err)
	return client
}
