package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/telemetryrelay"
)

const (
	MaxKept  = telemetryrelay.MaxKept
	MaxBatch = telemetryrelay.MaxBatch
)

type ControlPlane = telemetryrelay.ControlPlane

type Source interface {
	Sample(
		ctx context.Context,
		at time.Time,
		params *health.CheckParams,
	) instanceapi.TelemetrySample
}

type Relay struct {
	queue  *telemetryrelay.Relay
	params atomic.Pointer[health.CheckParams]
}

type Params struct {
	ControlPlane ControlPlane
	Source       Source
	Logger       *slog.Logger
	Now          func() time.Time
}

func New(p *Params) (*Relay, error) {
	if p.Source == nil {
		return nil, errors.New("telemetry: the relay needs a source of samples")
	}
	r := &Relay{}
	queue, err := telemetryrelay.New(&telemetryrelay.Params{
		ControlPlane: p.ControlPlane,
		Source: telemetryrelay.SourceFunc(
			func(ctx context.Context, at time.Time) instanceapi.TelemetrySample {
				return p.Source.Sample(ctx, at, r.params.Load())
			},
		),
		Logger: p.Logger,
		Now:    p.Now,
	})
	if err != nil {
		return nil, err
	}
	r.queue = queue
	return r, nil
}

func (r *Relay) Observe(p health.CheckParams) {
	r.params.Store(&p)
}

func (r *Relay) Run(ctx context.Context) error {
	return r.queue.Run(ctx)
}

func (r *Relay) Tick(ctx context.Context, at time.Time) {
	r.queue.Tick(ctx, at)
}
