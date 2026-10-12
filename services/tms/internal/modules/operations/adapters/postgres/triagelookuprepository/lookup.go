package triagelookuprepository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/commoditycol"
	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/equipmenttypecol"
	"github.com/emoss08/trenova/pkg/buncolgen/holdreasoncol"
	"github.com/emoss08/trenova/pkg/buncolgen/locationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/servicetypecol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/usstatecol"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) shipmentports.TriageLookupRepository {
	return &repository{db: p.DB}
}

func tenantJoin(right, left buncolgen.Column, rightOrg, leftOrg, rightBU, leftBU buncolgen.Column) string {
	return right.EqColumn(left) + " AND " + rightOrg.EqColumn(leftOrg) + " AND " + rightBU.EqColumn(leftBU)
}

func prefixPattern(text string) string {
	return stringutils.EscapeLikePattern(text) + "%"
}

func containsPattern(text string) string {
	return "%" + stringutils.EscapeLikePattern(text) + "%"
}

func (r *repository) PeekDetails(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) (map[pulid.ID]*shipmentports.TriagePeekDetail, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]*shipmentports.TriagePeekDetail, error) {
		details := make(map[pulid.ID]*shipmentports.TriagePeekDetail, len(shipmentIDs))
		if len(shipmentIDs) == 0 {
			return details, nil
		}

		sp := shipmentcol.ShipmentColumns
		cus := customercol.CustomerColumns
		et := equipmenttypecol.EquipmentTypeColumns
		rows := make([]*shipmentports.TriagePeekDetail, 0, len(shipmentIDs))
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*shipment.Shipment)(nil)).
			ColumnExpr(sp.ID.As("shipment_id")).
			ColumnExpr(sp.Status.As("status")).
			ColumnExpr(sp.ActualDeliveryDate.As("actual_delivery_date")).
			ColumnExpr(buncolgen.Expr("COALESCE({0}, '')", cus.Name) + " AS customer_name").
			ColumnExpr(buncolgen.Expr("COALESCE(NULLIF({0}, ''), {1}, '')", et.Description, et.Code) +
				" AS trailer_type").
			Join("LEFT JOIN "+customercol.CustomerTable.Name+" AS "+customercol.CustomerTable.Alias+
				" ON "+tenantJoin(cus.ID, sp.CustomerID, cus.OrganizationID, sp.OrganizationID,
				cus.BusinessUnitID, sp.BusinessUnitID)).
			Join("LEFT JOIN "+equipmenttypecol.EquipmentTypeTable.Name+" AS "+
				equipmenttypecol.EquipmentTypeTable.Alias+
				" ON "+tenantJoin(et.ID, sp.TrailerTypeID, et.OrganizationID, sp.OrganizationID,
				et.BusinessUnitID, sp.BusinessUnitID)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return shipmentcol.ShipmentScopeTenant(sq, tenantInfo).
					Where(sp.ID.In(), bun.List(shipmentIDs))
			}).
			Scan(ctx, &rows); err != nil {
			return nil, err
		}

		for _, row := range rows {
			details[row.ShipmentID] = row
		}

		return details, nil
	})
}

func (r *repository) CustomerShipments(
	ctx context.Context,
	req *shipmentports.TriageCustomerShipmentsQuery,
) ([]pulid.ID, int, error) {
	return dbtx.Read2(ctx, r.db, func(ctx context.Context) ([]pulid.ID, int, error) {
		sp := shipmentcol.ShipmentColumns
		db := r.db.DBForContext(ctx)
		scope := func(sq *bun.SelectQuery) *bun.SelectQuery {
			return shipmentcol.ShipmentScopeTenant(sq, req.TenantInfo).
				Where(sp.CustomerID.Eq(), req.CustomerID).
				Where(sp.Status.Ne(), shipment.StatusCanceled)
		}

		ids := make([]pulid.ID, 0, req.Limit)
		if err := db.NewSelect().
			Model((*shipment.Shipment)(nil)).
			Column(sp.ID.Bare()).
			WhereGroup(" AND ", scope).
			Order(sp.CreatedAt.OrderDesc(), sp.ID.OrderDesc()).
			Limit(int64(req.Limit)).
			Scan(ctx, &ids); err != nil {
			return nil, 0, err
		}

		thisWeek, err := db.NewSelect().
			Model((*shipment.Shipment)(nil)).
			WhereGroup(" AND ", scope).
			Where(sp.CreatedAt.Gte(), req.WeekStart).
			Count(ctx)
		if err != nil {
			return nil, 0, err
		}

		return ids, int(thisWeek), nil
	})
}

