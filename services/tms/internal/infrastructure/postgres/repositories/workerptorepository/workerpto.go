package workerptorepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) repositories.WorkerPTORepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-pto-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListPTORequest,
) *bun.SelectQuery {
	log := r.l.With(
		zap.String("operation", "filterQuery"),
		zap.Any("request", req),
	)

	q = querybuilder.ApplyFilters(
		q,
		"wpto",
		req.Filter,
		(*worker.WorkerPTO)(nil),
	)

	if req.Status != "" {
		ptoStatus, err := worker.PTOStatusFromString(req.Status)
		if err != nil {
			log.Warn("failed to convert status to PTO status", zap.Error(err))
			return q
		}

		q = q.Where("wpto.status = ?", ptoStatus)
	}

	if req.Type != "" {
		ptoType, err := worker.PTOTypeFromString(req.Type)
		if err != nil {
			log.Warn("failed to convert type to PTO type", zap.Error(err))
			return q
		}

		q = q.Where("wpto.type = ?", ptoType)
	}

	if !req.WorkerID.IsNil() {
		q = q.Where("wpto.worker_id = ?", req.WorkerID)
	}

	if req.StartDateFrom > 0 {
		q = q.Where("wpto.start_date >= ?", req.StartDateFrom)
	}

	if req.StartDateTo > 0 {
		q = q.Where("wpto.start_date <= ?", req.StartDateTo)
	}

	if req.IncludeWorker {
		q = q.Relation("Worker")
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListPTORequest,
) (*pagination.ListResult[*worker.WorkerPTO], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*worker.WorkerPTO, 0, req.Filter.Pagination.SafeLimit())

	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count PTO records", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*worker.WorkerPTO]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetPTOByIDRequest,
) (*worker.WorkerPTO, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(worker.WorkerPTO)
	q := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wpto.id = ?", req.ID).
				Where("wpto.organization_id = ?", req.TenantInfo.OrgID).
				Where("wpto.business_unit_id = ?", req.TenantInfo.BuID)
		})

	if req.IncludeWorker {
		q = q.Relation("Worker")
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get PTO record", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "WorkerPTO")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *worker.WorkerPTO,
) (*worker.WorkerPTO, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("workerId", entity.WorkerID.String()),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create PTO record", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req *repositories.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	log := r.l.With(
		zap.String("operation", "UpdatePTOStatus"),
		zap.String("id", req.ID.String()),
		zap.String("status", string(req.Status)),
	)

	entity := new(worker.WorkerPTO)
	q := r.db.DB().
		NewUpdate().
		Model(entity).
		Set("status = ?", req.Status).
		WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
			return sq.Where("wpto.id = ?", req.ID).
				Where("wpto.organization_id = ?", req.TenantInfo.OrgID).
				Where("wpto.business_unit_id = ?", req.TenantInfo.BuID)
		})

	switch req.Status { //nolint:exhaustive // only two cases are needed
	case worker.PTOStatusApproved:
		q = q.Set("approver_id = ?", req.UserID)
	case worker.PTOStatusRejected:
		q = q.Set("rejector_id = ?", req.UserID)
	}

	if _, err := q.Returning("*").Exec(ctx); err != nil {
		log.Error("failed to update PTO status", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) upcomingPTODateBoundaries(
	req *repositories.ListUpcomingPTORequest,
	timezone string,
	log *zap.Logger,
) (startFloor, endCeiling int64) {
	startFloor = req.StartDate
	endCeiling = req.EndDate
	if req.StartDate > 0 {
		dayStart, err := timeutils.DayStartUnix(req.StartDate, timezone)
		if err != nil {
			log.Error(
				"failed to calculate start date day boundary, defaulting to raw timestamp",
				zap.Error(err),
				zap.String("timezone", timezone),
			)
		} else {
			startFloor = dayStart
		}
	}
	if req.EndDate > 0 {
		dayEnd, err := timeutils.DayEndUnix(req.EndDate, timezone)
		if err != nil {
			log.Error(
				"failed to calculate end date day boundary, defaulting to raw timestamp",
				zap.Error(err),
				zap.String("timezone", timezone),
			)
		} else {
			endCeiling = dayEnd
		}
	}
	return startFloor, endCeiling
}

type upcomingPTOFilterParsed struct {
	Type        worker.PTOType
	HasType     bool
	Status      worker.PTOStatus
	HasStatus   bool
	WorkerID    pulid.ID
	HasWorkerID bool
}

func (r *repository) parseUpcomingPTOFilterParams(
	req *repositories.ListUpcomingPTORequest,
	log *zap.Logger,
) (*upcomingPTOFilterParsed, error) {
	out := &upcomingPTOFilterParsed{}
	if req.Type != "" {
		ptoType, err := worker.PTOTypeFromString(req.Type)
		if err != nil {
			log.Error(
				"failed to convert type to PTO type, returning original query",
				zap.Error(err),
			)
			return nil, err
		}
		out.Type = ptoType
		out.HasType = true
	}
	if req.Status != "" {
		ptoStatus, err := worker.PTOStatusFromString(req.Status)
		if err != nil {
			log.Error(
				"failed to convert status to PTO status, returning original query",
				zap.Error(err),
			)
			return nil, err
		}
		out.Status = ptoStatus
		out.HasStatus = true
	}
	if req.WorkerID != "" {
		wrkID, err := pulid.MustParse(req.WorkerID)
		if err != nil {
			log.Error(
				"failed to parse worker ID, returning original query",
				zap.Error(err),
				zap.String("workerId", req.WorkerID),
			)
			return nil, err
		}
		out.WorkerID = wrkID
		out.HasWorkerID = true
	}
	return out, nil
}

func (r *repository) applyUpcomingPTOWhereGroup(
	q *bun.SelectQuery,
	req *repositories.ListUpcomingPTORequest,
	startFloor, endCeiling int64,
	parsed *upcomingPTOFilterParsed,
) *bun.SelectQuery {
	return q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		sq = sq.
			Where("wpto.organization_id = ?", req.Filter.TenantInfo.OrgID).
			Where("wpto.business_unit_id = ?", req.Filter.TenantInfo.BuID)
		if req.StartDate > 0 {
			sq = sq.Where("wpto.start_date >= ?", startFloor)
		}
		if req.EndDate > 0 {
			sq = sq.Where("wpto.end_date <= ?", endCeiling)
		}
		if parsed.HasType {
			sq = sq.Where("wpto.type = ?", parsed.Type)
		}
		if parsed.HasStatus {
			sq = sq.Where("wpto.status = ?", parsed.Status)
		}
		if parsed.HasWorkerID {
			sq = sq.Where("wpto.worker_id = ?", parsed.WorkerID)
		}
		return sq
	})
}

