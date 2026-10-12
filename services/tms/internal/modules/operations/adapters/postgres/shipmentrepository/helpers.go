package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/commoditycol"
	"github.com/emoss08/trenova/pkg/buncolgen/locationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

func standardShipmentFilter(
	q *bun.SelectQuery,
	opts shipmentports.ShipmentOptions,
) *bun.SelectQuery {
	if opts.ExpandShipmentDetails {
		q = q.Relation(shipmentcol.ShipmentRelations.Customer).
			Relation(shipmentcol.ShipmentRelations.BillToCustomer)

		q = withCharges(q)

		q = q.RelationWithOpts(shipmentcol.ShipmentRelations.Commodities, bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation(shipmentcol.ShipmentCommodityRelations.Commodity).
					Relation(buncolgen.Rel(
						shipmentcol.ShipmentCommodityRelations.Commodity,
						commoditycol.CommodityRelations.HazardousMaterial))
			},
		})

		q = q.Relation(shipmentcol.ShipmentRelations.ServiceType).
			Relation(shipmentcol.ShipmentRelations.ShipmentType).
			Relation(shipmentcol.ShipmentRelations.FormulaTemplate).
			Relation(shipmentcol.ShipmentRelations.TractorType).
			Relation(shipmentcol.ShipmentRelations.TrailerType).
			Relation(shipmentcol.ShipmentRelations.CanceledBy).
			Relation(shipmentcol.ShipmentRelations.Owner)
	} else if opts.IncludeCustomer {
		q = q.Relation(shipmentcol.ShipmentRelations.Customer).
			Relation(shipmentcol.ShipmentRelations.BillToCustomer)
	}
	if opts.IncludeRoute && !opts.ExpandShipmentDetails {
		q = withRoute(q)
	}

	return q
}

func withRoute(q *bun.SelectQuery) *bun.SelectQuery {
	moves := shipmentcol.ShipmentRelations.Moves
	stops := buncolgen.Rel(moves, shipmentcol.ShipmentMoveRelations.Stops)

	return q.
		RelationWithOpts(moves, bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order(shipmentcol.ShipmentMoveColumns.Sequence.OrderAsc())
			},
		}).
		RelationWithOpts(stops, bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order(shipmentcol.StopColumns.Sequence.OrderAsc())
			},
		}).
		Relation(buncolgen.Rel(stops, shipmentcol.StopRelations.Location)).
		Relation(buncolgen.Rel(stops, shipmentcol.StopRelations.Location, locationcol.LocationRelations.State)).
		RelationWithOpts(buncolgen.Rel(moves, shipmentcol.ShipmentMoveRelations.Assignment),
			bun.RelationOpts{AdditionalJoinOnConditions: activeAssignmentJoin()}).
		Relation(buncolgen.Rel(
			moves,
			shipmentcol.ShipmentMoveRelations.Assignment,
			shipmentcol.AssignmentRelations.PrimaryWorker,
		)).
		Relation(buncolgen.Rel(
			moves,
			shipmentcol.ShipmentMoveRelations.Assignment,
			shipmentcol.AssignmentRelations.Tractor,
		)).
		RelationWithOpts(buncolgen.Rel(moves, shipmentcol.ShipmentMoveRelations.CarrierAssignment),
			bun.RelationOpts{AdditionalJoinOnConditions: activeCarrierAssignmentJoin()})
}

// A move holds one live assignment but keeps every one it had: an archived
// driver assignment or a canceled carrier assignment stays beside the live
// one. Joined as a has-one without these conditions, a move could come back
// carrying a canceled carrier and read as covered, or with its live
// assignment missing. The aliases are the ones bun gives a has-one joined
// under the has-many moves: the relation's field name.
func activeAssignmentJoin() []schema.QueryWithArgs {
	return []schema.QueryWithArgs{schema.SafeQuery("?.? IS NULL", []any{
		bun.Ident("assignment"), bun.Ident(shipmentcol.AssignmentColumns.ArchivedAt.Name),
	})}
}

func activeCarrierAssignmentJoin() []schema.QueryWithArgs {
	return []schema.QueryWithArgs{schema.SafeQuery("?.? != ?", []any{
		bun.Ident(
			"carrier_assignment",
		),
		bun.Ident(shipmentcol.CarrierAssignmentColumns.Status.Name),
		shipment.CarrierAssignmentStatusCanceled,
	})}
}

func cursorFilterQuery(
	q *bun.SelectQuery,
	dba bun.IDB,
	req *shipmentports.ListShipmentsRequest,
) (*bun.SelectQuery, error) {
	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return standardShipmentFilter(sq, req.ShipmentOptions)
	})

	q, err := querybuilder.ApplyCursorFilters(
		q,
		shipmentcol.ShipmentTable.Alias,
		req.Filter,
		req.Cursor,
		(*shipment.Shipment)(nil),
	)
	if err != nil {
		return q, err
	}
	q = applyShipmentOptionFilters(q, dba, req.ShipmentOptions)

	return q, nil
}

