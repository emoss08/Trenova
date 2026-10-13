package exceptionsourcerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/documentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/documenttypecol"
	"github.com/emoss08/trenova/pkg/buncolgen/locationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/tendercol"
	"github.com/emoss08/trenova/pkg/buncolgen/usstatecol"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

const (
	defaultLimit = 500
	maxLimit     = 2000

	newerTenderAlias     = "newer_tnd"
	latestStopAlias      = "last_stp"
	shipmentResourceType = "shipment"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) watchtowerports.ExceptionSourceReader {
	return &repository{db: p.DB}
}

func boundedLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	return min(limit, maxLimit)
}

func openShipmentStatuses() []shipment.Status {
	return []shipment.Status{
		shipment.StatusNew,
		shipment.StatusPartiallyAssigned,
		shipment.StatusAssigned,
		shipment.StatusInTransit,
		shipment.StatusDelayed,
		shipment.StatusPartiallyCompleted,
	}
}

func deliveredShipmentStatuses() []shipment.Status {
	return []shipment.Status{
		shipment.StatusPartiallyCompleted,
		shipment.StatusCompleted,
		shipment.StatusReadyToInvoice,
	}
}

func (r *repository) ListMissedStops(
	ctx context.Context,
	req *watchtowerports.ListMissedStopsRequest,
) ([]*watchtowerports.MissedStop, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*watchtowerports.MissedStop, error) {
		stp := shipmentcol.StopColumns
		sm := shipmentcol.ShipmentMoveColumns
		sp := shipmentcol.ShipmentColumns
		loc := locationcol.LocationColumns
		ust := usstatecol.UsStateColumns
		smTable := shipmentcol.ShipmentMoveTable
		spTable := shipmentcol.ShipmentTable
		locTable := locationcol.LocationTable
		ustTable := usstatecol.UsStateTable
		cutoff := buncolgen.Expr(
			"COALESCE({0}, {1})",
			stp.ScheduledWindowEnd,
			stp.ScheduledWindowStart,
		)

		limit := boundedLimit(req.Limit)
		rows := make([]*watchtowerports.MissedStop, 0, min(limit, 64))
		err := r.db.DBForContext(ctx).NewSelect().
			Model((*shipment.Stop)(nil)).
			ColumnExpr(stp.ID.As("stop_id")).
			ColumnExpr(sm.ID.As("move_id")).
			ColumnExpr(sp.ID.As("shipment_id")).
			ColumnExpr(sp.ProNumber.As("pro_number")).
			ColumnExpr(sp.OwnerID.As("owner_id")).
			ColumnExpr(stp.Type.As("stop_type")).
			ColumnExpr(cutoff+" AS cutoff").
			ColumnExpr(loc.Name.Expr("COALESCE({}, '')")+" AS location_name").
			ColumnExpr(loc.City.Expr("COALESCE({}, '')")+" AS city").
			ColumnExpr(ust.Abbreviation.Expr("COALESCE({}, '')")+" AS state_code").
			Join("JOIN "+smTable.As(smTable.Alias)).
			JoinOn(sm.ID.EqColumn(stp.ShipmentMoveID)).
			JoinOn(sm.OrganizationID.EqColumn(stp.OrganizationID)).
			JoinOn(sm.BusinessUnitID.EqColumn(stp.BusinessUnitID)).
			Join("JOIN "+spTable.As(spTable.Alias)).
			JoinOn(sp.ID.EqColumn(sm.ShipmentID)).
			JoinOn(sp.OrganizationID.EqColumn(sm.OrganizationID)).
			JoinOn(sp.BusinessUnitID.EqColumn(sm.BusinessUnitID)).
			Join("LEFT JOIN "+locTable.As(locTable.Alias)).
			JoinOn(loc.ID.EqColumn(stp.LocationID)).
			JoinOn(loc.OrganizationID.EqColumn(stp.OrganizationID)).
			JoinOn(loc.BusinessUnitID.EqColumn(stp.BusinessUnitID)).
			Join("LEFT JOIN "+ustTable.As(ustTable.Alias)).
			JoinOn(ust.ID.EqColumn(loc.StateID)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = shipmentcol.StopScopeTenant(sq, req.TenantInfo).
					Where(stp.Type.In(), bun.List(req.StopTypes)).
					Where(stp.ActualArrival.IsNull()).
					Where(stp.Status.NotIn(), bun.List([]shipment.StopStatus{
						shipment.StopStatusCompleted,
						shipment.StopStatusCanceled,
					})).
					Where(sm.Status.Ne(), shipment.MoveStatusCanceled).
					Where(sp.Status.In(), bun.List(openShipmentStatuses())).
					Where(cutoff+" < ?", req.ClosedBefore)
				if len(req.ShipmentIDs) > 0 {
					sq = sq.Where(sp.ID.In(), bun.List(req.ShipmentIDs))
				}
				return sq
			}).
			OrderExpr(cutoff+" ASC").
			Order(stp.ID.OrderAsc()).
			Limit(int64(limit)).
			Scan(ctx, &rows)
		if err != nil {
			return nil, fmt.Errorf("list missed stops: %w", err)
		}
		return rows, nil
	})
}