func (r *repository) workerRelationApplyForUpcoming(
	req *repositories.ListUpcomingPTORequest,
	log *zap.Logger,
) func(*bun.SelectQuery) *bun.SelectQuery {
	return func(sq *bun.SelectQuery) *bun.SelectQuery {
		if req.FleetCodeID == "" {
			return sq
		}
		fleetCodeID, err := pulid.MustParse(req.FleetCodeID)
		if err != nil {
			log.Error(
				"failed to parse fleet code ID, returning original query",
				zap.Error(err),
				zap.String("fleetCodeId", req.FleetCodeID),
			)
			return sq
		}
		return sq.Where("wrk.fleet_code_id = ?", fleetCodeID)
	}
}

func (r *repository) filterUpcomingPTOQuery(
	q *bun.SelectQuery,
	req *repositories.ListUpcomingPTORequest,
) *bun.SelectQuery {
	log := r.l.With(
		zap.String("operation", "filterUpcomingPTOQuery"),
		zap.Any("request", req),
	)
	timezone := timeutils.NormalizeTimezone(req.Timezone)
	startFloor, endCeiling := r.upcomingPTODateBoundaries(req, timezone, log)
	parsed, err := r.parseUpcomingPTOFilterParams(req, log)
	if err != nil {
		return q
	}
	q = r.applyUpcomingPTOWhereGroup(q, req, startFloor, endCeiling, parsed)
	q = q.Order("wpto.start_date ASC", "wpto.end_date ASC")
	q = q.RelationWithOpts("Worker", bun.RelationOpts{
		Apply: r.workerRelationApplyForUpcoming(req, log),
	})
	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) ListUpcoming(
	ctx context.Context,
	req *repositories.ListUpcomingPTORequest,
) (*pagination.ListResult[*worker.WorkerPTO], error) {
	log := r.l.With(
		zap.String("operation", "ListUpcoming"),
		zap.Any("request", req),
	)

	entities := make([]*worker.WorkerPTO, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterUpcomingPTOQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count upcoming PTOs", zap.Error(err), zap.Any("request", req))
		return nil, err
	}

	return &pagination.ListResult[*worker.WorkerPTO]{
		Items: entities,
		Total: total,
	}, nil
}

type ptoAggregateRow struct {
	Date    string `bun:"date"`
	Type    string `bun:"type"`
	Count   int    `bun:"count"`
	Workers string `bun:"workers"`
}

