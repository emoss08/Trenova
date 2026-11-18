package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"sync/atomic"
	"time"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const (
	MaxKept     = 120
	MaxBatch    = 30
	sendTimeout = 15 * time.Second
	keptFor     = 2*time.Hour - time.Minute
)

type ControlPlane interface {
	ReportTelemetry(ctx context.Context, report instanceapi.TelemetryReport) error
}

type Source interface {
	Sample(
		ctx context.Context,
		at time.Time,
		params *health.CheckParams,
	) instanceapi.TelemetrySample
}

type Relay struct {
	cp      ControlPlane
	source  Source
	logger  *slog.Logger
	now     func() time.Time
	params  atomic.Pointer[health.CheckParams]
	kept    []instanceapi.TelemetrySample
	waitFor time.Time
}

type Params struct {
	ControlPlane ControlPlane
	Source       Source
	Logger       *slog.Logger
	Now          func() time.Time
}

func New(p *Params) (*Relay, error) {
	switch {
	case p.ControlPlane == nil:
		return nil, errors.New("telemetry: the relay needs a control plane")
	case p.Source == nil:
		return nil, errors.New("telemetry: the relay needs a source of samples")
	case p.Logger == nil:
		return nil, errors.New("telemetry: the relay needs a logger")
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Relay{cp: p.ControlPlane, source: p.Source, logger: p.Logger, now: now}, nil
}

func (r *Relay) Observe(p health.CheckParams) {
	r.params.Store(&p)
}

func (r *Relay) Run(ctx context.Context) error {
	for {
		next := r.now().UTC().Truncate(time.Minute).Add(time.Minute)
		if !sleep(ctx, next.Sub(r.now())) {
			return ctx.Err()
		}
		r.Tick(ctx, next)
	}
}

func (r *Relay) Tick(ctx context.Context, at time.Time) {
	r.keep(r.source.Sample(ctx, at.UTC().Truncate(time.Minute), r.params.Load()))
	r.send(ctx)
}

func (r *Relay) keep(sample instanceapi.TelemetrySample) {
	if n := len(r.kept); n > 0 {
		last := r.kept[n-1].At
		switch {
		case sample.At.Equal(last):
			r.kept[n-1] = sample
			return
		case sample.At.Before(last):
			r.logger.Warn("telemetry: a sample older than the last one kept is dropped",
				"at", sample.At, "last", last)
			return
		}
	}
	if over := len(r.kept) + 1 - MaxKept; over > 0 {
		r.kept = slices.Delete(r.kept, 0, over)
	}
	r.kept = append(r.kept, sample)
}

func (r *Relay) send(ctx context.Context) {
	now := r.now()
	oldest := now.Add(-keptFor)
	stale := 0
	for stale < len(r.kept) && r.kept[stale].At.Before(oldest) {
		stale++
	}
	if stale > 0 {
		r.kept = slices.Delete(r.kept, 0, stale)
		r.logger.Warn(
			"telemetry: samples too old for the control plane are dropped",
			"dropped",
			stale,
		)
	}
	if len(r.kept) == 0 || now.Before(r.waitFor) {
		return
	}
	batch := slices.Clone(r.kept[:min(len(r.kept), MaxBatch)])
	run, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	err := r.cp.ReportTelemetry(run, instanceapi.TelemetryReport{Samples: batch})
	if err == nil {
		r.kept = slices.Delete(r.kept, 0, len(batch))
		return
	}
	var later *controlplane.RetryLaterError
	var refused *controlplane.ProblemError
	switch {
	case errors.As(err, &refused) && refused.Status == http.StatusBadRequest:
		r.kept = slices.Delete(r.kept, 0, len(batch))
		r.logger.Error("telemetry: the control plane refused the samples; they are dropped",
			"samples", len(batch),
			"first", batch[0].At,
			"code", refused.Code,
			"title", refused.Title,
			"detail", refused.Detail)
		return
	case errors.As(err, &later):
		r.waitFor = now.Add(later.RetryAfter)
	}
	r.logger.Warn(
		"telemetry: cannot report the samples; they are kept and sent again after the next one",
		"samples",
		len(batch),
		"kept",
		len(r.kept),
		"error",
		err,
	)
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(max(d, 0))
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