func (r *repository) ListMissingPODs(
	ctx context.Context,
	req *watchtowerports.ListMissingPODsRequest,
) ([]*watchtowerports.MissingPOD, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*watchtowerports.MissingPOD, error) {
		db := r.db.DBForContext(ctx)
		sp := shipmentcol.ShipmentColumns
		cus := customercol.CustomerColumns
		cusTable := customercol.CustomerTable
		delivered := shipmentcol.StopColumns.ActualArrival.WithAlias(latestStopAlias)

		limit := boundedLimit(req.Limit)
		rows := make([]*watchtowerports.MissingPOD, 0, min(limit, 64))
		err := db.NewSelect().
			Model((*shipment.Shipment)(nil)).
			ColumnExpr(sp.ID.As("shipment_id")).
			ColumnExpr(sp.ProNumber.As("pro_number")).
			ColumnExpr(sp.OwnerID.As("owner_id")).
			ColumnExpr(cus.Name.Expr("COALESCE({}, '')")+" AS customer_name").
			ColumnExpr(delivered.As("delivered_at")).
			Join("JOIN LATERAL (?) AS ? ON TRUE", lastDeliveryQuery(db), bun.Ident(latestStopAlias)).
			Join("LEFT JOIN "+cusTable.As(cusTable.Alias)).
			JoinOn(cus.ID.EqColumn(sp.CustomerID)).
			JoinOn(cus.OrganizationID.EqColumn(sp.OrganizationID)).
			JoinOn(cus.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = shipmentcol.ShipmentScopeTenant(sq, req.TenantInfo).
					Where(sp.Status.In(), bun.List(deliveredShipmentStatuses())).
					Where(delivered.Lt(), req.DeliveredBefore).
					Where(delivered.Gte(), req.DeliveredAfter).
					Where("NOT EXISTS (?)", podOnFileQuery(db))
				if len(req.ShipmentIDs) > 0 {
					sq = sq.Where(sp.ID.In(), bun.List(req.ShipmentIDs))
				}
				return sq
			}).
			Order(delivered.OrderAsc(), sp.ID.OrderAsc()).
			Limit(int64(limit)).
			Scan(ctx, &rows)
		if err != nil {
			return nil, fmt.Errorf("list shipments missing a pod: %w", err)
		}
		return rows, nil
	})
}

