package stopetarepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/pkg/buncolgen/locationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/stopetacol"
	"github.com/emoss08/trenova/pkg/buncolgen/usstatecol"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	MaxShipmentsPerRequest = 500
	defaultTrackedLimit    = 5000
	maxTrackedLimit        = 20000
	defaultAttentionLimit  = 5000
	maxAttentionLimit      = 20000
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) stopetaports.Repository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.stop-eta-repository"),
	}
}

func (r *repository) ReplaceForShipments(
	ctx context.Context,
	req *stopetaports.ReplaceForShipmentsRequest,
) error {
	if len(req.ShipmentIDs) == 0 {
		return nil
	}
	if len(req.ShipmentIDs) > MaxShipmentsPerRequest {
		return errortypes.NewBusinessError(
			"At most %d shipments can be refreshed at once",
			MaxShipmentsPerRequest,
		)
	}

	allowed := make(map[pulid.ID]struct{}, len(req.ShipmentIDs))
	for _, id := range req.ShipmentIDs {
		allowed[id] = struct{}{}
	}
	keep := make([]pulid.ID, 0, len(req.Rows))
	rows := make([]*stopeta.StopETA, 0, len(req.Rows))
	for _, row := range req.Rows {
		if row == nil {
			continue
		}
		if row.OrganizationID != req.TenantInfo.OrgID || row.BusinessUnitID != req.TenantInfo.BuID {
			return errortypes.NewBusinessError("A stop ETA belongs to another organization")
		}
		if _, ok := allowed[row.ShipmentID]; !ok {
			return errortypes.NewBusinessError("A stop ETA names a shipment outside the refresh")
		}
		if row.ID.IsNil() {
			row.ID = stopeta.NewID()
		}
		keep = append(keep, row.StopID)
		rows = append(rows, row)
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.DBForContext(ctx)
		cols := stopetacol.StopETAColumns

		_, err := db.NewDelete().
			Model((*stopeta.StopETA)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				dq = stopetacol.StopETAScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ShipmentID.In(), bun.List(req.ShipmentIDs))
				if len(keep) > 0 {
					dq = dq.Where(cols.StopID.NotIn(), bun.List(keep))
				}
				return dq
			}).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("delete stale stop etas: %w", err)
		}
		if len(rows) == 0 {
			return nil
		}

		_, err = db.NewInsert().
			Model(&rows).
			On("CONFLICT (organization_id, business_unit_id, stop_id) DO UPDATE").
			Set(cols.ShipmentID.SetExcluded()).
			Set(cols.ShipmentMoveID.SetExcluded()).
			Set(cols.MoveSequence.SetExcluded()).
			Set(cols.StopSequence.SetExcluded()).
			Set(cols.Arrived.SetExcluded()).
			Set(cols.EstimatedArrival.SetExcluded()).
			Set(cols.EstimatedDeparture.SetExcluded()).
			Set(cols.WindowEnd.SetExcluded()).
			Set(cols.SlackMinutes.SetExcluded()).
			Set(cols.Verdict.SetExcluded()).
			Set(cols.Basis.SetExcluded()).
			Set(cols.DriveMethod.SetExcluded()).
			Set(cols.MilesRemaining.SetExcluded()).
			Set(cols.DriveMinutes.SetExcluded()).
			Set(cols.RestMinutes.SetExcluded()).
			Set(cols.InputAt.SetExcluded()).
			Set(cols.InputStale.SetExcluded()).
			Set(cols.ThresholdMinutes.SetExcluded()).
			Set(cols.Reason.SetExcluded()).
			Set(cols.ComputedAt.SetExcluded()).
			Where(cols.ComputedAt.Qualified() + " <= EXCLUDED.computed_at").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("upsert stop etas: %w", err)
		}
		return nil
	})
}

func (r *repository) ListByShipmentIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) ([]*stopeta.StopETA, error) {
	if len(shipmentIDs) == 0 {
		return []*stopeta.StopETA{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*stopeta.StopETA, error) {
		cols := stopetacol.StopETAColumns
		entities := make([]*stopeta.StopETA, 0, len(shipmentIDs)*2)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return stopetacol.StopETAScopeTenant(sq, tenantInfo).
					Where(cols.ShipmentID.In(), bun.List(shipmentIDs))
			}).
			Order(
				cols.ShipmentID.OrderAsc(),
				cols.MoveSequence.OrderAsc(),
				cols.StopSequence.OrderAsc(),
			).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list stop etas by shipment: %w", err)
		}
		return entities, nil
	})
}

