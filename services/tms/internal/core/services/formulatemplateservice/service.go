package formulatemplateservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/typeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	DB             ports.DBConnection
	Repo           repositories.FormulaTemplateRepository
	VersionRepo    repositories.FormulaTemplateVersionRepository
	TestCaseRepo   repositories.FormulaTemplateTestCaseRepository
	ShipmentRepo   repositories.ShipmentRepository
	FormulaService *formula.Service
	AuditService   services.AuditService
	Notifications  *notificationservice.Service `optional:"true"`
}

type Service struct {
	l              *zap.Logger
	db             ports.DBConnection
	repo           repositories.FormulaTemplateRepository
	versionRepo    repositories.FormulaTemplateVersionRepository
	testCaseRepo   repositories.FormulaTemplateTestCaseRepository
	shipmentRepo   repositories.ShipmentRepository
	formulaService *formula.Service
	auditService   services.AuditService
	notifications  *notificationservice.Service
}

func New(p Params) *Service { //nolint:gocritic // fx param structs are passed by value
	return &Service{
		l:              p.Logger.Named("service.formulatemplate"),
		db:             p.DB,
		repo:           p.Repo,
		versionRepo:    p.VersionRepo,
		testCaseRepo:   p.TestCaseRepo,
		shipmentRepo:   p.ShipmentRepo,
		formulaService: p.FormulaService,
		auditService:   p.AuditService,
		notifications:  p.Notifications,
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

	// A template is born a draft whatever the payload claims. Active is
	// reached only through review, and the review stamps are set by the
	// reviewers, not by whoever typed the record in.
	entity.Status = formulatemplate.StatusDraft
	clearApprovalFields(entity)

	if err := s.validateTemplate(ctx, entity); err != nil {
		return nil, err
	}

	entity.CurrentVersionNumber = 1

	var createdEntity *formulatemplate.FormulaTemplate
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		created, txErr := s.repo.Create(txCtx, entity)
		if txErr != nil {
			log.Error("failed to create formula template", zap.Error(txErr))
			return txErr
		}

		if txErr = s.createVersionSnapshot(
			txCtx, created, 1, userID, "Initial version", nil,
		); txErr != nil {
			log.Error("failed to create version snapshot", zap.Error(txErr))
			return txErr
		}

		createdEntity = created
		return nil
	})
	if err != nil {
		return nil, err
	}

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

	if err := s.validateTemplate(ctx, entity); err != nil {
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

	if entity.Status != original.Status {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Status cannot be changed from %s to %s here; "+
					"submit, approve, reject, or archive the template instead",
				original.Status,
				entity.Status,
			),
		)
	}

	carryApprovalFields(entity, original)

	material := entity.HasMaterialChange(original)

	auditComment := "Formula template updated"
	revertsApproval := material &&
		(original.Status == formulatemplate.StatusActive ||
			original.Status == formulatemplate.StatusInReview)
	if revertsApproval {
		entity.Status = formulatemplate.StatusDraft
		clearApprovalFields(entity)
		auditComment = "Material change reverted approval"
	}

	entity.CurrentVersionNumber = original.CurrentVersionNumber
	var changeSummary map[string]jsonutils.FieldChange
	if material {
		entity.CurrentVersionNumber = original.CurrentVersionNumber + 1

		var diffErr error
		changeSummary, diffErr = jsonutils.JSONDiff(original, entity, nil)
		if diffErr != nil {
			log.Warn("failed to compute change summary for version snapshot", zap.Error(diffErr))
		}
	}

	var updatedEntity *formulatemplate.FormulaTemplate
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		updated, txErr := s.repo.Update(txCtx, entity)
		if txErr != nil {
			log.Error("failed to update formula template", zap.Error(txErr))
			return txErr
		}

		if material {
			if txErr = s.createVersionSnapshot(
				txCtx, updated, updated.CurrentVersionNumber, userID, "", changeSummary,
			); txErr != nil {
				log.Error("failed to create version snapshot", zap.Error(txErr))
				return txErr
			}
		}

		if revertsApproval {
			cleared, clearErr := s.clearScheduledVersions(txCtx, updated)
			if clearErr != nil {
				log.Error("failed to clear scheduled versions", zap.Error(clearErr))
				return clearErr
			}
			auditComment = withClearedSchedules(auditComment, cleared)
		}

		updatedEntity = updated
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logAuditAction(
		log,
		updatedEntity,
		permission.OpUpdate,
		userID,
		original,
		auditComment,
	)

	return updatedEntity, nil
}

