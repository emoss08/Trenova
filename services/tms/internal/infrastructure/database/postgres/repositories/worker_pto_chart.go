/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/bytedance/sonic"
)

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

	// Generate date series first to ensure we have all dates in the range
	var dateSeries []string
	_, err = dba.NewRaw(`
		WITH RECURSIVE date_series AS (
			SELECT to_timestamp(?::bigint) AT TIME ZONE 'UTC' AS date
			UNION ALL
			SELECT date + INTERVAL '1 day'
			FROM date_series
			WHERE date < to_timestamp(?::bigint) AT TIME ZONE 'UTC'
		)
		SELECT to_char(date, 'YYYY-MM-DD') as date_str
		FROM date_series
		ORDER BY date
	`, req.StartDate, req.EndDate).
		Exec(ctx, &dateSeries)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate date series")
		return nil, err
	}

	// Get PTO data aggregated by date and type with worker details
	type ptoAggregateRow struct {
		Date      string `bun:"date"`
		Type      string `bun:"type"`
		Count     int    `bun:"count"`
		Workers   string `bun:"workers"` // JSON array of worker details
	}

	var ptoData []ptoAggregateRow
	var queryArgs []interface{}
	baseQuery := `
		SELECT 
			to_char(to_timestamp(wpto.start_date) AT TIME ZONE 'UTC', 'YYYY-MM-DD') as date,
			wpto.type::text as type,
			COUNT(*)::int as count,
			json_agg(json_build_object(
				'id', wrk.id,
				'firstName', wrk.first_name,
				'lastName', wrk.last_name,
				'ptoType', wpto.type::text
			)) as workers
		FROM worker_pto wpto
		INNER JOIN workers wrk ON wpto.worker_id = wrk.id
		WHERE wpto.organization_id = ?
			AND wpto.business_unit_id = ?
			AND wpto.start_date >= ?
			AND wpto.start_date <= ?
			AND wpto.status = 'Approved'`

	queryArgs = append(queryArgs, req.Filter.TenantOpts.OrgID, req.Filter.TenantOpts.BuID, req.StartDate, req.EndDate)

	// Add type filter if specified
	if req.Type != "" && req.Type != "all" {
		baseQuery += " AND wpto.type = ?"
		queryArgs = append(queryArgs, req.Type)
	}

	baseQuery += `
		GROUP BY date, wpto.type
		ORDER BY date, wpto.type`

	err = dba.NewRaw(baseQuery, queryArgs...).
		Scan(ctx, &ptoData)
	if err != nil {
		log.Error().Err(err).Msg("failed to get PTO chart data")
		return nil, err
	}

	// Create a map for quick lookup of PTO data by date and type
	ptoMap := make(map[string]map[string]ptoAggregateRow)
	for _, row := range ptoData {
		if ptoMap[row.Date] == nil {
			ptoMap[row.Date] = make(map[string]ptoAggregateRow)
		}
		ptoMap[row.Date][row.Type] = row
	}

	// Build the result with all dates, filling in zeros where no data exists
	result := make([]*repositories.PTOChartDataPoint, 0, len(dateSeries))
	for _, date := range dateSeries {
		dataPoint := &repositories.PTOChartDataPoint{
			Date:    date,
			Workers: make(map[string][]repositories.WorkerDetail),
		}

		if dayData, exists := ptoMap[date]; exists {
			for ptoType, row := range dayData {
				// Parse workers JSON - using sonic for better performance
				var workers []repositories.WorkerDetail
				if err := sonic.UnmarshalString(row.Workers, &workers); err != nil {
					log.Warn().Err(err).Str("date", date).Str("type", ptoType).Msg("failed to parse worker details")
					workers = []repositories.WorkerDetail{}
				}

				// Set count and workers for this PTO type
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

	log.Debug().Int("count", len(result)).Msg("retrieved PTO chart data")
	return result, nil
}