func (r *repository) ListAttention(
	ctx context.Context,
	req *stopetaports.ListAttentionRequest,
) ([]*stopetaports.AttentionStop, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*stopetaports.AttentionStop, error) {
		rows, err := r.attentionRows(ctx, req)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return []*stopetaports.AttentionStop{}, nil
		}

		stopIDs := make([]pulid.ID, 0, len(rows))
		for _, row := range rows {
			stopIDs = append(stopIDs, row.StopID)
		}
		meta, err := r.stopMeta(ctx, req.TenantInfo, stopIDs)
		if err != nil {
			return nil, err
		}

		out := make([]*stopetaports.AttentionStop, 0, len(rows))
		for _, row := range rows {
			item := &stopetaports.AttentionStop{ETA: row}
			if m, ok := meta[row.StopID]; ok {
				item.ProNumber = m.ProNumber
				item.StopType = m.StopType
				item.LocationName = m.LocationName
				item.City = m.City
				item.StateCode = m.StateCode
			}
			out = append(out, item)
		}
		return out, nil
	})
}

func (r *repository) attentionRows(
	ctx context.Context,
	req *stopetaports.ListAttentionRequest,
) ([]*stopeta.StopETA, error) {
	cols := stopetacol.StopETAColumns
	sp := shipmentcol.ShipmentColumns
	spTable := shipmentcol.ShipmentTable

	entities := make([]*stopeta.StopETA, 0)
	err := r.db.DBForContext(ctx).NewSelect().
		Model(&entities).
		Join("JOIN "+spTable.As(spTable.Alias)).
		JoinOn(sp.ID.EqColumn(cols.ShipmentID)).
		JoinOn(sp.OrganizationID.EqColumn(cols.OrganizationID)).
		JoinOn(sp.BusinessUnitID.EqColumn(cols.BusinessUnitID)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = stopetacol.StopETAScopeTenant(sq, req.TenantInfo).
				Where(cols.Verdict.In(), bun.List([]stopeta.Verdict{
					stopeta.VerdictAtRisk,
					stopeta.VerdictLate,
				})).
				Where(cols.Arrived.IsFalse()).
				Where(sp.Status.In(), bun.List(req.Statuses))
			if len(req.ShipmentIDs) > 0 {
				sq = sq.Where(cols.ShipmentID.In(), bun.List(req.ShipmentIDs))
			}
			return sq
		}).
		Order(
			cols.ShipmentID.OrderAsc(),
			cols.MoveSequence.OrderAsc(),
			cols.StopSequence.OrderAsc(),
		).
		Limit(int64(boundedLimit(req.Limit, defaultAttentionLimit, maxAttentionLimit))).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list stop etas needing attention: %w", err)
	}
	return entities, nil
}

type stopMetaRow struct {
	StopID       pulid.ID          `bun:"stop_id"`
	StopType     shipment.StopType `bun:"stop_type"`
	ProNumber    string            `bun:"pro_number"`
	LocationName string            `bun:"location_name"`
	City         string            `bun:"city"`
	StateCode    string            `bun:"state_code"`
}

func (r *repository) stopMeta(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	stopIDs []pulid.ID,
) (map[pulid.ID]stopMetaRow, error) {
	stp := shipmentcol.StopColumns
	sm := shipmentcol.ShipmentMoveColumns
	sp := shipmentcol.ShipmentColumns
	loc := locationcol.LocationColumns
	ust := usstatecol.UsStateColumns
	smTable := shipmentcol.ShipmentMoveTable
	spTable := shipmentcol.ShipmentTable
	locTable := locationcol.LocationTable
	ustTable := usstatecol.UsStateTable

	rows := make([]stopMetaRow, 0, len(stopIDs))
	err := r.db.DBForContext(ctx).NewSelect().
		Model((*shipment.Stop)(nil)).
		ColumnExpr(stp.ID.As("stop_id")).
		ColumnExpr(stp.Type.As("stop_type")).
		ColumnExpr(sp.ProNumber.As("pro_number")).
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
			return shipmentcol.StopScopeTenant(sq, tenantInfo).
				Where(stp.ID.In(), bun.List(stopIDs))
		}).
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("list stop details for etas: %w", err)
	}

	out := make(map[pulid.ID]stopMetaRow, len(rows))
	for _, row := range rows {
		out[row.StopID] = row
	}
	return out, nil
}

