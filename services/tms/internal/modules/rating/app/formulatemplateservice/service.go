package formulatemplateservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type sourceChecker interface {
	CheckSource(ctx context.Context, req *tariff.SourceCheckRequest) (*tariff.SourceCheck, error)
	RunTests(ctx context.Context, req *tariff.TestRequest) (*rules.TariffTestRun, error)
	TestChecked(
		ctx context.Context,
		checked *tariff.SourceCheck,
		source string,
	) (*rules.TariffTestRun, error)
	ScaffoldTest(ctx context.Context, req *tariff.ScaffoldRequest) (*rules.TariffScaffold, error)
	CheckWithContext(
		ctx context.Context,
		req *tariff.ContextCheckRequest,
	) (*rules.TariffCheck, error)
	Context(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		libraries tariff.LibrarySet,
	) (*rules.TariffContext, error)
	TableStamp(ctx context.Context, tenantInfo pagination.TenantInfo) (string, error)
	TableCells(ctx context.Context, req *tariff.TableCellsRequest) (*tariff.TableCells, error)
}

type billingResolver interface {
	BillingVersion(
		ctx context.Context,
		template *formulatemplate.FormulaTemplate,
		tenantInfo pagination.TenantInfo,
		asOf int64,
	) (tariff.BillingResolution, error)
	PinnedLibraries(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		imports []tarifftypes.Import,
	) ([]rules.TariffLibrary, error)
	LatestLibraries(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		sources ...string,
	) (tariff.LibrarySet, error)
}

type approvedLibraries interface {
	ListLatestApproved(
		ctx context.Context,
		req *formulalibraryports.ListApprovedLibrariesRequest,
	) ([]*formulalibrary.FormulaLibraryVersion, error)
}

