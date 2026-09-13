package workertrainingrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultReminderPageSize = 500
	secondsPerDay           = int64(86400)
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

func New(p Params) repositories.WorkerTrainingRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-training-repository"),
	}
}

func duplicateCode() error {
	return errortypes.NewValidationError(
		"code",
		errortypes.ErrDuplicate,
		"A course with this code already exists",
	)
}

func duplicateOpenRecord() error {
	return errortypes.NewValidationError(
		"courseId",
		errortypes.ErrDuplicate,
		"This worker already has this course assigned",
	)
}

func orderCourses(sq *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.TrainingCourseColumns
	return sq.Order(cols.SortOrder.OrderAsc()).Order(cols.Name.OrderAsc())
}

func (r *repository) applyCourseFilters(
	q *bun.SelectQuery,
	req *repositories.ListTrainingCoursesRequest,
) *bun.SelectQuery {
	cols := buncolgen.TrainingCourseColumns
	if req.Status != "" {
		q = q.Where(cols.Status.Eq(), req.Status)
	}
	if req.Category != "" {
		q = q.Where(cols.Category.Eq(), req.Category)
	}
	return q
}

func (r *repository) ListCourses(
	ctx context.Context,
	req *repositories.ListTrainingCoursesRequest,
) (*pagination.CursorListResult[*worker.TrainingCourse], error) {
	log := r.l.With(zap.String("operation", "ListCourses"))

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*worker.TrainingCourse)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.TrainingCourseTable.Alias,
					req.Filter,
					(*worker.TrainingCourse)(nil),
				)
				return r.applyCourseFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count training courses", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*worker.TrainingCourse]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(items *[]*worker.TrainingCourse) *bun.SelectQuery {
				return dba.NewSelect().
					Model(items).
					ColumnExpr(buncolgen.TrainingCourseTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.TrainingCourseTable.Alias,
					req.Filter,
					req.Cursor,
					(*worker.TrainingCourse)(nil),
				)
				if applyErr != nil {
					return sq, applyErr
				}
				return r.applyCourseFilters(sq, req), nil
			},
		},
	)
	if err != nil {
		log.Error("failed to list training courses", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) ListActiveCourses(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.TrainingCourse, error) {
	cols := buncolgen.TrainingCourseColumns
	entities := make([]*worker.TrainingCourse, 0, 16)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TrainingCourseScopeTenant(sq, tenantInfo).
				Where(cols.Status.Eq(), domaintypes.StatusActive)
		}).
		Apply(orderCourses).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list active training courses", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) CourseSelectOptions(
	ctx context.Context,
	req *repositories.TrainingCourseSelectOptionsRequest,
) (*pagination.ListResult[*worker.TrainingCourse], error) {
	cols := buncolgen.TrainingCourseColumns
	return dbhelper.SelectOptions[*worker.TrainingCourse](
		ctx,
		r.db.DBForContext(ctx),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.Code,
				cols.Name,
				cols.Description,
				cols.Category,
				cols.Delivery,
				cols.DurationMinutes,
				cols.PassingScore,
				cols.DueDaysAfterAssignment,
				cols.SortOrder,
				cols.CreatedAt,
			},
			OrgColumnRef: &cols.OrganizationID,
			BuColumnRef:  &cols.BusinessUnitID,
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return orderCourses(q.Where(cols.Status.Eq(), domaintypes.StatusActive))
			},
			EntityName: "TrainingCourse",
			SearchColumnRefs: []buncolgen.Column{
				cols.Code,
				cols.Name,
				cols.Description,
			},
		},
	)
}

func (r *repository) GetCourseByID(
	ctx context.Context,
	req *repositories.GetTrainingCourseByIDRequest,
) (*worker.TrainingCourse, error) {
	entity := new(worker.TrainingCourse)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TrainingCourseScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.TrainingCourseColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "TrainingCourse")
	}

	return entity, nil
}

