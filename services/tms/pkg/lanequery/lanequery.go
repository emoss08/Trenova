package lanequery

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
)

const (
	OriginStateColumn      = "origin_state"
	DestinationStateColumn = "destination_state"
	StatusColumn           = "status"
	CountColumn            = "count"

	endpointsCTE             = "lane_endpoints"
	pairsCTE                 = "lane_pairs"
	pairsAlias               = "lp"
	originLocationAlias      = "loc_orig"
	destinationLocationAlias = "loc_dest"
	originStateAlias         = "ust_orig"
	destinationStateAlias    = "ust_dest"

	pickupLocationIDs     = "pickup_location_ids"
	deliveryLocationIDs   = "delivery_location_ids"
	originLocationID      = "origin_location_id"
	destinationLocationID = "destination_location_id"
	shipmentCount         = "shipment_count"
)

type Options struct {
	Tenant         *pagination.TenantInfo
	ShipmentFilter func(*bun.SelectQuery) *bun.SelectQuery
	GroupByStatus  bool
}

var (
	shipmentCols = buncolgen.ShipmentColumns
	moveCols     = buncolgen.ShipmentMoveColumns
	stopCols     = buncolgen.StopColumns

	moveFromShipmentJoin = "JOIN " + buncolgen.ShipmentMoveTable.As(
		buncolgen.ShipmentMoveTable.Alias) +
		" ON " + moveCols.ShipmentID.EqColumn(shipmentCols.ID) +
		" AND " + moveCols.OrganizationID.EqColumn(shipmentCols.OrganizationID) +
		" AND " + moveCols.BusinessUnitID.EqColumn(shipmentCols.BusinessUnitID)

	stopFromMoveJoin = "JOIN " + buncolgen.StopTable.As(buncolgen.StopTable.Alias) +
		" ON " + stopCols.ShipmentMoveID.EqColumn(moveCols.ID) +
		" AND " + stopCols.OrganizationID.EqColumn(moveCols.OrganizationID) +
		" AND " + stopCols.BusinessUnitID.EqColumn(moveCols.BusinessUnitID)

	pickupLocationIDsExpr   = stopLocationIDsExpr(pickupLocationIDs)
	deliveryLocationIDsExpr = stopLocationIDsExpr(deliveryLocationIDs)
	originLocationIDExpr    = pickupLocationIDs + "[1] AS " + originLocationID
	destinationLocationExpr = deliveryLocationIDs + "[cardinality(" + deliveryLocationIDs +
		")] AS " + destinationLocationID
	shipmentCountSumExpr = "SUM(" + pairsAlias + "." + shipmentCount + ")::int AS " + CountColumn

	pairOrganizationID = shipmentCols.OrganizationID.WithAlias(pairsAlias)
	pairBusinessUnitID = shipmentCols.BusinessUnitID.WithAlias(pairsAlias)
	pairStatus         = shipmentCols.Status.WithAlias(pairsAlias)

	originTenantLocationJoin = locationJoin(originLocationAlias, originLocationID) +
		locationTenantCondition(originLocationAlias)
	destinationTenantLocationJoin = locationJoin(destinationLocationAlias, destinationLocationID) +
		locationTenantCondition(destinationLocationAlias)
	originPairLocationJoin = locationJoin(originLocationAlias, originLocationID) +
		locationPairCondition(originLocationAlias)
	destinationPairLocationJoin = locationJoin(destinationLocationAlias, destinationLocationID) +
		locationPairCondition(destinationLocationAlias)
	originStateJoin      = stateJoin(originStateAlias, originLocationAlias)
	destinationStateJoin = stateJoin(destinationStateAlias, destinationLocationAlias)

	originAbbreviation      = buncolgen.UsStateColumns.Abbreviation.WithAlias(originStateAlias)
	destinationAbbreviation = buncolgen.UsStateColumns.Abbreviation.WithAlias(
		destinationStateAlias,
	)
)

