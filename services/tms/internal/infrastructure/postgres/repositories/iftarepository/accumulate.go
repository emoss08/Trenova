package iftarepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	kilometresPerMile   = "1.609344"
	milesScale          = "3"
	mismatchFloorMiles  = "1"
	mismatchTolerance   = "0.02"
	defaultRouteRowSize = 64
)

type tenantCols struct {
	org, bu buncolgen.Column
}

type moveMiles struct {
	MoveID pulid.ID        `bun:"move_id"`
	Miles  decimal.Decimal `bun:"miles"`
}

var (
	smCols   = buncolgen.ShipmentMoveColumns
	stpCols  = buncolgen.StopColumns
	jmCols   = buncolgen.ShipmentMoveJurisdictionMileColumns
	aCols    = buncolgen.AssignmentColumns
	ifmeCols = buncolgen.JurisdictionMileageEntryColumns
	ifjCols  = buncolgen.JurisdictionColumns

	accumulationPrefix = buildAccumulationPrefix()
	routeRowsSQL       = accumulationPrefix + buildRouteRowsBody()
	noTractorSQL       = accumulationPrefix + buildNoTractorBody()
	unattributedSQL    = accumulationPrefix + buildUnattributedBody()
	mismatchSQL        = accumulationPrefix + buildMismatchBody()
)

func normalisedMiles(distance, units buncolgen.Column) string {
	return "CASE WHEN " + units.Qualified() + " = ? THEN " +
		distance.Qualified() + " / " + kilometresPerMile +
		" ELSE " + distance.Qualified() + " END"
}

func rounded(expr string) string {
	return "ROUND((" + expr + ")::numeric, " + milesScale + ")"
}

func tenantJoin(left, right tenantCols) string {
	return left.org.EqColumn(right.org) + " AND " + left.bu.EqColumn(right.bu)
}

func buildAccumulationPrefix() string {
	inpID := buncolgen.NewColumn(smCols.ID.Name, "inp")
	inpOrg := buncolgen.NewColumn(smCols.OrganizationID.Name, "inp")
	inpBu := buncolgen.NewColumn(smCols.BusinessUnitID.Name, "inp")
	inpLoaded := buncolgen.NewColumn(smCols.Loaded.Name, "inp")
	inpDistance := buncolgen.NewColumn(smCols.Distance.Name, "inp")
	mcCompleted := buncolgen.NewColumn("completed_at", "mc")
	ovMove := buncolgen.NewColumn(ifmeCols.ShipmentMoveID.Name, "o")

	return "WITH mc AS (" +
		" SELECT " + smCols.ID.Qualified() + ", " + smCols.OrganizationID.Qualified() + ", " +
		smCols.BusinessUnitID.Qualified() + ", " + smCols.Loaded.Qualified() + ", " +
		normalisedMiles(smCols.Distance, smCols.DistanceUnits) + " AS " + smCols.Distance.Name + ", " +
		"COALESCE(MAX(COALESCE(" + stpCols.ActualDeparture.Qualified() + ", " +
		stpCols.ActualArrival.Qualified() + ")), " + smCols.UpdatedAt.Qualified() + ") AS completed_at" +
		" FROM " + buncolgen.ShipmentMoveTable.As(buncolgen.ShipmentMoveTable.Alias) +
		" LEFT JOIN " + buncolgen.StopTable.As(buncolgen.StopTable.Alias) +
		" ON " + stpCols.ShipmentMoveID.EqColumn(smCols.ID) +
		" AND " + tenantJoin(
		tenantCols{stpCols.OrganizationID, stpCols.BusinessUnitID},
		tenantCols{smCols.OrganizationID, smCols.BusinessUnitID},
	) +
		" WHERE " + smCols.OrganizationID.Eq() + " AND " + smCols.BusinessUnitID.Eq() +
		" AND " + smCols.Status.Eq() +
		" GROUP BY " + smCols.ID.Qualified() + ", " + smCols.OrganizationID.Qualified() + ", " +
		smCols.BusinessUnitID.Qualified() + ", " + smCols.Loaded.Qualified() + ", " +
		smCols.Distance.Qualified() + ", " + smCols.DistanceUnits.Qualified() + ", " +
		smCols.UpdatedAt.Qualified() +
		"), inp AS (" +
		" SELECT mc.* FROM mc WHERE " + mcCompleted.Gte() + " AND " + mcCompleted.Lt() +
		"), overridden AS (" +
		" SELECT DISTINCT " + ifmeCols.ShipmentMoveID.Qualified() +
		" FROM " + buncolgen.JurisdictionMileageEntryTable.As(
		buncolgen.JurisdictionMileageEntryTable.Alias,
	) +
		" WHERE " + ifmeCols.OrganizationID.Eq() + " AND " + ifmeCols.BusinessUnitID.Eq() +
		" AND " + ifmeCols.ShipmentMoveID.IsNotNull() +
		"), jm AS (" +
		" SELECT " + jmCols.ShipmentMoveID.Qualified() + ", " + jmCols.OrganizationID.Qualified() + ", " +
		jmCols.BusinessUnitID.Qualified() + ", " + jmCols.CountryCode.Qualified() + ", " +
		jmCols.JurisdictionCode.Qualified() + ", " + inpLoaded.Qualified() + " AS loaded, " +
		inpDistance.Qualified() + " AS move_distance, " +
		normalisedMiles(jmCols.Distance, jmCols.DistanceUnits) + " AS miles" +
		" FROM " + buncolgen.ShipmentMoveJurisdictionMileTable.As(
		buncolgen.ShipmentMoveJurisdictionMileTable.Alias,
	) +
		" JOIN inp ON " + inpID.EqColumn(jmCols.ShipmentMoveID) +
		" AND " + tenantJoin(
		tenantCols{inpOrg, inpBu},
		tenantCols{jmCols.OrganizationID, jmCols.BusinessUnitID},
	) +
		" WHERE " + jmCols.OrganizationID.Eq() + " AND " + jmCols.BusinessUnitID.Eq() +
		" AND NOT EXISTS (SELECT 1 FROM overridden o WHERE " +
		ovMove.EqColumn(jmCols.ShipmentMoveID) + ")" +
		") "
}