type chartFieldSetter struct {
	countSetter func(*repositories.PTOChartDataPoint, int)
	workerKey   string
}

var chartTypeSetters = map[worker.PTOType]chartFieldSetter{
	worker.PTOTypeVacation: {
		countSetter: func(dp *repositories.PTOChartDataPoint, count int) { dp.Vacation = count },
		workerKey:   worker.PTOTypeVacation.String(),
	},
	worker.PTOTypeSick: {
		countSetter: func(dp *repositories.PTOChartDataPoint, count int) { dp.Sick = count },
		workerKey:   worker.PTOTypeSick.String(),
	},
	worker.PTOTypeHoliday: {
		countSetter: func(dp *repositories.PTOChartDataPoint, count int) { dp.Holiday = count },
		workerKey:   worker.PTOTypeHoliday.String(),
	},
	worker.PTOTypeBereavement: {
		countSetter: func(dp *repositories.PTOChartDataPoint, count int) { dp.Bereavement = count },
		workerKey:   worker.PTOTypeBereavement.String(),
	},
	worker.PTOTypeMaternity: {
		countSetter: func(dp *repositories.PTOChartDataPoint, count int) { dp.Maternity = count },
		workerKey:   worker.PTOTypeMaternity.String(),
	},
	worker.PTOTypePaternity: {
		countSetter: func(dp *repositories.PTOChartDataPoint, count int) { dp.Paternity = count },
		workerKey:   worker.PTOTypePaternity.String(),
	},
	worker.PTOTypePersonal: {
		countSetter: func(dp *repositories.PTOChartDataPoint, count int) { dp.Personal = count },
		workerKey:   worker.PTOTypePersonal.String(),
	},
}

func (r *repository) GetChartData(
	ctx context.Context,
	req *repositories.PTOChartRequest,
) ([]*repositories.PTOChartDataPoint, error) {
	log := r.l.With(
		zap.String("operation", "GetPTOChart"),
		zap.Any("req", req),
	)

	db := r.db.DB()

	timezone := timeutils.NormalizeTimezone(req.Timezone)
	fromDayStartUnix, toDayEndUnix, err := r.calculateChartDateBoundaries(req, timezone)
	if err != nil {
		log.Error("failed to calculate chart date boundaries", zap.Error(err))
		return nil, err
	}

	dateSeries, err := r.generateDateSeries(ctx, db, req, timezone, log)
	if err != nil {
		return nil, err
	}

	ptoData, err := r.fetchPTOData(ctx, db, req, fromDayStartUnix, toDayEndUnix, log)
	if err != nil {
		return nil, err
	}

	ptoMap := r.aggregatePTOData(ptoData)

	return r.buildPTOChartDataQuery(dateSeries, ptoMap, log), nil
}

func (r *repository) generateDateSeries(
	ctx context.Context,
	db bun.IDB,
	req *repositories.PTOChartRequest,
	timezone string,
	log *zap.Logger,
) ([]string, error) {
	var dateSeries []string
	err := db.NewSelect().
		ColumnExpr("to_char(gs.day, 'YYYY-MM-DD')").
		TableExpr(
			`generate_series(
				date_trunc('day', to_timestamp(?::bigint) AT TIME ZONE ?),
				date_trunc('day', to_timestamp(?::bigint) AT TIME ZONE ?),
				interval '1 day'
			) AS gs(day)`,
			req.StartDateFrom,
			timezone,
			req.StartDateTo,
			timezone,
		).
		OrderExpr("gs.day ASC").
		Scan(ctx, &dateSeries)
	if err != nil {
		log.Error("failed to generate date series", zap.Error(err))
		return nil, err
	}

	return dateSeries, nil
}

