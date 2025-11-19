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
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type TemplateParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type templateRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewTemplateRepository(p TemplateParams) repositories.TemplateRepository {
	return &templateRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.workflow-template-repository"),
	}
}

func (r *templateRepository) addOptions(
	q *bun.SelectQuery,
	opts repositories.TemplateOptions,
) *bun.SelectQuery {
	if opts.IncludeVersions != nil && *opts.IncludeVersions {
		q = q.Relation("Versions", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order("version_number DESC")
		})
	}
	return q
}

func (r *templateRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListTemplateRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"wft",
		req.Filter,
		(*workflow.Template)(nil),
	)
	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.TemplateOptions)
	})
	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *templateRepository) List(
	ctx context.Context,
	req *repositories.ListTemplateRequest,
) (*pagination.ListResult[*workflow.Template], error) {
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

	entities := make([]*workflow.Template, 0, req.Filter.Limit)
	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan workflow templates", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*workflow.Template]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *templateRepository) GetByID(
	ctx context.Context,
	req *repositories.GetTemplateByIDRequest,
) (*workflow.Template, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(workflow.Template)
	query := db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wft.id = ?", req.ID).
				Where("wft.organization_id = ?", req.OrgID).
				Where("wft.business_unit_id = ?", req.BuID)
		})

	if req.IncludeVersions {
		query = query.Relation("Versions", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order("version_number DESC")
		})
	}

	err = query.Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Workflow Template")
	}

	return entity, nil
}

func (r *templateRepository) Create(
	ctx context.Context,
	entity *workflow.Template,
	userID pulid.ID,
) (*workflow.Template, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity.CreatedByID = userID
	entity.UpdatedByID = userID

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert workflow template", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *templateRepository) Update(
	ctx context.Context,
	entity *workflow.Template,
	userID pulid.ID,
) (*workflow.Template, error) {
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
	entity.UpdatedByID = userID

	results, rErr := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("wft.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update workflow template", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Workflow Template", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *templateRepository) Delete(
	ctx context.Context,
	req *repositories.DeleteTemplateRequest,
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

	entity := &workflow.Template{
		ID:             req.ID,
		OrganizationID: req.OrgID,
		BusinessUnitID: req.BuID,
	}

	results, dErr := db.NewDelete().Model(entity).WherePK().Exec(ctx)
	if dErr != nil {
		log.Error("failed to delete workflow template", zap.Error(dErr))
		return dErr
	}

	return dberror.CheckRowsAffected(results, "Workflow Template", req.ID.String())
}

func (r *templateRepository) fetchOriginalTemplate(
	ctx context.Context,
	db *bun.DB,
	templateID, orgID, buID pulid.ID,
) (*workflow.Template, error) {
	original := new(workflow.Template)
	err := db.NewSelect().Model(original).
		Relation("Versions", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("version_status = ?", workflow.VersionStatusPublished).
				Relation("Nodes").
				Relation("Connections")
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wft.id = ?", templateID).
				Where("wft.organization_id = ?", orgID).
				Where("wft.business_unit_id = ?", buID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Workflow Template")
	}
	return original, nil
}

func (r *templateRepository) createDuplicatedVersion(
	ctx context.Context,
	db *bun.DB,
	log *zap.Logger,
	publishedVersion *workflow.Version,
	newTemplateID, orgID, buID, userID pulid.ID,
	originalName string,
) error {
	newVersion := &workflow.Version{
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		WorkflowTemplateID: newTemplateID,
		VersionNumber:      1,
		VersionStatus:      workflow.VersionStatusDraft,
		Status:             publishedVersion.Status,
		TriggerType:        publishedVersion.TriggerType,
		ScheduleConfig:     publishedVersion.ScheduleConfig,
		TriggerConfig:      publishedVersion.TriggerConfig,
		ChangeDescription:  "Duplicated from " + originalName,
		CreatedByID:        userID,
	}

	if _, err := db.NewInsert().Model(newVersion).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert duplicated version", zap.Error(err))
		return err
	}

	if len(publishedVersion.Nodes) == 0 {
		return nil
	}

	versionRepo := &versionRepository{db: r.db, l: r.l}
	nodeIDMap, err := versionRepo.cloneNodes(
		ctx,
		db,
		log,
		publishedVersion.Nodes,
		newVersion.ID,
		orgID,
		buID,
	)
	if err != nil {
		return err
	}

	return versionRepo.cloneConnections(
		ctx,
		db,
		log,
		publishedVersion.Connections,
		newVersion.ID,
		orgID,
		buID,
		nodeIDMap,
	)
}

func (r *templateRepository) Duplicate(
	ctx context.Context,
	req *repositories.DuplicateTemplateRequest,
) (*workflow.Template, error) {
	log := r.l.With(
		zap.String("operation", "Duplicate"),
		zap.String("templateID", req.TemplateID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	original, err := r.fetchOriginalTemplate(ctx, db, req.TemplateID, req.OrgID, req.BuID)
	if err != nil {
		return nil, err
	}

	newTemplate := &workflow.Template{
		OrganizationID: req.OrgID,
		BusinessUnitID: req.BuID,
		Name:           req.NewName,
		Description:    original.Description,
		CreatedByID:    req.UserID,
		UpdatedByID:    req.UserID,
	}

	if _, err = db.NewInsert().Model(newTemplate).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert duplicated template", zap.Error(err))
		return nil, err
	}

	if len(original.Versions) > 0 {
		if err = r.createDuplicatedVersion(
			ctx,
			db,
			log,
			original.Versions[0],
			newTemplate.ID,
			req.OrgID,
			req.BuID,
			req.UserID,
			original.Name,
		); err != nil {
			return nil, err
		}
	}

	return newTemplate, nil
}

func (r *templateRepository) buildVersionQuery(
	query *bun.SelectQuery,
	req *repositories.ExportTemplateRequest,
) *bun.SelectQuery {
	switch {
	case req.IncludeAllVersions:
		return query.Relation("Versions", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order("version_number DESC").
				Relation("Nodes").
				Relation("Connections")
		})
	case req.VersionID != nil:
		return query.Relation("Versions", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("wfv.id = ?", *req.VersionID).
				Relation("Nodes").
				Relation("Connections")
		})
	default:
		return query.Relation("Versions", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("wfv.version_status = ?", workflow.VersionStatusPublished).
				Relation("Nodes").
				Relation("Connections")
		})
	}
}