func (r *repository) CourseCodeExists(
	ctx context.Context,
	req *repositories.TrainingCourseCodeExistsRequest,
) (bool, error) {
	cols := buncolgen.TrainingCourseColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.TrainingCourse)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TrainingCourseScopeTenant(sq, req.TenantInfo).
				Where("LOWER("+cols.Code.String()+") = ?", strings.ToLower(req.Code))
		})
	if !req.ExcludeID.IsNil() {
		q = q.Where(cols.ID.Ne(), req.ExcludeID)
	}

	exists, err := q.Exists(ctx)
	if err != nil {
		r.l.Error("failed to check training course code", zap.Error(err))
		return false, err
	}

	return exists, nil
}

func (r *repository) CreateCourse(
	ctx context.Context,
	entity *worker.TrainingCourse,
) (*worker.TrainingCourse, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateCode()
		}
		r.l.Error("failed to create training course", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateCourse(
	ctx context.Context,
	entity *worker.TrainingCourse,
) (*worker.TrainingCourse, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.TrainingCourseColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateCode()
		}
		r.l.Error("failed to update training course", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "TrainingCourse", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) CountRecordsByCourse(
	ctx context.Context,
	req *repositories.CountTrainingRecordsRequest,
) (int, error) {
	cols := buncolgen.WorkerTrainingRecordColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerTrainingRecord)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerTrainingRecordScopeTenant(sq, req.TenantInfo).
				Where(cols.CourseID.Eq(), req.CourseID)
		})
	if req.OpenOnly {
		q = q.Where(cols.Status.In(), bun.In([]worker.TrainingStatus{
			worker.TrainingStatusAssigned,
			worker.TrainingStatusInProgress,
		}))
	}

	count, err := q.Count(ctx)
	if err != nil {
		r.l.Error("failed to count training records", zap.Error(err))
		return 0, err
	}

	return count, nil
}

func (r *repository) CountRecordsByCourseIDs(
	ctx context.Context,
	req *repositories.CountTrainingRecordsByCourseIDsRequest,
) (map[pulid.ID]int, error) {
	if len(req.CourseIDs) == 0 {
		return map[pulid.ID]int{}, nil
	}

	cols := buncolgen.WorkerTrainingRecordColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerTrainingRecord)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerTrainingRecordScopeTenant(sq, req.TenantInfo).
				Where(cols.CourseID.In(), bun.In(req.CourseIDs))
		})
	if req.OpenOnly {
		q = q.Where(cols.Status.In(), bun.In([]worker.TrainingStatus{
			worker.TrainingStatusAssigned,
			worker.TrainingStatusInProgress,
		}))
	}

	counts, err := dbhelper.CountByID(ctx, q, cols.CourseID, len(req.CourseIDs))
	if err != nil {
		r.l.Error("failed to count training records by course", zap.Error(err))
		return nil, err
	}

	return counts, nil
}

