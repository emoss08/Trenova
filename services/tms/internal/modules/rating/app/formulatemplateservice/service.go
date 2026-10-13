package formulatemplateservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	DB             kernelports.DBConnection
	Repo           formulatemplateports.FormulaTemplateRepository
	VersionRepo    formulatemplateports.FormulaTemplateVersionRepository
	TestCaseRepo   formulatemplateports.FormulaTemplateTestCaseRepository
	ReviewRepo     formulatemplateports.FormulaTemplateReviewRepository
	ShipmentRepo   shipmentports.ShipmentRepository
	FormulaService *formula.Service
	AuditService   auditports.AuditService
	Notifications  *notificationservice.Service   `optional:"true"`
	Roles          permissionports.RoleRepository `optional:"true"`
}

type Service struct {
	l              *zap.Logger
	db             kernelports.DBConnection
	repo           formulatemplateports.FormulaTemplateRepository
	versionRepo    formulatemplateports.FormulaTemplateVersionRepository
	testCaseRepo   formulatemplateports.FormulaTemplateTestCaseRepository
	reviewRepo     formulatemplateports.FormulaTemplateReviewRepository
	shipmentRepo   shipmentports.ShipmentRepository
	formulaService *formula.Service
	auditService   auditports.AuditService
	notifications  *notificationservice.Service
	roles          permissionports.RoleRepository
}

func New(p Params) *Service { //nolint:gocritic // fx param structs are passed by value
	return &Service{
		l:              p.Logger.Named("service.formulatemplate"),
		db:             p.DB,
		repo:           p.Repo,
		versionRepo:    p.VersionRepo,
		testCaseRepo:   p.TestCaseRepo,
		reviewRepo:     p.ReviewRepo,
		shipmentRepo:   p.ShipmentRepo,
		formulaService: p.FormulaService,
		auditService:   p.AuditService,
		notifications:  p.Notifications,
		roles:          p.Roles,
	}
}

func (s *Service) Create(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
	userID pulid.ID,
) (*formulatemplate.FormulaTemplate, error) {
	return s.CreateWithTestCases(ctx, entity, userID, nil)
}

func (s *Service) CreateWithTestCases(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
	userID pulid.ID,
	testCases []*TestCaseInput,
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

	if err := validateHeldTestCases(testCases); err != nil {
		return nil, err
	}

	entity.CurrentVersionNumber = 1

	var createdEntity *formulatemplate.FormulaTemplate
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
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

		if txErr = s.createRecordedTestCases(
			txCtx,
			created,
			pagination.TenantInfo{
				OrgID:  created.OrganizationID,
				BuID:   created.BusinessUnitID,
				UserID: userID,
			},
			testCases,
		); txErr != nil {
			log.Error("failed to create held test scenarios", zap.Error(txErr))
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
	err = s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
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

func (s *Service) Duplicate(
	ctx context.Context,
	req *formulatemplateports.BulkDuplicateFormulaTemplateRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	log := s.l.With(
		zap.String("operation", "Duplicate"),
		zap.Any("request", req),
	)

	if err := validateBulkTemplateIDs(req.TemplateIDs); err != nil {
		return nil, err
	}

	sources, err := s.repo.GetByIDs(ctx, formulatemplateports.GetFormulaTemplatesByIDsRequest{
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
	err = s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
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
	req *formulatemplateports.BulkUpdateFormulaTemplateStatusRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	if req.Status != formulatemplate.StatusInactive {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Bulk status updates can only archive templates; "+
				"an archived template is reactivated by submitting it for review",
		)
	}

	if err := validateBulkTemplateIDs(req.TemplateIDs); err != nil {
		return nil, err
	}

	templates, err := s.repo.GetByIDs(ctx, formulatemplateports.GetFormulaTemplatesByIDsRequest{
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
			"Template {0} is {1}; only Active and Inactive templates can be bulk updated",
			template.Name,
			template.Status,
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
	req formulatemplateports.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) List(
	ctx context.Context,
	req *formulatemplateports.ListFormulaTemplatesRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *formulatemplateports.ListFormulaTemplateConnectionRequest,
) (*pagination.CursorListResult[*formulatemplate.FormulaTemplate], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) GetUsage(
	ctx context.Context,
	req *formulatemplateports.GetTemplateUsageRequest,
) (*formulatemplateports.GetTemplateUsageResponse, error) {
	return s.repo.CountUsages(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *formulatemplateports.FormulaTemplateSelectOptionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) getTemplateByID(
	ctx context.Context,
	id, orgID, buID pulid.ID,
) (*formulatemplate.FormulaTemplate, error) {
	return s.repo.GetByID(ctx, formulatemplateports.GetFormulaTemplateByIDRequest{
		TemplateID: id,
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	})
}

func (s *Service) getTemplateByIDWithTenant(
	ctx context.Context,
	id pulid.ID,
	tenant pagination.TenantInfo,
) (*formulatemplate.FormulaTemplate, error) {
	return s.repo.GetByID(ctx, formulatemplateports.GetFormulaTemplateByIDRequest{
		TemplateID: id,
		TenantInfo: tenant,
	})
}

func (s *Service) logAuditAction(
	log *zap.Logger,
	entity *formulatemplate.FormulaTemplate,
	operation permission.Operation,
	userID pulid.ID,
	previousState *formulatemplate.FormulaTemplate,
	comment string,
	extraOpts ...auditports.LogOption,
) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceFormulaTemplate,
		ResourceID:     entity.GetID().String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(entity),
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}

	opts := []auditports.LogOption{auditservice.WithComment(comment)}

	if previousState != nil {
		params.PreviousState = jsonutils.MustToJSON(previousState)
		opts = append(opts, auditservice.WithDiff(previousState, entity))
	}

	opts = append(opts, extraOpts...)

	if err := s.auditService.LogAction(params, opts...); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}
}

const MaxHeldTestCases = 50

func validateHeldTestCases(testCases []*TestCaseInput) error {
	multiErr := errortypes.NewMultiError()
	if len(testCases) > MaxHeldTestCases {
		multiErr.Add(
			"testCases",
			errortypes.ErrInvalid,
			"A new template can carry at most 50 test scenarios",
		)
		return multiErr
	}

	pendingTemplateID := pulid.MustNew("ft_")
	seen := make(map[string]int, len(testCases))
	for index, input := range testCases {
		caseErrs := multiErr.WithIndex("testCases", index)
		if input == nil {
			caseErrs.Add("name", errortypes.ErrRequired, "Name is required")
			continue
		}

		entity := buildTestCaseEntity(pendingTemplateID, pagination.TenantInfo{}, input)
		entity.Validate(caseErrs)
		ValidateScenarioContent(caseErrs, "candidate", input.Candidate)

		name := input.Name
		if strings.TrimSpace(name) == "" {
			continue
		}
		if first, dup := seen[name]; dup {
			caseErrs.Add(
				"name",
				errortypes.ErrDuplicate,
				"Scenario {0} has the same name as scenario {1}",
				index+1,
				first+1,
			)
			continue
		}
		seen[name] = index
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// maxBulkTemplateIDs bounds the id lists bulk operations accept. A duplicate
// of a hundred templates is already a strange request; an unbounded one is a
// way to hold a transaction open for as long as the caller likes.
const maxBulkTemplateIDs = 100
