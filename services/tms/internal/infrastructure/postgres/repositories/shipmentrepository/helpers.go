package shipmentrepository

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
)

func standardShipmentFilter(
	q *bun.SelectQuery,
	opts repositories.ShipmentOptions,
) *bun.SelectQuery {
	if opts.ExpandShipmentDetails {
		q = q.Relation(buncolgen.ShipmentRelations.Customer)

		q = q.RelationWithOpts(buncolgen.ShipmentRelations.AdditionalCharges, bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation(buncolgen.AdditionalChargeRelations.AccessorialCharge)
			},
		})

		q = q.RelationWithOpts(buncolgen.ShipmentRelations.Commodities, bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation(buncolgen.ShipmentCommodityRelations.Commodity).
					Relation(buncolgen.Rel(
						buncolgen.ShipmentCommodityRelations.Commodity,
						buncolgen.CommodityRelations.HazardousMaterial))
			},
		})

		q = q.Relation(buncolgen.ShipmentRelations.ServiceType).
			Relation(buncolgen.ShipmentRelations.ShipmentType).
			Relation(buncolgen.ShipmentRelations.FormulaTemplate).
			Relation(buncolgen.ShipmentRelations.TractorType).
			Relation(buncolgen.ShipmentRelations.TrailerType).
			Relation(buncolgen.ShipmentRelations.CanceledBy).
			Relation(buncolgen.ShipmentRelations.Owner)
	}

	return q
}

func filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListShipmentsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"sp",
		req.Filter,
		(*shipment.Shipment)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return standardShipmentFilter(sq, req.ShipmentOptions)
	})

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}