func countShipmentListQuery(
	q *bun.SelectQuery,
	dba bun.IDB,
	req *shipmentports.ListShipmentsRequest,
) *bun.SelectQuery {
	countReq := *req
	countReq.ShipmentOptions.ExpandShipmentDetails = false
	countReq.ShipmentOptions.IncludeCustomer = false
	countReq.ShipmentOptions.IncludeRoute = false

	return baseShipmentListQuery(q, dba, &countReq)
}

func baseShipmentListQuery(
	q *bun.SelectQuery,
	dba bun.IDB,
	req *shipmentports.ListShipmentsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFiltersWithoutSort(
		q,
		shipmentcol.ShipmentTable.Alias,
		req.Filter,
		(*shipment.Shipment)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return standardShipmentFilter(sq, req.ShipmentOptions)
	})

	return applyShipmentOptionFilters(q, dba, req.ShipmentOptions)
}

func applyShipmentOptionFilters(
	q *bun.SelectQuery,
	dba bun.IDB,
	opts shipmentports.ShipmentOptions,
) *bun.SelectQuery {
	if opts.Status != "" {
		q = q.Where(shipmentcol.ShipmentColumns.Status.Eq(), shipment.Status(opts.Status))
	}
	if len(opts.Statuses) > 0 {
		q = q.Where(shipmentcol.ShipmentColumns.Status.In(), bun.In(opts.Statuses))
	}
	if opts.HasActivityWindow() {
		q = q.Where("EXISTS (?)", activityWindowPredicate(dba, opts))
	}
	if opts.BillingTransferEligible {
		q = billingTransferCandidatePredicate(q)
	}
	if len(opts.CustomerIDs) > 0 {
		q = q.Where(shipmentcol.ShipmentColumns.CustomerID.In(), bun.In(opts.CustomerIDs))
	}
	if len(opts.StopLocationIDs) > 0 {
		q = q.Where("EXISTS (?)", stopLocationPredicate(dba, opts.StopLocationIDs))
	}
	for _, group := range opts.StopEachOf {
		if len(group) > 0 {
			q = q.Where("EXISTS (?)", stopLocationPredicate(dba, group))
		}
	}
	if len(opts.WorkerIDs) > 0 {
		q = q.Where("EXISTS (?)", assignedWorkerPredicate(dba, opts.WorkerIDs))
	}
	if len(opts.TractorIDs) > 0 {
		q = q.Where("EXISTS (?)", assignedTractorPredicate(dba, opts.TractorIDs))
	}
	if len(opts.TrailerIDs) > 0 {
		q = q.Where("EXISTS (?)", assignedTrailerPredicate(dba, opts.TrailerIDs))
	}
	if len(opts.CarrierIDs) > 0 {
		q = q.Where("EXISTS (?)", assignedCarrierPredicate(dba, opts.CarrierIDs))
	}
	if len(opts.CommodityIDs) > 0 {
		q = q.Where("EXISTS (?)", commodityPredicate(dba, opts.CommodityIDs))
	}
	if len(opts.ShipmentTypeIDs) > 0 {
		q = q.Where(shipmentcol.ShipmentColumns.ShipmentTypeID.In(), bun.In(opts.ShipmentTypeIDs))
	}

	return q
}

func commodityPredicate(dba bun.IDB, commodityIDs []pulid.ID) *bun.SelectQuery {
	cols := shipmentcol.ShipmentCommodityColumns

	return dba.NewSelect().
		TableExpr(shipmentcol.ShipmentCommodityTable.As("sc_com")).
		ColumnExpr("1").
		Where("sc_com."+cols.ShipmentID.Name+" = sp.id").
		Where("sc_com."+cols.OrganizationID.Name+" = sp.organization_id").
		Where("sc_com."+cols.BusinessUnitID.Name+" = sp.business_unit_id").
		Where("sc_com."+cols.CommodityID.Name+" IN (?)", bun.In(commodityIDs))
}

func assignedWorkerPredicate(dba bun.IDB, workerIDs []pulid.ID) *bun.SelectQuery {
	return liveAssignmentPredicate(dba).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("a_wrk.primary_worker_id IN (?)", bun.List(workerIDs)).
				WhereOr("a_wrk.secondary_worker_id IN (?)", bun.List(workerIDs))
		})
}

func assignedTractorPredicate(dba bun.IDB, tractorIDs []pulid.ID) *bun.SelectQuery {
	return liveAssignmentPredicate(dba).Where("a_wrk.tractor_id IN (?)", bun.List(tractorIDs))
}

func assignedTrailerPredicate(dba bun.IDB, trailerIDs []pulid.ID) *bun.SelectQuery {
	return liveAssignmentPredicate(dba).Where("a_wrk.trailer_id IN (?)", bun.List(trailerIDs))
}

