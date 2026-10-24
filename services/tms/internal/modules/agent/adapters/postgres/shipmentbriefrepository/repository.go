package shipmentbriefrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/shipmentbrief"
	"github.com/emoss08/trenova/internal/modules/agent/domain/shipmentbrief/shipmentbriefdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/shipmentbriefports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxDeleteBatch = 1000

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) shipmentbriefports.ShipmentBriefRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-brief-repository"),
	}
}

func (r *repository) Latest(
	ctx context.Context,
	req *shipmentbriefports.GetLatestShipmentBriefRequest,
) (*shipmentbrief.Brief, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipmentbrief.Brief, error) {
		return r.latest(ctx, req.BriefDate)
	})
}

func (r *repository) latest(ctx context.Context, briefDate string) (*shipmentbrief.Brief, error) {
	b := shipmentbriefdb.Briefs
	brief, err := b.Where(b.BriefDate.Eq(briefDate)).
		Order(b.Generation.Desc()).
		First(ctx, r.db.Keel(ctx))
	if dberror.IsNotFoundError(err) {
		return nil, nil //nolint:nilnil // no brief has been written for the day yet
	}
	if err != nil {
		return nil, fmt.Errorf("read latest shipment brief: %w", err)
	}

	return brief, nil
}

func (r *repository) Insert(
	ctx context.Context,
	brief *shipmentbrief.Brief,
) (*shipmentbrief.Brief, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipmentbrief.Brief, error) {
		b := shipmentbriefdb.Briefs
		inserted, err := b.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*shipmentbrief.Brief{brief},
			b.OrganizationID,
			b.BusinessUnitID,
			b.BriefDate,
			b.Generation,
		)
		if err != nil {
			r.l.Error("failed to insert shipment brief", zap.Error(err))
			return nil, fmt.Errorf("insert shipment brief: %w", err)
		}
		if inserted > 0 {
			return brief, nil
		}

		return r.latest(ctx, brief.BriefDate)
	})
}

func (r *repository) DeleteBefore(
	ctx context.Context,
	req *shipmentbriefports.DeleteShipmentBriefsBeforeRequest,
) (int, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"delete shipment board briefs past retention across every organization",
	)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxDeleteBatch {
			limit = maxDeleteBatch
		}

		b := shipmentbriefdb.Briefs
		stale := b.As("stale")
		n, err := b.AllTenants().
			Where(b.ID.InQuery(stale.AllTenants().
				Pick(stale.ID).
				Where(stale.BriefDate.Lt(req.BeforeDate)).
				Order(stale.BriefDate.Asc()).
				Limit(limit))).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("delete shipment briefs: %w", err)
		}

		return int(n), nil
	})
}
