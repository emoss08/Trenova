package iftaservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/services/referencedataguard"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	realtimeReturn       = "ifta_return"
	realtimeMileageEntry = "ifta_jurisdiction_mileage_entry"
	realtimeTaxRate      = "ifta_tax_rate"
)

type FuelAccumulator interface {
	AccumulateFuel(
		ctx context.Context,
		req *fuelpurchaseports.AccumulateFuelRequest,
	) ([]*fuelpurchaseports.FuelAccumulationRow, error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          iftaports.IFTARepository
	Fuel          fuelpurchaseports.FuelPurchaseRepository
	TractorRepo   tractorports.TractorRepository
	Organizations tenantports.OrganizationRepository
	AuditService  auditports.AuditService
	ReferenceData *referencedataguard.Guard
	Realtime      agentports.RealtimeService `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	repo          iftaports.IFTARepository
	fuel          FuelAccumulator
	tractorRepo   tractorports.TractorRepository
	organizations tenantports.OrganizationRepository
	auditService  auditports.AuditService
	referenceData *referencedataguard.Guard
	realtime      agentports.RealtimeService
	now           func() int64
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.ifta"),
		repo:          p.Repo,
		fuel:          p.Fuel,
		tractorRepo:   p.TractorRepo,
		organizations: p.Organizations,
		auditService:  p.AuditService,
		referenceData: p.ReferenceData,
		realtime:      p.Realtime,
		now:           timeutils.NowUnix,
	}
}

type Deps struct {
	Logger        *zap.Logger
	Repo          iftaports.IFTARepository
	Fuel          FuelAccumulator
	TractorRepo   tractorports.TractorRepository
	Organizations tenantports.OrganizationRepository
	AuditService  auditports.AuditService
	ReferenceData *referencedataguard.Guard
	Realtime      agentports.RealtimeService
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
		l:             logger.Named("service.ifta"),
		repo:          d.Repo,
		fuel:          d.Fuel,
		tractorRepo:   d.TractorRepo,
		organizations: d.Organizations,
		auditService:  d.AuditService,
		referenceData: d.ReferenceData,
		realtime:      d.Realtime,
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
	params := &auditports.LogActionParams{
		Resource:       p.resource,
		ResourceID:     p.resourceID,
		Operation:      p.operation,
		UserID:         p.userID,
		CurrentState:   jsonutils.MustToJSON(p.current),
		OrganizationID: p.tenant.OrgID,
		BusinessUnitID: p.tenant.BuID,
	}
	opts := []auditports.LogOption{auditservice.WithComment(p.comment)}
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
		ActorType:      permissionports.PrincipalTypeUser,
		ActorID:        userID,
		Resource:       resource,
		Action:         string(operation),
		RecordID:       recordID,
	}); err != nil {
		s.l.Warn("failed to publish ifta invalidation",
			zap.String("resource", resource),
			zap.Error(err))
	}
}

func (s *Service) tenantLocation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*time.Location, error) {
	return orgzone.Location(ctx, s.organizations, tenantInfo)
}

func versionMismatch(entity string) error {
	return errortypes.NewValidationError(
		"version",
		errortypes.ErrVersionMismatch,
		"This {0} was changed by someone else; reload it and try again", entity,
	)
}

func invalidOperation(field, message string) error {
	return errortypes.NewValidationError(field, errortypes.ErrInvalidOperation, message)
}
