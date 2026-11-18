package reconcile_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/telemetry"
)

type fakeTelemetry struct {
	mu       sync.Mutex
	observed []health.CheckParams
	run      func(ctx context.Context) error
}

func (f *fakeTelemetry) Run(ctx context.Context) error {
	f.mu.Lock()
	run := f.run
	f.mu.Unlock()
	if run != nil {
		return run(ctx)
	}
	<-ctx.Done()
	return ctx.Err()
}

func (f *fakeTelemetry) Observe(p health.CheckParams) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.observed = append(f.observed, p)
}

func (f *fakeTelemetry) last() (health.CheckParams, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.observed) == 0 {
		return health.CheckParams{}, false
	}
	return f.observed[len(f.observed)-1], true
}

type partsSource struct{}

func (partsSource) Sample(
	_ context.Context,
	at time.Time,
	params *health.CheckParams,
) instanceapi.TelemetrySample {
	sample := instanceapi.TelemetrySample{At: at, Parts: []instanceapi.PartUsage{}}
	if params != nil {
		sample.Parts = append(sample.Parts, instanceapi.PartUsage{Name: "api", Ready: true})
	}
	return sample
}

func TestTelemetryReadsTheSamePartsTheReportWasBuiltFrom(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)

	w.loop.Tick(t.Context())

	observed, ok := w.telem.last()
	require.True(t, ok)
	require.Equal(t, w.health.last(), observed)
}

func TestAFailingTelemetryRelayChangesNothingInTheStatusReport(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)
	w.cp.FailTelemetryNext(1000)
	start := time.Now()
	base := start.UTC().Truncate(time.Minute).Add(time.Minute - 50*time.Millisecond)
	relay, err := telemetry.New(&telemetry.Params{
		ControlPlane: w.client,
		Source:       partsSource{},
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:          func() time.Time { return base.Add(time.Since(start)) },
	})
	require.NoError(t, err)
	w.telem.mu.Lock()
	w.telem.run = func(ctx context.Context) error {
		relay.Observe(health.CheckParams{})
		return relay.Run(ctx)
	}
	w.telem.mu.Unlock()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- w.loop.Run(ctx) }()
	require.Eventually(t, func() bool {
		return w.cp.TelemetryCalls() >= 1 && len(w.cp.Reports()) >= 1
	}, 10*time.Second, 10*time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the loop did not stop with the telemetry relay beside it")
	}

	r := w.lastReport(t)
	require.True(t, r.Ready)
	require.Nil(t, r.Message)
	require.Empty(t, w.cp.TelemetrySamples(), "the control plane accepted nothing")
}
