package workerptorepository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	chartDateLayout   = "2006-01-02"
	chartDayFormat    = "YYYY-MM-DD"
	chartDaySeries    = "day_series"
	chartDaySeriesDay = "day"
	chartTruncUnit    = "day"
	chartDayStep      = "1 day"
)

var ptoErrors = dberror.Mapping{Entity: "WorkerPTO"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workerports.WorkerPTORepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-pto-repository"),
	}
}

func (r *repository) listFilter(req *workerports.ListPTORequest) []keel.Pred {
	p := workerdb.WorkerPTOs
	preds := make([]keel.Pred, 0, 5)

	if req.Status != "" {
		status, err := worker.PTOStatusFromString(req.Status)
		if err != nil {
			r.l.Warn("failed to convert status to PTO status", zap.Error(err))
		} else {
			preds = append(preds, p.Status.Eq(status))
		}
	}
	if req.Type != "" {
		ptoType, err := worker.PTOTypeFromString(req.Type)
		if err != nil {
			r.l.Warn("failed to convert type to PTO type", zap.Error(err))
		} else {
			preds = append(preds, p.Type.Eq(ptoType))
		}
	}
	if req.WorkerID.IsNotNil() {
		preds = append(preds, p.WorkerID.Eq(req.WorkerID))
	}
	if req.StartDateFrom > 0 {
		if req.Overlapping {
			preds = append(preds, p.EndDate.Gte(req.StartDateFrom))
		} else {
			preds = append(preds, p.StartDate.Gte(req.StartDateFrom))
		}
	}
	if req.StartDateTo > 0 {
		preds = append(preds, p.StartDate.Lte(req.StartDateTo))
	}

	return preds
}

func (r *repository) List(
	ctx context.Context,
	req *workerports.ListPTORequest,
) (*pagination.CursorListResult[*worker.WorkerPTO], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*worker.WorkerPTO], error) {
			p := workerdb.WorkerPTOs
			result, err := listquery.Connection(ctx, r.db.Keel(ctx), p.Where(r.listFilter(req)...),
				&listquery.Request[worker.WorkerPTO]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(q *keel.Query[worker.WorkerPTO]) *keel.Query[worker.WorkerPTO] {
						return q.Load(listquery.When(req.IncludeWorker, p.Worker())...)
					},
				})
			if err != nil {
				r.l.Error("failed to list PTO records", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *workerports.GetPTOByIDRequest,
) (*worker.WorkerPTO, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.WorkerPTO, error) {
		p := workerdb.WorkerPTOs
		entity, err := p.Where(p.ID.Eq(req.ID)).
			Load(listquery.When(req.IncludeWorker, p.Worker())...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, ptoErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *worker.WorkerPTO,
) (*worker.WorkerPTO, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerPTO, error) {
		if err := workerdb.WorkerPTOs.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to create PTO record",
				zap.String("workerId", entity.WorkerID.String()),
				zap.Error(err),
			)
			return nil, ptoErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *worker.WorkerPTO,
) (*worker.WorkerPTO, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerPTO, error) {
		p := workerdb.WorkerPTOs
		rows, err := p.Where(
			p.ID.Eq(entity.ID),
			p.Version.Eq(entity.Version),
			p.Status.Eq(worker.PTOStatusRequested),
		).UpdateReturning(ctx, r.db.Keel(ctx), p.Type.To(entity.Type), p.StartDate.To(entity.StartDate), p.EndDate.To(entity.EndDate), p.Reason.To(entity.Reason))
		if err != nil {
			r.l.Error(
				"failed to update PTO record",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, ptoErrors.Map(err, entity.ID.String())
		}
		if len(rows) == 0 {
			return nil, dberror.CreateVersionMismatchError("WorkerPTO", entity.ID.String())
		}

		return rows[0], nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req *workerports.GetPTOsByIDsRequest,
) ([]*worker.WorkerPTO, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.WorkerPTO, error) {
		if len(req.IDs) == 0 {
			return []*worker.WorkerPTO{}, nil
		}

		p := workerdb.WorkerPTOs
		entities, err := p.Where(p.ID.In(req.IDs...)).
			Limit(len(req.IDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get PTO records", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) HasOverlap(
	ctx context.Context,
	req *workerports.PTOOverlapRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		p := workerdb.WorkerPTOs
		exists, err := p.Where(
			p.WorkerID.Eq(req.WorkerID),
			p.Status.In(worker.PTOStatusRequested, worker.PTOStatusApproved),
			p.StartDate.Lte(req.EndDate),
			p.EndDate.Gte(req.StartDate),
			keel.When(req.ExcludeID.IsNotNil(), p.ID.Ne(req.ExcludeID)),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to check PTO overlap",
				zap.String("workerId", req.WorkerID.String()),
				zap.Error(err),
			)
			return false, err
		}

		return exists, nil
	})
}

func (r *repository) SetBalanceAfter(
	ctx context.Context,
	req *workerports.SetPTOBalanceAfterRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := workerdb.WorkerPTOs
		_, err := p.Where(p.ID.Eq(req.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			p.BalanceAfterDays.To(decimal.NewNullDecimal(req.BalanceAfterDays)),
			p.Version.Add(0),
		)
		if err != nil {
			r.l.Error("failed to set PTO balance after", zap.Error(err))
			return err
		}

		return nil
	})
}

