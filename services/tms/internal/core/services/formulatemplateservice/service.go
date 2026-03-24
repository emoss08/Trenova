package formulatemplateservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	Repo           repositories.FormulaTemplateRepository
	VersionRepo    repositories.FormulaTemplateVersionRepository
	FormulaService *formula.Service
	AuditService   services.AuditService
}

type Service struct {
	l              *zap.Logger
	repo           repositories.FormulaTemplateRepository
	versionRepo    repositories.FormulaTemplateVersionRepository
	formulaService *formula.Service
	auditService   services.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:              p.Logger.Named("service.formulatemplate"),
		repo:           p.Repo,
		versionRepo:    p.VersionRepo,
		formulaService: p.FormulaService,
		auditService:   p.AuditService,
	}
}

func (s *Service) Create(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
	userID pulid.ID,
) (*formulatemplate.FormulaTemplate, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
	)

	if err := s.validateTemplate(entity); err != nil {
		return nil, err
	}

	entity.CurrentVersionNumber = 1

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create formula template", zap.Error(err))
		return nil, err
	}

	s.createVersionSnapshot(ctx, log, createdEntity, 1, userID, "Initial version", nil)
	s.logAuditAction(
		log,
		createdEntity,
		permission.OpCreate,
		userID,
		nil,
		"Formula template created",
	)

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
	userID pulid.ID,
) (*formulatemplate.FormulaTemplate, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	if err := s.validateTemplate(entity); err != nil {
		return nil, err
	}

	original, err := s.getTemplateByID(
		ctx,
		entity.ID,
		entity.GetOrganizationID(),
		entity.GetBusinessUnitID(),
	)
	if err != nil {
		log.Error("failed to get original formula template", zap.Error(err))
		return nil, err
	}

	newVersionNumber := original.CurrentVersionNumber + 1
	entity.CurrentVersionNumber = newVersionNumber

	changeSummary, diffErr := jsonutils.JSONDiff(original, entity, nil)
	if diffErr != nil {
		log.Warn("failed to compute change summary for version snapshot", zap.Error(diffErr))
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update formula template", zap.Error(err))
		return nil, err
	}

	s.createVersionSnapshot(ctx, log, updatedEntity, newVersionNumber, userID, "", changeSummary)
	s.logAuditAction(
		log,
		updatedEntity,
		permission.OpUpdate,
		userID,
		original,
		"Formula template updated",
	)

	return updatedEntity, nil
}

func (s *Service) Duplicate(
	ctx context.Context,
	req *repositories.BulkDuplicateFormulaTemplateRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	log := s.l.With(
		zap.String("operation", "Duplicate"),
		zap.Any("request", req),
	)

	entities, err := s.repo.BulkDuplicate(ctx, req)
	if err != nil {
		log.Error("failed to duplicate formula template", zap.Error(err))
		return nil, err
	}

	for _, entity := range entities {
		s.logAuditAction(
			log,
			entity,
			permission.OpCreate,
			req.TenantInfo.UserID,
			nil,
			"Formula template duplicated",
		)
	}

	return entities, nil
}

func (s *Service) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateFormulaTemplateStatusRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return s.repo.BulkUpdateStatus(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListFormulaTemplatesRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) GetUsage(
	ctx context.Context,
	req *repositories.GetTemplateUsageRequest,
) (*repositories.GetTemplateUsageResponse, error) {
	return s.repo.CountUsages(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *repositories.FormulaTemplateSelectOptionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) CreateVersion(
	ctx context.Context,
	req *repositories.CreateVersionRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	log := s.l.With(
		zap.String("operation", "CreateVersion"),
		zap.String("templateID", req.TemplateID.String()),
	)

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get template", zap.Error(err))
		return nil, err
	}

	var changeSummary map[string]jsonutils.FieldChange
	if template.CurrentVersionNumber >= 1 {
		prevVersion, verr := s.versionRepo.GetByTemplateAndVersion(
			ctx,
			&repositories.GetVersionRequest{
				TenantInfo:    req.TenantInfo,
				TemplateID:    req.TemplateID,
				VersionNumber: template.CurrentVersionNumber,
			},
		)
		if verr == nil && prevVersion != nil {
			changeSummary, _ = jsonutils.JSONDiff(prevVersion, template, &jsonutils.DiffOptions{
				IgnoreFields: versionDiffIgnoreFields,
			})
		}
	}

	newVersionNumber := template.CurrentVersionNumber + 1
	template.CurrentVersionNumber = newVersionNumber

	if _, err = s.repo.Update(ctx, template); err != nil {
		log.Error("failed to update template version number", zap.Error(err))
		return nil, err
	}

	version := formulatemplate.NewVersionFromTemplate(
		template,
		newVersionNumber,
		req.TenantInfo.UserID,
		req.ChangeMessage,
		changeSummary,
	)

	createdVersion, err := s.versionRepo.Create(ctx, version)
	if err != nil {
		log.Error("failed to create version", zap.Error(err))
		return nil, err
	}

	return createdVersion, nil
}

