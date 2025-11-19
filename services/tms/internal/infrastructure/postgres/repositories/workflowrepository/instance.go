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

type InstanceParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type instanceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewInstanceRepository(p InstanceParams) repositories.WorkflowInstanceRepository {
	return &instanceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.workflow-instance-repository"),
	}
}

func (r *instanceRepository) addOptions(
	q *bun.SelectQuery,
	opts repositories.WorkflowInstanceOptions,
) *bun.SelectQuery {
	if opts.WorkflowTemplateID != nil {
		q = q.Where("wfi.workflow_template_id = ?", *opts.WorkflowTemplateID)
	}
	if opts.WorkflowVersionID != nil {
		q = q.Where("wfi.workflow_version_id = ?", *opts.WorkflowVersionID)
	}
	if opts.Status != "" {
		status, err := workflow.InstanceStatusFromString(opts.Status)
		if err != nil {
			r.l.Error("invalid instance status", zap.Error(err), zap.String("status", opts.Status))
			return q
		}
		q = q.Where("wfi.status = ?", status)
	}
	if opts.ExecutionMode != "" {
		mode, err := workflow.ExecutionModeFromString(opts.ExecutionMode)
		if err != nil {
			r.l.Error(
				"invalid execution mode",
				zap.Error(err),
				zap.String("executionMode", opts.ExecutionMode),
			)
			return q
		}
		q = q.Where("wfi.execution_mode = ?", mode)
	}
	return q
}

func (r *instanceRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListWorkflowInstanceRequest,
) *bun.SelectQuery {
	q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.
			Where("wfi.organization_id = ?", req.Filter.TenantOpts.OrgID).
			Where("wfi.business_unit_id = ?", req.Filter.TenantOpts.BuID)
	})

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.WorkflowInstanceOptions)
	})
	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset).Order("wfi.started_at DESC")
}

func (r *instanceRepository) List(
	ctx context.Context,
	req *repositories.ListWorkflowInstanceRequest,
) (*pagination.ListResult[*workflow.Instance], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*workflow.Instance, 0, req.Filter.Limit)
	total, err := db.NewSelect().
		Model(&entities).
		Relation("WorkflowTemplate").
		Relation("WorkflowVersion").
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan workflow instances", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*workflow.Instance]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *instanceRepository) GetByID(
	ctx context.Context,
	req *repositories.GetWorkflowInstanceByIDRequest,
) (*workflow.Instance, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(workflow.Instance)
	err = db.NewSelect().Model(entity).
		Relation("WorkflowTemplate").
		Relation("WorkflowVersion").
		Relation("NodeExecutions", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order("created_at ASC")
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfi.id = ?", req.ID).
				Where("wfi.organization_id = ?", req.OrgID).
				Where("wfi.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Workflow Instance")
	}

	return entity, nil
}

func (r *instanceRepository) Create(
	ctx context.Context,
	entity *workflow.Instance,
) (*workflow.Instance, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("templateID", entity.WorkflowTemplateID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert workflow instance", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *instanceRepository) Update(
	ctx context.Context,
	entity *workflow.Instance,
) (*workflow.Instance, error) {
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
		Where("wfi.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update workflow instance", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Workflow Instance", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *instanceRepository) GetNodeExecutions(
	ctx context.Context,
	instanceID, orgID, buID pulid.ID,
) ([]*workflow.NodeExecution, error) {
	log := r.l.With(
		zap.String("operation", "GetNodeExecutions"),
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
