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

	DB          *postgres.Connection
	VerisonRepo repositories.VersionRepository
	NodeRepo    repositories.WorkflowNodeRepository
	ConnRepo    repositories.WorkflowConnectionRepository
	Logger      *zap.Logger
}

type templateRepository struct {
	db          *postgres.Connection
	versionRepo repositories.VersionRepository
	nodeRepo    repositories.WorkflowNodeRepository
	connRepo    repositories.WorkflowConnectionRepository
	l           *zap.Logger
}

func NewTemplateRepository(p TemplateParams) repositories.TemplateRepository {
	return &templateRepository{
		db:          p.DB,
		versionRepo: p.VerisonRepo,
		nodeRepo:    p.NodeRepo,
		connRepo:    p.ConnRepo,
		l:           p.Logger.Named("postgres.workflow-template-repository"),
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

	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err = tx.NewInsert().Model(newTemplate).Returning("*").Exec(ctx); err != nil {
			log.Error("failed to insert duplicated template", zap.Error(err))
			return err
		}

		if len(original.Versions) > 0 {
			if err = r.nodeRepo.CloneVersionNodes(ctx, tx, &repositories.CloneVersionNodesRequest{
				SourceVersion: original.Versions[0],
				NewVersionID:  newTemplate.ID,
				OrgID:         req.OrgID,
				BuID:          req.BuID,
			}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		log.Error("failed to duplicate workflow template", zap.Error(err))
		return nil, err
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
	templateData map[string]any,
	orgID, buID, userID pulid.ID,
) (*workflow.Template, error) {
	log := r.l.With(
		zap.String("operation", "createTemplateFromImport"),
		zap.String("orgID", orgID.String()),
		zap.String("buID", buID.String()),
		zap.String("userID", userID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	name, err := maputils.GetString(templateData, "name")
	if err != nil {
		return nil, errortypes.NewValidationError(
			"name",
			errortypes.ErrInvalid,
			err.Error(),
		)
	}

	description, err := maputils.GetString(templateData, "description")
	if err != nil {
		return nil, errortypes.NewValidationError(
			"description",
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

func (r *templateRepository) ImportFromJSON(
	ctx context.Context,
	req *repositories.ImportTemplateRequest,
) (*workflow.Template, error) {
	templateData, err := r.parseTemplateData(req.TemplateData)
	if err != nil {
		return nil, err
	}

	newTemplate, err := r.createTemplateFromImport(
		ctx,
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

		if err = r.versionRepo.ImportVersion(ctx, &repositories.ImportVersionRequest{
			VersionData: versionData,
			TemplateID:  newTemplate.ID,
			OrgID:       req.OrgID,
			BuID:        req.BuID,
			UserID:      req.UserID,
		}); err != nil {
			return nil, err
		}
	}

	return newTemplate, nil
}