func (r *templateRepository) buildVersionExport(version *workflow.Version) map[string]any {
	nodes := make([]map[string]any, 0, len(version.Nodes))
	connections := make([]map[string]any, 0, len(version.Connections))

	nodeIDMap := make(map[pulid.ID]int)
	for i, node := range version.Nodes {
		nodeIDMap[node.ID] = i
		nodes = append(nodes, map[string]any{
			"name":     node.Name,
			"nodeType": node.NodeType,
			"config":   node.Config,
			"position": map[string]any{
				"x": node.PositionX,
				"y": node.PositionY,
			},
		})
	}

	for _, conn := range version.Connections {
		connections = append(connections, map[string]any{
			"sourceNodeIndex": nodeIDMap[conn.SourceNodeID],
			"targetNodeIndex": nodeIDMap[conn.TargetNodeID],
			"condition":       conn.Condition,
			"isDefaultBranch": conn.IsDefaultBranch,
		})
	}

	return map[string]any{
		"versionNumber":     version.VersionNumber,
		"versionStatus":     version.VersionStatus,
		"status":            version.Status,
		"triggerType":       version.TriggerType,
		"scheduleConfig":    version.ScheduleConfig,
		"triggerConfig":     version.TriggerConfig,
		"changeDescription": version.ChangeDescription,
		"nodes":             nodes,
		"connections":       connections,
	}
}

func (r *templateRepository) ExportToJSON(
	ctx context.Context,
	req *repositories.ExportTemplateRequest,
) (map[string]any, error) {
	log := r.l.With(
		zap.String("operation", "ExportToJSON"),
		zap.String("templateID", req.TemplateID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	template := new(workflow.Template)
	query := db.NewSelect().Model(template).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wft.id = ?", req.TemplateID).
				Where("wft.organization_id = ?", req.OrgID).
				Where("wft.business_unit_id = ?", req.BuID)
		})

	query = r.buildVersionQuery(query, req)

	err = query.Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Workflow Template")
	}

	versions := make([]map[string]any, 0, len(template.Versions))
	for _, version := range template.Versions {
		versions = append(versions, r.buildVersionExport(version))
	}

	return map[string]any{
		"template": map[string]any{
			"name":        template.Name,
			"description": template.Description,
		},
		"versions": versions,
	}, nil
}

