package parties

import (
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/carriercapacityrepository"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/carrierintelrepository"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/carrierrepository"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/customerrepository"
	"github.com/emoss08/trenova/internal/modules/parties/app/carriercapacityservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/customerservice"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			carriercapacityrepository.New,
			carrierrepository.New,
			carrierintelrepository.NewControlRepository,
			carrierintelrepository.NewSnapshotRepository,
			carrierintelrepository.NewRawPayloadRepository,
			carrierintelrepository.NewEventRepository,
			carrierintelrepository.NewEnrollmentRepository,
			carrierintelrepository.NewFeedStateRepository,
			carrierintelrepository.NewUsageRepository,
			carrierintelrepository.NewOverrideRepository,
			carrierintelrepository.NewVerificationRepository,
			carrierintelrepository.NewSubjectRepository,
			customerrepository.New,
			carrierservice.NewValidator,
			customerservice.NewValidator,
			carriercapacityservice.NewValidator,
			carrierservice.New,
			carrierintelservice.New,
			func(s *carrierintelservice.Service) carrierintelports.CarrierIntelGate { return s },
			func(s *carrierintelservice.Service) carrierintelports.CarrierLifecycleObserver { return s },
			customerservice.New,
			carriercapacityservice.New,
		),
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
