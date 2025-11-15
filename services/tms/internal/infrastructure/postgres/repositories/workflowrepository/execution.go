package workflowrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ExecutionParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type executionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewExecutionRepository(p ExecutionParams) repositories.WorkflowExecutionRepository {
	return &executionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.workflow-execution-repository"),
	}
}

func (r *executionRepository) List(
	ctx context.Context,
	req *repositories.ListWorkflowExecutionRequest,
) (*pagination.ListResult[*workflow.WorkflowExecution], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*workflow.WorkflowExecution, 0, req.Filter.Limit)

	total, err := db.NewSelect().
		Model(&entities).
		Relation("Workflow").
		Relation("WorkflowVersion").
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan workflow executions", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*workflow.WorkflowExecution]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *executionRepository) GetByID(
	ctx context.Context,
	req repositories.GetWorkflowExecutionByIDRequest,
) (*workflow.WorkflowExecution, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(workflow.WorkflowExecution)
	err = db.NewSelect().
		Model(entity).
		Relation("Workflow").
		Relation("WorkflowVersion").
		Relation("Steps", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order("wfxs.step_number ASC")
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("wfx.id = ?", req.ID).
				Where("wfx.organization_id = ?", req.OrgID).
				Where("wfx.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkflowExecution")
	}

	return entity, nil
}

func (r *executionRepository) Create(
	ctx context.Context,
	entity *workflow.WorkflowExecution,
) (*workflow.WorkflowExecution, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("executionID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert workflow execution", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *executionRepository) Update(
	ctx context.Context,
	entity *workflow.WorkflowExecution,
) (*workflow.WorkflowExecution, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("executionID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	_, err = db.NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update workflow execution", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// Step management

func (r *executionRepository) CreateStep(
	ctx context.Context,
	entity *workflow.WorkflowExecutionStep,
) (*workflow.WorkflowExecutionStep, error) {
	log := r.l.With(
		zap.String("operation", "CreateStep"),
		zap.String("stepID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert workflow execution step", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *executionRepository) UpdateStep(
	ctx context.Context,
	entity *workflow.WorkflowExecutionStep,
) (*workflow.WorkflowExecutionStep, error) {
	log := r.l.With(
		zap.String("operation", "UpdateStep"),
		zap.String("stepID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	_, err = db.NewUpdate().
		Model(entity).
		WherePK().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update workflow execution step", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *executionRepository) GetStepsByExecutionID(
	ctx context.Context,
	executionID, orgID, buID pulid.ID,
) ([]*workflow.WorkflowExecutionStep, error) {
	log := r.l.With(
		zap.String("operation", "GetStepsByExecutionID"),
		zap.String("executionID", executionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	steps := make([]*workflow.WorkflowExecutionStep, 0)
	err = db.NewSelect().
		Model(&steps).
		Where("wfxs.execution_id = ?", executionID).
		Where("wfxs.organization_id = ?", orgID).
		Where("wfxs.business_unit_id = ?", buID).
		Order("wfxs.step_number ASC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get workflow execution steps", zap.Error(err))
		return nil, err
	}

	return steps, nil
}

// Status management

func (r *executionRepository) UpdateStatus(
	ctx context.Context,
	id, orgID, buID pulid.ID,
	status workflow.ExecutionStatus,
) error {
	log := r.l.With(
		zap.String("operation", "UpdateStatus"),
		zap.String("executionID", id.String()),
		zap.String("status", status.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewUpdate().
		Model((*workflow.WorkflowExecution)(nil)).
		Set("status = ?", status).
		Where("id = ?", id).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to update workflow execution status", zap.Error(err))
		return err
	}

	return nil
}

func (r *executionRepository) CancelExecution(
	ctx context.Context,
	id, orgID, buID pulid.ID,
) error {
	return r.UpdateStatus(ctx, id, orgID, buID, workflow.ExecutionStatusCanceled)
}

// Temporal integration

func (r *executionRepository) GetByTemporalWorkflowID(
	ctx context.Context,
	temporalWorkflowID string,
	orgID, buID pulid.ID,
) (*workflow.WorkflowExecution, error) {
	log := r.l.With(
		zap.String("operation", "GetByTemporalWorkflowID"),
		zap.String("temporalWorkflowID", temporalWorkflowID),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(workflow.WorkflowExecution)
	err = db.NewSelect().
		Model(entity).
		Where("wfx.temporal_workflow_id = ?", temporalWorkflowID).
		Where("wfx.organization_id = ?", orgID).
		Where("wfx.business_unit_id = ?", buID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkflowExecution")
	}

	return entity, nil
}

func (r *executionRepository) UpdateTemporalInfo(
	ctx context.Context,
	id, orgID, buID pulid.ID,
	temporalWorkflowID, temporalRunID string,
) error {
	log := r.l.With(
		zap.String("operation", "UpdateTemporalInfo"),
		zap.String("executionID", id.String()),
		zap.String("temporalWorkflowID", temporalWorkflowID),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewUpdate().
		Model((*workflow.WorkflowExecution)(nil)).
		Set("temporal_workflow_id = ?", temporalWorkflowID).
		Set("temporal_run_id = ?", temporalRunID).
		Where("id = ?", id).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to update temporal info", zap.Error(err))
		return err
	}

	return nil
}
