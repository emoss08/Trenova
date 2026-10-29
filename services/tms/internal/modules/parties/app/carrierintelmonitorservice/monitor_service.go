package carrierintelmonitorservice

import (
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelenrollmentservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	Intel          *carrierintelservice.Service
	Enrollment     *carrierintelenrollmentservice.Service
	ControlRepo    carrierintelports.CarrierIntelControlRepository
	SnapshotRepo   carrierintelports.CarrierIntelSnapshotRepository
	RawRepo        carrierintelports.CarrierIntelRawPayloadRepository
	EventRepo      carrierintelports.CarrierIntelEventRepository
	EnrollmentRepo carrierintelports.CarrierMonitoringEnrollmentRepository
	FeedRepo       carrierintelports.CarrierIntelFeedStateRepository
	UsageRepo      carrierintelports.CarrierIntelUsageRepository
	SubjectRepo    carrierintelports.CarrierIntelSubjectRepository
}

type Service struct {
	l              *zap.Logger
	intel          *carrierintelservice.Service
	enrollment     *carrierintelenrollmentservice.Service
	controlRepo    carrierintelports.CarrierIntelControlRepository
	snapshotRepo   carrierintelports.CarrierIntelSnapshotRepository
	rawRepo        carrierintelports.CarrierIntelRawPayloadRepository
	eventRepo      carrierintelports.CarrierIntelEventRepository
	enrollmentRepo carrierintelports.CarrierMonitoringEnrollmentRepository
	feedRepo       carrierintelports.CarrierIntelFeedStateRepository
	usageRepo      carrierintelports.CarrierIntelUsageRepository
	subjectRepo    carrierintelports.CarrierIntelSubjectRepository
	now            func() int64
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		l:              p.Logger.Named("service.carrier-intelligence"),
		intel:          p.Intel,
		enrollment:     p.Enrollment,
		controlRepo:    p.ControlRepo,
		snapshotRepo:   p.SnapshotRepo,
		rawRepo:        p.RawRepo,
		eventRepo:      p.EventRepo,
		enrollmentRepo: p.EnrollmentRepo,
		feedRepo:       p.FeedRepo,
		usageRepo:      p.UsageRepo,
		subjectRepo:    p.SubjectRepo,
		now:            timeutils.NowUnix,
	}
}
