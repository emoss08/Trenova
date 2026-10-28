package workertrainingrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining/workertrainingdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workertrainingports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultReminderPageSize = 500
	secondsPerDay           = int64(86400)
)

var courseErrors = dberror.Mapping{Entity: "TrainingCourse"}

var recordErrors = dberror.Mapping{Entity: "WorkerTrainingRecord"}

var openTrainingStatuses = []workertraining.TrainingStatus{
	workertraining.TrainingStatusAssigned,
	workertraining.TrainingStatusInProgress,
}

type courseCount struct {
	CourseID pulid.ID
	Count    int64
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workertrainingports.WorkerTrainingRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-training-repository"),
	}
}

func courseOrder() []keel.Order {
	c := workertrainingdb.TrainingCourses
	return []keel.Order{c.SortOrder.Asc(), c.Name.Asc()}
}

func (r *repository) ListCourses(
	ctx context.Context,
	req *workertrainingports.ListTrainingCoursesRequest,
) (*pagination.CursorListResult[*workertraining.TrainingCourse], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*workertraining.TrainingCourse], error) {
			c := workertrainingdb.TrainingCourses
			q := c.Where(
				keel.When(req.Status != "", c.Status.Eq(domaintypes.Status(req.Status))),
				keel.When(
					req.Category != "",
					c.Category.Eq(workertraining.TrainingCategory(req.Category)),
				),
			)

			result, err := listquery.Connection(ctx, r.db.Keel(ctx), q,
				&listquery.Request[workertraining.TrainingCourse]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				})
			if err != nil {
				r.l.Error("failed to list training courses", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListActiveCourses(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*workertraining.TrainingCourse, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workertraining.TrainingCourse, error) {
			c := workertrainingdb.TrainingCourses
			entities, err := c.Where(c.Status.Eq(domaintypes.StatusActive)).
				Order(courseOrder()...).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list active training courses", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) CourseSelectOptions(
	ctx context.Context,
	req *workertrainingports.TrainingCourseSelectOptionsRequest,
) (*pagination.ListResult[*workertraining.TrainingCourse], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*workertraining.TrainingCourse], error) {
			c := workertrainingdb.TrainingCourses
			q := c.Pick(
				c.ID,
				c.Code,
				c.Name,
				c.Description,
				c.Category,
				c.Delivery,
				c.DurationMinutes,
				c.PassingScore,
				c.DueDaysAfterAssignment,
				c.ValidityMonths,
				c.SortOrder,
				c.CreatedAt,
			).
				Where(c.Status.Eq(domaintypes.StatusActive)).
				Order(courseOrder()...)

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						c.Code.ContainsFold(s),
						c.Name.ContainsFold(s),
						c.Description.ContainsFold(s),
					)
				},
			)
		},
	)
}

func (r *repository) GetCourseByID(
	ctx context.Context,
	req *workertrainingports.GetTrainingCourseByIDRequest,
) (*workertraining.TrainingCourse, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*workertraining.TrainingCourse, error) {
		entity, err := workertrainingdb.TrainingCourses.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, courseErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CourseCodeExists(
	ctx context.Context,
	req *workertrainingports.TrainingCourseCodeExistsRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		c := workertrainingdb.TrainingCourses
		exists, err := c.Where(
			c.Code.Lower().Eq(strings.ToLower(req.Code)),
			keel.When(req.ExcludeID.IsNotNil(), c.ID.Ne(req.ExcludeID)),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to check training course code", zap.Error(err))
			return false, err
		}

		return exists, nil
	})
}

func (r *repository) CreateCourse(
	ctx context.Context,
	entity *workertraining.TrainingCourse,
) (*workertraining.TrainingCourse, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*workertraining.TrainingCourse, error) {
		if err := workertrainingdb.TrainingCourses.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create training course", zap.Error(err))
			return nil, courseErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateCourse(
	ctx context.Context,
	entity *workertraining.TrainingCourse,
) (*workertraining.TrainingCourse, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*workertraining.TrainingCourse, error) {
		if err := workertrainingdb.TrainingCourses.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update training course", zap.Error(err))
			return nil, courseErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CountRecordsByCourse(
	ctx context.Context,
	req *workertrainingports.CountTrainingRecordsRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		tr := workertrainingdb.WorkerTrainingRecords
		count, err := tr.Where(
			tr.CourseID.Eq(req.CourseID),
			keel.When(req.OpenOnly, tr.Status.In(openTrainingStatuses...)),
		).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count training records", zap.Error(err))
			return 0, err
		}

		return int(count), nil
	})
}

func (r *repository) CountRecordsByCourseIDs(
	ctx context.Context,
	req *workertrainingports.CountTrainingRecordsByCourseIDsRequest,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		if len(req.CourseIDs) == 0 {
			return map[pulid.ID]int{}, nil
		}

		tr := workertrainingdb.WorkerTrainingRecords
		rows, err := keel.Into[courseCount](ctx, r.db.Keel(ctx), tr.
			Group(tr.CourseID).
			Pick(tr.CourseID, keel.Count()).
			Where(
				tr.CourseID.In(req.CourseIDs...),
				keel.When(req.OpenOnly, tr.Status.In(openTrainingStatuses...)),
			))
		if err != nil {
			r.l.Error("failed to count training records by course", zap.Error(err))
			return nil, err
		}

		counts := make(map[pulid.ID]int, len(rows))
		for _, row := range rows {
			counts[row.CourseID] = int(row.Count)
		}

		return counts, nil
	})
}

