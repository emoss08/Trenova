package carrierintelsourcingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelenrollmentservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierservice"
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
	SubjectRepo    carrierintelports.CarrierIntelSubjectRepository
	UsStateRepo    usstateports.UsStateRepository
	CarrierService *carrierservice.Service
	Features       platformcatalogports.FeatureGate
}

type Service struct {
	l              *zap.Logger
	intel          *carrierintelservice.Service
	enrollment     *carrierintelenrollmentservice.Service
	controlRepo    carrierintelports.CarrierIntelControlRepository
	subjectRepo    carrierintelports.CarrierIntelSubjectRepository
	usStateRepo    usstateports.UsStateRepository
	carrierService carrierintelservice.CarrierWriter
	features       platformcatalogports.FeatureGate
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
		features:       p.Features,
		now:            timeutils.NowUnix,
	}
}

func (s *Service) requirePaidLookups(ctx context.Context) error {
	return platformcatalogports.RequireFeature(
		ctx,
		s.features,
		platformcatalog.FeatureCarrierBrokerage,
	)
}