// clearScheduledVersions drops any pending effective dates on a template whose
// approved content is no longer what it was. A schedule set against the old
// approval would otherwise fire, unreviewed, the moment the template is
// approved again.
func (s *Service) clearScheduledVersions(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
) (int64, error) {
	return s.versionRepo.ClearScheduled(ctx, &repositories.ListScheduledVersionsRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: template.OrganizationID,
			BuID:  template.BusinessUnitID,
		},
		TemplateID: template.ID,
	})
}

func withClearedSchedules(comment string, cleared int64) string {
	switch cleared {
	case 0:
		return comment
	case 1:
		return comment + "; cleared 1 scheduled version"
	default:
		return fmt.Sprintf("%s; cleared %d scheduled versions", comment, cleared)
	}
}

func (s *Service) Duplicate(
	ctx context.Context,
	req *repositories.BulkDuplicateFormulaTemplateRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	log := s.l.With(
		zap.String("operation", "Duplicate"),
		zap.Any("request", req),
	)

	sources, err := s.repo.GetByIDs(ctx, repositories.GetFormulaTemplatesByIDsRequest{
		TenantInfo:  req.TenantInfo,
		TemplateIDs: req.TemplateIDs,
	})
	if err != nil {
		log.Error("failed to load source templates", zap.Error(err))
		return nil, err
	}

	sourceNames := make(map[pulid.ID]string, len(sources))
	for _, source := range sources {
		if vErr := s.validateExpression(ctx, source); vErr != nil {
			return nil, vErr
		}
		sourceNames[source.ID] = source.Name
	}

	var entities []*formulatemplate.FormulaTemplate
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		duplicated, txErr := s.repo.BulkDuplicate(txCtx, req)
		if txErr != nil {
			log.Error("failed to duplicate formula template", zap.Error(txErr))
			return txErr
		}

		for _, entity := range duplicated {
			changeMessage := "Duplicated"
			if entity.SourceTemplateID != nil {
				if sourceName, ok := sourceNames[*entity.SourceTemplateID]; ok {
					changeMessage = "Duplicated from " + sourceName
				}
			}

			if txErr = s.createVersionSnapshot(
				txCtx,
				entity,
				1,
				req.TenantInfo.UserID,
				changeMessage,
				nil,
			); txErr != nil {
				log.Error("failed to create version snapshot", zap.Error(txErr))
				return txErr
			}
		}

		entities = duplicated
		return nil
	})
	if err != nil {
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
	if req.Status != formulatemplate.StatusInactive {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Bulk status updates can only archive templates; "+
				"an archived template is reactivated by submitting it for review",
		)
	}

	templates, err := s.repo.GetByIDs(ctx, repositories.GetFormulaTemplatesByIDsRequest{
		TenantInfo:  req.TenantInfo,
		TemplateIDs: req.TemplateIDs,
	})
	if err != nil {
		return nil, err
	}

	for _, template := range templates {
		if template.Status == formulatemplate.StatusActive ||
			template.Status == formulatemplate.StatusInactive {
			continue
		}

		return nil, errortypes.NewValidationError(
			"templateIds",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Template %s is %s; only Active and Inactive templates can be bulk updated",
				template.Name,
				template.Status,
			),
		)
	}

	previousStates := make(map[pulid.ID]*formulatemplate.FormulaTemplate, len(templates))
	for _, template := range templates {
		previousStates[template.ID] = template
	}

	updated, err := s.repo.BulkUpdateStatus(ctx, req)
	if err != nil {
		return nil, err
	}

	log := s.l.With(zap.String("operation", "BulkUpdateStatus"))
	for _, template := range updated {
		s.logAuditAction(
			log,
			template,
			permission.OpUpdate,
			req.TenantInfo.UserID,
			previousStates[template.ID],
			fmt.Sprintf("Status changed to %s in bulk update", req.Status),
		)
	}

	return updated, nil
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

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListFormulaTemplateConnectionRequest,
) (*pagination.CursorListResult[*formulatemplate.FormulaTemplate], error) {
	return s.repo.ListConnection(ctx, req)
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

	var createdVersion *formulatemplate.FormulaTemplateVersion
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, txErr := s.repo.Update(txCtx, template); txErr != nil {
			log.Error("failed to update template version number", zap.Error(txErr))
			return txErr
		}

		version := formulatemplate.NewVersionFromTemplate(
			template,
			newVersionNumber,
			req.TenantInfo.UserID,
			req.ChangeMessage,
			changeSummary,
		)

		created, txErr := s.versionRepo.Create(txCtx, version)
		if txErr != nil {
			log.Error("failed to create version", zap.Error(txErr))
			return txErr
		}

		createdVersion = created
		return nil
	})
	if err != nil {
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

	resolved := currentTemplate.ApplyVersionFull(targetVersion)
	resolved.CurrentVersionNumber = currentTemplate.CurrentVersionNumber + 1

	changeMessage := req.ChangeMessage
	if changeMessage == "" {
		changeMessage = fmt.Sprintf("Rolled back to version %d", req.TargetVersion)
	}

	auditComment := changeMessage
	if (currentTemplate.Status == formulatemplate.StatusActive ||
		currentTemplate.Status == formulatemplate.StatusInReview) &&
		resolved.HasMaterialChange(currentTemplate) {
		resolved.Status = formulatemplate.StatusDraft
		clearApprovalFields(resolved)
		auditComment = changeMessage + "; rollback reverted approval"
	}

	var updatedTemplate *formulatemplate.FormulaTemplate
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		updated, txErr := s.repo.Update(txCtx, resolved)
		if txErr != nil {
			log.Error("failed to update template", zap.Error(txErr))
			return txErr
		}

		if txErr = s.createVersionSnapshot(
			txCtx,
			updated,
			updated.CurrentVersionNumber,
			req.TenantInfo.UserID,
			changeMessage,
			nil,
		); txErr != nil {
			log.Error("failed to create version snapshot", zap.Error(txErr))
			return txErr
		}

		updatedTemplate = updated
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logAuditAction(
		log,
		updatedTemplate,
		permission.OpUpdate,
		req.TenantInfo.UserID,
		currentTemplate,
		auditComment,
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
		BreakdownDefinitions: snapshot.BreakdownDefinitions,
		MinCharge:            snapshot.MinCharge,
		MaxCharge:            snapshot.MaxCharge,
		RoundingMode:         snapshot.RoundingMode,
		RoundingPrecision:    snapshot.RoundingPrecision,
		Metadata:             snapshot.Metadata,
		SourceTemplateID:     &req.SourceTemplateID,
		SourceVersionNumber:  &sourceVersionNum,
		CurrentVersionNumber: 1,
	}

	changeMessage := req.ChangeMessage
	if changeMessage == "" {
		changeMessage = fmt.Sprintf("Forked from template %s", sourceTemplate.Name)
	}

	var createdTemplate *formulatemplate.FormulaTemplate
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		created, txErr := s.repo.Create(txCtx, forkedTemplate)
		if txErr != nil {
			log.Error("failed to create forked template", zap.Error(txErr))
			return txErr
		}

		if txErr = s.createVersionSnapshot(
			txCtx, created, 1, req.TenantInfo.UserID, changeMessage, nil,
		); txErr != nil {
			log.Error("failed to create version snapshot", zap.Error(txErr))
			return txErr
		}

		createdTemplate = created
		return nil
	})
	if err != nil {
		return nil, err
	}

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

	req.Tags = sliceutils.DedupeStrings(req.Tags)
	for _, tag := range req.Tags {
		if !formulatemplate.VersionTag(tag).IsValid() {
			return nil, errortypes.NewValidationError(
				"tags",
				errortypes.ErrInvalid,
				fmt.Sprintf("Invalid tag: %s", tag),
			)
		}
	}

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get template", zap.Error(err))
		return nil, err
	}

	version, err := s.versionRepo.UpdateTags(ctx, req)
	if err != nil {
		log.Error("failed to update version tags", zap.Error(err))
		return nil, err
	}

	s.logAuditAction(
		log,
		template,
		permission.OpUpdate,
		req.TenantInfo.UserID,
		nil,
		fmt.Sprintf(
			"Version %d tags updated to [%s]",
			req.VersionNumber,
			strings.Join(req.Tags, ", "),
		),
	)

	return version, nil
}

