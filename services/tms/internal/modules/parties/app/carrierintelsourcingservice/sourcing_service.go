package carrierintelsourcingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelenrollmentservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierservice"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/pagination"
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
	SubjectRepo    carrierintelports.CarrierIntelSubjectRepository
	UsStateRepo    usstateports.UsStateRepository
	CarrierService *carrierservice.Service
	Plans          platformplanports.PlanService `optional:"true"`
}

type Service struct {
	l              *zap.Logger
	intel          *carrierintelservice.Service
	enrollment     *carrierintelenrollmentservice.Service
	controlRepo    carrierintelports.CarrierIntelControlRepository
	subjectRepo    carrierintelports.CarrierIntelSubjectRepository
	usStateRepo    usstateports.UsStateRepository
	carrierService carrierintelservice.CarrierWriter
	plans          platformplanports.PlanService
	now            func() int64
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		l:              p.Logger.Named("service.carrier-intelligence"),
		intel:          p.Intel,
		enrollment:     p.Enrollment,
		controlRepo:    p.ControlRepo,
		subjectRepo:    p.SubjectRepo,
		usStateRepo:    p.UsStateRepo,
		carrierService: p.CarrierService,
		plans:          p.Plans,
		now:            timeutils.NowUnix,
	}
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
