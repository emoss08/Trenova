package workerrepository

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type ptoAggregateRow struct {
	Date    string `bun:"date"`
	Type    string `bun:"type"`
	Count   int    `bun:"count"`
	Workers string `bun:"workers"`
}

func (r *repository) GetPTOChartData(
	ctx context.Context,
	req *repositories.PTOChartDataRequest,
) ([]*repositories.PTOChartDataPoint, error) {
	log := r.l.With(
		zap.String("operation", "GetPTOChartData"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	timezone := r.normalizeTimezone(req.Timezone)

	dateSeries, err := r.generateDateSeries(ctx, db, req, timezone, log)
	if err != nil {
		return nil, err
	}

	ptoData, err := r.fetchPTOData(ctx, db, req, timezone, log)
	if err != nil {
		return nil, err
	}

	ptoMap := r.aggregatePTOData(ptoData)

	return r.buildPTOChartDataQuery(dateSeries, ptoMap, log), nil
}

func (r *repository) normalizeTimezone(timezone string) string {
	if timezone == "" {
		return "UTC"
	}
	return timezone
}

func (r *repository) generateDateSeries(
	ctx context.Context,
	db bun.IDB,
	req *repositories.PTOChartDataRequest,
	timezone string,
	log *zap.Logger,
) ([]string, error) {
	var dateSeries []string
	cte := db.NewSelect().
		WithRecursive("date_series",
			db.NewSelect().
				ColumnExpr("date_trunc('day', to_timestamp(?::bigint) AT TIME ZONE ?) AS date", req.StartDate, timezone).
				UnionAll(
					db.NewSelect().
						ColumnExpr("date + INTERVAL '1 day' AS date").
						TableExpr("date_series").
						Where("date < date_trunc('day', to_timestamp(?::bigint) AT TIME ZONE ?)", req.EndDate, timezone),
				),
		).
		ColumnExpr("to_char(date, 'YYYY-MM-DD') AS date_str").
		Order("date").
		Table("date_series")

	_, err := cte.Exec(ctx, &dateSeries)
	if err != nil {
		log.Error("failed to generate date series", zap.Error(err))
		return nil, err
	}

	return dateSeries, nil
}

func (r *repository) fetchPTOData(
	ctx context.Context,
	db bun.IDB,
	req *repositories.PTOChartDataRequest,
	timezone string,
	log *zap.Logger,
) ([]ptoAggregateRow, error) {
	q := r.buildPTOQuery(db, req, timezone)

	if err := r.applyPTOFilters(q, req, log); err != nil {
		return nil, err
	}

	q = q.Group("date", "wpto.type")
	q = q.Order("date", "wpto.type")

	var ptoData []ptoAggregateRow
	err := q.Scan(ctx, &ptoData)
	if err != nil {
		log.Error("failed to get PTO chart data", zap.Error(err))
		return nil, err
	}

	return ptoData, nil
}

func (r *repository) buildPTOQuery(
	db bun.IDB,
	req *repositories.PTOChartDataRequest,
	timezone string,
) *bun.SelectQuery {
	return db.NewSelect().
		ColumnExpr("to_char(to_timestamp(wpto.start_date) AT TIME ZONE ?, 'YYYY-MM-DD') as date", timezone).
		ColumnExpr("wpto.type::text as type").
		ColumnExpr("COUNT(*)::int as count").
		ColumnExpr("json_agg(json_build_object('id', wrk.id, 'firstName', wrk.first_name, 'lastName', wrk.last_name, 'ptoType', wpto.type::text)) as workers").
		TableExpr("worker_pto wpto").
		Join("INNER JOIN workers wrk ON wpto.worker_id = wrk.id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wpto.organization_id = ?", req.Filter.TenantOpts.OrgID).
				Where("wpto.business_unit_id = ?", req.Filter.TenantOpts.BuID).
				Where("wpto.status = ?", worker.PTOStatusApproved).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Where(`
						date_trunc('day', to_timestamp(wpto.start_date) AT TIME ZONE ?) >=
						date_trunc('day', to_timestamp(?) AT TIME ZONE ?)`,
						timezone, req.StartDate, timezone).
						Where(`
						date_trunc('day', to_timestamp(wpto.start_date) AT TIME ZONE ?) <=
						date_trunc('day', to_timestamp(?) AT TIME ZONE ?)`,
							timezone, req.EndDate, timezone)
				})
		})
}

func (r *repository) applyPTOFilters(
	q *bun.SelectQuery,
	req *repositories.PTOChartDataRequest,
	log *zap.Logger,
) error {
	if req.Type != "" && req.Type != "all" {
		ptoType, err := worker.PTOTypeFromString(req.Type)
		if err != nil {
			log.Error("invalid PTO type", zap.Error(err), zap.String("type", req.Type))
			return err
		}
		q.Where("wpto.type = ?", ptoType)
	}

	if req.WorkerID != "" {
		wrkID, err := pulid.MustParse(req.WorkerID)
		if err != nil {
			log.Error(
				"failed to parse worker ID",
				zap.Error(err),
				zap.String("workerId", req.WorkerID),
			)
			return err
		}
		q.Where("wpto.worker_id = ?", wrkID)
	}

	return nil
}

func (r *repository) aggregatePTOData(
	ptoData []ptoAggregateRow,
) map[string]map[string]ptoAggregateRow {
	ptoMap := make(map[string]map[string]ptoAggregateRow)
	for _, row := range ptoData {
		if ptoMap[row.Date] == nil {
			ptoMap[row.Date] = make(map[string]ptoAggregateRow)
		}
		ptoMap[row.Date][row.Type] = row
	}
	return ptoMap
}

func (r *repository) buildPTOChartDataQuery(
	dateSeries []string,
	ptoMap map[string]map[string]ptoAggregateRow,
	log *zap.Logger,
) []*repositories.PTOChartDataPoint {
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
					log.Error(
						"failed to parse worker details",
						zap.Error(err),
						zap.String("date", date),
						zap.String("type", ptoType),
					)
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
				case "Personal":
					dataPoint.Personal = row.Count
					dataPoint.Workers["Personal"] = workers
				}
			}
		}

		result = append(result, dataPoint)
	}

	return result
}
