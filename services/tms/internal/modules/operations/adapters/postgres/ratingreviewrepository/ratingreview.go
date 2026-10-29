package ratingreviewrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/ratingreviewports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
)

var shipmentRatingErrors = dberror.Mapping{Entity: "Shipment"}

type RatingReviewParams struct {
	fx.In

	DB *postgres.Connection
}

type ratingReviewRepository struct {
	db *postgres.Connection
}

//nolint:gocritic // fx param structs are passed by value
func NewRatingReviewRepository(p RatingReviewParams) ratingreviewports.RatingReviewRepository {
	return &ratingReviewRepository{db: p.DB}
}

func (r *ratingReviewRepository) GetForRatingReview(
	ctx context.Context,
	req *ratingreviewports.ApproveRatingReviewRequest,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		sp := shipmentdb.Shipments
		entity, err := sp.Where(sp.ID.Eq(req.ShipmentID)).
			Lock(keel.ForUpdate).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, shipmentRatingErrors.Map(err, req.ShipmentID.String())
		}
		return entity, nil
	})
}

func (r *ratingReviewRepository) SaveRatingReview(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		sp := shipmentdb.Shipments
		if err := sp.Save(ctx, r.db.Keel(ctx), entity, sp.RatingDetail); err != nil {
			return nil, shipmentRatingErrors.Map(err, entity.ID.String())
		}
		return entity, nil
	})
}