func statusAssignments(req *workerports.UpdatePTOStatusRequest) []keel.Assignment {
	p := workerdb.WorkerPTOs
	sets := []keel.Assignment{p.Status.To(req.Status)}

	switch req.Status { //nolint:exhaustive // Requested is never a transition target
	case worker.PTOStatusApproved:
		sets = append(sets, p.ApproverID.To(req.UserID))
	case worker.PTOStatusRejected:
		sets = append(sets, p.RejectorID.To(req.UserID), p.RejectionReason.To(req.Reason))
	case worker.PTOStatusCancelled:
		sets = append(sets, p.CancelledByID.To(req.UserID), p.CancellationReason.To(req.Reason))
	}

	return sets
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req *workerports.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerPTO, error) {
		sources := worker.PTOStatusSourcesFor(req.Status)
		if len(sources) == 0 {
			return nil, errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"PTO cannot be moved to {0}", req.Status,
			)
		}

		p := workerdb.WorkerPTOs
		rows, err := p.Where(
			p.ID.Eq(req.ID),
			p.Status.In(sources...),
			keel.When(req.ExpectedVersion > 0, p.Version.Eq(req.ExpectedVersion)),
		).UpdateReturning(ctx, r.db.Keel(ctx), statusAssignments(req)...)
		if err != nil {
			r.l.Error(
				"failed to update PTO status",
				zap.String("id", req.ID.String()),
				zap.String("status", string(req.Status)),
				zap.Error(err),
			)
			return nil, err
		}
		if len(rows) == 0 {
			return nil, r.explainStatusConflict(ctx, req, sources)
		}

		return rows[0], nil
	})
}

func (r *repository) explainStatusConflict(
	ctx context.Context,
	req *workerports.UpdatePTOStatusRequest,
	sources []worker.PTOStatus,
) error {
	current, err := r.GetByID(ctx, &workerports.GetPTOByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}

	if req.ExpectedVersion > 0 && current.Version != req.ExpectedVersion {
		return dberror.CreateVersionMismatchError("WorkerPTO", req.ID.String())
	}

	labels := make([]string, 0, len(sources))
	for _, src := range sources {
		labels = append(labels, string(src))
	}

	return errortypes.NewValidationError(
		"status",
		errortypes.ErrInvalidOperation,
		"PTO is {0} and can only be {1} from {2}",
		strings.ToLower(string(current.Status)),
		strings.ToLower(string(req.Status)),
		strings.Join(labels, " or "),
	)
}

