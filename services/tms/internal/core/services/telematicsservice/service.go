package telematicsservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/customfieldservice"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultPositionMaxAgeSeconds = int64(3600)
	violationLookbackSeconds     = int64(48 * 3600)
	dvirLookbackSeconds          = int64(48 * 3600)
	dvirSafetyStatusResolved     = "resolved"
	webhookMaxSkew               = 5 * time.Minute
)

type Params struct {
	fx.In

	Repo                repositories.TelematicsRepository
	ProviderFactory     services.TelematicsProviderFactory
	IntegrationService  *integrationservice.Service
	EncryptionService   *encryptionservice.Service
	RealtimeService     services.RealtimeService
	Notifications       *notificationservice.Service
	AssignmentRepo      repositories.AssignmentRepository
	ShipmentRepo        repositories.ShipmentRepository
	ShipmentMoveRepo    repositories.ShipmentMoveRepository
	ShipmentMoveService services.ShipmentMoveService
	DispatchControlRepo repositories.DispatchControlRepository
	CustomFieldValues   *customfieldservice.ValuesService
	Logger              *zap.Logger
	ProviderOverride    services.TelematicsProvider `optional:"true"`
}

type Service struct {
	repo                repositories.TelematicsRepository
	providerFactory     services.TelematicsProviderFactory
	integrationService  *integrationservice.Service
	encryptionService   *encryptionservice.Service
	realtimeService     services.RealtimeService
	notifications       *notificationservice.Service
	assignmentRepo      repositories.AssignmentRepository
	shipmentRepo        repositories.ShipmentRepository
	shipmentMoveRepo    repositories.ShipmentMoveRepository
	shipmentMoveService services.ShipmentMoveService
	dispatchControlRepo repositories.DispatchControlRepository
	customFieldValues   *customfieldservice.ValuesService
	providerOverride    services.TelematicsProvider
	l                   *zap.Logger
}

func New(p Params) *Service { //nolint:gocritic // dependency injection
	return &Service{
		repo:                p.Repo,
		providerFactory:     p.ProviderFactory,
		integrationService:  p.IntegrationService,
		encryptionService:   p.EncryptionService,
		realtimeService:     p.RealtimeService,
		notifications:       p.Notifications,
		assignmentRepo:      p.AssignmentRepo,
		shipmentRepo:        p.ShipmentRepo,
		shipmentMoveRepo:    p.ShipmentMoveRepo,
		shipmentMoveService: p.ShipmentMoveService,
		dispatchControlRepo: p.DispatchControlRepo,
		customFieldValues:   p.CustomFieldValues,
		providerOverride:    p.ProviderOverride,
		l:                   p.Logger.Named("telematics-service"),
	}
}

func (s *Service) resolveProvider(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (services.TelematicsProvider, error) {
	if s.providerOverride != nil {
		return s.providerOverride, nil
	}
	return s.providerFactory.ProviderFor(ctx, tenantInfo)
}

func (s *Service) publishInvalidation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource string,
) {
	if s.realtimeService == nil {
		return
	}
	err := s.realtimeService.PublishResourceInvalidation(
		ctx,
		&services.PublishResourceInvalidationRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Resource:       resource,
			Action:         "updated",
		},
	)
	if err != nil {
		s.l.Warn("failed to publish telematics invalidation",
			zap.String("resource", resource),
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Error(err))
	}
}
