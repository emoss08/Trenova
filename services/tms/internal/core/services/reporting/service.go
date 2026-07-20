package reporting

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
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
	DefinitionRepo   repositories.ReportDefinitionRepository
	RunRepo          repositories.ReportRunRepository
	ScheduleRepo     repositories.ReportScheduleRepository
	OrganizationRepo repositories.OrganizationRepository
	UserRepo         repositories.UserRepository
	Compiler         services.ReportCompiler
	Executor         services.ReportDatasetExecutor
	WorkflowStarter  services.WorkflowStarter
	Storage          storage.Client
	AuditService     services.AuditService `optional:"true"`
	Metrics          *metrics.Registry     `optional:"true"`
	Config           *config.Config
}

type Service struct {
	l            *zap.Logger
	defRepo      repositories.ReportDefinitionRepository
	runRepo      repositories.ReportRunRepository
	scheduleRepo repositories.ReportScheduleRepository
	orgRepo      repositories.OrganizationRepository
	userRepo     repositories.UserRepository
	compiler     services.ReportCompiler
	executor     services.ReportDatasetExecutor
	workflows    services.WorkflowStarter
	storage      storage.Client
	audit        services.AuditService
	metrics      *metrics.Report
	cfg          *config.ReportingConfig
	previews     *previewLimiter
	canned       *canned.Registry
}

//nolint:gocritic // fx.In parameter structs must be passed by value
func New(p Params) *Service {
	var reportMetrics *metrics.Report
	if p.Metrics != nil {
		reportMetrics = p.Metrics.Report
	}

	return &Service{
		l:            p.Logger.Named("service.reporting"),
		defRepo:      p.DefinitionRepo,
		runRepo:      p.RunRepo,
		scheduleRepo: p.ScheduleRepo,
		orgRepo:      p.OrganizationRepo,
		userRepo:     p.UserRepo,
		compiler:     p.Compiler,
		executor:     p.Executor,
		workflows:    p.WorkflowStarter,
		storage:      p.Storage,
		audit:        p.AuditService,
		metrics:      reportMetrics,
		cfg:          p.Config.GetReportingConfig(),
		previews:     newPreviewLimiter(),
		canned:       canned.Default(),
	}
}

type Request struct {
	TenantInfo pagination.TenantInfo
	Principal  services.PrincipalInfo
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
	org, err := s.orgRepo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenant,
	})
	if err != nil || org.Timezone == "" {
		return "UTC"
	}
	return org.Timezone
}

func (s *Service) validateDefinition(
	ctx context.Context,
	req *SaveDefinitionRequest,
) error {
	_, err := s.compiler.ValidateAndAuthorize(ctx, &services.ReportCompileRequest{
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

	entity := &report.ReportDefinition{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		Name:           req.Name,
		Description:    req.Description,
		Category:       req.Category,
		Tags:           req.Tags,
		Kind:           report.DefinitionKindCustom,
		OwnerID:        req.TenantInfo.UserID,
		Visibility:     defaultVisibility(req.Visibility),
		Status:         defaultStatus(req.Status),
		CatalogVersion: reportcatalog.Version,
		Definition:     req.Definition,
		DefaultFormat:  defaultFormat(req.DefaultFormat),
	}

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

	existing, err := s.defRepo.GetByID(ctx, &repositories.GetReportDefinitionRequest{
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

	existing.Name = req.Name
	existing.Description = req.Description
	existing.Category = req.Category
	existing.Tags = req.Tags
	existing.Visibility = defaultVisibility(req.Visibility)
	existing.Status = defaultStatus(req.Status)
	existing.Definition = req.Definition
	existing.CatalogVersion = reportcatalog.Version
	existing.DefaultFormat = defaultFormat(req.DefaultFormat)
	existing.Diagnostics = nil
	existing.Version = req.Version

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
	entity, err := s.defRepo.GetByID(ctx, &repositories.GetReportDefinitionRequest{
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
	entities, err := s.defRepo.List(ctx, &repositories.ListReportDefinitionsRequest{
		TenantInfo: req.TenantInfo,
		Statuses:   req.Statuses,
		Limit:      req.Limit,
		Offset:     req.Offset,
	})
	if err != nil {
		return nil, err
	}

	visible := make([]*report.ReportDefinition, 0, len(entities))
	for _, entity := range entities {
		if s.checkDefinitionVisibility(entity, req.TenantInfo) == nil {
			visible = append(visible, entity)
		}
	}

	return visible, nil
}

func (s *Service) DeleteDefinition(ctx context.Context, req *GetDefinitionRequest) error {
	existing, err := s.defRepo.GetByID(ctx, &repositories.GetReportDefinitionRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: req.DefinitionID,
	})
	if err != nil {
		return err
	}
	if existing.OwnerID != req.TenantInfo.UserID {
		return errortypes.NewAuthorizationError("Only the report owner can delete this report")
	}

	return s.defRepo.Delete(ctx, &repositories.DeleteReportDefinitionRequest{
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

	return s.defRepo.ListRevisions(ctx, &repositories.ListReportRevisionsRequest{
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
	revisions, err := s.defRepo.ListRevisions(ctx, &repositories.ListReportRevisionsRequest{
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
	req *repositories.ListReportDefinitionConnectionRequest,
) (*pagination.CursorListResult[*report.ReportDefinition], error) {
	return s.defRepo.ListConnection(ctx, req)
}

func (s *Service) ListRunsConnection(
	ctx context.Context,
	req *repositories.ListReportRunConnectionRequest,
) (*pagination.CursorListResult[*report.ReportRun], error) {
	return s.runRepo.ListConnection(ctx, req)
}