func accumulationArgs(req *repositories.AccumulateMilesRequest) []any {
	return []any{
		shipment.JurisdictionDistanceUnitsKilometers,
		req.TenantInfo.OrgID, req.TenantInfo.BuID, shipment.MoveStatusCompleted,
		req.Start, req.End,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
		shipment.JurisdictionDistanceUnitsKilometers,
		req.TenantInfo.OrgID, req.TenantInfo.BuID,
	}
}

func buildRouteRowsBody() string {
	jmMove := buncolgen.NewColumn(jmCols.ShipmentMoveID.Name, "jm")
	jmOrg := buncolgen.NewColumn(jmCols.OrganizationID.Name, "jm")
	jmBu := buncolgen.NewColumn(jmCols.BusinessUnitID.Name, "jm")
	jmCountry := buncolgen.NewColumn(jmCols.CountryCode.Name, "jm")
	jmCode := buncolgen.NewColumn(jmCols.JurisdictionCode.Name, "jm")
	jmLoaded := buncolgen.NewColumn("loaded", "jm")
	jmMiles := buncolgen.NewColumn("miles", "jm")

	return "SELECT " + aCols.TractorID.As("tractor_id") + ", " +
		jmCountry.As("country_code") + ", " + jmCode.As("jurisdiction_code") + ", " +
		ifjCols.ID.As("jurisdiction_id") + ", " +
		rounded("COALESCE(SUM("+jmMiles.Qualified()+"), 0)") + " AS miles, " +
		rounded("COALESCE(SUM("+jmMiles.Qualified()+") FILTER (WHERE "+jmLoaded.Qualified()+"), 0)") +
		" AS loaded_miles, " +
		rounded("COALESCE(SUM("+jmMiles.Qualified()+") FILTER (WHERE NOT "+jmLoaded.Qualified()+"), 0)") +
		" AS empty_miles, " +
		buncolgen.CountDistinct(jmMove, "move_count") +
		" FROM jm" +
		" LEFT JOIN " + buncolgen.AssignmentTable.As(buncolgen.AssignmentTable.Alias) +
		" ON " + aCols.ShipmentMoveID.EqColumn(jmMove) +
		" AND " + tenantJoin(
		tenantCols{aCols.OrganizationID, aCols.BusinessUnitID},
		tenantCols{jmOrg, jmBu},
	) +
		" AND " + aCols.ArchivedAt.IsNull() +
		" LEFT JOIN " + buncolgen.JurisdictionTable.As(buncolgen.JurisdictionTable.Alias) +
		" ON " + ifjCols.CountryCode.EqColumn(jmCountry) +
		" AND " + ifjCols.Code.EqColumn(jmCode) +
		" GROUP BY " + aCols.TractorID.Qualified() + ", " + jmCountry.Qualified() + ", " +
		jmCode.Qualified() + ", " + ifjCols.ID.Qualified() +
		" ORDER BY " + aCols.TractorID.Qualified() + " NULLS FIRST, " +
		jmCountry.Qualified() + ", " + jmCode.Qualified()
}

