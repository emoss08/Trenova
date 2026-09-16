package networkpulserepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/lanequery"
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

	cols := buncolgen.ShipmentColumns
	if err := lanequery.New(r.db.DB(), lanequery.Options{
		ShipmentFilter: func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(cols.Status.In(), bun.List(activeLaneStatuses))
		},
		GroupByStatus: true,
	}).
		OrderExpr(lanequery.CountColumn+" DESC").
		OrderExpr(lanequery.OriginStateColumn+" ASC").
		OrderExpr(lanequery.DestinationStateColumn+" ASC").
		OrderExpr(lanequery.StatusColumn+" ASC").
		Limit(laneLimit).
		Scan(ctx, &laneRows); err != nil {
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
