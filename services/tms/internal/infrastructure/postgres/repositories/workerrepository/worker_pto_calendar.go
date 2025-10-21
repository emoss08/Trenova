package workerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (r *repository) GetPTOCalendarData(
	ctx context.Context,
	req *repositories.PTOCalendarDataRequest,
) ([]*repositories.PTOCalendarEvent, error) {
	log := r.l.With(
		zap.String("operation", "GetPTOCalendarData"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	query := db.NewSelect().
		ColumnExpr("CONCAT(wrk.first_name, ' ', wrk.last_name) as worker_name").
		ColumnExpr("wpto.worker_id as worker_id").
		ColumnExpr("wpto.id as id").
		ColumnExpr("wpto.start_date as start_date").
		ColumnExpr("wpto.end_date as end_date").
		ColumnExpr("wpto.type as type").
		ColumnExpr("wpto.status as status").
		ColumnExpr("wpto.reason as reason").
		TableExpr("worker_pto wpto").
		Join("INNER JOIN workers wrk ON wpto.worker_id = wrk.id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wpto.organization_id = ?", req.Filter.TenantOpts.OrgID).
				Where("wpto.business_unit_id = ?", req.Filter.TenantOpts.BuID).
				Where("wpto.status = ?", worker.PTOStatusApproved).
				WhereOr("wpto.start_date <= ? AND wpto.end_date >= ?", req.StartDate, req.EndDate).
				WhereOr("wpto.start_date >= ? AND wpto.start_date <= ?", req.StartDate, req.EndDate).
				WhereOr("wpto.end_date >= ? AND wpto.end_date <= ?", req.StartDate, req.EndDate)
		})

	if req.Type != "" && req.Type != "all" {
		ptoType, ptoErr := worker.PTOTypeFromString(req.Type)
		if ptoErr != nil {
			log.Error("invalid PTO type", zap.Error(ptoErr), zap.String("type", req.Type))
			return nil, ptoErr
		}
		query = query.Where("wpto.type = ?", ptoType)
	}

	query = query.Order("wpto.start_date ASC", "wrk.last_name ASC", "wrk.first_name ASC")

	var events []*repositories.PTOCalendarEvent
	err = query.Scan(ctx, &events)
	if err != nil {
		log.Error("failed to get PTO calendar data", zap.Error(err))
		return nil, err
	}

	return events, nil
}
