package fuelpurchaseservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	ImportDocumentResourceType   = "fuel_purchase_import"
	PurchaseDocumentResourceType = "fuel_purchase"

	realtimeCard     = "fuel_card"
	realtimePurchase = "fuel_purchase"
	realtimeImport   = "fuel_purchase_import"
)

type JurisdictionReader interface {
	GetJurisdictionByID(ctx context.Context, id pulid.ID) (*ifta.Jurisdiction, error)
	GetJurisdictionByCode(
		ctx context.Context,
		countryCode, code string,
	) (*ifta.Jurisdiction, error)
	FindJurisdictionsByCodes(
		ctx context.Context,
		keys []string,
	) (map[string]*ifta.Jurisdiction, error)
}

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.FuelPurchaseRepository
	IFTARepo     repositories.IFTARepository
	TractorRepo  repositories.TractorRepository
	WorkerRepo   repositories.WorkerRepository
	Documents    services.InvoiceDocumentService
	AuditService services.AuditService
	Realtime     services.RealtimeService `optional:"true"`
	Validator    *Validator
}

type Service struct {
	l             *zap.Logger
	repo          repositories.FuelPurchaseRepository
	jurisdictions JurisdictionReader
	tractorRepo   repositories.TractorRepository
	workerRepo    repositories.WorkerRepository
	documents     services.InvoiceDocumentService
	auditService  services.AuditService
	realtime      services.RealtimeService
	validator     *Validator
	now           func() int64
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.fuel-purchase"),
		repo:          p.Repo,
		jurisdictions: p.IFTARepo,
		tractorRepo:   p.TractorRepo,
		workerRepo:    p.WorkerRepo,
		documents:     p.Documents,
		auditService:  p.AuditService,
		realtime:      p.Realtime,
		validator:     p.Validator,
		now:           timeutils.NowUnix,
	}
}

type Deps struct {
	Logger        *zap.Logger
	Repo          repositories.FuelPurchaseRepository
	Jurisdictions JurisdictionReader
	TractorRepo   repositories.TractorRepository
	WorkerRepo    repositories.WorkerRepository
	Documents     services.InvoiceDocumentService
	AuditService  services.AuditService
	Realtime      services.RealtimeService
	Validator     *Validator
	Now           func() int64
}

func NewWithDeps(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	now := d.Now
	if now == nil {
		now = timeutils.NowUnix
	}
	return &Service{
		l:             logger.Named("service.fuel-purchase"),
		repo:          d.Repo,
		jurisdictions: d.Jurisdictions,
		tractorRepo:   d.TractorRepo,
		workerRepo:    d.WorkerRepo,
		documents:     d.Documents,
		auditService:  d.AuditService,
		realtime:      d.Realtime,
		validator:     d.Validator,
		now:           now,
	}
}

type auditParams struct {
	resource   permission.Resource
	resourceID string
	operation  permission.Operation
	userID     pulid.ID
	tenant     pagination.TenantInfo
	current    any
	previous   any
	comment    string
}

func (s *Service) audit(p *auditParams) {
	if s.auditService == nil || p.userID.IsNil() {
		return
	}
	params := &services.LogActionParams{
		Resource:       p.resource,
		ResourceID:     p.resourceID,
		Operation:      p.operation,
		UserID:         p.userID,
		CurrentState:   jsonutils.MustToJSON(p.current),
		OrganizationID: p.tenant.OrgID,
		BusinessUnitID: p.tenant.BuID,
	}
	opts := []services.LogOption{auditservice.WithComment(p.comment)}
	if p.previous != nil {
		params.PreviousState = jsonutils.MustToJSON(p.previous)
		opts = append(opts, auditservice.WithDiff(p.previous, p.current))
	}
	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}
}

func (s *Service) publish(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource string,
	operation permission.Operation,
	recordID pulid.ID,
	userID pulid.ID,
) {
	if s.realtime == nil {
		return
	}
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    userID,
		ActorType:      services.PrincipalTypeUser,
		ActorID:        userID,
		Resource:       resource,
		Action:         string(operation),
		RecordID:       recordID,
	}); err != nil {
		s.l.Warn("failed to publish fuel purchase invalidation",
			zap.String("resource", resource),
			zap.Error(err))
	}
}
