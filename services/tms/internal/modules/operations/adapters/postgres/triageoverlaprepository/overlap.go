package triageoverlaprepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/buncolgen/locationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/usstatecol"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

const maxLoadsChecked = 20

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) triageports.TenderOverlapFinder {
	return &repository{db: p.DB}
}

func (r *repository) FindOverlappingLoads(
	ctx context.Context,
	req triageports.FindOverlappingLoadsRequest,
) ([]triageports.OverlappingLoad, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]triageports.OverlappingLoad, error) {
		found := make([]triageports.OverlappingLoad, 0, len(req.Loads))
		if req.CustomerID.IsNil() {
			return found, nil
		}

		db := r.db.DBForContext(ctx)
		for i, load := range req.Loads {
			if i >= maxLoadsChecked {
				break
			}
			if !checkable(load) {
				continue
			}

			ids := make([]pulid.ID, 0, 1)
			if err := overlapQuery(db, req.TenantInfo, req.CustomerID, load).
				Scan(ctx, &ids); err != nil {
				return nil, err
			}
			if len(ids) > 0 {
				found = append(found, triageports.OverlappingLoad{
					LoadIndex:  load.Index,
					ShipmentID: ids[0],
				})
			}
		}

		return found, nil
	})
}

func checkable(load triageports.TenderLoad) bool {
	return load.OriginCity != "" && load.OriginState != "" &&
		load.DestinationCity != "" && load.DestinationState != "" &&
		load.PickupFrom > 0 && load.PickupTo > load.PickupFrom
}

func overlapQuery(
	db bun.IDB,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
	load triageports.TenderLoad,
) *bun.SelectQuery {
	cols := shipmentcol.ShipmentColumns

	pickup := stopAt(db, tenantInfo, stopAtRequest{
		types: []shipment.StopType{shipment.StopTypePickup, shipment.StopTypeSplitPickup},
		city:  load.OriginCity,
		state: load.OriginState,
		from:  load.PickupFrom,
		to:    load.PickupTo,
	})
	delivery := stopAt(db, tenantInfo, stopAtRequest{
		types: []shipment.StopType{shipment.StopTypeDelivery, shipment.StopTypeSplitDelivery},
		city:  load.DestinationCity,
		state: load.DestinationState,
	})

	return db.NewSelect().
		Model((*shipment.Shipment)(nil)).
		ColumnExpr(cols.ID.Qualified()).
		Apply(shipmentcol.ShipmentApplyTenant(tenantInfo)).
		Where(cols.CustomerID.Eq(), customerID).
		Where(cols.Status.NotEq(), shipment.StatusCanceled).
		Where("EXISTS (?)", pickup).
		Where("EXISTS (?)", delivery).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(1)
}

type stopAtRequest struct {
	types []shipment.StopType
	city  string
	state string
	from  int64
	to    int64
}

func stopAt(
	db bun.IDB,
	tenantInfo pagination.TenantInfo,
	req stopAtRequest,
) *bun.SelectQuery {
	ship := shipmentcol.ShipmentColumns
	move := shipmentcol.ShipmentMoveColumns
	stop := shipmentcol.StopColumns
	loc := locationcol.LocationColumns
	state := usstatecol.UsStateColumns

	query := db.NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		ColumnExpr("1").
		Join("JOIN "+shipmentcol.StopTable.Name+" AS "+shipmentcol.StopTable.Alias).
		JoinOn(stop.ShipmentMoveID.EqColumn(move.ID)).
		JoinOn(stop.OrganizationID.EqColumn(move.OrganizationID)).
		JoinOn(stop.BusinessUnitID.EqColumn(move.BusinessUnitID)).
		Join("JOIN "+locationcol.LocationTable.Name+" AS "+locationcol.LocationTable.Alias).
		JoinOn(loc.ID.EqColumn(stop.LocationID)).
		JoinOn(loc.OrganizationID.EqColumn(stop.OrganizationID)).
		JoinOn(loc.BusinessUnitID.EqColumn(stop.BusinessUnitID)).
		Join("JOIN "+usstatecol.UsStateTable.Name+" AS "+usstatecol.UsStateTable.Alias).
		JoinOn(state.ID.EqColumn(loc.StateID)).
		Where(move.ShipmentID.EqColumn(ship.ID)).
		Where(move.OrganizationID.Eq(), tenantInfo.OrgID).
		Where(move.BusinessUnitID.Eq(), tenantInfo.BuID).
		Where(stop.Type.In(), bun.In(req.types)).
		Where(stop.Status.NotEq(), shipment.StopStatusCanceled).
		Where(loc.City.Expr("LOWER(BTRIM({})) = ?"), strings.ToLower(strings.TrimSpace(req.city))).
		Where(state.Abbreviation.Expr("UPPER({}) = ?"), strings.ToUpper(req.state))
	if req.from > 0 && req.to > req.from {
		query = query.
			Where(stop.ScheduledWindowStart.Gte(), req.from).
			Where(stop.ScheduledWindowStart.Lt(), req.to)
	}

	return query
}
