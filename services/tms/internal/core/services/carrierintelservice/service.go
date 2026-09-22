package carrierintelservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carrierservice"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/restx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type IntegrationConfigReader interface {
	GetRuntimeConfig(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		typ integration.Type,
	) (*integrationservice.RuntimeConfig, error)
}

type Params struct {
	fx.In

	Logger         *zap.Logger
	Config         *config.Config
	DB             ports.DBConnection
	Integrations   *integrationservice.Service
	Connectors     []services.CarrierIntelConnector `group:"carrierIntelConnectors"`
	Limiter        restx.Limiter                    `name:"carrierIntelLimiter"`
	ControlRepo    repositories.CarrierIntelControlRepository
	SnapshotRepo   repositories.CarrierIntelSnapshotRepository
	RawRepo        repositories.CarrierIntelRawPayloadRepository
	EventRepo      repositories.CarrierIntelEventRepository
	EnrollmentRepo repositories.CarrierMonitoringEnrollmentRepository
	FeedRepo       repositories.CarrierIntelFeedStateRepository
	UsageRepo      repositories.CarrierIntelUsageRepository
	OverrideRepo   repositories.CarrierIntelOverrideRepository
	VerifyRepo     repositories.CarrierEquipmentVerificationRepository
	SubjectRepo    repositories.CarrierIntelSubjectRepository
	CarrierRepo    repositories.CarrierRepository
	AssignmentRepo repositories.CarrierAssignmentRepository
	UsStateRepo    repositories.UsStateRepository
	CarrierService *carrierservice.Service
	Notifications  *notificationservice.Service
	AuditService   services.AuditService
	Realtime       services.RealtimeService
	// Watchtower puts a change on a carrier's authority, insurance or
	// safety record on the feed and takes it off once it is resolved.
	Watchtower services.WatchtowerProjector `optional:"true"`
	// Publisher wakes whichever agent covers carrier risk on a finding the
	// tower would show a person.
	Publisher services.AgentEventPublisher `optional:"true"`
}

type Service struct {
	l               *zap.Logger
	db              ports.DBConnection
	integrations    IntegrationConfigReader
	connectors      map[integration.Type]services.CarrierIntelConnector
	limiter         restx.Limiter
	controlRepo     repositories.CarrierIntelControlRepository
	snapshotRepo    repositories.CarrierIntelSnapshotRepository
	rawRepo         repositories.CarrierIntelRawPayloadRepository
	eventRepo       repositories.CarrierIntelEventRepository
	enrollmentRepo  repositories.CarrierMonitoringEnrollmentRepository
	feedRepo        repositories.CarrierIntelFeedStateRepository
	usageRepo       repositories.CarrierIntelUsageRepository
	overrideRepo    repositories.CarrierIntelOverrideRepository
	verifyRepo      repositories.CarrierEquipmentVerificationRepository
	subjectRepo     repositories.CarrierIntelSubjectRepository
	carrierRepo     repositories.CarrierRepository
	assignmentRepo  repositories.CarrierAssignmentRepository
	usStateRepo     repositories.UsStateRepository
	carrierService  CarrierWriter
	notifications   NotificationSender
	auditService    services.AuditService
	realtime        services.RealtimeService
	watchtower      services.WatchtowerProjector
	publisher       services.AgentEventPublisher
	interactiveWait time.Duration
	breaker         *circuitBreaker
	now             func() int64
}

var (
	_ services.CarrierIntelGate         = (*Service)(nil)
	_ services.CarrierLifecycleObserver = (*Service)(nil)
)

func New(p Params) *Service {
	connectors := make(map[integration.Type]services.CarrierIntelConnector, len(p.Connectors))
	for _, connector := range p.Connectors {
		if connector != nil {
			connectors[connector.IntegrationType()] = connector
		}
	}

	svc := &Service{
		l:               p.Logger.Named("service.carrier-intelligence"),
		db:              p.DB,
		integrations:    p.Integrations,
		connectors:      connectors,
		limiter:         p.Limiter,
		controlRepo:     p.ControlRepo,
		snapshotRepo:    p.SnapshotRepo,
		rawRepo:         p.RawRepo,
		eventRepo:       p.EventRepo,
		enrollmentRepo:  p.EnrollmentRepo,
		feedRepo:        p.FeedRepo,
		usageRepo:       p.UsageRepo,
		overrideRepo:    p.OverrideRepo,
		verifyRepo:      p.VerifyRepo,
		subjectRepo:     p.SubjectRepo,
		carrierRepo:     p.CarrierRepo,
		assignmentRepo:  p.AssignmentRepo,
		usStateRepo:     p.UsStateRepo,
		carrierService:  p.CarrierService,
		notifications:   p.Notifications,
		auditService:    p.AuditService,
		realtime:        p.Realtime,
		watchtower:      p.Watchtower,
		publisher:       p.Publisher,
		interactiveWait: p.Config.CarrierIntelligence.GetInteractiveWait(),
		breaker:         newCircuitBreaker(),
		now:             nowUnix,
	}

	if p.CarrierService != nil {
		p.CarrierService.SetLifecycleObserver(svc)
	}

	return svc
}

func (s *Service) Connector(typ integration.Type) (services.CarrierIntelConnector, bool) {
	connector, ok := s.connectors[typ]
	return connector, ok
}

func (s *Service) Control(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*carrierintel.CarrierIntelControl, error) {
	return s.controlRepo.GetOrCreate(ctx, tenantInfo)
}
