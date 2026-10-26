package reportjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	catalogHeartbeatEvery     = 5 * time.Minute
	catalogHeartbeatTimeout   = 10 * time.Second
	catalogVersionAliveFor    = 20 * time.Minute
	catalogVersionForgetAfter = 30 * 24 * time.Hour
)

type CatalogHeartbeatParams struct {
	fx.In

	LC       fx.Lifecycle
	Versions reportports.ReportCatalogVersionRepository
	Engine   reportports.ReportEngine
	Logger   *zap.Logger
}

type catalogHeartbeat struct {
	versions reportports.ReportCatalogVersionRepository
	engine   reportports.ReportEngine
	l        *zap.Logger
}

//nolint:gocritic // fx.In parameter structs must be passed by value
func StartCatalogHeartbeat(p CatalogHeartbeatParams) {
	h := &catalogHeartbeat{
		versions: p.Versions,
		engine:   p.Engine,
		l:        p.Logger.Named("reporting.catalog-heartbeat"),
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	p.LC.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go h.run(ctx, done)
			return nil
		},
		OnStop: func(stop context.Context) error {
			cancel()
			select {
			case <-done:
			case <-stop.Done():
			}
			return nil
		},
	})
}

func (h *catalogHeartbeat) run(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(catalogHeartbeatEvery)
	defer ticker.Stop()

	for {
		h.beat(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (h *catalogHeartbeat) beat(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, catalogHeartbeatTimeout)
	defer cancel()

	version := h.engine.Catalog().Version()
	if err := recordCatalogVersion(ctx, h.versions, version, timeutils.NowUnix()); err != nil {
		h.l.Warn("failed to record the report catalog version",
			zap.String("catalogVersion", version), zap.Error(err))
	}
}

func recordCatalogVersion(
	ctx context.Context,
	versions reportports.ReportCatalogVersionRepository,
	version string,
	now int64,
) error {
	return versions.Heartbeat(ctx, &reportports.ReportCatalogHeartbeatRequest{
		Version:      version,
		Now:          now,
		ForgetBefore: now - int64(catalogVersionForgetAfter.Seconds()),
	})
}

func catalogVersionAliveSince(now int64) int64 {
	return now - int64(catalogVersionAliveFor.Seconds())
}