func buildNoTractorBody() string {
	jmMove := buncolgen.NewColumn(jmCols.ShipmentMoveID.Name, "jm")
	jmOrg := buncolgen.NewColumn(jmCols.OrganizationID.Name, "jm")
	jmBu := buncolgen.NewColumn(jmCols.BusinessUnitID.Name, "jm")
	jmMiles := buncolgen.NewColumn("miles", "jm")

	return "SELECT " + jmMove.As("move_id") + ", " +
		rounded("COALESCE(SUM("+jmMiles.Qualified()+"), 0)") + " AS miles" +
		" FROM jm" +
		" LEFT JOIN " + buncolgen.AssignmentTable.As(buncolgen.AssignmentTable.Alias) +
		" ON " + aCols.ShipmentMoveID.EqColumn(jmMove) +
		" AND " + tenantJoin(
		tenantCols{aCols.OrganizationID, aCols.BusinessUnitID},
		tenantCols{jmOrg, jmBu},
	) +
		" AND " + aCols.ArchivedAt.IsNull() +
		" WHERE " + aCols.TractorID.IsNull() +
		" GROUP BY " + jmMove.Qualified() +
		" ORDER BY " + jmMove.Qualified()
}

func buildUnattributedBody() string {
	inpID := buncolgen.NewColumn(smCols.ID.Name, "inp")
	inpOrg := buncolgen.NewColumn(smCols.OrganizationID.Name, "inp")
	inpBu := buncolgen.NewColumn(smCols.BusinessUnitID.Name, "inp")
	inpDistance := buncolgen.NewColumn(smCols.Distance.Name, "inp")
	inpCompleted := buncolgen.NewColumn("completed_at", "inp")
	ovMove := buncolgen.NewColumn(ifmeCols.ShipmentMoveID.Name, "o")

	return "SELECT " + inpID.As("move_id") + ", " +
		rounded(inpDistance.Qualified()) + " AS miles" +
		" FROM inp" +
		" WHERE " + inpDistance.Qualified() + " > 0" +
		" AND NOT EXISTS (SELECT 1 FROM " +
		buncolgen.ShipmentMoveJurisdictionMileTable.As(
			buncolgen.ShipmentMoveJurisdictionMileTable.Alias,
		) +
		" WHERE " + jmCols.ShipmentMoveID.EqColumn(inpID) +
		" AND " + tenantJoin(
		tenantCols{jmCols.OrganizationID, jmCols.BusinessUnitID},
		tenantCols{inpOrg, inpBu},
	) + ")" +
		" AND NOT EXISTS (SELECT 1 FROM overridden o WHERE " + ovMove.EqColumn(inpID) + ")" +
		" ORDER BY " + inpCompleted.Qualified() + ", " + inpID.Qualified()
}

func buildMismatchBody() string {
	jmMove := buncolgen.NewColumn(jmCols.ShipmentMoveID.Name, "jm")
	jmMiles := buncolgen.NewColumn("miles", "jm")
	jmDistance := buncolgen.NewColumn("move_distance", "jm")
	sum := "SUM(" + jmMiles.Qualified() + ")"

	return "SELECT " + jmMove.As("move_id") + ", " +
		rounded(jmDistance.Qualified()) + " AS move_distance, " +
		rounded(sum) + " AS attributed_miles" +
		" FROM jm" +
		" WHERE " + jmDistance.IsNotNull() +
		" GROUP BY " + jmMove.Qualified() + ", " + jmDistance.Qualified() +
		" HAVING ABS(" + sum + " - " + jmDistance.Qualified() + ") > GREATEST(" +
		mismatchFloorMiles + ", " + jmDistance.Qualified() + " * " + mismatchTolerance + ")" +
		" ORDER BY " + jmMove.Qualified()
}