type TestExpressionRequest struct {
	Expression string
	SchemaID   string
	Variables  map[string]any
	ShipmentID *pulid.ID
	TenantInfo pagination.TenantInfo
	Breakdowns []*formulatypes.BreakdownDefinition
	MinCharge  decimal.NullDecimal
	MaxCharge  decimal.NullDecimal
	// RoundingMode and RoundingPrecision are the policy under test. An empty
	// mode means the default policy, exactly as it does on a stored template.
	RoundingMode      ratetypes.RoundingMode
	RoundingPrecision int32
}

func (r *TestExpressionRequest) chargePolicy() formulatypes.ChargePolicy {
	return formulatypes.ChargePolicy{
		MinCharge:         r.MinCharge,
		MaxCharge:         r.MaxCharge,
		RoundingMode:      r.RoundingMode,
		RoundingPrecision: r.RoundingPrecision,
	}
}

const (
	msgExpressionValidationFailed = "Expression validation failed"

	// previewEvaluationTimeout is the leash on an interactive preview. It runs
	// on every keystroke, so a runaway expression should fail fast rather than
	// hold the Studio for the engine's full batch ceiling.
	previewEvaluationTimeout = 2 * time.Second
)

type TestExpressionResponse struct {
	Valid             bool                                   `json:"valid"`
	Result            any                                    `json:"result,omitempty"`
	Error             string                                 `json:"error,omitempty"`
	Message           string                                 `json:"message"`
	Breakdown         []formulatemplatetypes.BreakdownAmount `json:"breakdown,omitempty"`
	ResolvedVariables map[string]any                         `json:"resolvedVariables,omitempty"`
	Guardrail         *formulatemplatetypes.GuardrailResult  `json:"guardrail,omitempty"`
	Rounding          *formulatemplatetypes.RoundingResult   `json:"rounding,omitempty"`
}

