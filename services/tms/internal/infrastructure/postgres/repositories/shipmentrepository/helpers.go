package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
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

func cursorFilterQuery(
	q *bun.SelectQuery,
	req *repositories.ListShipmentsRequest,
) (*bun.SelectQuery, error) {
	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return standardShipmentFilter(sq, req.ShipmentOptions)
	})

	q, err := querybuilder.ApplyCursorFilters(
		q,
		buncolgen.ShipmentTable.Alias,
		req.Filter,
		req.Cursor,
		(*shipment.Shipment)(nil),
	)
	if err != nil {
		return q, err
	}
	if req.ShipmentOptions.Status != "" {
		q = q.Where(buncolgen.ShipmentColumns.Status.Eq(), shipment.Status(req.ShipmentOptions.Status))
	}

	return q, nil
}

func countShipmentListQuery(
	q *bun.SelectQuery,
	req *repositories.ListShipmentsRequest,
) *bun.SelectQuery {
	countReq := *req
	countReq.ShipmentOptions.ExpandShipmentDetails = false

	return baseShipmentListQuery(q, &countReq)
}

func baseShipmentListQuery(
	q *bun.SelectQuery,
	req *repositories.ListShipmentsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.ShipmentTable.Alias,
		req.Filter,
		(*shipment.Shipment)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return standardShipmentFilter(sq, req.ShipmentOptions)
	})

	if req.ShipmentOptions.Status != "" {
		q = q.Where(buncolgen.ShipmentColumns.Status.Eq(), shipment.Status(req.ShipmentOptions.Status))
	}

	return q
}

func unassignedShipmentListQuery(
	q *bun.SelectQuery,
	dba bun.IDB,
	req *repositories.GetUnassignedShipmentsRequest,
) (*bun.SelectQuery, error) {
	q = q.Relation(buncolgen.ShipmentRelations.Customer).
		Where(buncolgen.ShipmentColumns.Status.Eq(), shipment.StatusNew).
		Where("NOT EXISTS (?)", unassignedShipmentPredicate(dba))

	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.ShipmentTable.Alias,
		req.Filter,
		req.Cursor,
		(*shipment.Shipment)(nil),
	)
}

func unassignedShipmentPredicate(dba bun.IDB) *bun.SelectQuery {
	return dba.NewSelect().
		TableExpr(`"shipment_moves" AS "sm"`).
		ColumnExpr("1").
		Join(`JOIN "assignments" AS "a"`).
		JoinOn("a.shipment_move_id = sm.id").
		JoinOn("a.organization_id = sm.organization_id").
		JoinOn("a.business_unit_id = sm.business_unit_id").
		JoinOn("a.archived_at IS NULL").
		JoinOn("a.status != ?", shipment.AssignmentStatusCanceled).
		Where("sm.shipment_id = sp.id").
		Where("sm.organization_id = sp.organization_id").
		Where("sm.business_unit_id = sp.business_unit_id").
		Where("sm.status != ?", shipment.MoveStatusCanceled)
}

func (r *repository) hydrateMoves(
	ctx context.Context,
	shipments []*shipment.Shipment,
) error {
	for _, entity := range shipments {
		if entity == nil || entity.ID.IsNil() {
			continue
		}

		moves, err := r.moveRepository.GetMovesByShipmentID(
			ctx,
			&repositories.GetMovesByShipmentIDRequest{
				ShipmentID: entity.ID,
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
				ExpandMoveDetails: true,
			},
		)
		if err != nil {
			return err
		}

		entity.Moves = moves
	}

	return nil
}
