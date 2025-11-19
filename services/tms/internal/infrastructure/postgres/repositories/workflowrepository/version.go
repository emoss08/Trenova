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

type VersionParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type versionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewVersionRepository(p VersionParams) repositories.VersionRepository {
	return &versionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.workflow-version-repository"),
	}
}

func (r *versionRepository) addOptions(
	q *bun.SelectQuery,
	opts repositories.VersionOptions,
) *bun.SelectQuery {
	if opts.Status != "" {
		status, err := workflow.VersionStatusFromString(opts.Status)
		if err != nil {
			r.l.Error("invalid version status", zap.Error(err), zap.String("status", opts.Status))
			return q
		}
		q = q.Where("wfv.version_status = ?", status)
	}
	if opts.IncludeNodes != nil && *opts.IncludeNodes {
		q = q.Relation("Nodes")
	}
	if opts.IncludeConnections != nil && *opts.IncludeConnections {
		q = q.Relation("Connections")
	}
	return q
}

func (r *versionRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListVersionRequest,
) *bun.SelectQuery {
	q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.
			Where("wfv.workflow_template_id = ?", req.TemplateID).
			Where("wfv.organization_id = ?", req.OrgID).
			Where("wfv.business_unit_id = ?", req.BuID)
	})

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.VersionOptions)
	})
	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset).Order("wfv.version_number DESC")
}

func (r *versionRepository) List(
	ctx context.Context,
	req *repositories.ListVersionRequest,
) (*pagination.ListResult[*workflow.Version], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("templateID", req.TemplateID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*workflow.Version, 0, req.Filter.Limit)
	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan workflow versions", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*workflow.Version]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *versionRepository) GetByID(
	ctx context.Context,
	req *repositories.GetVersionByIDRequest,
) (*workflow.Version, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(workflow.Version)
	query := db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfv.id = ?", req.ID).
				Where("wfv.organization_id = ?", req.OrgID).
				Where("wfv.business_unit_id = ?", req.BuID)
		})

	if req.IncludeNodes {
		query = query.Relation("Nodes")
	}
	if req.IncludeConnections {
		query = query.Relation("Connections")
	}

	err = query.Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Workflow Version")
	}

	return entity, nil
}

func (r *versionRepository) getNextVersionNumber(
	ctx context.Context,
	db *bun.DB,
	templateID, orgID, buID pulid.ID,
) (int, error) {
	var maxVersion int
	err := db.NewSelect().
		Model((*workflow.Version)(nil)).
		Column("version_number").
		Where("workflow_template_id = ?", templateID).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Order("version_number DESC").
		Limit(1).
		Scan(ctx, &maxVersion)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return 0, err
	}
	return maxVersion + 1, nil
}

func (r *versionRepository) cloneNodes(
	ctx context.Context,
	db *bun.DB,
	log *zap.Logger,
	sourceNodes []*workflow.Node,
	newVersionID, orgID, buID pulid.ID,
) (map[pulid.ID]pulid.ID, error) {
	nodeIDMap := make(map[pulid.ID]pulid.ID)
	for _, node := range sourceNodes {
		newNode := &workflow.Node{
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			WorkflowVersionID: newVersionID,
			Name:              node.Name,
			NodeType:          node.NodeType,
			Config:            node.Config,
			PositionX:         node.PositionX,
			PositionY:         node.PositionY,
		}

		if _, err := db.NewInsert().Model(newNode).Returning("*").Exec(ctx); err != nil {
			log.Error("failed to insert cloned node", zap.Error(err))
			return nil, err
		}

		nodeIDMap[node.ID] = newNode.ID
	}
	return nodeIDMap, nil
}

func (r *versionRepository) cloneConnections(
	ctx context.Context,
	db *bun.DB,
	log *zap.Logger,
	sourceConnections []*workflow.Connection,
	newVersionID, orgID, buID pulid.ID,
	nodeIDMap map[pulid.ID]pulid.ID,
) error {
	for _, conn := range sourceConnections {
		newConn := &workflow.Connection{
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			WorkflowVersionID: newVersionID,
			SourceNodeID:      nodeIDMap[conn.SourceNodeID],
			TargetNodeID:      nodeIDMap[conn.TargetNodeID],
			Condition:         conn.Condition,
			IsDefaultBranch:   conn.IsDefaultBranch,
		}

		if _, err := db.NewInsert().Model(newConn).Returning("*").Exec(ctx); err != nil {
			log.Error("failed to insert cloned connection", zap.Error(err))
			return err
		}
	}
	return nil
}