type shipmentIDRow struct {
	ShipmentID pulid.ID `bun:"shipment_id"`
}

func (r *repository) ListTrackedShipmentIDs(
	ctx context.Context,
	req *stopetaports.ListTrackedShipmentIDsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		db := r.db.DBForContext(ctx)
		sp := shipmentcol.ShipmentColumns
		cols := stopetacol.StopETAColumns
		spTable := shipmentcol.ShipmentTable

		tracked := make([]shipmentIDRow, 0)
		err := db.NewSelect().
			Model((*shipment.Shipment)(nil)).
			ColumnExpr(sp.ID.As("shipment_id")).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return shipmentcol.ShipmentScopeTenant(sq, req.TenantInfo).
					Where(sp.Status.In(), bun.List(req.Statuses))
			}).
			Order(sp.ID.OrderAsc()).
			Limit(int64(boundedLimit(req.Limit, defaultTrackedLimit, maxTrackedLimit))).
			Scan(ctx, &tracked)
		if err != nil {
			return nil, fmt.Errorf("list tracked shipments: %w", err)
		}

		leftover := make([]shipmentIDRow, 0)
		err = db.NewSelect().
			Model((*stopeta.StopETA)(nil)).
			DistinctOn(cols.ShipmentID.Qualified()).
			ColumnExpr(cols.ShipmentID.As("shipment_id")).
			Join("JOIN "+spTable.As(spTable.Alias)).
			JoinOn(sp.ID.EqColumn(cols.ShipmentID)).
			JoinOn(sp.OrganizationID.EqColumn(cols.OrganizationID)).
			JoinOn(sp.BusinessUnitID.EqColumn(cols.BusinessUnitID)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return stopetacol.StopETAScopeTenant(sq, req.TenantInfo).
					Where(sp.Status.NotIn(), bun.List(req.Statuses))
			}).
			Order(cols.ShipmentID.OrderAsc()).
			Limit(int64(maxTrackedLimit)).
			Scan(ctx, &leftover)
		if err != nil {
			return nil, fmt.Errorf("list shipments with stale stop etas: %w", err)
		}

		ids := make([]pulid.ID, 0, len(tracked)+len(leftover))
		for _, row := range tracked {
			ids = append(ids, row.ShipmentID)
		}
		for _, row := range leftover {
			ids = append(ids, row.ShipmentID)
		}
		return ids, nil
	})
}

type tenantRow struct {
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

func (r *repository) ListActiveTenants(
	ctx context.Context,
	statuses []shipment.Status,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with shipments whose arrivals are projected")

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		db := r.db.DBForContext(ctx)
		sp := shipmentcol.ShipmentColumns
		cols := stopetacol.StopETAColumns

		fromShipments := make([]tenantRow, 0)
		err := db.NewSelect().
			Model((*shipment.Shipment)(nil)).
			Distinct().
			ColumnExpr(sp.OrganizationID.As("organization_id")).
			ColumnExpr(sp.BusinessUnitID.As("business_unit_id")).
			Where(sp.Status.In(), bun.List(statuses)).
			Scan(ctx, &fromShipments)
		if err != nil {
			return nil, fmt.Errorf("list tenants with tracked shipments: %w", err)
		}

		fromEtas := make([]tenantRow, 0)
		err = db.NewSelect().
			Model((*stopeta.StopETA)(nil)).
			Distinct().
			ColumnExpr(cols.OrganizationID.As("organization_id")).
			ColumnExpr(cols.BusinessUnitID.As("business_unit_id")).
			Scan(ctx, &fromEtas)
		if err != nil {
			return nil, fmt.Errorf("list tenants with stop etas: %w", err)
		}

		seen := make(map[tenantRow]struct{}, len(fromShipments)+len(fromEtas))
		tenants := make([]pagination.TenantInfo, 0, len(fromShipments)+len(fromEtas))
		for _, rows := range [][]tenantRow{fromShipments, fromEtas} {
			for _, row := range rows {
				if _, ok := seen[row]; ok {
					continue
				}
				seen[row] = struct{}{}
				tenants = append(tenants, pagination.TenantInfo{
					OrgID: row.OrganizationID,
					BuID:  row.BusinessUnitID,
				})
			}
		}
		return tenants, nil
	})
}

func boundedLimit(limit, fallback, ceiling int) int {
	if limit <= 0 {
		return fallback
	}
	return min(limit, ceiling)
}
