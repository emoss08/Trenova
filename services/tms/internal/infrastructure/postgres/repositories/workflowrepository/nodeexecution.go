package workflowrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type NodeExecutionParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type nodeExecutionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewNodeExecutionRepository(
	p NodeExecutionParams,
) repositories.WorkflowNodeExecutionRepository {
	return &nodeExecutionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.workflow-nodeexecution-repository"),
	}
}

func (r *nodeExecutionRepository) Create(
	ctx context.Context,
	entity *workflow.NodeExecution,
) (*workflow.NodeExecution, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("instanceID", entity.WorkflowInstanceID.String()),
		zap.String("nodeID", entity.WorkflowNodeID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert node execution", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *nodeExecutionRepository) Update(
	ctx context.Context,
	entity *workflow.NodeExecution,
) (*workflow.NodeExecution, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	results, rErr := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("wfne.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update node execution", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Node Execution", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *nodeExecutionRepository) GetByInstanceID(
	ctx context.Context,
	instanceID, orgID, buID pulid.ID,
) ([]*workflow.NodeExecution, error) {
	log := r.l.With(
		zap.String("operation", "GetByInstanceID"),
		zap.String("instanceID", instanceID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	executions := make([]*workflow.NodeExecution, 0)
	err = db.NewSelect().Model(&executions).
		Relation("WorkflowNode").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfne.workflow_instance_id = ?", instanceID).
				Where("wfne.organization_id = ?", orgID).
				Where("wfne.business_unit_id = ?", buID)
		}).
		Order("wfne.created_at ASC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to scan node executions", zap.Error(err))
		return nil, err
	}

	return executions, nil
}
