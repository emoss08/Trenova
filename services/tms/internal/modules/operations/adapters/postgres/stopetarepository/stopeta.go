package stopetarepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta/stopetadb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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
		e := stopetadb.StopETAs
		stale := e.Where(e.ShipmentID.In(req.ShipmentIDs...))
		if len(keep) > 0 {
			stale = stale.Where(e.StopID.NotIn(keep...))
		}
		if _, err := stale.Delete(ctx, r.db.Keel(ctx)); err != nil {
			return fmt.Errorf("delete stale stop etas: %w", err)
		}
		if len(rows) == 0 {
			return nil
		}

		if _, err := e.Upsert(ctx, r.db.Keel(ctx), rows, newerETA()); err != nil {
			return fmt.Errorf("upsert stop etas: %w", err)
		}
		return nil
	})
}

func newerETA() keel.Conflict {
	e := stopetadb.StopETAs
	newer := keel.Int64Of(e.ComputedAt).LteExpr(keel.Int64Of(e.ComputedAt.Excluded()))
	text := func(c keel.StringCol) keel.Assignment {
		return c.ToExpr(keel.CaseString().When(newer, keel.StringOf(c.Excluded())).Else(c))
	}
	bigint := func(c keel.Int64Col) keel.Assignment {
		return c.ToExpr(keel.CaseInt64().When(newer, keel.Int64Of(c.Excluded())).Else(c))
	}
	boolean := func(c keel.BoolCol) keel.Assignment {
		return c.ToExpr(keel.CaseBool().When(newer, keel.BoolOf(c.Excluded())).Else(c))
	}
	float := func(c keel.Float64Col) keel.Assignment {
		return c.ToExpr(keel.CaseFloat64().When(newer, keel.Float64Of(c.Excluded())).Else(c))
	}

	return keel.OnConflict(e.OrganizationID, e.BusinessUnitID, e.StopID).Set(
		text(e.ShipmentID.StringCol),
		text(e.ShipmentMoveID.StringCol),
		bigint(e.MoveSequence),
		bigint(e.StopSequence),
		boolean(e.Arrived),
		bigint(e.EstimatedArrival),
		bigint(e.EstimatedDeparture),
		bigint(e.WindowEnd),
		bigint(e.SlackMinutes),
		text(e.Verdict.StringCol),
		text(e.Basis.StringCol),
		text(e.DriveMethod.StringCol),
		float(e.MilesRemaining),
		bigint(e.DriveMinutes),
		bigint(e.RestMinutes),
		bigint(e.InputAt),
		boolean(e.InputStale),
		bigint(e.ThresholdMinutes),
		text(e.Reason),
		bigint(e.ComputedAt),
	)
}