func (r *templateRepository) parseTemplateData(data map[string]any) (map[string]any, error) {
	templateData, err := maputils.GetMap(data, "template")
	if err != nil {
		return nil, errortypes.NewValidationError(
			"template",
			errortypes.ErrInvalid,
			"invalid template data structure",
		)
	}
	return templateData, nil
}

func (r *templateRepository) createTemplateFromImport(
	ctx context.Context,
	db *bun.DB,
	log *zap.Logger,
	templateData map[string]any,
	orgID, buID, userID pulid.ID,
) (*workflow.Template, error) {
	name, err := maputils.GetString(templateData, "name")
	if err != nil {
		return nil, errortypes.NewValidationError(
			"template.name",
			errortypes.ErrInvalid,
			err.Error(),
		)
	}

	description, err := maputils.GetString(templateData, "description")
	if err != nil {
		return nil, errortypes.NewValidationError(
			"template.description",
			errortypes.ErrInvalid,
			err.Error(),
		)
	}

	newTemplate := &workflow.Template{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Name:           name,
		Description:    description,
		CreatedByID:    userID,
		UpdatedByID:    userID,
	}

	if _, err = db.NewInsert().Model(newTemplate).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert imported template", zap.Error(err))
		return nil, err
	}

	return newTemplate, nil
}

func (r *templateRepository) importVersion(
	ctx context.Context,
	db *bun.DB,
	log *zap.Logger,
	versionData map[string]any,
	templateID, orgID, buID, userID pulid.ID,
) error {
	versionNumber, err := maputils.GetInt(versionData, "versionNumber")
	if err != nil {
		return errortypes.NewValidationError(
			"version.versionNumber",
			errortypes.ErrInvalid,
			err.Error(),
		)
	}

	status, err := maputils.GetString(versionData, "status")
	if err != nil {
		return errortypes.NewValidationError("version.status", errortypes.ErrInvalid, err.Error())
	}

	triggerType, err := maputils.GetString(versionData, "triggerType")
	if err != nil {
		return errortypes.NewValidationError(
			"version.triggerType",
			errortypes.ErrInvalid,
			err.Error(),
		)
	}

	scheduleConfig, err := maputils.GetMap(versionData, "scheduleConfig")
	if err != nil {
		return errortypes.NewValidationError(
			"version.scheduleConfig",
			errortypes.ErrInvalid,
			err.Error(),
		)
	}

	triggerConfig, err := maputils.GetMap(versionData, "triggerConfig")
	if err != nil {
		return errortypes.NewValidationError(
			"version.triggerConfig",
			errortypes.ErrInvalid,
			err.Error(),
		)
	}

	changeDescription, err := maputils.GetString(versionData, "changeDescription")
	if err != nil {
		return errortypes.NewValidationError(
			"version.changeDescription",
			errortypes.ErrInvalid,
			err.Error(),
		)
	}

	newVersion := &workflow.Version{
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		WorkflowTemplateID: templateID,
		VersionNumber:      versionNumber,
		VersionStatus:      workflow.VersionStatusDraft,
		Status:             workflow.Status(status),
		TriggerType:        workflow.TriggerType(triggerType),
		ScheduleConfig:     scheduleConfig,
		TriggerConfig:      triggerConfig,
		ChangeDescription:  changeDescription,
		CreatedByID:        userID,
	}

	if _, err = db.NewInsert().Model(newVersion).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert imported version", zap.Error(err))
		return err
	}

	nodes, _ := maputils.GetArray(versionData, "nodes")
	if len(nodes) == 0 {
		return nil
	}

	nodeIDs, err := r.importNodes(ctx, db, log, nodes, newVersion.ID, orgID, buID)
	if err != nil {
		return err
	}

	connections, _ := maputils.GetArray(versionData, "connections")
	return r.importConnections(ctx, db, log, connections, newVersion.ID, orgID, buID, nodeIDs)
}