func (s *Service) ListVersions(
	ctx context.Context,
	req *repositories.ListVersionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplateVersion], error) {
	return s.versionRepo.List(ctx, req)
}

func (s *Service) GetVersion(
	ctx context.Context,
	req *repositories.GetVersionRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return s.versionRepo.GetByTemplateAndVersion(ctx, req)
}

func (s *Service) Rollback(
	ctx context.Context,
	req *repositories.RollbackRequest,
) (*formulatemplate.FormulaTemplate, error) {
	log := s.l.With(
		zap.String("operation", "Rollback"),
		zap.String("templateID", req.TemplateID.String()),
		zap.Int64("targetVersion", req.TargetVersion),
	)

	targetVersion, err := s.versionRepo.GetByTemplateAndVersion(
		ctx,
		&repositories.GetVersionRequest{
			TenantInfo:    req.TenantInfo,
			TemplateID:    req.TemplateID,
			VersionNumber: req.TargetVersion,
		},
	)
	if err != nil {
		log.Error("failed to get target version", zap.Error(err))
		return nil, err
	}

	currentTemplate, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get current template", zap.Error(err))
		return nil, err
	}

	applyVersionToTemplate(currentTemplate, targetVersion)

	newVersionNumber := currentTemplate.CurrentVersionNumber + 1
	currentTemplate.CurrentVersionNumber = newVersionNumber

	updatedTemplate, err := s.repo.Update(ctx, currentTemplate)
	if err != nil {
		log.Error("failed to update template", zap.Error(err))
		return nil, err
	}

	changeMessage := req.ChangeMessage
	if changeMessage == "" {
		changeMessage = fmt.Sprintf("Rolled back to version %d", req.TargetVersion)
	}

	s.createVersionSnapshot(
		ctx,
		log,
		updatedTemplate,
		newVersionNumber,
		req.TenantInfo.UserID,
		changeMessage,
		nil,
	)
	s.logAuditAction(
		log,
		updatedTemplate,
		permission.OpUpdate,
		req.TenantInfo.UserID,
		nil,
		changeMessage,
	)

	return updatedTemplate, nil
}

func (s *Service) Fork(
	ctx context.Context,
	req *repositories.ForkTemplateRequest,
) (*formulatemplate.FormulaTemplate, error) {
	log := s.l.With(
		zap.String("operation", "Fork"),
		zap.String("sourceTemplateID", req.SourceTemplateID.String()),
	)

	sourceTemplate, err := s.getTemplateByIDWithTenant(ctx, req.SourceTemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get source template", zap.Error(err))
		return nil, err
	}

	snapshot, sourceVersionNum := s.resolveTemplateSnapshot(
		ctx,
		log,
		sourceTemplate,
		req.SourceVersion,
		req.TenantInfo,
	)

	forkedTemplate := &formulatemplate.FormulaTemplate{
		OrganizationID:       req.TenantInfo.OrgID,
		BusinessUnitID:       req.TenantInfo.BuID,
		Name:                 req.NewName,
		Description:          snapshot.Description,
		Type:                 snapshot.Type,
		Expression:           snapshot.Expression,
		Status:               formulatemplate.StatusDraft,
		SchemaID:             snapshot.SchemaID,
		VariableDefinitions:  snapshot.VariableDefinitions,
		Metadata:             snapshot.Metadata,
		SourceTemplateID:     &req.SourceTemplateID,
		SourceVersionNumber:  &sourceVersionNum,
		CurrentVersionNumber: 1,
	}

	createdTemplate, err := s.repo.Create(ctx, forkedTemplate)
	if err != nil {
		log.Error("failed to create forked template", zap.Error(err))
		return nil, err
	}

	changeMessage := req.ChangeMessage
	if changeMessage == "" {
		changeMessage = fmt.Sprintf("Forked from template %s", sourceTemplate.Name)
	}

	s.createVersionSnapshot(ctx, log, createdTemplate, 1, req.TenantInfo.UserID, changeMessage, nil)
	s.logAuditAction(
		log,
		createdTemplate,
		permission.OpCreate,
		req.TenantInfo.UserID,
		nil,
		changeMessage,
	)

	return createdTemplate, nil
}

