package workersafetyrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"go.uber.org/zap"
)

var violationErrors = dberror.Mapping{
	Entity: "WorkerSafetyViolation",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_worker_safety_violations_event",
			Field:   "safetyEventId",
			Message: "Safety event does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_worker_safety_violations_worker",
			Field:   "workerId",
			Message: "Worker does not exist or belongs to a different organization",
		},
	},
}

func violationsQuery(
	req *workerports.ListWorkerSafetyViolationsRequest,
) *keel.Query[worker.WorkerSafetyViolation] {
	v := workerdb.WorkerSafetyViolations

	return v.Where(
		keel.When(len(req.SafetyEventIDs) > 0, v.SafetyEventID.In(req.SafetyEventIDs...)),
		keel.When(req.WorkerID.IsNotNil(), v.WorkerID.Eq(req.WorkerID)),
		keel.When(req.Since > 0, v.SafetyEvent().OccurredAt.Gte(req.Since)),
	).
		Load(listquery.When(req.IncludeEvent, v.SafetyEvent())...).
		Order(v.Basic.Asc(), v.SeverityWeight.Desc())
}

func (r *repository) ListViolations(
	ctx context.Context,
	req *workerports.ListWorkerSafetyViolationsRequest,
) ([]*worker.WorkerSafetyViolation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.WorkerSafetyViolation, error) {
		entities, err := violationsQuery(req).All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list worker safety violations", zap.Error(err))
			return nil, fmt.Errorf("list worker safety violations: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetViolationByID(
	ctx context.Context,
	req *workerports.GetWorkerSafetyViolationByIDRequest,
) (*worker.WorkerSafetyViolation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.WorkerSafetyViolation, error) {
		entity, err := workerdb.WorkerSafetyViolations.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, violationErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateViolation(
	ctx context.Context,
	entity *worker.WorkerSafetyViolation,
) (*worker.WorkerSafetyViolation, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerSafetyViolation, error) {
		if err := workerdb.WorkerSafetyViolations.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create worker safety violation", zap.Error(err))
			return nil, violationErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateViolation(
	ctx context.Context,
	entity *worker.WorkerSafetyViolation,
) (*worker.WorkerSafetyViolation, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerSafetyViolation, error) {
		if err := workerdb.WorkerSafetyViolations.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update worker safety violation", zap.Error(err))
			return nil, violationErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) DeleteViolation(
	ctx context.Context,
	req *workerports.GetWorkerSafetyViolationByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		v := workerdb.WorkerSafetyViolations
		rows, err := v.Where(v.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete worker safety violation", zap.Error(err))
			return violationErrors.Map(err, req.ID.String())
		}

		return dberror.CheckAffected(rows, violationErrors.Entity, req.ID.String())
	})
}
