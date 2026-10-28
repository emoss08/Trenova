package triagelookuprepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype/equipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/triagelookupports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason/holdreasondb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) triagelookupports.TriageLookupRepository {
	return &repository{db: p.DB}
}

func text(e keel.Expr) keel.StringExpr {
	return keel.StringOf(e).Coalesce("")
}

func lowerEq(col keel.StringCol, value string) keel.Pred {
	return col.Lower().EqExpr(keel.Text(value).Lower())
}

func (r *repository) PeekDetails(
	ctx context.Context,
	_ pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) (map[pulid.ID]*triagelookupports.TriagePeekDetail, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*triagelookupports.TriagePeekDetail, error) {
			details := make(map[pulid.ID]*triagelookupports.TriagePeekDetail, len(shipmentIDs))
			if len(shipmentIDs) == 0 {
				return details, nil
			}

			sp := shipmentdb.Shipments
			cus := customerdb.Customers
			et := equipmenttypedb.EquipmentTypes
			rows, err := keel.Into[triagelookupports.TriagePeekDetail](ctx, r.db.Keel(ctx), sp.
				LeftJoin(cus, cus.ID.EqCol(sp.CustomerID)).
				LeftJoin(et, et.ID.EqCol(sp.TrailerTypeID)).
				Pick(
					sp.ID,
					sp.Status,
					text(cus.Name),
					et.Description.NullIf("").Coalesce(et.Code, ""),
					sp.ActualDeliveryDate,
				).
				Where(sp.ID.In(shipmentIDs...)))
			if err != nil {
				return nil, err
			}

			for i := range rows {
				details[rows[i].ShipmentID] = &rows[i]
			}

			return details, nil
		},
	)
}

func (r *repository) CustomerShipments(
	ctx context.Context,
	req *triagelookupports.TriageCustomerShipmentsQuery,
) ([]pulid.ID, int, error) {
	return dbtx.Read2(ctx, r.db, func(ctx context.Context) ([]pulid.ID, int, error) {
		sp := shipmentdb.Shipments
		scope := sp.Where(sp.CustomerID.Eq(req.CustomerID), sp.Status.Ne(shipment.StatusCanceled))

		var ids []pulid.ID
		var thisWeek int64
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoSlice(scope.Clone().
				Pick(sp.ID).
				Order(sp.CreatedAt.Desc(), sp.ID.Desc()).
				Limit(req.Limit), &ids),
			keel.CountInto(scope.Where(sp.CreatedAt.Gte(req.WeekStart)), &thisWeek),
		)
		if err != nil {
			return nil, 0, err
		}

		return ids, int(thisWeek), nil
	})
}

func (r *repository) CustomerDefaults(
	ctx context.Context,
	_ pagination.TenantInfo,
	customerID pulid.ID,
) (*triagelookupports.TriageShipmentDefaults, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*triagelookupports.TriageShipmentDefaults, error) {
			sp := shipmentdb.Shipments
			defaults, err := keel.IntoOne[triagelookupports.TriageShipmentDefaults](
				ctx,
				r.db.Keel(ctx),
				sp.Pick(
					text(sp.ServiceTypeID),
					text(sp.ShipmentTypeID),
					text(sp.FormulaTemplateID),
					text(sp.TractorTypeID),
					text(sp.TrailerTypeID),
				).
					Where(sp.CustomerID.Eq(customerID), sp.Status.Ne(shipment.StatusCanceled)).
					Order(sp.CreatedAt.Desc()).
					Limit(1),
			)
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			if err != nil {
				return nil, err
			}

			return &defaults, nil
		},
	)
}

func (r *repository) ActiveServiceTypes(
	ctx context.Context,
	_ pagination.TenantInfo,
	limit int,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		st := servicetypedb.ServiceTypes
		return keel.Into[pulid.ID](ctx, r.db.Keel(ctx), st.
			Pick(st.ID).
			Where(st.Status.Eq(domaintypes.StatusActive)).
			Order(st.ID.Asc()).
			Limit(limit))
	})
}

func (r *repository) FindCustomers(
	ctx context.Context,
	req *triagelookupports.TriageLookupQuery,
) ([]triagelookupports.TriageRecordMatch, error) {
	query := strings.TrimSpace(req.Text)

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]triagelookupports.TriageRecordMatch, error) {
			c := customerdb.Customers
			exact := keel.Or(lowerEq(c.Code, query), lowerEq(c.Name, query))
			return keel.Into[triagelookupports.TriageRecordMatch](ctx, r.db.Keel(ctx), c.
				Pick(c.ID, c.Name, exact).
				Where(
					c.Status.Eq(domaintypes.StatusActive),
					keel.Or(exact, c.Name.HasPrefixFold(query)),
				).
				Order(keel.Desc(exact), c.Name.Asc()).
				Limit(req.Limit))
		},
	)
}