func (r *repository) AccumulateMiles(
	ctx context.Context,
	req *repositories.AccumulateMilesRequest,
) (*repositories.MileAccumulation, error) {
	dba := r.db.DBForContext(ctx)
	args := accumulationArgs(req)
	out := &repositories.MileAccumulation{
		RouteRows:  make([]*repositories.MileRow, 0, defaultRouteRowSize),
		ManualRows: make([]*repositories.MileRow, 0, defaultRouteRowSize),
		Mismatches: make([]repositories.MoveMismatch, 0),
	}

	if err := dba.NewRaw(routeRowsSQL, args...).Scan(ctx, &out.RouteRows); err != nil {
		r.l.Error("failed to accumulate route miles", zap.Error(err))
		return nil, fmt.Errorf("accumulate route miles: %w", err)
	}

	noTractor := make([]moveMiles, 0)
	if err := dba.NewRaw(noTractorSQL, args...).Scan(ctx, &noTractor); err != nil {
		r.l.Error("failed to accumulate no-tractor miles", zap.Error(err))
		return nil, fmt.Errorf("accumulate no-tractor miles: %w", err)
	}
	for i := range noTractor {
		out.NoTractor.Add(noTractor[i].MoveID, noTractor[i].Miles)
	}

	unattributed := make([]moveMiles, 0)
	if err := dba.NewRaw(unattributedSQL, args...).Scan(ctx, &unattributed); err != nil {
		r.l.Error("failed to list unattributed moves", zap.Error(err))
		return nil, fmt.Errorf("list unattributed moves: %w", err)
	}
	for i := range unattributed {
		out.Unattributed.Add(unattributed[i].MoveID, unattributed[i].Miles)
	}

	if err := dba.NewRaw(mismatchSQL, args...).Scan(ctx, &out.Mismatches); err != nil {
		r.l.Error("failed to detect jurisdiction mileage mismatches", zap.Error(err))
		return nil, fmt.Errorf("detect mileage mismatches: %w", err)
	}

	manual, err := r.manualRows(ctx, dba, req)
	if err != nil {
		return nil, err
	}
	out.ManualRows = manual

	return out, nil
}

func (r *repository) manualRows(
	ctx context.Context,
	dba bun.IDB,
	req *repositories.AccumulateMilesRequest,
) ([]*repositories.MileRow, error) {
	rows := make([]*repositories.MileRow, 0, defaultRouteRowSize)
	err := dba.NewSelect().
		Model((*ifta.JurisdictionMileageEntry)(nil)).
		ColumnExpr(ifmeCols.TractorID.As("tractor_id")).
		ColumnExpr(ifjCols.CountryCode.As("country_code")).
		ColumnExpr(ifjCols.Code.As("jurisdiction_code")).
		ColumnExpr(ifmeCols.JurisdictionID.As("jurisdiction_id")).
		ColumnExpr(ifmeCols.Miles.Expr("COALESCE(SUM({}), 0) AS miles")).
		ColumnExpr(buncolgen.Expr(
			"COALESCE(SUM({0}) FILTER (WHERE {1}), 0) AS loaded_miles",
			ifmeCols.Miles, ifmeCols.Loaded,
		)).
		ColumnExpr(buncolgen.Expr(
			"COALESCE(SUM({0}) FILTER (WHERE NOT {1}), 0) AS empty_miles",
			ifmeCols.Miles, ifmeCols.Loaded,
		)).
		ColumnExpr(buncolgen.Count("move_count")).
		Join("JOIN "+buncolgen.JurisdictionTable.As(buncolgen.JurisdictionTable.Alias)+
			" ON "+ifjCols.ID.EqColumn(ifmeCols.JurisdictionID)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.JurisdictionMileageEntryScopeTenant(sq, req.TenantInfo).
				Where(ifmeCols.Year.Eq(), req.Year).
				Where(ifmeCols.Quarter.Eq(), req.Quarter)
		}).
		GroupExpr(ifmeCols.TractorID.Qualified()).
		GroupExpr(ifjCols.CountryCode.Qualified()).
		GroupExpr(ifjCols.Code.Qualified()).
		GroupExpr(ifmeCols.JurisdictionID.Qualified()).
		Order(ifmeCols.TractorID.OrderAsc()).
		Order(ifjCols.CountryCode.OrderAsc()).
		Order(ifjCols.Code.OrderAsc()).
		Scan(ctx, &rows)
	if err != nil {
		r.l.Error("failed to accumulate manual miles", zap.Error(err))
		return nil, fmt.Errorf("accumulate manual miles: %w", err)
	}

	return rows, nil
}
