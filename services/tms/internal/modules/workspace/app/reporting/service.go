package reporting

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger           *zap.Logger
	DefinitionRepo   reportports.ReportDefinitionRepository
	DashboardRepo    reportports.ReportDashboardRepository
	RunRepo          reportports.ReportRunRepository
	ScheduleRepo     reportports.ReportScheduleRepository
	ViewRepo         reportports.ReportViewRepository
	OrganizationRepo tenantports.OrganizationRepository
	UserRepo         tenantports.UserRepository
	Compiler         reportports.ReportCompiler
	Executor         reportports.ReportDatasetExecutor
	PermissionEngine permissionports.PermissionEngine
	FieldAccess      permissionports.FieldAccess
	ReferenceViews   permissionports.ReferenceViews
	WorkflowStarter  kernelports.WorkflowStarter
	Storage          kernelports.Client
	AuditService     auditports.AuditService `optional:"true"`
	Metrics          *metrics.Registry       `optional:"true"`
	Config           *config.Config
}

type Service struct {
	l             *zap.Logger
	defRepo       reportports.ReportDefinitionRepository
	dashboardRepo reportports.ReportDashboardRepository
	runRepo       reportports.ReportRunRepository
	scheduleRepo  reportports.ReportScheduleRepository
	viewRepo      reportports.ReportViewRepository
	orgRepo       tenantports.OrganizationRepository
	userRepo      tenantports.UserRepository
	compiler      reportports.ReportCompiler
	executor      reportports.ReportDatasetExecutor
	perms         permissionports.PermissionEngine
	access        *reportaccess.Resolver
	workflows     kernelports.WorkflowStarter
	storage       kernelports.Client
	audit         auditports.AuditService
	metrics       *metrics.Report
	cfg           *config.ReportingConfig
	previews      *previewLimiter
	canned        *canned.Registry
}

//nolint:gocritic // fx.In parameter structs must be passed by value
func New(p Params) *Service {
	var reportMetrics *metrics.Report
	if p.Metrics != nil {
		reportMetrics = p.Metrics.Report
	}

	registry := permission.NewRegistry()

	return &Service{
		l:             p.Logger.Named("service.reporting"),
		defRepo:       p.DefinitionRepo,
		dashboardRepo: p.DashboardRepo,
		runRepo:       p.RunRepo,
		scheduleRepo:  p.ScheduleRepo,
		viewRepo:      p.ViewRepo,
		orgRepo:       p.OrganizationRepo,
		userRepo:      p.UserRepo,
		compiler:      p.Compiler,
		executor:      p.Executor,
		perms:         p.PermissionEngine,
		access: reportaccess.New(reportaccess.Params{
			Catalog:  &reportcatalog.Default,
			Engine:   p.PermissionEngine,
			Registry: registry,
			Views:    p.ReferenceViews,
			Fields:   p.FieldAccess,
		}),
		workflows: p.WorkflowStarter,
		storage:   p.Storage,
		audit:     p.AuditService,
		metrics:   reportMetrics,
		cfg:       p.Config.GetReportingConfig(),
		previews:  newPreviewLimiter(),
		canned:    canned.Default(),
	}
}

type Request struct {
	TenantInfo pagination.TenantInfo
	Principal  kernelports.PrincipalInfo
}

type SaveDefinitionRequest struct {
	Request

	DefinitionID  pulid.ID
	Name          string
	Description   string
	Category      string
	Tags          []string
	Visibility    report.Visibility
	Status        report.DefinitionStatus
	DefaultFormat report.Format
	Definition    *report.Definition
	Version       int64
}

func (s *Service) orgTimezone(ctx context.Context, tenant pagination.TenantInfo) string {
	org, err := s.orgRepo.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenant,
	})
	if err != nil || org.Timezone == "" {
		return "UTC"
	}
	return org.Timezone
}

// ValidateDefinition compiles a definition as the caller may see it, without
// saving anything. It is what an agent checks a proposal against before a
// person is asked to approve it.
func (s *Service) ValidateDefinition(
	ctx context.Context,
	req *SaveDefinitionRequest,
) error {
	return s.validateDefinition(ctx, req)
}

func (s *Service) validateDefinition(
	ctx context.Context,
	req *SaveDefinitionRequest,
) error {
	_, err := s.compiler.ValidateAndAuthorize(ctx, &reportports.ReportCompileRequest{
		Definition:  req.Definition,
		Tenant:      req.TenantInfo,
		OrgTimezone: s.orgTimezone(ctx, req.TenantInfo),
		NowUnix:     timeutils.NowUnix(),
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordCompileError("validation")
		}
		return err
	}
	return nil
}

