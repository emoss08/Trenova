package reportrepository

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report/reportdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type CatalogVersionParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type catalogVersionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewCatalogVersionRepository(
	p CatalogVersionParams,
) reportports.ReportCatalogVersionRepository {
	return &catalogVersionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.report-catalog-version-repository"),
	}
}

func (r *catalogVersionRepository) Heartbeat(
	ctx context.Context,
	req *reportports.ReportCatalogHeartbeatRequest,
) error {
	ctx = dbscope.WithSystem(ctx, "record the report catalog version this worker runs")
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		v := reportdb.ReportCatalogVersions
		db := r.db.Keel(ctx)
		if _, err := v.Upsert(
			ctx,
			db,
			[]*report.ReportCatalogVersion{{
				Version:     req.Version,
				FirstSeenAt: req.Now,
				LastSeenAt:  req.Now,
			}},
			keel.OnConflict(v.Version).Update(v.LastSeenAt),
		); err != nil {
			r.l.Error("failed to record the report catalog version", zap.Error(err))
			return err
		}

		if _, err := v.Where(v.LastSeenAt.Lt(req.ForgetBefore)).Delete(ctx, db); err != nil {
			r.l.Error("failed to forget old report catalog versions", zap.Error(err))
			return err
		}

		return nil
	})
}

func (r *catalogVersionRepository) Newest(ctx context.Context, aliveSince int64) (string, error) {
	ctx = dbscope.WithSystem(ctx, "find the newest report catalog version a worker still runs")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (string, error) {
		v := reportdb.ReportCatalogVersions
		row, err := v.Where(v.LastSeenAt.Gte(aliveSince)).
			Order(v.FirstSeenAt.Desc(), v.Version.Desc()).
			First(ctx, r.db.Keel(ctx))
		if errors.Is(err, keel.ErrNotFound) {
			return "", nil
		}
		if err != nil {
			r.l.Error("failed to read the newest report catalog version", zap.Error(err))
			return "", err
		}

		return row.Version, nil
	})
}