func (s *Service) DescribeSchema(
	schemaID string,
) (*formulatemplatetypes.SchemaDescription, error) {
	return s.formulaService.DescribeSchema(schemaID)
}

func (s *Service) TestExpression(
	ctx context.Context,
	req *TestExpressionRequest,
) *TestExpressionResponse {
	ctx = engine.WithEvaluationTimeout(ratetablecache.With(ctx), previewEvaluationTimeout)

	err := s.formulaService.ValidateLookupTables(ctx, req.Expression, req.TenantInfo)
	if err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: msgExpressionValidationFailed,
		}
	}

	if req.ShipmentID != nil {
		return s.testExpressionAgainstShipment(ctx, req)
	}

	lookup, err := s.lookupForTest(ctx, req)
	if err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: "Failed to load rate tables",
		}
	}

	return s.testExpressionWithEnv(ctx, req, lookup)
}

// lookupForTest loads the tenant's rate tables only when the expression or a
// breakdown line names one. A preview re-runs on every keystroke, and most
// formulas never touch a table; loading every matrix for them would turn the
// live preview into the slowest thing on the page. When a table is named the
// real tables are used, so the number an author sees is the number a shipment
// would be charged.
func (s *Service) lookupForTest(
	ctx context.Context,
	req *TestExpressionRequest,
) (formulatemplatetypes.RateTableLookup, error) {
	if !referencesLookupTables(req.Expression, req.Breakdowns) {
		return nil, nil
	}

	return s.formulaService.BuildLookup(ctx, req.TenantInfo)
}

func referencesLookupTables(
	expression string,
	breakdowns []*formulatypes.BreakdownDefinition,
) bool {
	if tables, err := engine.ExtractLookupTables(expression); err == nil && len(tables) > 0 {
		return true
	}

	for _, def := range breakdowns {
		if def == nil {
			continue
		}
		if tables, err := engine.ExtractLookupTables(def.Expression); err == nil &&
			len(tables) > 0 {
			return true
		}
	}

	return false
}

func (s *Service) testExpressionAgainstShipment(
	ctx context.Context,
	req *TestExpressionRequest,
) *TestExpressionResponse {
	entity, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:              *req.ShipmentID,
		TenantInfo:      req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{ExpandShipmentDetails: true},
	})
	if err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: "Failed to load shipment",
		}
	}

	resp, err := s.formulaService.EvaluateExpression(ctx, &formula.EvaluateExpressionRequest{
		Expression: req.Expression,
		Entity:     entity,
		SchemaID:   req.SchemaID,
		Variables:  req.Variables,
		Breakdowns: req.Breakdowns,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: "Expression evaluation failed",
		}
	}

	amount, guardrail, rounding := formula.ApplyChargePolicy(req.chargePolicy(), resp.Amount)

	return &TestExpressionResponse{
		Valid:             true,
		Result:            amount,
		Breakdown:         resp.Breakdown,
		ResolvedVariables: maputils.WithoutFuncValues(resp.Variables),
		Guardrail:         guardrail,
		Rounding:          rounding,
		Message:           "Expression evaluated against shipment",
	}
}