func New(db bun.IDB, opts Options) *bun.SelectQuery {
	endpoints := db.NewSelect().
		Model((*shipment.Shipment)(nil)).
		ColumnExpr(pickupLocationIDsExpr, bun.List(shipment.PickupStopTypes())).
		ColumnExpr(deliveryLocationIDsExpr, bun.List(shipment.DeliveryStopTypes())).
		Join(moveFromShipmentJoin).
		Join(stopFromMoveJoin).
		GroupExpr(shipmentCols.ID.Qualified())

	pairs := db.NewSelect().
		TableExpr(endpointsCTE).
		ColumnExpr(originLocationIDExpr).
		ColumnExpr(destinationLocationExpr).
		ColumnExpr(buncolgen.Count(shipmentCount)).
		Where(pickupLocationIDs + " IS NOT NULL").
		Where(deliveryLocationIDs + " IS NOT NULL").
		GroupExpr(originLocationID).
		GroupExpr(destinationLocationID)

	lanes := db.NewSelect().
		TableExpr(pairsCTE + " AS " + pairsAlias).
		ColumnExpr(originAbbreviation.As(OriginStateColumn)).
		ColumnExpr(destinationAbbreviation.As(DestinationStateColumn)).
		ColumnExpr(shipmentCountSumExpr).
		GroupExpr(originAbbreviation.Qualified()).
		GroupExpr(destinationAbbreviation.Qualified())

	if opts.Tenant != nil {
		endpoints = endpoints.Apply(buncolgen.ShipmentApplyTenant(*opts.Tenant))
		lanes = lanes.
			Join(originTenantLocationJoin, opts.Tenant.OrgID, opts.Tenant.BuID).
			Join(originStateJoin).
			Join(destinationTenantLocationJoin, opts.Tenant.OrgID, opts.Tenant.BuID).
			Join(destinationStateJoin)
	} else {
		endpoints = endpoints.
			ColumnExpr(shipmentCols.OrganizationID.Qualified()).
			ColumnExpr(shipmentCols.BusinessUnitID.Qualified()).
			GroupExpr(shipmentCols.OrganizationID.Qualified()).
			GroupExpr(shipmentCols.BusinessUnitID.Qualified())
		pairs = pairs.
			ColumnExpr(shipmentCols.OrganizationID.Bare()).
			ColumnExpr(shipmentCols.BusinessUnitID.Bare()).
			GroupExpr(shipmentCols.OrganizationID.Bare()).
			GroupExpr(shipmentCols.BusinessUnitID.Bare())
		lanes = lanes.
			Join(originPairLocationJoin).
			Join(originStateJoin).
			Join(destinationPairLocationJoin).
			Join(destinationStateJoin)
	}

	if opts.GroupByStatus {
		endpoints = endpoints.
			ColumnExpr(shipmentCols.Status.Qualified()).
			GroupExpr(shipmentCols.Status.Qualified())
		pairs = pairs.
			ColumnExpr(shipmentCols.Status.Bare()).
			GroupExpr(shipmentCols.Status.Bare())
		lanes = lanes.
			ColumnExpr(pairStatus.Qualified()).
			GroupExpr(pairStatus.Qualified())
	}

	if opts.ShipmentFilter != nil {
		endpoints = endpoints.Apply(opts.ShipmentFilter)
	}

	return lanes.
		With(endpointsCTE, endpoints).
		With(pairsCTE, pairs)
}

func stopLocationIDsExpr(label string) string {
	return buncolgen.Expr(
		"array_agg({0} ORDER BY {1}, {2}) FILTER (WHERE {3} IN (?)) AS "+label,
		stopCols.LocationID,
		moveCols.Sequence,
		stopCols.Sequence,
		stopCols.Type,
	)
}

func locationJoin(alias, pairLocationID string) string {
	return "JOIN " + buncolgen.LocationTable.As(alias) +
		" ON " + buncolgen.LocationColumns.ID.WithAlias(alias).Qualified() +
		" = " + pairsAlias + "." + pairLocationID
}

func locationTenantCondition(alias string) string {
	cols := buncolgen.LocationColumns
	return " AND " + cols.OrganizationID.WithAlias(alias).Eq() +
		" AND " + cols.BusinessUnitID.WithAlias(alias).Eq()
}

func locationPairCondition(alias string) string {
	cols := buncolgen.LocationColumns
	return " AND " + cols.OrganizationID.WithAlias(alias).EqColumn(pairOrganizationID) +
		" AND " + cols.BusinessUnitID.WithAlias(alias).EqColumn(pairBusinessUnitID)
}

func stateJoin(stateAlias, locationAlias string) string {
	return "JOIN " + buncolgen.UsStateTable.As(stateAlias) +
		" ON " + buncolgen.UsStateColumns.ID.WithAlias(stateAlias).EqColumn(
		buncolgen.LocationColumns.StateID.WithAlias(locationAlias),
	)
}