func (r *repository) ListForWorker(
	ctx context.Context,
	req *repositories.ListWorkerTrainingRequest,
) ([]*worker.WorkerTrainingRecord, error) {
	cols := buncolgen.WorkerTrainingRecordColumns
	entities := make([]*worker.WorkerTrainingRecord, 0, 16)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerTrainingRecordScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID)
			if !req.IncludeClosed {
				sq = sq.Where(cols.Status.In(), bun.In([]worker.TrainingStatus{
					worker.TrainingStatusAssigned,
					worker.TrainingStatusInProgress,
				}))
			}
			return sq
		}).
		Order(cols.AssignedAt.OrderDesc()).
		Order(cols.CreatedAt.OrderDesc())
	if req.IncludeCourse {
		q = q.Relation(buncolgen.WorkerTrainingRecordRelations.Course)
	}
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerTrainingRecordRelations.Document)
	}
	if req.IncludeActors {
		q = q.Relation(buncolgen.WorkerTrainingRecordRelations.AssignedBy).
			Relation(buncolgen.WorkerTrainingRecordRelations.RecordedBy)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list worker training records", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetWorkerTrainingByIDRequest,
) (*worker.WorkerTrainingRecord, error) {
	entity := new(worker.WorkerTrainingRecord)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerTrainingRecordScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerTrainingRecordColumns.ID.Eq(), req.ID)
		})
	if req.IncludeCourse {
		q = q.Relation(buncolgen.WorkerTrainingRecordRelations.Course)
	}
	if req.IncludeWorker {
		q = q.Relation(
			buncolgen.WorkerTrainingRecordRelations.Worker,
			func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation(buncolgen.WorkerRelations.Profile)
			},
		)
	}
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerTrainingRecordRelations.Document)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerTrainingRecord")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *worker.WorkerTrainingRecord,
) (*worker.WorkerTrainingRecord, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateOpenRecord()
		}
		r.l.Error("failed to create worker training record", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *worker.WorkerTrainingRecord,
) (*worker.WorkerTrainingRecord, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerTrainingRecordColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateOpenRecord()
		}
		r.l.Error("failed to update worker training record", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "WorkerTrainingRecord", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) reminderQuery(
	ctx context.Context,
	req *repositories.ListTrainingRemindersRequest,
	entities *[]*worker.WorkerTrainingRecord,
) *bun.SelectQuery {
	cols := buncolgen.WorkerTrainingRecordColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultReminderPageSize
	}
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entities).
		Relation(buncolgen.WorkerTrainingRecordRelations.Course).
		Relation(buncolgen.WorkerTrainingRecordRelations.Worker, func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Relation(buncolgen.WorkerRelations.Profile)
		}).
		Where(buncolgen.WorkerColumns.Status.WithAlias("worker").Eq(), domaintypes.StatusActive).
		Where(buncolgen.TrainingCourseColumns.Status.WithAlias("course").Eq(), domaintypes.StatusActive).
		Order(cols.ID.OrderAsc()).
		Limit(limit)
	if !req.TenantInfo.OrgID.IsNil() {
		q = q.Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID)
	}
	if !req.TenantInfo.BuID.IsNil() {
		q = q.Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID)
	}
	if !req.AfterID.IsNil() {
		q = q.Where(cols.ID.Gt(), req.AfterID)
	}
	return q
}

func reminderWindow(req *repositories.ListTrainingRemindersRequest) (int64, int64) {
	now := timeutils.NowUnix()
	return now - int64(req.GraceDays)*secondsPerDay, now + int64(req.HorizonDays)*secondsPerDay
}

func (r *repository) ListDue(
	ctx context.Context,
	req *repositories.ListTrainingRemindersRequest,
) ([]*worker.WorkerTrainingRecord, error) {
	cols := buncolgen.WorkerTrainingRecordColumns
	grace, horizon := reminderWindow(req)
	entities := make([]*worker.WorkerTrainingRecord, 0, defaultReminderPageSize)
	q := r.reminderQuery(ctx, req, &entities).
		Where(cols.Status.In(), bun.In([]worker.TrainingStatus{
			worker.TrainingStatusAssigned,
			worker.TrainingStatusInProgress,
		})).
		Where(cols.DueAt.Between(), grace, horizon)

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list due training", zap.Error(err))
		return nil, fmt.Errorf("list due training: %w", err)
	}

	return entities, nil
}

func (r *repository) ListExpiring(
	ctx context.Context,
	req *repositories.ListTrainingRemindersRequest,
) ([]*worker.WorkerTrainingRecord, error) {
	cols := buncolgen.WorkerTrainingRecordColumns
	grace, horizon := reminderWindow(req)
	entities := make([]*worker.WorkerTrainingRecord, 0, defaultReminderPageSize)
	q := r.reminderQuery(ctx, req, &entities).
		Where(cols.Status.Eq(), worker.TrainingStatusCompleted).
		Where(cols.ExpiresAt.Between(), grace, horizon)

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list expiring training", zap.Error(err))
		return nil, fmt.Errorf("list expiring training: %w", err)
	}

	return entities, nil
}