func (r *repository) CustomerDefaults(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) (*shipmentports.TriageShipmentDefaults, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipmentports.TriageShipmentDefaults, error) {
		sp := shipmentcol.ShipmentColumns
		defaults := new(shipmentports.TriageShipmentDefaults)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*shipment.Shipment)(nil)).
			Column(
				sp.ServiceTypeID.Bare(),
				sp.ShipmentTypeID.Bare(),
				sp.FormulaTemplateID.Bare(),
				sp.TractorTypeID.Bare(),
				sp.TrailerTypeID.Bare(),
			).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return shipmentcol.ShipmentScopeTenant(sq, tenantInfo).
					Where(sp.CustomerID.Eq(), customerID).
					Where(sp.Status.Ne(), shipment.StatusCanceled)
			}).
			Order(sp.CreatedAt.OrderDesc()).
			Limit(1).
			Scan(ctx, defaults)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}

		return defaults, nil
	})
}

func (r *repository) ActiveServiceTypes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		cols := servicetypecol.ServiceTypeColumns
		ids := make([]pulid.ID, 0, limit)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*servicetype.ServiceType)(nil)).
			Column(cols.ID.Bare()).
			Where(cols.Status.Eq(), domaintypes.StatusActive).
			Apply(servicetypecol.ServiceTypeApplyTenant(tenantInfo)).
			Order(cols.ID.OrderAsc()).
			Limit(int64(limit)).
			Scan(ctx, &ids); err != nil {
			return nil, err
		}

		return ids, nil
	})
}

func (r *repository) FindCustomers(
	ctx context.Context,
	req *shipmentports.TriageLookupQuery,
) ([]shipmentports.TriageRecordMatch, error) {
	text := strings.TrimSpace(req.Text)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]shipmentports.TriageRecordMatch, error) {
		cols := customercol.CustomerColumns
		matches := make([]shipmentports.TriageRecordMatch, 0, req.Limit)
		exact := "(" + cols.Code.Expr("lower({})") + " = lower(?) OR " +
			cols.Name.Expr("lower({})") + " = lower(?))"
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*customer.Customer)(nil)).
			ColumnExpr(cols.ID.As("id")).
			ColumnExpr(cols.Name.As("label")).
			ColumnExpr(exact+" AS exact", text, text).
			Where(cols.Status.Eq(), domaintypes.StatusActive).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where(exact, text, text).
					WhereOr(cols.Name.ILike(), prefixPattern(text))
			}).
			Apply(customercol.CustomerApplyTenant(req.TenantInfo)).
			OrderExpr("exact DESC").
			Order(cols.Name.OrderAsc()).
			Limit(int64(req.Limit)).
			Scan(ctx, &matches); err != nil {
			return nil, err
		}

		return matches, nil
	})
}

func (r *repository) FindLocations(
	ctx context.Context,
	req *shipmentports.TriageLocationQuery,
) ([]shipmentports.TriageLocationMatch, error) {
	name := strings.TrimSpace(req.Name)
	city := strings.TrimSpace(req.City)
	state := strings.TrimSpace(req.State)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]shipmentports.TriageLocationMatch, error) {
		loc := locationcol.LocationColumns
		st := usstatecol.UsStateColumns
		matches := make([]shipmentports.TriageLocationMatch, 0, req.Limit)

		nameExact := "(" + loc.Code.Expr("lower({})") + " = lower(?) OR " +
			loc.Name.Expr("lower({})") + " = lower(?))"
		query := r.db.DBForContext(ctx).
			NewSelect().
			Model((*location.Location)(nil)).
			ColumnExpr(loc.ID.As("id")).
			ColumnExpr(buncolgen.Expr("concat_ws(' · ', {0}, concat_ws(', ', {1}, {2}))",
				loc.Name, loc.City, st.Abbreviation) + " AS label").
			ColumnExpr(buncolgen.Expr("COALESCE({0}, '')", loc.Timezone) + " AS timezone").
			Join("LEFT JOIN "+usstatecol.UsStateTable.Name+" AS "+usstatecol.UsStateTable.Alias+
				" ON "+st.ID.EqColumn(loc.StateID)).
			Where(loc.Status.Eq(), domaintypes.StatusActive).
			Apply(locationcol.LocationApplyTenant(req.TenantInfo))

		switch {
		case name != "":
			query = query.ColumnExpr(nameExact+" AS exact", name, name).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Where(nameExact, name, name).
						WhereOr(loc.Name.ILike(), prefixPattern(name))
				})
		case city != "":
			query = query.ColumnExpr("TRUE AS exact").
				Where(loc.City.Expr("lower({})")+" = lower(?)", city)
		default:
			return matches, nil
		}
		if city != "" && name != "" {
			query = query.Where(loc.City.Expr("lower({})")+" = lower(?)", city)
		}
		if state != "" {
			query = query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where(st.Abbreviation.Expr("upper({})")+" = upper(?)", state).
					WhereOr(st.Name.Expr("lower({})")+" = lower(?)", state)
			})
		}

		if err := query.
			OrderExpr("exact DESC").
			Order(loc.Name.OrderAsc()).
			Limit(int64(req.Limit)).
			Scan(ctx, &matches); err != nil {
			return nil, err
		}

		return matches, nil
	})
}

