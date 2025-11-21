package workflowrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/maputils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type VersionParams struct {
	fx.In

	DB       *postgres.Connection
	NodeRepo repositories.WorkflowNodeRepository
	ConnRepo repositories.WorkflowConnectionRepository
	Logger   *zap.Logger
}

type versionRepository struct {
	db       *postgres.Connection
	nodeRepo repositories.WorkflowNodeRepository
	connRepo repositories.WorkflowConnectionRepository
	l        *zap.Logger
}

func NewVersionRepository(p VersionParams) repositories.VersionRepository {
	return &versionRepository{
		db:       p.DB,
		nodeRepo: p.NodeRepo,
		connRepo: p.ConnRepo,
		l:        p.Logger.Named("postgres.workflow-version-repository"),
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

	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err = tx.NewInsert().Model(newVersion).Returning("*").Exec(ctx); err != nil {
			log.Error("failed to insert workflow version", zap.Error(err))
			return err
		}

		if sourceVersion != nil && len(sourceVersion.Nodes) > 0 {
			if err = r.nodeRepo.CloneVersionNodes(ctx, tx, &repositories.CloneVersionNodesRequest{
				SourceVersion: sourceVersion,
				NewVersionID:  newVersion.ID,
				OrgID:         req.OrgID,
				BuID:          req.BuID,
			}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		log.Error("failed to create workflow version", zap.Error(err))
		return nil, err
	}

	return newVersion, nil
}

func (r *versionRepository) CreateEntity(
	ctx context.Context,
	entity *workflow.Version,
) (*workflow.Version, error) {
	log := r.l.With(
		zap.String("operation", "CreateEntity"),
		zap.String("entityID", entity.ID.String()),
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

func (r *versionRepository) Update(
	ctx context.Context,
	entity *workflow.Version,
) (*workflow.Version, error) {
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
		Where("wfv.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update workflow version", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Workflow Version", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
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

func (r *versionRepository) ImportVersion(
	ctx context.Context,
	req *repositories.ImportVersionRequest,
) error {
	log := r.l.With(
		zap.String("operation", "importVersion"),
		zap.Any("req", req),
	)

	versionData, err := r.parseVersionData(req.VersionData)
	if err != nil {
		return err
	}

	newVersion := &workflow.Version{
		OrganizationID:     req.OrgID,
		BusinessUnitID:     req.BuID,
		WorkflowTemplateID: req.TemplateID,
		VersionNumber:      versionData.VersionNumber,
		VersionStatus:      workflow.VersionStatusDraft,
		Status:             workflow.Status(versionData.Status),
		TriggerType:        workflow.TriggerType(versionData.TriggerType),
		ScheduleConfig:     versionData.ScheduleConfig,
		TriggerConfig:      versionData.TriggerConfig,
		ChangeDescription:  versionData.ChangeDescription,
		CreatedByID:        req.UserID,
	}

	_, err = r.CreateEntity(ctx, newVersion)
	if err != nil {
		log.Error("failed to create imported version", zap.Error(err))
		return err
	}

	nodes, _ := maputils.GetArray(req.VersionData, "nodes")
	if len(nodes) == 0 {
		return nil
	}

	nodeIDs, err := r.nodeRepo.ImportNodes(ctx, &repositories.ImportNodesRequest{
		Nodes:     nodes,
		VersionID: newVersion.ID,
		OrgID:     req.OrgID,
		BuID:      req.BuID,
	})
	if err != nil {
		return err
	}

	connections, _ := maputils.GetArray(req.VersionData, "connections")
	return r.connRepo.ImportConnections(ctx, &repositories.ImportConnectionsRequest{
		Connections: connections,
		VersionID:   newVersion.ID,
		OrgID:       req.OrgID,
		BuID:        req.BuID,
		NodeIDs:     nodeIDs,
	})
}

type VersionDataResponse struct {
	VersionNumber     int            `json:"versionNumber"`
	Status            string         `json:"status"`
	TriggerType       string         `json:"triggerType"`
	ScheduleConfig    map[string]any `json:"scheduleConfig"`
	TriggerConfig     map[string]any `json:"triggerConfig"`
	ChangeDescription string         `json:"changeDescription"`
}

func (r *versionRepository) parseVersionData(data map[string]any) (*VersionDataResponse, error) {
	versionNumber, err := maputils.GetInt(data, "versionNumber")
	if err != nil {
		return nil, errortypes.NewValidationError(
			"versionNumber",
			errortypes.ErrInvalid,
			err.Error(),
		)
	}

	status, err := maputils.GetString(data, "status")
	if err != nil {
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalid, err.Error())
	}

	triggerType, err := maputils.GetString(data, "triggerType")
	if err != nil {
		return nil, errortypes.NewValidationError(
			"triggerType",
			errortypes.ErrInvalid,
			err.Error(),
		)
	}

	scheduleConfig, err := maputils.GetMap(data, "scheduleConfig")
	if err != nil {
		return nil, errortypes.NewValidationError(
			"scheduleConfig",
			errortypes.ErrInvalid,
			err.Error(),
		)
	}

	triggerConfig, err := maputils.GetMap(data, "triggerConfig")
	if err != nil {
		return nil, errortypes.NewValidationError(
			"triggerConfig",
			errortypes.ErrInvalid,
			err.Error(),
		)
	}

	changeDescription, err := maputils.GetString(data, "changeDescription")
	if err != nil {
		return nil, errortypes.NewValidationError(
			"changeDescription",
			errortypes.ErrInvalid,
			err.Error(),
		)
	}

	return &VersionDataResponse{
		VersionNumber:     versionNumber,
		Status:            status,
		TriggerType:       triggerType,
		ScheduleConfig:    scheduleConfig,
		TriggerConfig:     triggerConfig,
		ChangeDescription: changeDescription,
	}, nil
}