func (r *repository) ListForWorker(
	ctx context.Context,
	req *workertrainingports.ListWorkerTrainingRequest,
) ([]*workertraining.WorkerTrainingRecord, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workertraining.WorkerTrainingRecord, error) {
			tr := workertrainingdb.WorkerTrainingRecords
			loads := make([]keel.Loader, 0, 4)
			loads = append(loads, listquery.When(req.IncludeCourse, tr.Course())...)
			loads = append(loads, listquery.When(req.IncludeDocument, tr.Document())...)
			loads = append(
				loads,
				listquery.When(req.IncludeActors, tr.AssignedBy(), tr.RecordedBy())...)

			entities, err := tr.Where(
				tr.WorkerID.Eq(req.WorkerID),
				keel.When(!req.IncludeClosed, tr.Status.In(openTrainingStatuses...)),
			).
				Load(loads...).
				Order(tr.AssignedAt.Desc(), tr.CreatedAt.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list worker training records", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *workertrainingports.GetWorkerTrainingByIDRequest,
) (*workertraining.WorkerTrainingRecord, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*workertraining.WorkerTrainingRecord, error) {
			tr := workertrainingdb.WorkerTrainingRecords
			loads := make([]keel.Loader, 0, 3)
			loads = append(loads, listquery.When(req.IncludeCourse, tr.Course())...)
			loads = append(loads, listquery.When(req.IncludeWorker, tr.Worker().Profile())...)
			loads = append(loads, listquery.When(req.IncludeDocument, tr.Document())...)

			entity, err := tr.Where(tr.ID.Eq(req.ID)).Load(loads...).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, recordErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *workertraining.WorkerTrainingRecord,
) (*workertraining.WorkerTrainingRecord, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workertraining.WorkerTrainingRecord, error) {
			if err := workertrainingdb.WorkerTrainingRecords.Create(ctx, r.db.Keel(ctx), entity); err != nil {
				r.l.Error("failed to create worker training record", zap.Error(err))
				return nil, recordErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *workertraining.WorkerTrainingRecord,
) (*workertraining.WorkerTrainingRecord, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workertraining.WorkerTrainingRecord, error) {
			if err := workertrainingdb.WorkerTrainingRecords.Save(ctx, r.db.Keel(ctx), entity); err != nil {
				r.l.Error("failed to update worker training record", zap.Error(err))
				return nil, recordErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func reminderQuery(
	ctx context.Context,
	req *workertrainingports.ListTrainingRemindersRequest,
	window keel.Pred,
) *keel.Query[*workertraining.WorkerTrainingRecord] {
	tr := workertrainingdb.WorkerTrainingRecords
	limit := req.Limit
	if limit <= 0 {
		limit = defaultReminderPageSize
	}

	q := tr.Where(
		window,
		tr.Worker().Status.Eq(domaintypes.StatusActive),
		tr.Course().Status.Eq(domaintypes.StatusActive),
		keel.When(req.TenantInfo.OrgID.IsNotNil(), tr.OrganizationID.Eq(req.TenantInfo.OrgID)),
		keel.When(req.TenantInfo.BuID.IsNotNil(), tr.BusinessUnitID.Eq(req.TenantInfo.BuID)),
		keel.When(req.AfterID.IsNotNil(), tr.ID.Gt(req.AfterID)),
	).
		Load(tr.Course(), tr.Worker().Profile()).
		Order(tr.ID.Asc()).
		Limit(limit)
	if dbscope.IsSystem(ctx) {
		q = q.AllTenants()
	}

	return q
}

func reminderWindow(req *workertrainingports.ListTrainingRemindersRequest) (int64, int64) {
	now := timeutils.NowUnix()
	return now - int64(req.GraceDays)*secondsPerDay, now + int64(req.HorizonDays)*secondsPerDay
}

func (r *repository) ListDue(
	ctx context.Context,
	req *workertrainingports.ListTrainingRemindersRequest,
) ([]*workertraining.WorkerTrainingRecord, error) {
	if req.TenantInfo.OrgID.IsNil() {
		ctx = dbscope.WithSystem(
			ctx,
			"list training due across every organization for the reminder sweep",
		)
	}
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workertraining.WorkerTrainingRecord, error) {
			tr := workertrainingdb.WorkerTrainingRecords
			grace, horizon := reminderWindow(req)
			window := keel.And(
				tr.Status.In(openTrainingStatuses...),
				tr.DueAt.Between(grace, horizon),
			)

			entities, err := reminderQuery(ctx, req, window).List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list due training", zap.Error(err))
				return nil, fmt.Errorf("list due training: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) ListExpiring(
	ctx context.Context,
	req *workertrainingports.ListTrainingRemindersRequest,
) ([]*workertraining.WorkerTrainingRecord, error) {
	if req.TenantInfo.OrgID.IsNil() {
		ctx = dbscope.WithSystem(
			ctx,
			"list expiring training across every organization for the reminder sweep",
		)
	}
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workertraining.WorkerTrainingRecord, error) {
			tr := workertrainingdb.WorkerTrainingRecords
			grace, horizon := reminderWindow(req)
			window := keel.And(
				tr.Status.Eq(workertraining.TrainingStatusCompleted),
				tr.ExpiresAt.Between(grace, horizon),
			)

			entities, err := reminderQuery(ctx, req, window).List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list expiring training", zap.Error(err))
				return nil, fmt.Errorf("list expiring training: %w", err)
			}

			return entities, nil
		},
	)
}