func (r *repository) FindTrailerTypes(
	ctx context.Context,
	req *shipmentports.TriageLookupQuery,
) ([]shipmentports.TriageRecordMatch, error) {
	text := strings.TrimSpace(req.Text)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]shipmentports.TriageRecordMatch, error) {
		cols := equipmenttypecol.EquipmentTypeColumns
		matches := make([]shipmentports.TriageRecordMatch, 0, req.Limit)
		exact := "(" + cols.Code.Expr("lower({})") + " = lower(?) OR " +
			cols.Description.Expr("lower({})") + " = lower(?))"
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*equipmenttype.EquipmentType)(nil)).
			ColumnExpr(cols.ID.As("id")).
			ColumnExpr(buncolgen.Expr("COALESCE(NULLIF({0}, ''), {1})", cols.Description, cols.Code) +
				" AS label").
			ColumnExpr(exact+" AS exact", text, text).
			Where(cols.Status.Eq(), domaintypes.StatusActive).
			Where(cols.Class.In(), bun.List([]equipmenttype.Class{
				equipmenttype.ClassTrailer,
				equipmenttype.ClassContainer,
			})).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where(exact, text, text).
					WhereOr(cols.Description.ILike(), containsPattern(text))
			}).
			Apply(equipmenttypecol.EquipmentTypeApplyTenant(req.TenantInfo)).
			OrderExpr("exact DESC").
			Order(cols.Code.OrderAsc()).
			Limit(int64(req.Limit)).
			Scan(ctx, &matches); err != nil {
			return nil, err
		}

		return matches, nil
	})
}

func (r *repository) FindCommodities(
	ctx context.Context,
	req *shipmentports.TriageLookupQuery,
) ([]shipmentports.TriageRecordMatch, error) {
	text := strings.TrimSpace(req.Text)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]shipmentports.TriageRecordMatch, error) {
		cols := commoditycol.CommodityColumns
		matches := make([]shipmentports.TriageRecordMatch, 0, req.Limit)
		exact := cols.Name.Expr("lower({})") + " = lower(?)"
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*commodity.Commodity)(nil)).
			ColumnExpr(cols.ID.As("id")).
			ColumnExpr(cols.Name.As("label")).
			ColumnExpr("("+exact+") AS exact", text).
			Where(cols.Status.Eq(), domaintypes.StatusActive).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where(exact, text).
					WhereOr(cols.Name.ILike(), prefixPattern(text))
			}).
			Apply(commoditycol.CommodityApplyTenant(req.TenantInfo)).
			OrderExpr("exact DESC").
			Order(cols.Name.OrderAsc()).
			Limit(int64(req.Limit)).
			Scan(ctx, &matches); err != nil {
			return nil, err
		}

		return matches, nil
	})
}

func (r *repository) HoldReasonByCode(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	code string,
) (pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (pulid.ID, error) {
		cols := holdreasoncol.HoldReasonColumns
		var id pulid.ID
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*holdreason.HoldReason)(nil)).
			Column(cols.ID.Bare()).
			Where(cols.Code.Expr("upper({})")+" = upper(?)", code).
			Where(cols.Active.IsTrue()).
			Apply(holdreasoncol.HoldReasonApplyTenant(tenantInfo)).
			Limit(1).
			Scan(ctx, &id)
		if errors.Is(err, sql.ErrNoRows) {
			return pulid.Nil, nil
		}

		return id, err
	})
}
