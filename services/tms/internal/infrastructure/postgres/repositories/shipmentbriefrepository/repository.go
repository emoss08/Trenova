package shipmentbriefrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipmentbrief"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
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

func New(p Params) repositories.ShipmentBriefRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-brief-repository"),
	}
}

func (r *repository) Latest(
	ctx context.Context,
	req *repositories.GetLatestShipmentBriefRequest,
) (*shipmentbrief.Brief, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipmentbrief.Brief, error) {
		return latest(ctx, r.db.DBForContext(ctx), req)
	})
}

func latest(
	ctx context.Context,
	db bun.IDB,
	req *repositories.GetLatestShipmentBriefRequest,
) (*shipmentbrief.Brief, error) {
	cols := buncolgen.BriefColumns
	brief := new(shipmentbrief.Brief)
	err := db.NewSelect().
		Model(brief).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.BriefDate.Eq(), req.BriefDate).
		Order(cols.Generation.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
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
		cols := buncolgen.BriefColumns
		db := r.db.DBForContext(ctx)
		result, err := db.NewInsert().
			Model(brief).
			On("CONFLICT (" + cols.OrganizationID.Bare() + ", " + cols.BusinessUnitID.Bare() +
				", " + cols.BriefDate.Bare() + ", " + cols.Generation.Bare() + ") DO NOTHING").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to insert shipment brief", zap.Error(err))
			return nil, fmt.Errorf("insert shipment brief: %w", err)
		}
		if affected, _ := result.RowsAffected(); affected > 0 {
			return brief, nil
		}

		return latest(ctx, db, &repositories.GetLatestShipmentBriefRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: brief.OrganizationID,
				BuID:  brief.BusinessUnitID,
			},
			BriefDate: brief.BriefDate,
		})
	})
}

func (r *repository) DeleteBefore(
	ctx context.Context,
	req *repositories.DeleteShipmentBriefsBeforeRequest,
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

		cols := buncolgen.BriefColumns
		res, err := r.db.DBForContext(ctx).NewDelete().
			Model((*shipmentbrief.Brief)(nil)).
			Where(
				cols.ID.Qualified()+" IN (SELECT "+cols.ID.Qualified()+" FROM "+
					buncolgen.BriefTable.Name+" AS "+buncolgen.BriefTable.Alias+
					" WHERE "+cols.BriefDate.Qualified()+" < ? ORDER BY "+
					cols.BriefDate.Qualified()+" LIMIT ?)",
				req.BeforeDate,
				limit,
			).
			Exec(ctx)
		if err != nil {
			return 0, fmt.Errorf("delete shipment briefs: %w", err)
		}
		affected, _ := res.RowsAffected()

		return int(affected), nil
	})
}
