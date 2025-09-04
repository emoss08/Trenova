/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package worker

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

type ptoAggregateRow struct {
	Date    string `bun:"date"`
	Type    string `bun:"type"`
	Count   int    `bun:"count"`
	Workers string `bun:"workers"`
}

func (wr *workerRepository) GetPTOChartData(
	ctx context.Context,
	req *repositories.PTOChartDataRequest,
) ([]*repositories.PTOChartDataPoint, error) {
	dba, err := wr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := wr.l.With().
		Str("operation", "GetPTOChartData").
		Interface("req", req).
		Logger()

	var dateSeries []string
	cte := dba.NewSelect().
		WithRecursive("date_series",
			dba.NewSelect().
				ColumnExpr("to_timestamp(?::bigint) AT TIME ZONE 'UTC' AS date", req.StartDate).
				UnionAll(
					dba.NewSelect().
						ColumnExpr("date + INTERVAL '1 day' AS date").
						TableExpr("date_series").
						Where("date < to_timestamp(?::bigint) AT TIME ZONE 'UTC'", req.EndDate),
				),
		).
		ColumnExpr("to_char(date, 'YYYY-MM-DD') AS date_str").
		Order("date").
		Table("date_series")
	_, err = cte.Exec(ctx, &dateSeries)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate date series")
		return nil, err
	}

	var ptoData []ptoAggregateRow
	q := dba.NewSelect().
		ColumnExpr("to_char(to_timestamp(wpto.start_date) AT TIME ZONE 'UTC', 'YYYY-MM-DD') as date").
		ColumnExpr("wpto.type::text as type").
		ColumnExpr("COUNT(*)::int as count").
		ColumnExpr("json_agg(json_build_object('id', wrk.id, 'firstName', wrk.first_name, 'lastName', wrk.last_name, 'ptoType', wpto.type::text)) as workers").
		TableExpr("worker_pto wpto").
		Join("INNER JOIN workers wrk ON wpto.worker_id = wrk.id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wpto.organization_id = ?", req.Filter.TenantOpts.OrgID).
				Where("wpto.business_unit_id = ?", req.Filter.TenantOpts.BuID).
				Where("wpto.status = 'Approved'").
				Where("wpto.start_date >= ?", req.StartDate).
				Where("wpto.start_date <= ?", req.EndDate)
		})

	if req.Type != "" && req.Type != "all" {
		ptoType, err := worker.PTOTypeFromString(req.Type)
		if err != nil {
			log.Error().
				Err(err).
				Str("type", req.Type).
				Msg("invalid PTO type")
			return nil, err
		}

		q = q.Where("wpto.type = ?", ptoType)
	}

	q = q.Group("date", "wpto.type")
	q = q.Order("date", "wpto.type")

	err = q.Scan(ctx, &ptoData)
	if err != nil {
		log.Error().Err(err).Msg("failed to get PTO chart data")
		return nil, err
	}

	ptoMap := make(map[string]map[string]ptoAggregateRow)
	for _, row := range ptoData {
		if ptoMap[row.Date] == nil {
			ptoMap[row.Date] = make(map[string]ptoAggregateRow)
		}
		ptoMap[row.Date][row.Type] = row
	}

	result, err := wr.buildPTOChartDataQuery(dateSeries, ptoMap, &log)
	if err != nil {
		log.Error().Err(err).Msg("failed to build PTO chart data")
		return nil, err
	}

	log.Debug().Int("count", len(result)).Msg("retrieved PTO chart data")
	return result, nil
}

func (wr *workerRepository) buildPTOChartDataQuery(
	dateSeries []string,
	ptoMap map[string]map[string]ptoAggregateRow,
	log *zerolog.Logger,
) ([]*repositories.PTOChartDataPoint, error) {
	result := make([]*repositories.PTOChartDataPoint, 0, len(dateSeries))
	for _, date := range dateSeries {
		dataPoint := &repositories.PTOChartDataPoint{
			Date:    date,
			Workers: make(map[string][]repositories.WorkerDetail),
		}

		if dayData, exists := ptoMap[date]; exists {
			for ptoType, row := range dayData {
				var workers []repositories.WorkerDetail
				if err := sonic.UnmarshalString(row.Workers, &workers); err != nil {
					log.Warn().
						Err(err).
						Str("date", date).
						Str("type", ptoType).
						Msg("failed to parse worker details")
					workers = []repositories.WorkerDetail{}
				}

				switch ptoType {
				case "Vacation":
					dataPoint.Vacation = row.Count
					dataPoint.Workers["Vacation"] = workers
				case "Sick":
					dataPoint.Sick = row.Count
					dataPoint.Workers["Sick"] = workers
				case "Holiday":
					dataPoint.Holiday = row.Count
					dataPoint.Workers["Holiday"] = workers
				case "Bereavement":
					dataPoint.Bereavement = row.Count
					dataPoint.Workers["Bereavement"] = workers
				case "Maternity":
					dataPoint.Maternity = row.Count
					dataPoint.Workers["Maternity"] = workers
				case "Paternity":
					dataPoint.Paternity = row.Count
					dataPoint.Workers["Paternity"] = workers
				}
			}
		}

		result = append(result, dataPoint)
	}

	return result, nil
}