func (s *Service) CreateDefinition(
	ctx context.Context,
	req *SaveDefinitionRequest,
) (*report.ReportDefinition, error) {
	log := s.l.With(zap.String("operation", "CreateDefinition"), zap.String("name", req.Name))

	if err := s.validateDefinition(ctx, req); err != nil {
		return nil, err
	}

	entity := NewDefinition(req)

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.defRepo.Create(ctx, entity, req.TenantInfo.UserID)
	if err != nil {
		log.Error("failed to create report definition", zap.Error(err))
		return nil, err
	}

	return created, nil
}

func (s *Service) UpdateDefinition(
	ctx context.Context,
	req *SaveDefinitionRequest,
) (*report.ReportDefinition, error) {
	log := s.l.With(
		zap.String("operation", "UpdateDefinition"),
		zap.String("id", req.DefinitionID.String()),
	)

	existing, err := s.defRepo.GetByID(ctx, &reportports.GetReportDefinitionRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: req.DefinitionID,
	})
	if err != nil {
		return nil, err
	}
	if existing.OwnerID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError(
			"Only the report owner can modify this report",
		)
	}

	if err = s.validateDefinition(ctx, req); err != nil {
		return nil, err
	}

	ApplyDefinitionSave(existing, req)

	multiErr := errortypes.NewMultiError()
	existing.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.defRepo.Update(ctx, existing, req.TenantInfo.UserID)
	if err != nil {
		log.Error("failed to update report definition", zap.Error(err))
		return nil, err
	}

	return updated, nil
}

type GetDefinitionRequest struct {
	Request

	DefinitionID pulid.ID
}

func (s *Service) GetDefinition(
	ctx context.Context,
	req *GetDefinitionRequest,
) (*report.ReportDefinition, error) {
	entity, err := s.defRepo.GetByID(ctx, &reportports.GetReportDefinitionRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: req.DefinitionID,
	})
	if err != nil {
		return nil, err
	}

	if err = s.checkDefinitionVisibility(entity, req.TenantInfo); err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *Service) checkDefinitionVisibility(
	entity *report.ReportDefinition,
	tenant pagination.TenantInfo,
) error {
	if entity.Visibility == report.VisibilityShared || entity.OwnerID == tenant.UserID {
		return nil
	}
	return errortypes.NewNotFoundError("ReportDefinition not found")
}

type ListDefinitionsRequest struct {
	Request

	Statuses []report.DefinitionStatus
	Limit    int
	Offset   int
}

func (s *Service) ListDefinitions(
	ctx context.Context,
	req *ListDefinitionsRequest,
) ([]*report.ReportDefinition, error) {
	return s.defRepo.List(ctx, &reportports.ListReportDefinitionsRequest{
		TenantInfo: req.TenantInfo,
		ViewerID:   req.TenantInfo.UserID,
		Statuses:   req.Statuses,
		Limit:      req.Limit,
		Offset:     req.Offset,
	})
}

func (s *Service) DeleteDefinition(ctx context.Context, req *GetDefinitionRequest) error {
	if _, err := s.PlanDeleteDefinition(ctx, req); err != nil {
		return err
	}

	return s.defRepo.Delete(ctx, &reportports.DeleteReportDefinitionRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: req.DefinitionID,
	})
}

type ListRevisionsRequest struct {
	Request

	DefinitionID pulid.ID
	Limit        int
}

func (s *Service) ListRevisions(
	ctx context.Context,
	req *ListRevisionsRequest,
) ([]*report.ReportDefinitionRevision, error) {
	if _, err := s.GetDefinition(ctx, &GetDefinitionRequest{
		Request:      req.Request,
		DefinitionID: req.DefinitionID,
	}); err != nil {
		return nil, err
	}

	return s.defRepo.ListRevisions(ctx, &reportports.ListReportRevisionsRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: req.DefinitionID,
		Limit:        req.Limit,
	})
}

func (s *Service) headRevision(
	ctx context.Context,
	tenant pagination.TenantInfo,
	definitionID pulid.ID,
) (*report.ReportDefinitionRevision, error) {
	revisions, err := s.defRepo.ListRevisions(ctx, &reportports.ListReportRevisionsRequest{
		TenantInfo:   tenant,
		DefinitionID: definitionID,
		Limit:        1,
	})
	if err != nil {
		return nil, err
	}
	if len(revisions) == 0 {
		return nil, fmt.Errorf("definition %s has no revisions", definitionID)
	}
	return revisions[0], nil
}

func defaultVisibility(v report.Visibility) report.Visibility {
	if !v.IsValid() {
		return report.VisibilityPrivate
	}
	return v
}

func defaultStatus(s report.DefinitionStatus) report.DefinitionStatus {
	if !s.IsValid() {
		return report.DefinitionStatusActive
	}
	return s
}

func defaultFormat(f report.Format) report.Format {
	if !f.IsValid() {
		return report.FormatCSV
	}
	return f
}

func (s *Service) ListDefinitionsConnection(
	ctx context.Context,
	req *reportports.ListReportDefinitionConnectionRequest,
) (*pagination.CursorListResult[*report.ReportDefinition], error) {
	return s.defRepo.ListConnection(ctx, req)
}