func (r *repository) fetchPTOData(
	ctx context.Context,
	db bun.IDB,
	req *repositories.PTOChartRequest,
	fromDayStartUnix int64,
	toDayEndUnix int64,
	log *zap.Logger,
) ([]ptoAggregateRow, error) {
	q := r.buildPTOQuery(db, req, fromDayStartUnix, toDayEndUnix)

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
	req *repositories.PTOChartRequest,
	fromDayStartUnix int64,
	toDayEndUnix int64,
) *bun.SelectQuery {
	timezone := timeutils.NormalizeTimezone(req.Timezone)

	return db.NewSelect().
		ColumnExpr("to_char(day_series.day, 'YYYY-MM-DD') as date").
		ColumnExpr("wpto.type::text as type").
		ColumnExpr("COUNT(DISTINCT wpto.worker_id)::int as count").
		ColumnExpr("json_agg(DISTINCT jsonb_build_object('id', wrk.id, 'firstName', wrk.first_name, 'lastName', wrk.last_name, 'ptoType', wpto.type::text))::text as workers").
		TableExpr("worker_pto wpto").
		Join("INNER JOIN workers wrk ON wpto.worker_id = wrk.id AND wpto.organization_id = wrk.organization_id AND wpto.business_unit_id = wrk.business_unit_id").
		Join(
			`CROSS JOIN LATERAL generate_series(
				GREATEST(
					date_trunc('day', to_timestamp(wpto.start_date) AT TIME ZONE ?),
					date_trunc('day', to_timestamp(?::bigint) AT TIME ZONE ?)
				),
				LEAST(
					date_trunc('day', to_timestamp(wpto.end_date) AT TIME ZONE ?),
					date_trunc('day', to_timestamp(?::bigint) AT TIME ZONE ?)
				),
				interval '1 day'
			) AS day_series(day)`,
			timezone,
			req.StartDateFrom,
			timezone,
			timezone,
			req.StartDateTo,
			timezone,
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wpto.organization_id = ?", req.Filter.TenantInfo.OrgID).
				Where("wpto.business_unit_id = ?", req.Filter.TenantInfo.BuID).
				Where("wpto.status = ?", worker.PTOStatusApproved).
				WhereGroup(" AND ", func(innerSq *bun.SelectQuery) *bun.SelectQuery {
					return innerSq.
						Where("wpto.end_date >= ?", fromDayStartUnix).
						Where("wpto.start_date <= ?", toDayEndUnix)
				})
		})
}

func (r *repository) calculateChartDateBoundaries(
	req *repositories.PTOChartRequest,
	timezone string,
) (fromDayStartUnix, toDayEndUnix int64, err error) {
	fromDayStartUnix, err = timeutils.DayStartUnix(req.StartDateFrom, timezone)
	if err != nil {
		return 0, 0, fmt.Errorf("calculate start day boundary: %w", err)
	}
	toDayEndUnix, err = timeutils.DayEndUnix(req.StartDateTo, timezone)
	if err != nil {
		return 0, 0, fmt.Errorf("calculate end day boundary: %w", err)
	}
	return fromDayStartUnix, toDayEndUnix, nil
}

func (r *repository) applyPTOFilters(
	q *bun.SelectQuery,
	req *repositories.PTOChartRequest,
	log *zap.Logger,
) error {
	if req.Type != "" && !strings.EqualFold(req.Type, "all") {
		ptoType, err := worker.PTOTypeFromString(req.Type)
		if err != nil {
			log.Error("invalid PTO type", zap.Error(err), zap.String("type", req.Type))
			return errortypes.NewValidationError(
				"type",
				errortypes.ErrInvalid,
				"type must be one of: all, Personal, Vacation, Sick, Holiday, Bereavement, Maternity, Paternity",
			)
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

			return errortypes.NewValidationError(
				"workerId",
				errortypes.ErrInvalidFormat,
				"workerId must be a valid ID",
			)
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
			Workers: make(map[string][]repositories.PTOChartWorker),
		}
		for _, ptoType := range []worker.PTOType{
			worker.PTOTypeVacation,
			worker.PTOTypeSick,
			worker.PTOTypeHoliday,
			worker.PTOTypeBereavement,
			worker.PTOTypeMaternity,
			worker.PTOTypePaternity,
			worker.PTOTypePersonal,
		} {
			dataPoint.Workers[ptoType.String()] = []repositories.PTOChartWorker{}
		}

		if dayData, exists := ptoMap[date]; exists {
			for rawType, row := range dayData {
				var workers []repositories.PTOChartWorker
				if err := sonic.UnmarshalString(row.Workers, &workers); err != nil {
					log.Error(
						"failed to parse worker details",
						zap.Error(err),
						zap.String("date", date),
						zap.String("type", rawType),
					)

					workers = []repositories.PTOChartWorker{}
				}

				ptoType, err := worker.PTOTypeFromString(rawType)
				if err != nil {
					log.Warn("unsupported PTO type in chart data", zap.String("type", rawType))
					continue
				}

				setter, ok := chartTypeSetters[ptoType]
				if !ok {
					log.Warn("missing chart type setter", zap.String("type", rawType))
					continue
				}

				setter.countSetter(dataPoint, row.Count)
				dataPoint.Workers[setter.workerKey] = workers
			}
		}

		result = append(result, dataPoint)
	}

	return result
}
