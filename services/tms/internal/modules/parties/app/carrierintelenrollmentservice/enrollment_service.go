package carrierintelenrollmentservice

import (
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierservice"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	Intel          *carrierintelservice.Service
	Integrations   *integrationservice.Service
	ControlRepo    carrierintelports.CarrierIntelControlRepository
	SnapshotRepo   carrierintelports.CarrierIntelSnapshotRepository
	EnrollmentRepo carrierintelports.CarrierMonitoringEnrollmentRepository
	FeedRepo       carrierintelports.CarrierIntelFeedStateRepository
	SubjectRepo    carrierintelports.CarrierIntelSubjectRepository
	CarrierService *carrierservice.Service
	AuditService   auditports.AuditService
}

type Service struct {
	l              *zap.Logger
	intel          *carrierintelservice.Service
	integrations   carrierintelservice.IntegrationConfigReader
	controlRepo    carrierintelports.CarrierIntelControlRepository
	snapshotRepo   carrierintelports.CarrierIntelSnapshotRepository
	enrollmentRepo carrierintelports.CarrierMonitoringEnrollmentRepository
	feedRepo       carrierintelports.CarrierIntelFeedStateRepository
	subjectRepo    carrierintelports.CarrierIntelSubjectRepository
	auditService   auditports.AuditService
	now            func() int64
}

var _ carrierintelports.CarrierLifecycleObserver = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	svc := &Service{
		l:              p.Logger.Named("service.carrier-intelligence"),
		intel:          p.Intel,
		integrations:   p.Integrations,
		controlRepo:    p.ControlRepo,
		snapshotRepo:   p.SnapshotRepo,
		enrollmentRepo: p.EnrollmentRepo,
		feedRepo:       p.FeedRepo,
		subjectRepo:    p.SubjectRepo,
		auditService:   p.AuditService,
		now:            timeutils.NowUnix,
	}

	if p.CarrierService != nil {
		p.CarrierService.SetLifecycleObserver(svc)
	}

	return svc
}