func assignedCarrierPredicate(dba bun.IDB, carrierIDs []pulid.ID) *bun.SelectQuery {
	return dba.NewSelect().
		TableExpr(`"shipment_moves" AS "sm_car"`).
		ColumnExpr("1").
		Join(`JOIN "carrier_assignments" AS "ca_car"`).
		JoinOn("ca_car.shipment_move_id = sm_car.id").
		JoinOn("ca_car.organization_id = sm_car.organization_id").
		JoinOn("ca_car.business_unit_id = sm_car.business_unit_id").
		JoinOn("ca_car.status != ?", shipment.CarrierAssignmentStatusCanceled).
		Where("sm_car.shipment_id = sp.id").
		Where("sm_car.organization_id = sp.organization_id").
		Where("sm_car.business_unit_id = sp.business_unit_id").
		Where("sm_car.status != ?", shipment.MoveStatusCanceled).
		Where("ca_car.carrier_id IN (?)", bun.List(carrierIDs))
}

func liveAssignmentPredicate(dba bun.IDB) *bun.SelectQuery {
	return dba.NewSelect().
		TableExpr(`"shipment_moves" AS "sm_wrk"`).
		ColumnExpr("1").
		Join(`JOIN "assignments" AS "a_wrk"`).
		JoinOn("a_wrk.shipment_move_id = sm_wrk.id").
		JoinOn("a_wrk.organization_id = sm_wrk.organization_id").
		JoinOn("a_wrk.business_unit_id = sm_wrk.business_unit_id").
		JoinOn("a_wrk.archived_at IS NULL").
		JoinOn("a_wrk.status != ?", shipment.AssignmentStatusCanceled).
		Where("sm_wrk.shipment_id = sp.id").
		Where("sm_wrk.organization_id = sp.organization_id").
		Where("sm_wrk.business_unit_id = sp.business_unit_id").
		Where("sm_wrk.status != ?", shipment.MoveStatusCanceled)
}

func stopLocationPredicate(dba bun.IDB, locationIDs []pulid.ID) *bun.SelectQuery {
	return dba.NewSelect().
		TableExpr(`"shipment_moves" AS "sm_loc"`).
		ColumnExpr("1").
		Join(`JOIN "stops" AS "stp_loc"`).
		JoinOn("stp_loc.shipment_move_id = sm_loc.id").
		JoinOn("stp_loc.organization_id = sm_loc.organization_id").
		JoinOn("stp_loc.business_unit_id = sm_loc.business_unit_id").
		Where("sm_loc.shipment_id = sp.id").
		Where("sm_loc.organization_id = sp.organization_id").
		Where("sm_loc.business_unit_id = sp.business_unit_id").
		Where("sm_loc.status != ?", shipment.MoveStatusCanceled).
		Where("stp_loc.status != ?", shipment.StopStatusCanceled).
		Where("stp_loc.location_id IN (?)", bun.In(locationIDs))
}

func activityWindowPredicate(
	dba bun.IDB,
	opts shipmentports.ShipmentOptions,
) *bun.SelectQuery {
	return dba.NewSelect().
		TableExpr(`"shipment_moves" AS "sm_aw"`).
		ColumnExpr("1").
		Join(`JOIN "stops" AS "stp_aw"`).
		JoinOn("stp_aw.shipment_move_id = sm_aw.id").
		JoinOn("stp_aw.organization_id = sm_aw.organization_id").
		JoinOn("stp_aw.business_unit_id = sm_aw.business_unit_id").
		Where("sm_aw.shipment_id = sp.id").
		Where("sm_aw.organization_id = sp.organization_id").
		Where("sm_aw.business_unit_id = sp.business_unit_id").
		Where("sm_aw.status != ?", shipment.MoveStatusCanceled).
		Where("stp_aw.scheduled_window_start > 0").
		Where("stp_aw.scheduled_window_start <= ?", opts.ActivityWindowEnd).
		Where(
			"COALESCE(stp_aw.scheduled_window_end, stp_aw.scheduled_window_start) >= ?",
			opts.ActivityWindowStart,
		)
}

func unassignedShipmentListQuery(
	q *bun.SelectQuery,
	dba bun.IDB,
	req *shipmentports.GetUnassignedShipmentsRequest,
) (*bun.SelectQuery, error) {
	q = q.Relation(shipmentcol.ShipmentRelations.Customer).
		Where(shipmentcol.ShipmentColumns.Status.Eq(), shipment.StatusNew).
		Where("NOT EXISTS (?)", unassignedShipmentPredicate(dba))

	return querybuilder.ApplyCursorFilters(
		q,
		shipmentcol.ShipmentTable.Alias,
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
			&shipmentports.GetMovesByShipmentIDRequest{
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

// withCharges loads the additional charges and the charge allocations that
// divide a shipment among its payers.
func withCharges(q *bun.SelectQuery) *bun.SelectQuery {
	q = q.RelationWithOpts(shipmentcol.ShipmentRelations.AdditionalCharges, bun.RelationOpts{
		Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Relation(shipmentcol.AdditionalChargeRelations.AccessorialCharge)
		},
	})

	return q.RelationWithOpts(shipmentcol.ShipmentRelations.ChargeAllocations, bun.RelationOpts{
		Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
			cols := shipmentcol.ChargeAllocationColumns
			return sq.Relation(shipmentcol.ChargeAllocationRelations.BillToCustomer).
				Order(cols.Sequence.OrderAsc(), cols.ID.OrderAsc())
		},
	})
}