func lastDeliveryQuery(db bun.IDB) *bun.SelectQuery {
	stp := shipmentcol.StopColumns
	sm := shipmentcol.ShipmentMoveColumns
	sp := shipmentcol.ShipmentColumns
	smTable := shipmentcol.ShipmentMoveTable

	return db.NewSelect().
		Model((*shipment.Stop)(nil)).
		ColumnExpr(stp.ActualArrival.Qualified()).
		Join("JOIN "+smTable.As(smTable.Alias)).
		JoinOn(sm.ID.EqColumn(stp.ShipmentMoveID)).
		JoinOn(sm.OrganizationID.EqColumn(stp.OrganizationID)).
		JoinOn(sm.BusinessUnitID.EqColumn(stp.BusinessUnitID)).
		Where(sm.ShipmentID.EqColumn(sp.ID)).
		Where(sm.OrganizationID.EqColumn(sp.OrganizationID)).
		Where(sm.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
		Where(sm.Status.Ne(), shipment.MoveStatusCanceled).
		Where(stp.Type.In(), bun.List([]shipment.StopType{
			shipment.StopTypeDelivery,
			shipment.StopTypeSplitDelivery,
		})).
		Where(stp.Status.Ne(), shipment.StopStatusCanceled).
		Where(stp.ActualArrival.IsNotNull()).
		Order(sm.Sequence.OrderDesc(), stp.Sequence.OrderDesc()).
		Limit(1)
}

func podOnFileQuery(db bun.IDB) *bun.SelectQuery {
	doc := documentcol.DocumentColumns
	dt := documenttypecol.DocumentTypeColumns
	sp := shipmentcol.ShipmentColumns
	dtTable := documenttypecol.DocumentTypeTable
	docTable := documentcol.DocumentTable

	return db.NewSelect().
		TableExpr(docTable.As(docTable.Alias)).
		ColumnExpr("1").
		Join("JOIN "+dtTable.As(dtTable.Alias)).
		JoinOn(dt.ID.EqColumn(doc.DocumentTypeID)).
		JoinOn(dt.OrganizationID.EqColumn(doc.OrganizationID)).
		JoinOn(dt.BusinessUnitID.EqColumn(doc.BusinessUnitID)).
		Where(doc.OrganizationID.EqColumn(sp.OrganizationID)).
		Where(doc.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
		Where(doc.ResourceType.Eq(), shipmentResourceType).
		Where(doc.ResourceID.EqColumn(sp.ID)).
		Where(doc.IsCurrentVersion.IsTrue()).
		Where(doc.Status.NotIn(), bun.List([]document.Status{
			document.StatusRejected,
			document.StatusArchived,
		})).
		Where(dt.Code.Eq(), documenttype.CodePOD)
}

func (r *repository) ListExhaustedTenders(
	ctx context.Context,
	req *watchtowerports.ListExhaustedTendersRequest,
) ([]*watchtowerports.ExhaustedTender, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*watchtowerports.ExhaustedTender, error) {
			db := r.db.DBForContext(ctx)
			tc := tendercol.TenderColumns
			sm := shipmentcol.ShipmentMoveColumns
			sp := shipmentcol.ShipmentColumns
			ca := shipmentcol.CarrierAssignmentColumns
			smTable := shipmentcol.ShipmentMoveTable
			spTable := shipmentcol.ShipmentTable
			caTable := shipmentcol.CarrierAssignmentTable
			tenderTable := tendercol.TenderTable
			newerMove := tc.ShipmentMoveID.WithAlias(newerTenderAlias)
			newerOrg := tc.OrganizationID.WithAlias(newerTenderAlias)
			newerBu := tc.BusinessUnitID.WithAlias(newerTenderAlias)
			newerCreated := tc.CreatedAt.WithAlias(newerTenderAlias)
			newerID := tc.ID.WithAlias(newerTenderAlias)

			newer := db.NewSelect().
				TableExpr(tenderTable.Name+" AS "+newerTenderAlias).
				ColumnExpr("1").
				Where(newerMove.EqColumn(tc.ShipmentMoveID)).
				Where(newerOrg.EqColumn(tc.OrganizationID)).
				Where(newerBu.EqColumn(tc.BusinessUnitID)).
				Where(
					"(?, ?) > (?, ?)",
					bun.Safe(newerCreated.Qualified()),
					bun.Safe(newerID.Qualified()),
					bun.Safe(tc.CreatedAt.Qualified()),
					bun.Safe(tc.ID.Qualified()),
				)

			covered := db.NewSelect().
				TableExpr(caTable.As(caTable.Alias)).
				ColumnExpr("1").
				Where(ca.ShipmentMoveID.EqColumn(sm.ID)).
				Where(ca.OrganizationID.EqColumn(sm.OrganizationID)).
				Where(ca.BusinessUnitID.EqColumn(sm.BusinessUnitID)).
				Where(ca.Status.Ne(), shipment.CarrierAssignmentStatusCanceled)

			limit := boundedLimit(req.Limit)
			rows := make([]*watchtowerports.ExhaustedTender, 0, min(limit, 32))
			err := db.NewSelect().
				TableExpr(tenderTable.As(tenderTable.Alias)).
				ColumnExpr(tc.ID.As("tender_id")).
				ColumnExpr(sm.ID.As("move_id")).
				ColumnExpr(sp.ID.As("shipment_id")).
				ColumnExpr(sp.ProNumber.As("pro_number")).
				ColumnExpr(sp.OwnerID.As("owner_id")).
				ColumnExpr(buncolgen.Expr("COALESCE({0}, {1}) AS exhausted_at", tc.ExhaustedAt, tc.UpdatedAt)).
				Join("JOIN "+smTable.As(smTable.Alias)).
				JoinOn(sm.ID.EqColumn(tc.ShipmentMoveID)).
				JoinOn(sm.OrganizationID.EqColumn(tc.OrganizationID)).
				JoinOn(sm.BusinessUnitID.EqColumn(tc.BusinessUnitID)).
				Join("JOIN "+spTable.As(spTable.Alias)).
				JoinOn(sp.ID.EqColumn(sm.ShipmentID)).
				JoinOn(sp.OrganizationID.EqColumn(sm.OrganizationID)).
				JoinOn(sp.BusinessUnitID.EqColumn(sm.BusinessUnitID)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					sq = tendercol.TenderScopeTenant(sq, req.TenantInfo).
						Where(tc.Status.Eq(), tender.StatusExhausted).
						Where(sm.Status.Eq(), shipment.MoveStatusNew).
						Where(sp.Status.In(), bun.List(openShipmentStatuses())).
						Where("NOT EXISTS (?)", newer).
						Where("NOT EXISTS (?)", covered)
					if len(req.ShipmentIDs) > 0 {
						sq = sq.Where(sp.ID.In(), bun.List(req.ShipmentIDs))
					}
					return sq
				}).
				Order(tc.CreatedAt.OrderAsc(), tc.ID.OrderAsc()).
				Limit(int64(limit)).
				Scan(ctx, &rows)
			if err != nil {
				return nil, fmt.Errorf("list exhausted tenders: %w", err)
			}
			return rows, nil
		},
	)
}
