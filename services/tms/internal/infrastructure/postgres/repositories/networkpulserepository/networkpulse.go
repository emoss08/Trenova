package networkpulserepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

func New(p Params) repositories.NetworkPulseRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.network-pulse-repository"),
	}
}

// activeLaneStatuses matches the analytics provider's notion of an active shipment, so a
// lane on the sign-in screen means the same thing as a lane on the dashboard.
var activeLaneStatuses = []shipment.Status{
	shipment.StatusNew,
	shipment.StatusPartiallyAssigned,
	shipment.StatusAssigned,
	shipment.StatusInTransit,
	shipment.StatusDelayed,
}

// activeLanesQuery resolves each active shipment's first pickup state and last delivery
// state, then groups. It is the same origin/destination derivation the in-app lane
// heatmap uses — first stop by move sequence then stop sequence, last stop by the
// reverse — so the two cannot disagree about where a load is running.
//
// The organization_id / business_unit_id predicates inside the subqueries are join
// integrity, not tenant scoping: they correlate each stop back to its own shipment's
// tenant so a row from another organization can never be picked up. The outer query is
// deliberately unscoped.
const activeLanesQuery = `
WITH shipment_lanes AS (
	SELECT
		sp.status,
		(
			SELECT ust_orig.abbreviation
			FROM shipment_moves sm_orig
			INNER JOIN stops stp_orig
				ON stp_orig.shipment_move_id = sm_orig.id
				AND stp_orig.organization_id = sm_orig.organization_id
				AND stp_orig.business_unit_id = sm_orig.business_unit_id
			INNER JOIN locations loc_orig
				ON loc_orig.id = stp_orig.location_id
				AND loc_orig.organization_id = stp_orig.organization_id
				AND loc_orig.business_unit_id = stp_orig.business_unit_id
			INNER JOIN us_states ust_orig
				ON ust_orig.id = loc_orig.state_id
			WHERE sm_orig.shipment_id = sp.id
				AND sm_orig.organization_id = sp.organization_id
				AND sm_orig.business_unit_id = sp.business_unit_id
				AND stp_orig.type IN (?, ?)
			ORDER BY sm_orig.sequence ASC, stp_orig.sequence ASC
			LIMIT 1
		) AS origin_state,
		(
			SELECT ust_dest.abbreviation
			FROM shipment_moves sm_dest
			INNER JOIN stops stp_dest
				ON stp_dest.shipment_move_id = sm_dest.id
				AND stp_dest.organization_id = sm_dest.organization_id
				AND stp_dest.business_unit_id = sm_dest.business_unit_id
			INNER JOIN locations loc_dest
				ON loc_dest.id = stp_dest.location_id
				AND loc_dest.organization_id = stp_dest.organization_id
				AND loc_dest.business_unit_id = stp_dest.business_unit_id
			INNER JOIN us_states ust_dest
				ON ust_dest.id = loc_dest.state_id
			WHERE sm_dest.shipment_id = sp.id
				AND sm_dest.organization_id = sp.organization_id
				AND sm_dest.business_unit_id = sp.business_unit_id
				AND stp_dest.type IN (?, ?)
			ORDER BY sm_dest.sequence DESC, stp_dest.sequence DESC
			LIMIT 1
		) AS destination_state
	FROM shipments sp
	WHERE sp.status IN (?)
)
SELECT origin_state, destination_state, status, COUNT(*)::int AS count
FROM shipment_lanes
WHERE origin_state IS NOT NULL
	AND destination_state IS NOT NULL
GROUP BY origin_state, destination_state, status
ORDER BY count DESC, origin_state ASC, destination_state ASC, status ASC
LIMIT ?`

// GetNetworkPulse deliberately applies no organization or business unit predicate: the
// sign-in screen has no tenant, and the figure it shows is the instance as a whole.
//
// Every part reuses the definitions the in-app shipment analytics already use, so the
// login screen and the dashboard cannot disagree: a load is in motion when its shipment
// is InTransit, and a delivery is on time when a completed delivery stop's actual
// arrival lands at or before its scheduled window end (falling back to the window start
// when no end is set).
func (r *repository) GetNetworkPulse(
	ctx context.Context,
	since int64,
	laneLimit int,
) (*repositories.NetworkPulseCounts, error) {
	counts := &repositories.NetworkPulseCounts{Lanes: []repositories.NetworkPulseLane{}}

	var inMotion struct {
		LoadsInMotion int `bun:"loads_in_motion"`
	}
	if err := r.db.DB().NewSelect().
		TableExpr("shipments sp").
		ColumnExpr("COUNT(*)::int AS loads_in_motion").
		Where("sp.status = ?", shipment.StatusInTransit).
		Scan(ctx, &inMotion); err != nil {
		r.l.Error("failed to count loads in motion", zap.Error(err))
		return nil, err
	}
	counts.LoadsInMotion = inMotion.LoadsInMotion

	var onTime struct {
		OnTimeCount int `bun:"on_time_count"`
		OnTimeTotal int `bun:"on_time_total"`
	}
	if err := r.db.DB().NewSelect().
		TableExpr("stops stp").
		ColumnExpr("COUNT(*)::int AS on_time_total").
		ColumnExpr(
			"COUNT(*) FILTER ("+
				"WHERE stp.actual_arrival <= "+
				"COALESCE(stp.scheduled_window_end, stp.scheduled_window_start)"+
				")::int AS on_time_count",
		).
		Where("stp.status = ?", shipment.StopStatusCompleted).
		Where("stp.type IN (?)", bun.List([]shipment.StopType{
			shipment.StopTypeDelivery,
			shipment.StopTypeSplitDelivery,
		})).
		Where("stp.actual_arrival IS NOT NULL").
		Where("stp.actual_arrival >= ?", since).
		Where("stp.scheduled_window_start > 0").
		Scan(ctx, &onTime); err != nil {
		r.l.Error("failed to score on-time deliveries", zap.Error(err))
		return nil, err
	}
	counts.OnTimeCount = onTime.OnTimeCount
	counts.OnTimeTotal = onTime.OnTimeTotal

	if laneLimit <= 0 {
		return counts, nil
	}

	laneRows := make([]struct {
		OriginState      string `bun:"origin_state"`
		DestinationState string `bun:"destination_state"`
		Status           string `bun:"status"`
		Count            int    `bun:"count"`
	}, 0, laneLimit)

	if err := r.db.DB().NewRaw(
		activeLanesQuery,
		shipment.StopTypePickup,
		shipment.StopTypeSplitPickup,
		shipment.StopTypeDelivery,
		shipment.StopTypeSplitDelivery,
		bun.List(activeLaneStatuses),
		laneLimit,
	).Scan(ctx, &laneRows); err != nil {
		r.l.Error("failed to load active lanes", zap.Error(err))
		return nil, err
	}

	counts.Lanes = make([]repositories.NetworkPulseLane, 0, len(laneRows))
	for _, row := range laneRows {
		counts.Lanes = append(counts.Lanes, repositories.NetworkPulseLane{
			OriginState:      row.OriginState,
			DestinationState: row.DestinationState,
			Status:           row.Status,
			Count:            row.Count,
		})
	}

	return counts, nil
}