func (s *Service) CompareVersions(
	ctx context.Context,
	req *repositories.CompareVersionsRequest,
) (*formulatemplate.VersionDiff, error) {
	log := s.l.With(
		zap.String("operation", "CompareVersions"),
		zap.String("templateID", req.TemplateID.String()),
		zap.Int64("fromVersion", req.FromVersion),
		zap.Int64("toVersion", req.ToVersion),
	)

	versions, err := s.versionRepo.GetVersionRange(ctx, &repositories.GetVersionRangeRequest{
		TenantInfo:  req.TenantInfo,
		TemplateID:  req.TemplateID,
		FromVersion: req.FromVersion,
		ToVersion:   req.ToVersion,
	})
	if err != nil {
		log.Error("failed to get version range", zap.Error(err))
		return nil, err
	}

	if len(versions) != 2 {
		return nil, errortypes.NewValidationError(
			"versions",
			errortypes.ErrInvalid,
			"Both versions must exist for comparison",
		)
	}

	fromVer, toVer := extractVersionPair(versions, req.FromVersion, req.ToVersion)
	if fromVer == nil || toVer == nil {
		return nil, errortypes.NewValidationError(
			"versions",
			errortypes.ErrNotFound,
			"One or both versions not found in the retrieved range",
		)
	}

	changes, err := jsonutils.JSONDiff(fromVer, toVer, &jsonutils.DiffOptions{
		IgnoreFields: versionDiffIgnoreFields,
	})
	if err != nil {
		log.Error("failed to compute diff", zap.Error(err))
		return nil, err
	}

	return &formulatemplate.VersionDiff{
		FromVersion: req.FromVersion,
		ToVersion:   req.ToVersion,
		Changes:     changes,
		ChangeCount: len(changes),
	}, nil
}

func (s *Service) GetLineage(
	ctx context.Context,
	req *repositories.GetLineageRequest,
) (*formulatemplate.ForkLineage, error) {
	log := s.l.With(
		zap.String("operation", "GetLineage"),
		zap.String("templateID", req.TemplateID.String()),
	)

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get template", zap.Error(err))
		return nil, err
	}

	forkedTemplates, err := s.versionRepo.GetForkedTemplates(
		ctx,
		&repositories.GetForkedTemplatesRequest{
			TenantInfo:       req.TenantInfo,
			SourceTemplateID: req.TemplateID,
		},
	)
	if err != nil {
		log.Error("failed to get forked templates", zap.Error(err))
		return nil, err
	}

	return buildLineage(template, forkedTemplates), nil
}

func (s *Service) UpdateVersionTags(
	ctx context.Context,
	req *repositories.UpdateVersionTagsRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	log := s.l.With(
		zap.String("operation", "UpdateVersionTags"),
		zap.String("templateID", req.TemplateID.String()),
		zap.Int64("versionNumber", req.VersionNumber),
	)

	for _, tag := range req.Tags {
		if !formulatemplate.VersionTag(tag).IsValid() {
			return nil, errortypes.NewValidationError(
				"tags",
				errortypes.ErrInvalid,
				fmt.Sprintf("Invalid tag: %s", tag),
			)
		}
	}

	version, err := s.versionRepo.UpdateTags(ctx, req)
	if err != nil {
		log.Error("failed to update version tags", zap.Error(err))
		return nil, err
	}

	return version, nil
}

type TestExpressionRequest struct {
	Expression string
	SchemaID   string
	Variables  map[string]any
}

type TestExpressionResponse struct {
	Valid   bool   `json:"valid"`
	Result  any    `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message"`
}