func (r *templateRepository) importNodes(
	ctx context.Context,
	db *bun.DB,
	log *zap.Logger,
	nodes []any,
	versionID, orgID, buID pulid.ID,
) ([]pulid.ID, error) {
	nodeIDs := make([]pulid.ID, len(nodes))

	for i, n := range nodes {
		nodeData, ok := n.(map[string]any)
		if !ok {
			return nil, errortypes.NewValidationError(
				"node",
				errortypes.ErrInvalid,
				"invalid node data",
			)
		}

		name, err := maputils.GetString(nodeData, "name")
		if err != nil {
			return nil, errortypes.NewValidationError(
				"node.name",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		nodeType, err := maputils.GetString(nodeData, "nodeType")
		if err != nil {
			return nil, errortypes.NewValidationError(
				"node.nodeType",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		config, err := maputils.GetMap(nodeData, "config")
		if err != nil {
			return nil, errortypes.NewValidationError(
				"node.config",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		position, err := maputils.GetMap(nodeData, "position")
		if err != nil {
			return nil, errortypes.NewValidationError(
				"node.position",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		posX, err := maputils.GetInt(position, "x")
		if err != nil {
			return nil, errortypes.NewValidationError(
				"node.position.x",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		posY, err := maputils.GetInt(position, "y")
		if err != nil {
			return nil, errortypes.NewValidationError(
				"node.position.y",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		newNode := &workflow.Node{
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			WorkflowVersionID: versionID,
			Name:              name,
			NodeType:          workflow.NodeType(nodeType),
			Config:            config,
			PositionX:         posX,
			PositionY:         posY,
		}

		if _, err = db.NewInsert().Model(newNode).Returning("*").Exec(ctx); err != nil {
			log.Error("failed to insert imported node", zap.Error(err))
			return nil, err
		}

		nodeIDs[i] = newNode.ID
	}

	return nodeIDs, nil
}

func (r *templateRepository) importConnections(
	ctx context.Context,
	db *bun.DB,
	log *zap.Logger,
	connections []any,
	versionID, orgID, buID pulid.ID,
	nodeIDs []pulid.ID,
) error {
	for _, c := range connections {
		connData, ok := c.(map[string]any)
		if !ok {
			continue
		}

		sourceIdx, err := maputils.GetInt(connData, "sourceNodeIndex")
		if err != nil {
			return errortypes.NewValidationError(
				"connection.sourceNodeIndex",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		targetIdx, err := maputils.GetInt(connData, "targetNodeIndex")
		if err != nil {
			return errortypes.NewValidationError(
				"connection.targetNodeIndex",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		condition, err := maputils.GetMap(connData, "condition")
		if err != nil {
			return errortypes.NewValidationError(
				"connection.condition",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		isDefaultBranch, err := maputils.GetBool(connData, "isDefaultBranch")
		if err != nil {
			return errortypes.NewValidationError(
				"connection.isDefaultBranch",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		newConn := &workflow.Connection{
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			WorkflowVersionID: versionID,
			SourceNodeID:      nodeIDs[sourceIdx],
			TargetNodeID:      nodeIDs[targetIdx],
			Condition:         condition,
			IsDefaultBranch:   isDefaultBranch,
		}

		if _, err = db.NewInsert().Model(newConn).Returning("*").Exec(ctx); err != nil {
			log.Error("failed to insert imported connection", zap.Error(err))
			return err
		}
	}

	return nil
}

func (r *templateRepository) ImportFromJSON(
	ctx context.Context,
	req *repositories.ImportTemplateRequest,
) (*workflow.Template, error) {
	log := r.l.With(
		zap.String("operation", "ImportFromJSON"),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	templateData, err := r.parseTemplateData(req.TemplateData)
	if err != nil {
		return nil, err
	}

	newTemplate, err := r.createTemplateFromImport(
		ctx,
		db,
		log,
		templateData,
		req.OrgID,
		req.BuID,
		req.UserID,
	)
	if err != nil {
		return nil, err
	}

	versions, _ := maputils.GetArray(req.TemplateData, "versions")
	if len(versions) == 0 {
		return newTemplate, nil
	}

	for _, v := range versions {
		versionData, ok := v.(map[string]any)
		if !ok {
			continue
		}

		if err = r.importVersion(ctx, db, log, versionData, newTemplate.ID, req.OrgID, req.BuID, req.UserID); err != nil {
			return nil, err
		}
	}

	return newTemplate, nil
}
