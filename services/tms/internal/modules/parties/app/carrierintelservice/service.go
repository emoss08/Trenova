package carrierintelservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/emoss08/trenova/shared/timeutils"
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
	DB             kernelports.DBConnection
	Integrations   *integrationservice.Service
	Connectors     []carrierintelports.CarrierIntelConnector `group:"carrierIntelConnectors"`
	Limiter        restx.Limiter                             `                               name:"carrierIntelLimiter"`
	ControlRepo    carrierintelports.CarrierIntelControlRepository
	SnapshotRepo   carrierintelports.CarrierIntelSnapshotRepository
	RawRepo        carrierintelports.CarrierIntelRawPayloadRepository
	EventRepo      carrierintelports.CarrierIntelEventRepository
	EnrollmentRepo carrierintelports.CarrierMonitoringEnrollmentRepository
	FeedRepo       carrierintelports.CarrierIntelFeedStateRepository
	UsageRepo      carrierintelports.CarrierIntelUsageRepository
	OverrideRepo   carrierintelports.CarrierIntelOverrideRepository
	VerifyRepo     carrierintelports.CarrierEquipmentVerificationRepository
	SubjectRepo    carrierintelports.CarrierIntelSubjectRepository
	CarrierRepo    carrierports.CarrierRepository
	AssignmentRepo shipmentports.CarrierAssignmentRepository
	UsStateRepo    usstateports.UsStateRepository
	CarrierService *carrierservice.Service
	Notifications  *notificationservice.Service
	AuditService   auditports.AuditService
	Realtime       agentports.RealtimeService
	// Watchtower puts a change on a carrier's authority, insurance or
	// safety record on the feed and takes it off once it is resolved.
	Watchtower watchtowerports.WatchtowerProjector `optional:"true"`
	// Publisher wakes whichever agent covers carrier risk on a finding the
	// tower would show a person.
	Publisher agentports.AgentEventPublisher `optional:"true"`
	Plans     platformplanports.PlanService  `optional:"true"`
}

type Service struct {
	l               *zap.Logger
	db              kernelports.DBConnection
	integrations    IntegrationConfigReader
	connectors      map[integration.Type]carrierintelports.CarrierIntelConnector
	limiter         restx.Limiter
	controlRepo     carrierintelports.CarrierIntelControlRepository
	snapshotRepo    carrierintelports.CarrierIntelSnapshotRepository
	rawRepo         carrierintelports.CarrierIntelRawPayloadRepository
	eventRepo       carrierintelports.CarrierIntelEventRepository
	enrollmentRepo  carrierintelports.CarrierMonitoringEnrollmentRepository
	feedRepo        carrierintelports.CarrierIntelFeedStateRepository
	usageRepo       carrierintelports.CarrierIntelUsageRepository
	overrideRepo    carrierintelports.CarrierIntelOverrideRepository
	verifyRepo      carrierintelports.CarrierEquipmentVerificationRepository
	subjectRepo     carrierintelports.CarrierIntelSubjectRepository
	carrierRepo     carrierports.CarrierRepository
	assignmentRepo  shipmentports.CarrierAssignmentRepository
	usStateRepo     usstateports.UsStateRepository
	carrierService  CarrierWriter
	notifications   NotificationSender
	auditService    auditports.AuditService
	realtime        agentports.RealtimeService
	watchtower      watchtowerports.WatchtowerProjector
	publisher       agentports.AgentEventPublisher
	interactiveWait time.Duration
	breaker         *circuitBreaker
	plans           platformplanports.PlanService
	now             func() int64
}

var (
	_ carrierintelports.CarrierIntelGate         = (*Service)(nil)
	_ carrierintelports.CarrierLifecycleObserver = (*Service)(nil)
)

func New(p Params) *Service {
	connectors := make(
		map[integration.Type]carrierintelports.CarrierIntelConnector,
		len(p.Connectors),
	)
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
		plans:           p.Plans,
		now:             timeutils.NowUnix,
	}

	if p.CarrierService != nil {
		p.CarrierService.SetLifecycleObserver(svc)
	}

	return svc
}

func (s *Service) requirePaidLookups(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	return planservice.RequireCapability(
		ctx,
		s.plans,
		tenantInfo,
		platformplan.CapabilityCarrierIntelligencePaid,
	)
}

func (s *Service) Connector(typ integration.Type) (carrierintelports.CarrierIntelConnector, bool) {
	connector, ok := s.connectors[typ]
	return connector, ok
}

func (s *Service) Control(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*carrierintel.CarrierIntelControl, error) {
	return s.controlRepo.GetOrCreate(ctx, tenantInfo)
}