type Params struct {
	fx.In

	Logger          *zap.Logger
	DB              kernelports.DBConnection
	Repo            formulatemplateports.FormulaTemplateRepository
	VersionRepo     formulatemplateports.FormulaTemplateVersionRepository
	ReviewRepo      formulatemplateports.FormulaTemplateReviewRepository
	LibraryVersions formulalibraryports.FormulaLibraryVersionRepository
	Checker         *tariff.Checker
	Resolver        *tariff.TemplateResolver
	Rater           tariffports.Rater
	Dependents      rateagreementports.TemplateInputsLister
	Records         tariffports.RecordBuilder  `optional:"true"`
	RatedShipments  tariffports.RatedShipments `optional:"true"`
	AuditService    auditports.AuditService
	Notifications   *notificationservice.Service   `optional:"true"`
	Roles           permissionports.RoleRepository `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	db            kernelports.DBConnection
	repo          formulatemplateports.FormulaTemplateRepository
	versionRepo   formulatemplateports.FormulaTemplateVersionRepository
	reviewRepo    formulatemplateports.FormulaTemplateReviewRepository
	libraries     approvedLibraries
	checker       sourceChecker
	resolver      billingResolver
	rater         tariffports.Rater
	dependents    rateagreementports.TemplateInputsLister
	records       tariffports.RecordBuilder
	rated         tariffports.RatedShipments
	auditService  auditports.AuditService
	notifications *notificationservice.Service
	roles         permissionports.RoleRepository
}

func New(p Params) *Service { //nolint:gocritic // fx param structs are passed by value
	return &Service{
		l:             p.Logger.Named("service.formulatemplate"),
		db:            p.DB,
		repo:          p.Repo,
		versionRepo:   p.VersionRepo,
		reviewRepo:    p.ReviewRepo,
		libraries:     p.LibraryVersions,
		checker:       p.Checker,
		resolver:      p.Resolver,
		rater:         p.Rater,
		dependents:    p.Dependents,
		records:       p.Records,
		rated:         p.RatedShipments,
		auditService:  p.AuditService,
		notifications: p.Notifications,
		roles:         p.Roles,
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

	entity.Status = formulatemplate.StatusDraft
	clearApprovalFields(entity)

	if err := s.prepareContent(ctx, entity); err != nil {
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

		createdEntity = created
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logAuditAction(
		ctx,
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
	payload *formulatemplate.FormulaTemplate,
	userID pulid.ID,
) (*formulatemplate.FormulaTemplate, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("id", payload.ID.String()),
	)

	original, err := s.getTemplateByID(
		ctx,
		payload.ID,
		payload.GetOrganizationID(),
		payload.GetBusinessUnitID(),
	)
	if err != nil {
		log.Error("failed to get original formula template", zap.Error(err))
		return nil, err
	}

	entity := withContent(original, payload)
	if err = s.prepareContent(ctx, entity); err != nil {
		return nil, err
	}

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
		ctx,
		log,
		updatedEntity,
		permission.OpUpdate,
		userID,
		original,
		auditComment,
		criticalWhen(material)...,
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
			ctx,
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

	if err = s.guardArchivingInUse(ctx, req, templates); err != nil {
		return nil, err
	}

	previousStates := make(map[pulid.ID]*formulatemplate.FormulaTemplate, len(templates))
	for _, template := range templates {
		previousStates[template.ID] = template
	}

	var updated []*formulatemplate.FormulaTemplate
	cleared := make(map[pulid.ID]int64, len(templates))
	err = s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		saved, txErr := s.repo.BulkUpdateStatus(txCtx, req)
		if txErr != nil {
			return txErr
		}

		for _, template := range saved {
			count, clearErr := s.clearScheduledVersions(txCtx, template)
			if clearErr != nil {
				return clearErr
			}
			cleared[template.ID] = count
		}

		updated = saved
		return nil
	})
	if err != nil {
		return nil, err
	}

	log := s.l.With(zap.String("operation", "BulkUpdateStatus"))
	for _, template := range updated {
		s.logAuditAction(
			ctx,
			log,
			template,
			permission.OpUpdate,
			req.TenantInfo.UserID,
			previousStates[template.ID],
			withClearedSchedules(
				fmt.Sprintf("Status changed to %s in bulk update", req.Status),
				cleared[template.ID],
			),
			auditservice.WithCritical(),
		)
	}

	return updated, nil
}

func (s *Service) guardArchivingInUse(
	ctx context.Context,
	req *formulatemplateports.BulkUpdateFormulaTemplateStatusRequest,
	templates []*formulatemplate.FormulaTemplate,
) error {
	inUse := make([]string, 0, len(templates))
	for _, template := range templates {
		if template.Status != formulatemplate.StatusActive {
			continue
		}

		usage, err := s.repo.CountUsages(ctx, &formulatemplateports.GetTemplateUsageRequest{
			TemplateID: template.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return err
		}
		if usage == nil || !usage.InUse {
			continue
		}

		inUse = append(inUse, template.Name+" ("+describeUsages(usage.Usages)+")")
	}

	if len(inUse) == 0 {
		return nil
	}

	if !req.MayArchiveInUse {
		return errortypes.NewAuthorizationError(
			"Archiving a formula template that is still in use needs approve permission",
		)
	}

	if !req.ConfirmInUse {
		return errortypes.NewValidationError(
			"confirmInUse",
			errortypes.ErrInvalid,
			"Archiving stops these templates rating anything: {0}. Confirm to archive them anyway",
			strings.Join(inUse, "; "),
		)
	}

	return nil
}

var usageNouns = map[string][2]string{
	"shipment":                 {"shipment", "shipments"},
	"rate_agreement_rule":      {"rate agreement rule", "rate agreement rules"},
	"billing_control_fallback": {"billing fallback", "billing fallbacks"},
}

func describeUsages(usages []formulatemplateports.TemplateUsageCount) string {
	parts := make([]string, 0, len(usages))
	for _, usage := range usages {
		if usage.Count <= 0 {
			continue
		}
		noun, ok := usageNouns[usage.Type]
		if !ok {
			noun = [2]string{usage.Type, usage.Type}
		}
		parts = append(parts, stringutils.CountNoun(usage.Count, noun[0], noun[1]))
	}
	return strings.Join(parts, ", ")
}

func (s *Service) GetByID(
	ctx context.Context,
	req formulatemplateports.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	template, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	billing, err := s.resolver.BillingVersion(
		ctx,
		template,
		req.TenantInfo,
		timeutils.NowUnix(),
	)
	if err != nil {
		return nil, err
	}
	switch {
	case billing.InForce != nil:
		template.Billing = &formulatemplate.BillingVersion{
			VersionNumber: billing.InForce.VersionNumber,
			EffectiveFrom: billing.InForce.EffectiveStart(),
		}
	case billing.Scheduled != nil:
		template.Billing = &formulatemplate.BillingVersion{
			VersionNumber: billing.Scheduled.VersionNumber,
			EffectiveFrom: billing.Scheduled.EffectiveStart(),
			Pending:       true,
		}
	}

	return template, nil
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
	ctx context.Context,
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

	opts := []auditports.LogOption{
		auditservice.WithComment(comment),
		auditservice.WithRequest(ctx),
	}

	if previousState != nil {
		params.PreviousState = jsonutils.MustToJSON(previousState)
		opts = append(opts, auditservice.WithDiff(previousState, entity))
	}

	opts = append(opts, extraOpts...)

	if err := s.auditService.LogAction(params, opts...); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}
}

func criticalWhen(critical bool) []auditports.LogOption {
	if !critical {
		return nil
	}
	return []auditports.LogOption{auditservice.WithCritical()}
}

const maxBulkTemplateIDs = 100