func (r *versionRepository) fetchSourceVersion(
	ctx context.Context,
	db *bun.DB,
	sourceVersionID, orgID, buID pulid.ID,
) (*workflow.Version, error) {
	sourceVersion := new(workflow.Version)
	err := db.NewSelect().Model(sourceVersion).
		Relation("Nodes").
		Relation("Connections").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfv.id = ?", sourceVersionID).
				Where("wfv.organization_id = ?", orgID).
				Where("wfv.business_unit_id = ?", buID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Source Workflow Version")
	}
	return sourceVersion, nil
}

func (r *versionRepository) cloneVersionNodes(
	ctx context.Context,
	db *bun.DB,
	log *zap.Logger,
	sourceVersion *workflow.Version,
	newVersionID, orgID, buID pulid.ID,
) error {
	nodeIDMap, err := r.cloneNodes(ctx, db, log, sourceVersion.Nodes, newVersionID, orgID, buID)
	if err != nil {
		return err
	}

	return r.cloneConnections(
		ctx,
		db,
		log,
		sourceVersion.Connections,
		newVersionID,
		orgID,
		buID,
		nodeIDMap,
	)
}

func (r *versionRepository) Create(
	ctx context.Context,
	req *repositories.CreateVersionRequest,
) (*workflow.Version, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("templateID", req.TemplateID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	versionNumber, err := r.getNextVersionNumber(ctx, db, req.TemplateID, req.OrgID, req.BuID)
	if err != nil {
		log.Error("failed to get next version number", zap.Error(err))
		return nil, err
	}

	newVersion := &workflow.Version{
		OrganizationID:     req.OrgID,
		BusinessUnitID:     req.BuID,
		WorkflowTemplateID: req.TemplateID,
		VersionNumber:      versionNumber,
		VersionStatus:      workflow.VersionStatusDraft,
		Status:             workflow.StatusDraft,
		TriggerType:        workflow.TriggerTypeManual,
		ScheduleConfig:     make(map[string]any),
		TriggerConfig:      make(map[string]any),
		ChangeDescription:  req.ChangeDescription,
		CreatedByID:        req.UserID,
	}

	var sourceVersion *workflow.Version
	if req.CloneFromVersionID != nil {
		sourceVersion, err = r.fetchSourceVersion(
			ctx,
			db,
			*req.CloneFromVersionID,
			req.OrgID,
			req.BuID,
		)
		if err != nil {
			return nil, err
		}

		newVersion.Status = sourceVersion.Status
		newVersion.TriggerType = sourceVersion.TriggerType
		newVersion.ScheduleConfig = sourceVersion.ScheduleConfig
		newVersion.TriggerConfig = sourceVersion.TriggerConfig
	}

	if _, err = db.NewInsert().Model(newVersion).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert workflow version", zap.Error(err))
		return nil, err
	}

	if sourceVersion != nil && len(sourceVersion.Nodes) > 0 {
		if err = r.cloneVersionNodes(ctx, db, log, sourceVersion, newVersion.ID, req.OrgID, req.BuID); err != nil {
			return nil, err
		}
	}

	return newVersion, nil
}

