package telematicsservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/services/customfieldservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
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

	Repo                telematicsports.TelematicsRepository
	ProviderFactory     telematicsports.TelematicsProviderFactory
	IntegrationService  *integrationservice.Service
	EncryptionService   *encryptionservice.Service
	RealtimeService     agentports.RealtimeService
	Notifications       *notificationservice.Service
	DriverNotify        *drivernotificationservice.Service `optional:"true"`
	AssignmentRepo      shipmentports.AssignmentRepository
	ShipmentRepo        shipmentports.ShipmentRepository
	ShipmentMoveRepo    shipmentports.ShipmentMoveRepository
	ShipmentMoveService shipmentports.ShipmentMoveService
	DispatchControlRepo dispatchcontrolports.DispatchControlRepository
	CustomFieldValues   *customfieldservice.ValuesService
	Logger              *zap.Logger
	ProviderOverride    telematicsports.TelematicsProvider  `optional:"true"`
	Watchtower          watchtowerports.WatchtowerProjector `optional:"true"`
	Waits               agentwaitports.AgentWaitNotifier    `optional:"true"`
}

type Service struct {
	repo                telematicsports.TelematicsRepository
	providerFactory     telematicsports.TelematicsProviderFactory
	integrationService  *integrationservice.Service
	encryptionService   *encryptionservice.Service
	realtimeService     agentports.RealtimeService
	notifications       *notificationservice.Service
	driverNotify        *drivernotificationservice.Service
	assignmentRepo      shipmentports.AssignmentRepository
	shipmentRepo        shipmentports.ShipmentRepository
	shipmentMoveRepo    shipmentports.ShipmentMoveRepository
	shipmentMoveService shipmentports.ShipmentMoveService
	dispatchControlRepo dispatchcontrolports.DispatchControlRepository
	customFieldValues   *customfieldservice.ValuesService
	providerOverride    telematicsports.TelematicsProvider
	watchtower          watchtowerports.WatchtowerProjector
	waits               agentwaitports.AgentWaitNotifier
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
		driverNotify:        p.DriverNotify,
		assignmentRepo:      p.AssignmentRepo,
		shipmentRepo:        p.ShipmentRepo,
		shipmentMoveRepo:    p.ShipmentMoveRepo,
		shipmentMoveService: p.ShipmentMoveService,
		dispatchControlRepo: p.DispatchControlRepo,
		customFieldValues:   p.CustomFieldValues,
		providerOverride:    p.ProviderOverride,
		watchtower:          p.Watchtower,
		waits:               p.Waits,
		l:                   p.Logger.Named("telematics-service"),
	}
}

func (s *Service) resolveProvider(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (telematicsports.TelematicsProvider, error) {
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
		&agentports.PublishResourceInvalidationRequest{
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
