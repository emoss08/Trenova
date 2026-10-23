package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type RatingReviewParams struct {
	fx.In

	DB *postgres.Connection
}

type ratingReviewRepository struct {
	db *postgres.Connection
}

//nolint:gocritic // fx param structs are passed by value
func NewRatingReviewRepository(p RatingReviewParams) shipmentports.RatingReviewRepository {
	return &ratingReviewRepository{db: p.DB}
}

func (r *ratingReviewRepository) GetForRatingReview(
	ctx context.Context,
	req *shipmentports.ApproveRatingReviewRequest,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		entity := new(shipment.Shipment)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return shipmentcol.ShipmentScopeTenant(sq, req.TenantInfo).
					Where(shipmentcol.ShipmentColumns.ID.Eq(), req.ShipmentID)
			}).
			For("UPDATE").
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Shipment")
		}
		return entity, nil
	})
}

func (r *ratingReviewRepository) SaveRatingReview(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		sp := shipmentcol.ShipmentColumns
		ov := entity.Version
		entity.Version++

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			Column(sp.RatingDetail.Bare(), sp.Version.Bare(), sp.UpdatedAt.Bare()).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return shipmentcol.ShipmentScopeTenantUpdate(uq, pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				}).
					Where(sp.ID.Eq(), entity.ID).
					Where(sp.Version.Eq(), ov)
			}).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, err
		}
		if err = dberror.CheckRowsAffected(results, "Shipment", entity.ID.String()); err != nil {
			return nil, err
		}
		return entity, nil
	})
}
