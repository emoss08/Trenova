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
		buncolgen.ShipmentTable.Alias,
		req.Filter,
		(*shipment.Shipment)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return standardShipmentFilter(sq, req.ShipmentOptions)
	})

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func unassignedFilterQuery(
	q *bun.SelectQuery,
	dba bun.IDB,
	req *repositories.GetUnassignedShipmentsRequest,
) *bun.SelectQuery {
	sp := buncolgen.ShipmentColumns
	sm := buncolgen.ShipmentMoveColumns
	a := buncolgen.AssignmentColumns

	q = querybuilder.ApplyFilters(
		q,
		buncolgen.ShipmentTable.Alias,
		req.Filter,
		(*shipment.Shipment)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return standardShipmentFilter(sq, req.ShipmentOptions)
	})

	activeAssignmentSubquery := dba.NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		ColumnExpr("1").
		Join(
			"JOIN ? AS ?",
			bun.Ident(buncolgen.AssignmentTable.Name),
			bun.Ident(buncolgen.AssignmentTable.Alias),
		).
		JoinOn(a.ShipmentMoveID.EqColumn(sm.ID)).
		JoinOn(a.OrganizationID.EqColumn(sm.OrganizationID)).
		JoinOn(a.BusinessUnitID.EqColumn(sm.BusinessUnitID)).
		JoinOn(a.ArchivedAt.IsNull()).
		Where(sm.ShipmentID.EqColumn(sp.ID)).
		Where(sm.OrganizationID.EqColumn(sp.OrganizationID)).
		Where(sm.BusinessUnitID.EqColumn(sp.BusinessUnitID))

	q = q.Where("NOT EXISTS (?)", activeAssignmentSubquery)

	if len(req.Filter.Sort) == 0 {
		q = q.Order(sp.CreatedAt.OrderDesc())
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}
