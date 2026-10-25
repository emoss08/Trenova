package dispatchconsolerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics/telematicsdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/shared/timeutils"
)

const atRiskLeadSeconds = int64(4 * 3600)

const positionFreshnessSeconds = int64(4 * 3600)

const metersPerMile = 1609.344

type moveSummary struct {
	UncoveredMoves  int
	CoveredMoves    int
	LateMoves       int
	AtRiskMoves     int
	AssignedToday   int
	AverageDeadhead float64
}

type driverSummary struct {
	AvailableDrivers int
	UnseatedDrivers  int
}

func (r *repository) GetBoardSummary(
	ctx context.Context,
	filter *shipmentports.DispatchBoardFilter,
) (*shipmentports.BoardSummary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipmentports.BoardSummary, error) {
		now := timeutils.NowUnix()
		dayStart := filter.DayStartUnix
		if dayStart <= 0 {
			var err error
			dayStart, err = timeutils.DayStartUnix(now, "")
			if err != nil {
				return nil, fmt.Errorf("resolve dispatch board day start: %w", err)
			}
		}

		var moves moveSummary
		var drivers driverSummary
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoValue(moveSummaryQuery(filter, now, dayStart), &moves),
			keel.IntoValue(driverSummaryQuery(filter), &drivers),
		)
		if err != nil {
			return nil, fmt.Errorf("scan dispatch board summary: %w", err)
		}

		summary := &shipmentports.BoardSummary{
			UncoveredMoves:   moves.UncoveredMoves,
			CoveredMoves:     moves.CoveredMoves,
			LateMoves:        moves.LateMoves,
			AtRiskMoves:      moves.AtRiskMoves,
			UnseatedDrivers:  drivers.UnseatedDrivers,
			AvailableDrivers: drivers.AvailableDrivers,
			AssignedToday:    moves.AssignedToday,
			AverageDeadhead:  moves.AverageDeadhead,
		}
		if summary.AvailableDrivers > 0 {
			seated := summary.AvailableDrivers - summary.UnseatedDrivers
			summary.UtilizationPct = float64(seated) / float64(summary.AvailableDrivers) * 100
		}

		return summary, nil
	})
}

func countIf(preds ...keel.Pred) keel.Expr {
	return keel.Count().Filter(keel.And(preds...))
}

func moveSummaryQuery(
	filter *shipmentports.DispatchBoardFilter,
	now, dayStart int64,
) *moveQuery {
	asn := assignments
	ca := carrierAssignments
	pos := telematicsdb.VehiclePositions
	windowStart := origin.stop.ScheduledWindowStart

	uncovered := keel.And(asn.ID.IsNull(), ca.ID.IsNull())
	covered := keel.Or(asn.ID.IsNotNull(), ca.ID.IsNotNull())
	scheduled := keel.DynOf(windowStart).Coalesce(0).Gt(0)

	return boardMoves().
		LeftJoin(pos,
			pos.TractorID.EqCol(asn.TractorID),
			pos.RecordedAt.Gte(now-positionFreshnessSeconds),
		).
		Pick(
			countIf(uncovered),
			countIf(covered),
			countIf(uncovered, scheduled, keel.DynOf(windowStart).Lt(now)),
			countIf(
				uncovered,
				scheduled,
				keel.DynOf(windowStart).Gte(now),
				keel.DynOf(windowStart).Lte(now+atRiskLeadSeconds),
			),
			countIf(
				covered,
				keel.DynOf(asn.CreatedAt).Coalesce(ca.CreatedAt).Gte(dayStart),
			),
			keel.Raw(
				"coalesce(avg(ST_DistanceSphere(ST_MakePoint(?, ?), ST_MakePoint(?, ?)) / ?) "+
					"FILTER (WHERE ? IS NOT NULL AND ? IS NOT NULL "+
					"AND ? IS NOT NULL AND ? IS NOT NULL), 0)::float8",
				pos.Longitude, pos.Latitude,
				origin.loc.Longitude, origin.loc.Latitude,
				metersPerMile,
				asn.ID, pos.TractorID,
				origin.loc.Latitude, origin.loc.Longitude,
			),
		).
		Where(moveFilters(filter)...)
}

func driverSummaryQuery(filter *shipmentports.DispatchBoardFilter) *keel.Query[worker.Worker] {
	w := workerdb.Workers
	return w.
		Pick(
			keel.Count(),
			countIf(keel.DynOf(openAssignments(w)).Coalesce(0).Eq(0)),
		).
		Where(driverFilters(filter)...).
		Where(w.AvailableForDispatch.IsTrue())
}
