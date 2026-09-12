package iftaservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
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
		req *repositories.AccumulateFuelRequest,
	) ([]*repositories.FuelAccumulationRow, error)
}

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.IFTARepository
	Fuel         repositories.FuelPurchaseRepository
	TractorRepo  repositories.TractorRepository
	OrgCacheRepo repositories.OrganizationCacheRepository
	AuditService services.AuditService
	Realtime     services.RealtimeService `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         repositories.IFTARepository
	fuel         FuelAccumulator
	tractorRepo  repositories.TractorRepository
	orgCacheRepo repositories.OrganizationCacheRepository
	auditService services.AuditService
	realtime     services.RealtimeService
	now          func() int64
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.ifta"),
		repo:         p.Repo,
		fuel:         p.Fuel,
		tractorRepo:  p.TractorRepo,
		orgCacheRepo: p.OrgCacheRepo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
		now:          timeutils.NowUnix,
	}
}

type Deps struct {
	Logger       *zap.Logger
	Repo         repositories.IFTARepository
	Fuel         FuelAccumulator
	TractorRepo  repositories.TractorRepository
	OrgCacheRepo repositories.OrganizationCacheRepository
	AuditService services.AuditService
	Realtime     services.RealtimeService
	Now          func() int64
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
		l:            logger.Named("service.ifta"),
		repo:         d.Repo,
		fuel:         d.Fuel,
		tractorRepo:  d.TractorRepo,
		orgCacheRepo: d.OrgCacheRepo,
		auditService: d.AuditService,
		realtime:     d.Realtime,
		now:          now,
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
		s.l.Warn("failed to publish ifta invalidation",
			zap.String("resource", resource),
			zap.Error(err))
	}
}

func (s *Service) tenantLocation(ctx context.Context, orgID pulid.ID) *time.Location {
	if s.orgCacheRepo == nil {
		return time.UTC
	}
	org, err := s.orgCacheRepo.GetByID(ctx, orgID)
	if err != nil || org == nil {
		return time.UTC
	}

	loc, err := time.LoadLocation(timeutils.NormalizeTimezone(org.Timezone))
	if err != nil {
		return time.UTC
	}

	return loc
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