func (r *versionRepository) Update(
	ctx context.Context,
	req *repositories.UpdateVersionRequest,
) (*workflow.Version, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", req.Version.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := req.Version.Version
	req.Version.Version++

	results, rErr := db.NewUpdate().
		Model(req.Version).
		WherePK().
		Where("wfv.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update workflow version", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Workflow Version", req.Version.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return req.Version, nil
}

func (r *versionRepository) Delete(
	ctx context.Context,
	req *repositories.DeleteVersionRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	entity := &workflow.Version{
		ID:             req.ID,
		OrganizationID: req.OrgID,
		BusinessUnitID: req.BuID,
	}

	results, dErr := db.NewDelete().Model(entity).WherePK().Exec(ctx)
	if dErr != nil {
		log.Error("failed to delete workflow version", zap.Error(dErr))
		return dErr
	}

	return dberror.CheckRowsAffected(results, "Workflow Version", req.ID.String())
}

func (r *versionRepository) Publish(
	ctx context.Context,
	req *repositories.PublishVersionRequest,
) (*workflow.Version, error) {
	log := r.l.With(
		zap.String("operation", "Publish"),
		zap.String("versionID", req.VersionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	version := new(workflow.Version)
	err = db.NewSelect().Model(version).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfv.id = ?", req.VersionID).
				Where("wfv.organization_id = ?", req.OrgID).
				Where("wfv.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Workflow Version")
	}

	if !version.VersionStatus.IsDraft() {
		return nil, ErrOnlyDraftVersionsCanBePublished
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		_, err = tx.NewUpdate().
			Model((*workflow.Version)(nil)).
			Set("version_status = ?", workflow.VersionStatusArchived).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("wfv.workflow_template_id = ?", version.WorkflowTemplateID).
					Where("wfv.version_status = ?", workflow.VersionStatusPublished).
					Where("wfv.organization_id = ?", req.OrgID).
					Where("wfv.business_unit_id = ?", req.BuID)
			}).
			Exec(c)
		if err != nil {
			log.Error("failed to archive current published version", zap.Error(err))
			return err
		}

		version.VersionStatus = workflow.VersionStatusPublished
		version.Version++

		results, rErr := tx.NewUpdate().
			Model(version).
			WherePK().
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error("failed to publish version", zap.Error(rErr))
			return rErr
		}

		if err = dberror.CheckRowsAffected(results, "Workflow Version", version.ID.String()); err != nil {
			return err
		}

		_, err = tx.NewUpdate().
			Model((*workflow.Template)(nil)).
			Set("published_version_id = ?", version.ID).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("wft.id = ?", version.WorkflowTemplateID).
					Where("wft.organization_id = ?", req.OrgID).
					Where("wft.business_unit_id = ?", req.BuID)
			}).
			Exec(c)
		if err != nil {
			log.Error("failed to update template published version", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to publish version", zap.Error(err))
		return nil, err
	}

	return version, nil
}

func (r *versionRepository) Archive(
	ctx context.Context,
	req *repositories.ArchiveVersionRequest,
) (*workflow.Version, error) {
	log := r.l.With(
		zap.String("operation", "Archive"),
		zap.String("versionID", req.VersionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	version := new(workflow.Version)
	err = db.NewSelect().Model(version).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfv.id = ?", req.VersionID).
				Where("wfv.organization_id = ?", req.OrgID).
				Where("wfv.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Workflow Version")
	}

	if version.VersionStatus.IsArchived() {
		return version, nil
	}

	version.VersionStatus = workflow.VersionStatusArchived
	version.Version++

	results, err := db.NewUpdate().
		Model(version).
		WherePK().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to archive version", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Workflow Version", version.ID.String()); err != nil {
		return nil, err
	}

	return version, nil
}

func (r *versionRepository) Rollback(
	ctx context.Context,
	req *repositories.RollbackVersionRequest,
) (*workflow.Version, error) {
	log := r.l.With(
		zap.String("operation", "Rollback"),
		zap.String("targetVersionID", req.TargetVersionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	targetVersion := new(workflow.Version)
	err = db.NewSelect().Model(targetVersion).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfv.id = ?", req.TargetVersionID).
				Where("wfv.workflow_template_id = ?", req.TemplateID).
				Where("wfv.organization_id = ?", req.OrgID).
				Where("wfv.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Target Workflow Version")
	}

	if targetVersion.VersionStatus.IsPublished() {
		return targetVersion, nil
	}

	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err = tx.NewUpdate().
			Model((*workflow.Version)(nil)).
			Set("version_status = ?", workflow.VersionStatusArchived).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("wfv.workflow_template_id = ?", req.TemplateID).
					Where("wfv.version_status = ?", workflow.VersionStatusPublished).
					Where("wfv.organization_id = ?", req.OrgID).
					Where("wfv.business_unit_id = ?", req.BuID)
			}).
			Exec(ctx)
		if err != nil {
			log.Error("failed to archive current published version", zap.Error(err))
			return err
		}

		targetVersion.VersionStatus = workflow.VersionStatusPublished
		targetVersion.Version++

		results, rErr := tx.NewUpdate().
			Model(targetVersion).
			WherePK().
			Returning("*").
			Exec(ctx)
		if rErr != nil {
			log.Error("failed to publish target version", zap.Error(rErr))
			return rErr
		}

		if err = dberror.CheckRowsAffected(results, "Workflow Version", targetVersion.ID.String()); err != nil {
			return err
		}

		_, err = tx.NewUpdate().
			Model((*workflow.Template)(nil)).
			Set("published_version_id = ?", targetVersion.ID).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("wft.id = ?", req.TemplateID).
					Where("wft.organization_id = ?", req.OrgID).
					Where("wft.business_unit_id = ?", req.BuID)
			}).
			Exec(ctx)
		if err != nil {
			log.Error("failed to update template published version", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to rollback version", zap.Error(err))
		return nil, err
	}

	return targetVersion, nil
}