func (r *repository) ListByShipmentIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) ([]*stopeta.StopETA, error) {
	if len(shipmentIDs) == 0 {
		return []*stopeta.StopETA{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*stopeta.StopETA, error) {
		e := stopetadb.StopETAs
		entities, err := e.Where(e.ShipmentID.In(shipmentIDs...)).
			Order(e.ShipmentID.Asc(), e.MoveSequence.Asc(), e.StopSequence.Asc()).
			All(ctx, r.db.Keel(ctx))
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
		meta, err := r.stopMeta(ctx, stopIDs)
		if err != nil {
			return nil, err
		}

		out := make([]*stopetaports.AttentionStop, 0, len(rows))
		for _, row := range rows {
			item := &stopetaports.AttentionStop{ETA: row}
			if m, ok := meta[row.StopID]; ok {
				item.ProNumber = m.ProNumber
				item.OwnerID = m.OwnerID
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
	e := stopetadb.StopETAs
	sp := shipmentdb.Shipments

	q := e.Join(sp, sp.ID.EqCol(e.ShipmentID)).Where(
		e.Verdict.In(stopeta.VerdictAtRisk, stopeta.VerdictLate),
		e.Arrived.IsFalse(),
		sp.Status.In(req.Statuses...),
	)
	if len(req.ShipmentIDs) > 0 {
		q = q.Where(e.ShipmentID.In(req.ShipmentIDs...))
	}

	entities, err := q.
		Order(e.ShipmentID.Asc(), e.MoveSequence.Asc(), e.StopSequence.Asc()).
		Limit(boundedLimit(req.Limit, defaultAttentionLimit, maxAttentionLimit)).
		List(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, fmt.Errorf("list stop etas needing attention: %w", err)
	}
	return entities, nil
}

type stopMetaRow struct {
	StopID       pulid.ID
	StopType     shipmentvocab.StopType
	ProNumber    string
	OwnerID      pulid.ID
	LocationName string
	City         string
	StateCode    string
}

func (r *repository) stopMeta(
	ctx context.Context,
	stopIDs []pulid.ID,
) (map[pulid.ID]stopMetaRow, error) {
	st := shipmentdb.Stops
	sm := shipmentdb.ShipmentMoves
	sp := shipmentdb.Shipments
	loc := st.Location()
	ust := loc.State()

	rows, err := keel.Into[stopMetaRow](ctx, r.db.Keel(ctx), st.
		Join(sm, sm.ID.EqCol(st.ShipmentMoveID)).
		Join(sp, sp.ID.EqCol(sm.ShipmentID)).
		Pick(
			st.ID,
			st.Type,
			sp.ProNumber,
			sp.OwnerID.Coalesce(""),
			loc.Name.Coalesce(""),
			loc.City.Coalesce(""),
			ust.Abbreviation.Coalesce(""),
		).
		Where(st.ID.In(stopIDs...)).
		Limit(len(stopIDs)))
	if err != nil {
		return nil, fmt.Errorf("list stop details for etas: %w", err)
	}

	out := make(map[pulid.ID]stopMetaRow, len(rows))
	for _, row := range rows {
		out[row.StopID] = row
	}
	return out, nil
}

func (r *repository) ListTrackedShipmentIDs(
	ctx context.Context,
	req *stopetaports.ListTrackedShipmentIDsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		sp := shipmentdb.Shipments
		e := stopetadb.StopETAs

		var tracked, leftover []pulid.ID
		err := keel.Batch(
			ctx,
			r.db.Keel(ctx),
			keel.IntoSlice(sp.Pick(sp.ID).
				Where(sp.Status.In(req.Statuses...)).
				Order(sp.ID.Asc()).
				Limit(boundedLimit(req.Limit, defaultTrackedLimit, maxTrackedLimit)), &tracked),
			keel.IntoSlice(e.DistinctOn(e.ShipmentID).
				Pick(e.ShipmentID).
				Join(sp, sp.ID.EqCol(e.ShipmentID)).
				Where(sp.Status.NotIn(req.Statuses...)).
				Order(e.ShipmentID.Asc()).
				Limit(maxTrackedLimit), &leftover),
		)
		if err != nil {
			return nil, fmt.Errorf("list tracked shipments: %w", err)
		}

		ids := make([]pulid.ID, 0, len(tracked)+len(leftover))
		ids = append(ids, tracked...)
		ids = append(ids, leftover...)
		return ids, nil
	})
}

type tenantRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func (r *repository) ListActiveTenants(
	ctx context.Context,
	statuses []shipmentvocab.Status,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with shipments whose arrivals are projected")

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		sp := shipmentdb.Shipments
		e := stopetadb.StopETAs

		var fromShipments, fromEtas []tenantRow
		err := keel.Batch(
			ctx,
			r.db.Keel(ctx),
			keel.IntoSlice(sp.AllTenants().
				Group(sp.OrganizationID, sp.BusinessUnitID).
				Pick(sp.OrganizationID, sp.BusinessUnitID).
				Where(sp.Status.In(statuses...)), &fromShipments),
			keel.IntoSlice(e.AllTenants().
				Group(e.OrganizationID, e.BusinessUnitID).
				Pick(e.OrganizationID, e.BusinessUnitID), &fromEtas),
		)
		if err != nil {
			return nil, fmt.Errorf("list tenants with tracked shipments: %w", err)
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