func (r *repository) upcomingPTODateBoundaries(
	req *workerports.ListUpcomingPTORequest,
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
	req *workerports.ListUpcomingPTORequest,
) (*upcomingPTOFilterParsed, error) {
	out := &upcomingPTOFilterParsed{}
	if req.Type != "" {
		ptoType, err := worker.PTOTypeFromString(req.Type)
		if err != nil {
			r.l.Error("failed to convert type to PTO type", zap.Error(err))
			return nil, err
		}
		out.Type = ptoType
		out.HasType = true
	}

	if req.Status != "" {
		ptoStatus, err := worker.PTOStatusFromString(req.Status)
		if err != nil {
			r.l.Error("failed to convert status to PTO status", zap.Error(err))
			return nil, err
		}
		out.Status = ptoStatus
		out.HasStatus = true
	}

	if req.WorkerID != "" {
		wrkID, err := pulid.MustParse(req.WorkerID)
		if err != nil {
			r.l.Error(
				"failed to parse worker ID",
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

func (r *repository) upcomingFilter(req *workerports.ListUpcomingPTORequest) []keel.Pred {
	log := r.l.With(zap.String("operation", "ListUpcoming"))
	timezone := timeutils.NormalizeTimezone(req.Timezone)
	startFloor, endCeiling := r.upcomingPTODateBoundaries(req, timezone, log)

	parsed, err := r.parseUpcomingPTOFilterParams(req)
	if err != nil {
		return nil
	}

	p := workerdb.WorkerPTOs
	preds := []keel.Pred{
		keel.When(req.StartDate > 0, p.StartDate.Gte(startFloor)),
		keel.When(req.EndDate > 0, p.EndDate.Lte(endCeiling)),
		keel.When(parsed.HasType, p.Type.Eq(parsed.Type)),
		keel.When(parsed.HasStatus, p.Status.Eq(parsed.Status)),
		keel.When(parsed.HasWorkerID, p.WorkerID.Eq(parsed.WorkerID)),
	}

	if req.FleetCodeID != "" {
		fleetCodeID, parseErr := pulid.MustParse(req.FleetCodeID)
		if parseErr != nil {
			log.Error(
				"failed to parse fleet code ID",
				zap.Error(parseErr),
				zap.String("fleetCodeId", req.FleetCodeID),
			)
		} else {
			preds = append(preds, p.Worker().FleetCodeID.Eq(fleetCodeID))
		}
	}

	return preds
}

func upcomingPTOCursorFilter(req *workerports.ListUpcomingPTORequest) *pagination.QueryOptions {
	filter := pagination.QueryOptions{}
	if req.Filter != nil {
		filter = *req.Filter
	}
	filter.Sort = []domaintypes.SortField{
		{
			Field:     "startDate",
			Direction: dbtype.SortDirectionAsc,
		},
		{
			Field:     "endDate",
			Direction: dbtype.SortDirectionAsc,
		},
	}

	return &filter
}

func (r *repository) ListUpcoming(
	ctx context.Context,
	req *workerports.ListUpcomingPTORequest,
) (*pagination.CursorListResult[*worker.WorkerPTO], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*worker.WorkerPTO], error) {
			p := workerdb.WorkerPTOs
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				p.Where(r.upcomingFilter(req)...),
				&listquery.Request[worker.WorkerPTO]{
					Filter: upcomingPTOCursorFilter(req),
					Cursor: req.Cursor,
					Shape: func(q *keel.Query[worker.WorkerPTO]) *keel.Query[worker.WorkerPTO] {
						return q.Load(p.Worker())
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list upcoming PTOs", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

type ptoDayRow struct {
	Date      string
	Type      worker.PTOType
	WorkerID  pulid.ID
	FirstName string
	LastName  string
}

type ptoAggregate struct {
	Count   int
	Workers []workerports.PTOChartWorker
}

type chartFieldSetter struct {
	countSetter func(*workerports.PTOChartDataPoint, int)
	workerKey   string
}

var chartTypeSetters = map[worker.PTOType]chartFieldSetter{
	worker.PTOTypeVacation: {
		countSetter: func(dp *workerports.PTOChartDataPoint, count int) { dp.Vacation = count },
		workerKey:   worker.PTOTypeVacation.String(),
	},
	worker.PTOTypeSick: {
		countSetter: func(dp *workerports.PTOChartDataPoint, count int) { dp.Sick = count },
		workerKey:   worker.PTOTypeSick.String(),
	},
	worker.PTOTypeHoliday: {
		countSetter: func(dp *workerports.PTOChartDataPoint, count int) { dp.Holiday = count },
		workerKey:   worker.PTOTypeHoliday.String(),
	},
	worker.PTOTypeBereavement: {
		countSetter: func(dp *workerports.PTOChartDataPoint, count int) { dp.Bereavement = count },
		workerKey:   worker.PTOTypeBereavement.String(),
	},
	worker.PTOTypeMaternity: {
		countSetter: func(dp *workerports.PTOChartDataPoint, count int) { dp.Maternity = count },
		workerKey:   worker.PTOTypeMaternity.String(),
	},
	worker.PTOTypePaternity: {
		countSetter: func(dp *workerports.PTOChartDataPoint, count int) { dp.Paternity = count },
		workerKey:   worker.PTOTypePaternity.String(),
	},
	worker.PTOTypePersonal: {
		countSetter: func(dp *workerports.PTOChartDataPoint, count int) { dp.Personal = count },
		workerKey:   worker.PTOTypePersonal.String(),
	},
}

func (r *repository) GetChartData(
	ctx context.Context,
	req *workerports.PTOChartRequest,
) ([]*workerports.PTOChartDataPoint, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workerports.PTOChartDataPoint, error) {
			log := r.l.With(zap.String("operation", "GetPTOChart"))
			timezone := timeutils.NormalizeTimezone(req.Timezone)
			fromDayStartUnix, toDayEndUnix, err := r.calculateChartDateBoundaries(req, timezone)
			if err != nil {
				log.Error("failed to calculate chart date boundaries", zap.Error(err))
				return nil, err
			}

			ptoType, workerID, err := chartFilters(req)
			if err != nil {
				log.Error("invalid PTO chart filter", zap.Error(err))
				return nil, err
			}

			loc, err := time.LoadLocation(timezone)
			if err != nil {
				log.Error("failed to load chart timezone", zap.Error(err))
				return nil, fmt.Errorf("load timezone %q: %w", timezone, err)
			}

			ptoData, err := keel.Into[ptoDayRow](
				ctx,
				r.db.Keel(ctx),
				chartDaysQuery(&chartDaysParams{
					timezone:     timezone,
					from:         req.StartDateFrom,
					to:           req.StartDateTo,
					overlapStart: fromDayStartUnix,
					overlapEnd:   toDayEndUnix,
					ptoType:      ptoType,
					workerID:     workerID,
				}),
			)
			if err != nil {
				log.Error("failed to get PTO chart data", zap.Error(err))
				return nil, err
			}

			return r.buildPTOChartDataQuery(
				chartDates(req.StartDateFrom, req.StartDateTo, loc),
				r.aggregatePTOData(ptoData),
			), nil
		},
	)
}

type chartDaysParams struct {
	timezone     string
	from         int64
	to           int64
	overlapStart int64
	overlapEnd   int64
	ptoType      worker.PTOType
	workerID     pulid.ID
}

func chartDaysQuery(p *chartDaysParams) *keel.Query[worker.WorkerPTO] {
	pto := workerdb.WorkerPTOs
	wrk := workerdb.Workers
	localDay := func(ts any) keel.DynExpr {
		return keel.DynFunc(
			"date_trunc",
			keel.DynConst(chartTruncUnit),
			keel.DynFunc("timezone", p.timezone, keel.DynFunc("to_timestamp", ts)),
		)
	}
	series := keel.Rows(
		chartDaySeries,
		[]string{chartDaySeriesDay},
		keel.DynFunc(
			"generate_series",
			keel.DynFunc(
				"greatest",
				localDay(pto.StartDate),
				localDay(keel.DynLit(p.from).Cast(keel.TypeInt64)),
			),
			keel.DynFunc(
				"least",
				localDay(pto.EndDate),
				localDay(keel.DynLit(p.to).Cast(keel.TypeInt64)),
			),
			keel.DynConst(chartDayStep),
		),
	)
	day := keel.DynFunc("to_char", series.Col(chartDaySeriesDay), chartDayFormat)

	return pto.
		Join(wrk, wrk.ID.EqCol(pto.WorkerID)).
		JoinLateral(series).
		DistinctOn(day, pto.Type, pto.WorkerID).
		Pick(day, pto.Type, pto.WorkerID, wrk.FirstName, wrk.LastName).
		Where(
			pto.Status.Eq(worker.PTOStatusApproved),
			pto.EndDate.Gte(p.overlapStart),
			pto.StartDate.Lte(p.overlapEnd),
			keel.When(p.ptoType != "", pto.Type.Eq(p.ptoType)),
			keel.When(!p.workerID.IsNil(), pto.WorkerID.Eq(p.workerID)),
		).
		Order(day.Asc(), pto.Type.Asc(), pto.WorkerID.Asc())
}

func chartDates(from, to int64, loc *time.Location) []string {
	start := calendarDay(from, loc)
	end := calendarDay(to, loc)
	dates := make([]string, 0)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		dates = append(dates, day.Format(chartDateLayout))
	}

	return dates
}

func calendarDay(ts int64, loc *time.Location) time.Time {
	local := time.Unix(ts, 0).In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

func chartFilters(
	req *workerports.PTOChartRequest,
) (ptoType worker.PTOType, workerID pulid.ID, err error) {
	if req.Type != "" && !strings.EqualFold(req.Type, "all") {
		parsed, parseErr := worker.PTOTypeFromString(req.Type)
		if parseErr != nil {
			return "", pulid.Nil, errortypes.NewValidationError(
				"type",
				errortypes.ErrInvalid,
				"type must be one of: all, Personal, Vacation, Sick, Holiday, Bereavement, Maternity, Paternity",
			)
		}
		ptoType = parsed
	}

	if req.WorkerID != "" {
		parsed, parseErr := pulid.MustParse(req.WorkerID)
		if parseErr != nil {
			return "", pulid.Nil, errortypes.NewValidationError(
				"workerId",
				errortypes.ErrInvalidFormat,
				"workerId must be a valid ID",
			)
		}
		workerID = parsed
	}

	return ptoType, workerID, nil
}

func (r *repository) calculateChartDateBoundaries(
	req *workerports.PTOChartRequest,
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

func (r *repository) aggregatePTOData(
	rows []ptoDayRow,
) map[string]map[worker.PTOType]*ptoAggregate {
	ptoMap := make(map[string]map[worker.PTOType]*ptoAggregate)
	for _, row := range rows {
		byType := ptoMap[row.Date]
		if byType == nil {
			byType = make(map[worker.PTOType]*ptoAggregate)
			ptoMap[row.Date] = byType
		}
		agg := byType[row.Type]
		if agg == nil {
			agg = &ptoAggregate{}
			byType[row.Type] = agg
		}
		agg.Count++
		agg.Workers = append(agg.Workers, workerports.PTOChartWorker{
			ID:        row.WorkerID.String(),
			FirstName: row.FirstName,
			LastName:  row.LastName,
			PTOType:   row.Type.String(),
		})
	}

	return ptoMap
}

func (r *repository) buildPTOChartDataQuery(
	dateSeries []string,
	ptoMap map[string]map[worker.PTOType]*ptoAggregate,
) []*workerports.PTOChartDataPoint {
	result := make([]*workerports.PTOChartDataPoint, 0, len(dateSeries))
	for _, date := range dateSeries {
		dataPoint := &workerports.PTOChartDataPoint{
			Date:    date,
			Workers: make(map[string][]workerports.PTOChartWorker),
		}
		for ptoType := range chartTypeSetters {
			dataPoint.Workers[ptoType.String()] = []workerports.PTOChartWorker{}
		}

		for ptoType, agg := range ptoMap[date] {
			setter, ok := chartTypeSetters[ptoType]
			if !ok {
				r.l.Warn("missing chart type setter", zap.String("type", ptoType.String()))
				continue
			}

			setter.countSetter(dataPoint, agg.Count)
			dataPoint.Workers[setter.workerKey] = agg.Workers
		}

		result = append(result, dataPoint)
	}

	return result
}