func (s *Service) TestExpression(req *TestExpressionRequest) *TestExpressionResponse {
	env, err := s.formulaService.BuildValidationEnvironment(req.SchemaID, req.Variables)
	if err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: "Expression validation failed",
		}
	}

	if err = s.formulaService.ValidateExpressionWithEnv(req.Expression, env); err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: "Expression validation failed",
		}
	}

	result, err := s.formulaService.EvaluateWithEnv(req.Expression, env)
	if err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: "Expression evaluation failed",
		}
	}

	return &TestExpressionResponse{
		Valid:   true,
		Result:  result.Amount,
		Message: "Expression is valid",
	}
}

func (s *Service) validateTemplate(entity *formulatemplate.FormulaTemplate) error {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return s.validateExpression(entity)
}

func (s *Service) validateExpression(entity *formulatemplate.FormulaTemplate) error {
	variables := make(map[string]any, len(entity.VariableDefinitions))
	for _, varDef := range entity.VariableDefinitions {
		if varDef.DefaultValue != nil {
			variables[varDef.Name] = varDef.DefaultValue
			continue
		}

		variables[varDef.Name] = typeutils.DefaultValueForType(string(varDef.Type))
	}

	env, err := s.formulaService.BuildValidationEnvironment(entity.SchemaID, variables)
	if err != nil {
		return err
	}

	return s.formulaService.ValidateExpressionWithEnv(entity.Expression, env)
}

func (s *Service) getTemplateByID(
	ctx context.Context,
	id, orgID, buID pulid.ID,
) (*formulatemplate.FormulaTemplate, error) {
	return s.repo.GetByID(ctx, repositories.GetFormulaTemplateByIDRequest{
		TemplateID: id,
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	})
}

func (s *Service) getTemplateByIDWithTenant(
	ctx context.Context,
	id pulid.ID,
	tenant pagination.TenantInfo,
) (*formulatemplate.FormulaTemplate, error) {
	return s.repo.GetByID(ctx, repositories.GetFormulaTemplateByIDRequest{
		TemplateID: id,
		TenantInfo: tenant,
	})
}

func (s *Service) createVersionSnapshot(
	ctx context.Context,
	log *zap.Logger,
	template *formulatemplate.FormulaTemplate,
	versionNumber int64,
	userID pulid.ID,
	changeMessage string,
	changeSummary map[string]jsonutils.FieldChange,
) {
	version := formulatemplate.NewVersionFromTemplate(
		template,
		versionNumber,
		userID,
		changeMessage,
		changeSummary,
	)
	if _, err := s.versionRepo.Create(ctx, version); err != nil {
		log.Error("failed to create version snapshot", zap.Error(err))
	}
}

func (s *Service) logAuditAction(
	log *zap.Logger,
	entity *formulatemplate.FormulaTemplate,
	operation permission.Operation,
	userID pulid.ID,
	previousState *formulatemplate.FormulaTemplate,
	comment string,
) {
	params := &services.LogActionParams{
		Resource:       permission.ResourceFormulaTemplate,
		ResourceID:     entity.GetID().String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(entity),
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}

	opts := []services.LogOption{auditservice.WithComment(comment)}

	if previousState != nil {
		params.PreviousState = jsonutils.MustToJSON(previousState)
		opts = append(opts, auditservice.WithDiff(previousState, entity))
	}

	if err := s.auditService.LogAction(params, opts...); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}
}

func (s *Service) resolveTemplateSnapshot(
	ctx context.Context,
	log *zap.Logger,
	template *formulatemplate.FormulaTemplate,
	requestedVersion *int64,
	tenant pagination.TenantInfo,
) (snapshot templateSnapshot, versionNumber int64) {
	if requestedVersion != nil {
		version, err := s.versionRepo.GetByTemplateAndVersion(ctx, &repositories.GetVersionRequest{
			TenantInfo:    tenant,
			TemplateID:    template.ID,
			VersionNumber: *requestedVersion,
		})
		if err == nil {
			return snapshotFromVersion(version), version.VersionNumber
		}
		log.Warn("failed to get requested version, falling back to template", zap.Error(err))
	}

	version, err := s.versionRepo.GetLatestVersion(ctx, template.ID, tenant)
	if err == nil {
		return snapshotFromVersion(version), version.VersionNumber
	}

	return snapshotFromTemplate(template), template.CurrentVersionNumber
}