func (r *versionRepository) GetPublished(
	ctx context.Context,
	req *repositories.GetPublishedVersionRequest,
) (*workflow.Version, error) {
	log := r.l.With(
		zap.String("operation", "GetPublished"),
		zap.String("templateID", req.TemplateID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(workflow.Version)
	query := db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfv.workflow_template_id = ?", req.TemplateID).
				Where("wfv.version_status = ?", workflow.VersionStatusPublished).
				Where("wfv.organization_id = ?", req.OrgID).
				Where("wfv.business_unit_id = ?", req.BuID)
		})

	if req.IncludeNodes {
		query = query.Relation("Nodes")
	}
	if req.IncludeConnections {
		query = query.Relation("Connections")
	}

	err = query.Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Published Workflow Version")
	}

	return entity, nil
}

func (r *versionRepository) GetNodes(
	ctx context.Context,
	versionID, orgID, buID pulid.ID,
) ([]*workflow.Node, error) {
	log := r.l.With(
		zap.String("operation", "GetNodes"),
		zap.String("versionID", versionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	nodes := make([]*workflow.Node, 0)
	err = db.NewSelect().Model(&nodes).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfn.workflow_version_id = ?", versionID).
				Where("wfn.organization_id = ?", orgID).
				Where("wfn.business_unit_id = ?", buID)
		}).Scan(ctx)
	if err != nil {
		log.Error("failed to scan workflow nodes", zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (r *versionRepository) GetConnections(
	ctx context.Context,
	versionID, orgID, buID pulid.ID,
) ([]*workflow.Connection, error) {
	log := r.l.With(
		zap.String("operation", "GetConnections"),
		zap.String("versionID", versionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	connections := make([]*workflow.Connection, 0)
	err = db.NewSelect().Model(&connections).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfc.workflow_version_id = ?", versionID).
				Where("wfc.organization_id = ?", orgID).
				Where("wfc.business_unit_id = ?", buID)
		}).Scan(ctx)
	if err != nil {
		log.Error("failed to scan workflow connections", zap.Error(err))
		return nil, err
	}

	return connections, nil
}

func (r *versionRepository) CreateNode(
	ctx context.Context,
	entity *workflow.Node,
) (*workflow.Node, error) {
	log := r.l.With(
		zap.String("operation", "CreateNode"),
		zap.String("versionID", entity.WorkflowVersionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert workflow node", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *versionRepository) UpdateNode(
	ctx context.Context,
	entity *workflow.Node,
) (*workflow.Node, error) {
	log := r.l.With(
		zap.String("operation", "UpdateNode"),
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
		Where("wfn.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update workflow node", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Workflow Node", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *versionRepository) DeleteNode(
	ctx context.Context,
	nodeID, orgID, buID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "DeleteNode"),
		zap.String("nodeID", nodeID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	entity := &workflow.Node{
		ID:             nodeID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}

	results, dErr := db.NewDelete().Model(entity).WherePK().Exec(ctx)
	if dErr != nil {
		log.Error("failed to delete workflow node", zap.Error(dErr))
		return dErr
	}

	return dberror.CheckRowsAffected(results, "Workflow Node", nodeID.String())
}

func (r *versionRepository) CreateConnection(
	ctx context.Context,
	entity *workflow.Connection,
) (*workflow.Connection, error) {
	log := r.l.With(
		zap.String("operation", "CreateConnection"),
		zap.String("versionID", entity.WorkflowVersionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert workflow connection", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *versionRepository) UpdateConnection(
	ctx context.Context,
	entity *workflow.Connection,
) (*workflow.Connection, error) {
	log := r.l.With(
		zap.String("operation", "UpdateConnection"),
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
		Where("wfc.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update workflow connection", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Workflow Connection", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *versionRepository) DeleteConnection(
	ctx context.Context,
	connectionID, orgID, buID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "DeleteConnection"),
		zap.String("connectionID", connectionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	entity := &workflow.Connection{
		ID:             connectionID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}

	results, dErr := db.NewDelete().Model(entity).WherePK().Exec(ctx)
	if dErr != nil {
		log.Error("failed to delete workflow connection", zap.Error(dErr))
		return dErr
	}

	return dberror.CheckRowsAffected(results, "Workflow Connection", connectionID.String())
}
