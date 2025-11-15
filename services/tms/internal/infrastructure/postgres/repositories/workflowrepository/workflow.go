package workflowrepository

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
	"github.com/emoss08/trenova/pkg/utils/workflowutils"
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

func NewRepository(p Params) repositories.WorkflowRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.workflow-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListWorkflowRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"wf",
		req.Filter,
		(*workflow.Workflow)(nil),
	)

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListWorkflowRequest,
) (*pagination.ListResult[*workflow.Workflow], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*workflow.Workflow, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan workflows", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*workflow.Workflow]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetWorkflowByIDRequest,
) (*workflow.Workflow, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(workflow.Workflow)
	err = db.NewSelect().
		Model(entity).
		Relation("CurrentVersion").
		Relation("PublishedVersion").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("wf.id = ?", req.ID).
				Where("wf.organization_id = ?", req.OrgID).
				Where("wf.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Workflow")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *workflow.Workflow,
) (*workflow.Workflow, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("workflowID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert workflow", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *workflow.Workflow,
) (*workflow.Workflow, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("workflowID", entity.ID.String()),
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
		log.Error("failed to update workflow", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	id, orgID, buID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("workflowID", id.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewDelete().
		Model((*workflow.Workflow)(nil)).
		Where("id = ?", id).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete workflow", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) CreateVersion(
	ctx context.Context,
	entity *workflow.WorkflowVersion,
) (*workflow.WorkflowVersion, error) {
	log := r.l.With(
		zap.String("operation", "CreateVersion"),
		zap.String("versionID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert workflow version", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) translateEdgeReferencesToNodeKeys(version *workflow.WorkflowVersion) error {
	if len(version.Edges) == 0 || len(version.Nodes) == 0 {
		return nil
	}

	mapper := workflowutils.NewNodeKeyMapper(version.Nodes)
	return mapper.TranslateEdgesToNodeKeys(version.Edges)
}

func (r *repository) GetVersionByID(
	ctx context.Context,
	id, orgID, buID pulid.ID,
) (*workflow.WorkflowVersion, error) {
	log := r.l.With(
		zap.String("operation", "GetVersionByID"),
		zap.String("versionID", id.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(workflow.WorkflowVersion)
	err = db.NewSelect().
		Model(entity).
		Relation("Nodes").
		Relation("Edges").
		Where("wfv.id = ?", id).
		Where("wfv.organization_id = ?", orgID).
		Where("wfv.business_unit_id = ?", buID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkflowVersion")
	}

	if err = r.translateEdgeReferencesToNodeKeys(entity); err != nil {
		log.Error("failed to translate edge references", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetVersionsByWorkflowID(
	ctx context.Context,
	workflowID, orgID, buID pulid.ID,
) ([]*workflow.WorkflowVersion, error) {
	log := r.l.With(
		zap.String("operation", "GetVersionsByWorkflowID"),
		zap.String("workflowID", workflowID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	versions := make([]*workflow.WorkflowVersion, 0)
	err = db.NewSelect().
		Model(&versions).
		Where("wfv.workflow_id = ?", workflowID).
		Where("wfv.organization_id = ?", orgID).
		Where("wfv.business_unit_id = ?", buID).
		Order("wfv.version_number DESC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get workflow versions", zap.Error(err))
		return nil, err
	}

	return versions, nil
}

func (r *repository) GetLatestVersion(
	ctx context.Context,
	workflowID, orgID, buID pulid.ID,
) (*workflow.WorkflowVersion, error) {
	log := r.l.With(
		zap.String("operation", "GetLatestVersion"),
		zap.String("workflowID", workflowID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(workflow.WorkflowVersion)
	err = db.NewSelect().
		Model(entity).
		Relation("Nodes").
		Relation("Edges").
		Where("wfv.workflow_id = ?", workflowID).
		Where("wfv.organization_id = ?", orgID).
		Where("wfv.business_unit_id = ?", buID).
		Order("wfv.version_number DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkflowVersion")
	}

	return entity, nil
}

func (r *repository) PublishVersion(
	ctx context.Context,
	workflowID, versionID, orgID, buID, userID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "PublishVersion"),
		zap.String("workflowID", workflowID.String()),
		zap.String("versionID", versionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	now := time.Now().Unix()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return err
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Error("failed to rollback transaction", zap.Error(rbErr))
			}
		}
	}()

	_, err = tx.NewUpdate().
		Model((*workflow.WorkflowVersion)(nil)).
		Set("is_published = ?", false).
		Where("workflow_id = ?", workflowID).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to unpublish previous versions", zap.Error(err))
		return err
	}

	_, err = tx.NewUpdate().
		Model((*workflow.WorkflowVersion)(nil)).
		Set("is_published = ?", true).
		Set("published_at = ?", now).
		Set("published_by = ?", userID).
		Where("id = ?", versionID).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to publish version", zap.Error(err))
		return err
	}

	_, err = tx.NewUpdate().
		Model((*workflow.Workflow)(nil)).
		Set("published_version_id = ?", versionID).
		Where("id = ?", workflowID).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to update workflow published version", zap.Error(err))
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Error("failed to commit transaction", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) CreateNodes(
	ctx context.Context,
	nodes []*workflow.WorkflowNode,
) error {
	log := r.l.With(
		zap.String("operation", "CreateNodes"),
		zap.Int("count", len(nodes)),
	)

	if len(nodes) == 0 {
		return nil
	}

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewInsert().Model(&nodes).Exec(ctx)
	if err != nil {
		log.Error("failed to insert workflow nodes", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) GetNodesByVersionID(
	ctx context.Context,
	versionID, orgID, buID pulid.ID,
) ([]*workflow.WorkflowNode, error) {
	log := r.l.With(
		zap.String("operation", "GetNodesByVersionID"),
		zap.String("versionID", versionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	nodes := make([]*workflow.WorkflowNode, 0)
	err = db.NewSelect().
		Model(&nodes).
		Where("wfn.workflow_version_id = ?", versionID).
		Where("wfn.organization_id = ?", orgID).
		Where("wfn.business_unit_id = ?", buID).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get workflow nodes", zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (r *repository) DeleteNodesByVersionID(
	ctx context.Context,
	versionID, orgID, buID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "DeleteNodesByVersionID"),
		zap.String("versionID", versionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewDelete().
		Model((*workflow.WorkflowNode)(nil)).
		Where("workflow_version_id = ?", versionID).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete workflow nodes", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) CreateEdges(
	ctx context.Context,
	edges []*workflow.WorkflowEdge,
) error {
	log := r.l.With(
		zap.String("operation", "CreateEdges"),
		zap.Int("count", len(edges)),
	)

	if len(edges) == 0 {
		return nil
	}

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewInsert().Model(&edges).Exec(ctx)
	if err != nil {
		log.Error("failed to insert workflow edges", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) GetEdgesByVersionID(
	ctx context.Context,
	versionID, orgID, buID pulid.ID,
) ([]*workflow.WorkflowEdge, error) {
	log := r.l.With(
		zap.String("operation", "GetEdgesByVersionID"),
		zap.String("versionID", versionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	edges := make([]*workflow.WorkflowEdge, 0)
	err = db.NewSelect().
		Model(&edges).
		Where("wfe.workflow_version_id = ?", versionID).
		Where("wfe.organization_id = ?", orgID).
		Where("wfe.business_unit_id = ?", buID).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get workflow edges", zap.Error(err))
		return nil, err
	}

	return edges, nil
}

func (r *repository) DeleteEdgesByVersionID(
	ctx context.Context,
	versionID, orgID, buID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "DeleteEdgesByVersionID"),
		zap.String("versionID", versionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewDelete().
		Model((*workflow.WorkflowEdge)(nil)).
		Where("workflow_version_id = ?", versionID).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete workflow edges", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	id, orgID, buID pulid.ID,
	status workflow.WorkflowStatus,
) error {
	log := r.l.With(
		zap.String("operation", "UpdateStatus"),
		zap.String("workflowID", id.String()),
		zap.String("status", status.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewUpdate().
		Model((*workflow.Workflow)(nil)).
		Set("status = ?", status).
		Where("id = ?", id).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to update workflow status", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) GetActiveWorkflowsByTrigger(
	ctx context.Context,
	triggerType workflow.TriggerType,
	orgID, buID pulid.ID,
) ([]*workflow.Workflow, error) {
	log := r.l.With(
		zap.String("operation", "GetActiveWorkflowsByTrigger"),
		zap.String("triggerType", triggerType.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	workflows := make([]*workflow.Workflow, 0)
	err = db.NewSelect().
		Model(&workflows).
		Relation("PublishedVersion").
		Where("wf.status = ?", workflow.WorkflowStatusActive).
		Where("wf.trigger_type = ?", triggerType).
		Where("wf.organization_id = ?", orgID).
		Where("wf.business_unit_id = ?", buID).
		Where("wf.published_version_id IS NOT NULL").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get active workflows", zap.Error(err))
		return nil, err
	}

	return workflows, nil
}
