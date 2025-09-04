/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package worker

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/uptrace/bun"
)

func (wr *workerRepository) GetPTOCalendarData(
	ctx context.Context,
	req *repositories.PTOCalendarDataRequest,
) ([]*repositories.PTOCalendarEvent, error) {
	dba, err := wr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := wr.l.With().
		Str("operation", "GetPTOCalendarData").
		Interface("req", req).
		Logger()

	query := dba.NewSelect().
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
		ptoType, err := worker.PTOTypeFromString(req.Type)
		if err != nil {
			log.Error().
				Err(err).
				Str("type", req.Type).
				Msg("invalid PTO type")
			return nil, fmt.Errorf("invalid PTO type: %s", req.Type)
		}
		query = query.Where("wpto.type = ?", ptoType)
	}

	query = query.Order("wpto.start_date ASC", "wrk.last_name ASC", "wrk.first_name ASC")

	var events []*repositories.PTOCalendarEvent
	err = query.Scan(ctx, &events)
	if err != nil {
		log.Error().Err(err).Msg("failed to get PTO calendar data")
		return nil, err
	}

	return events, nil
}
