package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const (
	DataEvery       = 5 * time.Minute
	DataStale       = 30 * time.Minute
	dataTimeout     = 10 * time.Second
	missingLogEvery = time.Hour
)

type DataSource interface {
	DataUsage(ctx context.Context) (tms.DataUsage, error)
}

type trenovaData struct {
	source  DataSource
	logger  *slog.Logger
	asked   time.Time
	held    *tms.DataUsage
	heldAt  time.Time
	missing time.Time
}

func (d *trenovaData) read(ctx context.Context, at time.Time) instanceapi.DataUsage {
	if d.asked.IsZero() || at.Sub(d.asked) >= DataEvery || at.Before(d.asked) {
		d.asked = at
		d.ask(ctx, at)
	}
	if d.held == nil || at.Sub(d.heldAt) > DataStale {
		return instanceapi.DataUsage{}
	}
	return instanceapi.DataUsage{
		DatabaseBytes:  d.held.DatabaseBytes,
		DocumentsBytes: d.held.DocumentsBytes,
		SearchBytes:    d.held.SearchBytes,
	}
}

func (d *trenovaData) ask(ctx context.Context, at time.Time) {
	run, cancel := context.WithTimeout(ctx, dataTimeout)
	defer cancel()
	got, err := d.source.DataUsage(run)
	switch {
	case err == nil:
		d.held, d.heldAt = &got, at
	case errors.Is(err, tms.ErrNoDataUsage):
		if d.missing.IsZero() || at.Sub(d.missing) >= missingLogEvery {
			d.missing = at
			d.logger.Info("telemetry: this Trenova release does not measure its data; the samples leave it out")
		}
	default:
		d.logger.Warn("telemetry: cannot read what the instance's data takes; the last answer is kept for 30 minutes",
			"error", err)
	}
}