func (r *repository) FindLocations(
	ctx context.Context,
	req *triagelookupports.TriageLocationQuery,
) ([]triagelookupports.TriageLocationMatch, error) {
	name := strings.TrimSpace(req.Name)
	city := strings.TrimSpace(req.City)
	state := strings.TrimSpace(req.State)

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]triagelookupports.TriageLocationMatch, error) {
			loc := locationdb.Locations
			us := usstatedb.UsStates

			var exact keel.Pred
			var match keel.Pred
			switch {
			case name != "":
				exact = keel.Or(lowerEq(loc.Code, name), lowerEq(loc.Name, name))
				match = keel.Or(exact, loc.Name.HasPrefixFold(name))
			case city != "":
				exact = keel.True()
			default:
				return []triagelookupports.TriageLocationMatch{}, nil
			}

			var stateMatch keel.Pred
			if state != "" {
				stateMatch = keel.Or(
					us.Abbreviation.Upper().EqExpr(keel.Text(state).Upper()),
					lowerEq(us.Name, state),
				)
			}

			return keel.Into[triagelookupports.TriageLocationMatch](ctx, r.db.Keel(ctx), loc.
				LeftJoin(us, us.ID.EqCol(loc.StateID)).
				Pick(
					loc.ID,
					keel.ConcatWS(" · ", loc.Name, keel.ConcatWS(", ", loc.City, us.Abbreviation)),
					text(loc.Timezone),
					exact,
				).
				Where(
					loc.Status.Eq(domaintypes.StatusActive),
					match,
					keel.When(city != "", lowerEq(loc.City, city)),
					stateMatch,
				).
				Order(keel.Desc(exact), loc.Name.Asc()).
				Limit(req.Limit))
		},
	)
}

func (r *repository) FindTrailerTypes(
	ctx context.Context,
	req *triagelookupports.TriageLookupQuery,
) ([]triagelookupports.TriageRecordMatch, error) {
	query := strings.TrimSpace(req.Text)

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]triagelookupports.TriageRecordMatch, error) {
			et := equipmenttypedb.EquipmentTypes
			exact := keel.Or(lowerEq(et.Code, query), lowerEq(et.Description, query))
			return keel.Into[triagelookupports.TriageRecordMatch](ctx, r.db.Keel(ctx), et.
				Pick(et.ID, et.Description.NullIf("").Coalesce(et.Code), exact).
				Where(
					et.Status.Eq(domaintypes.StatusActive),
					et.Class.In(equipmenttype.ClassTrailer, equipmenttype.ClassContainer),
					keel.Or(exact, et.Description.ContainsFold(query)),
				).
				Order(keel.Desc(exact), et.Code.Asc()).
				Limit(req.Limit))
		},
	)
}

func (r *repository) FindCommodities(
	ctx context.Context,
	req *triagelookupports.TriageLookupQuery,
) ([]triagelookupports.TriageRecordMatch, error) {
	query := strings.TrimSpace(req.Text)

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]triagelookupports.TriageRecordMatch, error) {
			c := commoditydb.Commodities
			exact := lowerEq(c.Name, query)
			return keel.Into[triagelookupports.TriageRecordMatch](ctx, r.db.Keel(ctx), c.
				Pick(c.ID, c.Name, exact).
				Where(
					c.Status.Eq(domaintypes.StatusActive),
					keel.Or(exact, c.Name.HasPrefixFold(query)),
				).
				Order(keel.Desc(exact), c.Name.Asc()).
				Limit(req.Limit))
		},
	)
}

func (r *repository) HoldReasonByCode(
	ctx context.Context,
	_ pagination.TenantInfo,
	code string,
) (pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (pulid.ID, error) {
		hr := holdreasondb.HoldReasons
		id, err := keel.IntoOne[pulid.ID](ctx, r.db.Keel(ctx), hr.
			Pick(hr.ID).
			Where(hr.Code.Upper().EqExpr(keel.Text(code).Upper()), hr.Active.IsTrue()).
			Limit(1))
		if dberror.IsNotFoundError(err) {
			return pulid.Nil, nil
		}

		return id, err
	})
}