func (s *Service) testExpressionWithEnv(
	ctx context.Context,
	req *TestExpressionRequest,
	lookup formulatemplatetypes.RateTableLookup,
) *TestExpressionResponse {
	env, err := s.formulaService.BuildValidationEnvironment(req.SchemaID, req.Variables)
	if err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: msgExpressionValidationFailed,
		}
	}

	if err = s.formulaService.ValidateExpressionWithEnv(ctx, req.Expression, env); err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: msgExpressionValidationFailed,
		}
	}

	result, err := s.formulaService.EvaluateWithEnv(ctx, &formulatemplatetypes.EnvEvaluationRequest{
		Expression: req.Expression,
		Env:        env,
		Lookup:     lookup,
	})
	if err != nil {
		return &TestExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: "Expression evaluation failed",
		}
	}

	amount, guardrail, rounding := formula.ApplyChargePolicy(req.chargePolicy(), result.Amount)

	resp := &TestExpressionResponse{
		Valid:     true,
		Result:    amount,
		Guardrail: guardrail,
		Rounding:  rounding,
		Message:   "Expression is valid",
	}

	if len(req.Breakdowns) > 0 {
		resp.Breakdown = s.evaluateBreakdownsWithEnv(ctx, req.Breakdowns, env, lookup)
	}

	return resp
}

func (s *Service) evaluateBreakdownsWithEnv(
	ctx context.Context,
	defs []*formulatypes.BreakdownDefinition,
	env map[string]any,
	lookup formulatemplatetypes.RateTableLookup,
) []formulatemplatetypes.BreakdownAmount {
	items := make([]formulatemplatetypes.BreakdownAmount, 0, len(defs))
	for _, def := range defs {
		if def == nil {
			continue
		}

		item := formulatemplatetypes.BreakdownAmount{Name: def.Name, Label: def.Label}
		result, err := s.formulaService.EvaluateWithEnv(
			ctx,
			&formulatemplatetypes.EnvEvaluationRequest{
				Expression: def.Expression,
				Env:        env,
				Lookup:     lookup,
			},
		)
		if err != nil {
			item.Error = err.Error()
		} else {
			item.Amount = result.Amount
		}

		items = append(items, item)
	}

	return items
}

func (s *Service) validateTemplate(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) error {
	entity.NormalizeRounding()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return s.validateExpression(ctx, entity)
}

func (s *Service) validateExpression(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) error {
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
		return errortypes.NewValidationError(
			"schemaId",
			errortypes.ErrInvalid,
			expressionErrorMessage(err),
		)
	}

	outcome := s.formulaService.ValidateExpressionDetailed(ctx, entity.Expression, env)
	if outcome.Err != nil {
		return errortypes.NewValidationError(
			"expression",
			errortypes.ErrInvalid,
			expressionErrorMessage(outcome.Err),
		)
	}
	if outcome.Warning != "" {
		s.l.Warn("expression produced a runtime error against the synthetic validation environment",
			zap.String("expression", entity.Expression),
			zap.String("warning", outcome.Warning),
		)
	}

	multiErr := errortypes.NewMultiError()
	for i, def := range entity.BreakdownDefinitions {
		if def == nil {
			continue
		}
		defOutcome := s.formulaService.ValidateExpressionDetailed(ctx, def.Expression, env)
		if defOutcome.Err != nil {
			multiErr.Add(
				fmt.Sprintf("breakdownDefinitions[%d].expression", i),
				errortypes.ErrInvalid,
				expressionErrorMessage(defOutcome.Err),
			)
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	if err = s.formulaService.ValidateLookupTables(ctx, entity.Expression, tenantInfo); err != nil {
		return err
	}

	for _, def := range entity.BreakdownDefinitions {
		if def == nil {
			continue
		}
		if err = s.formulaService.ValidateLookupTables(
			ctx,
			def.Expression,
			tenantInfo,
		); err != nil {
			return err
		}
	}

	return nil
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
	template *formulatemplate.FormulaTemplate,
	versionNumber int64,
	userID pulid.ID,
	changeMessage string,
	changeSummary map[string]jsonutils.FieldChange,
) error {
	version := formulatemplate.NewVersionFromTemplate(
		template,
		versionNumber,
		userID,
		changeMessage,
		changeSummary,
	)

	_, err := s.versionRepo.Create(ctx, version)
	return err
}

func (s *Service) logAuditAction(
	log *zap.Logger,
	entity *formulatemplate.FormulaTemplate,
	operation permission.Operation,
	userID pulid.ID,
	previousState *formulatemplate.FormulaTemplate,
	comment string,
	extraOpts ...services.LogOption,
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

	opts = append(opts, extraOpts...)

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
