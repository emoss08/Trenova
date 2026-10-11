package rating

import (
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulatemplaterepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulatemplatereviewrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulatemplatetestcaserepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulatemplateversionrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/fuelsurchargerepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/rateagreementrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/rateconfirmationrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/rateimportrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/ratematrixrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/ratequoterepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/ratesimulationrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/ratezonerepository"
	"github.com/emoss08/trenova/internal/modules/rating/app/carrierassignmentservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/fuelsurchargeservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateagreementservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateconfirmationservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateengine"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateimportservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratematrixservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratequoteservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratesimulationservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratezoneservice"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateconfirmationports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratesimulationports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			formulatemplaterepository.New,
			formulatemplatetestcaserepository.New,
			formulatemplatereviewrepository.New,
			formulatemplateversionrepository.New,
			rateconfirmationrepository.New,
			rateagreementrepository.New,
			ratezonerepository.New,
			ratematrixrepository.New,
			ratesimulationrepository.New,
			rateimportrepository.New,
			ratematrixrepository.NewDensityScaleRepository,
			ratequoterepository.New,
			fuelsurchargerepository.NewIndexRepository,
			fuelsurchargerepository.NewPriceRepository,
			fuelsurchargerepository.NewProgramRepository,
			rateagreementservice.NewValidator,
			ratezoneservice.NewValidator,
			ratematrixservice.NewValidator,
			carrierassignmentservice.New,
			func(s *carrierassignmentservice.Service) tenderports.CarrierMoveAssigner { return s },
			rateconfirmationservice.New,
			func(s *rateconfirmationservice.Service) rateconfirmationports.RateConfirmationIssuer { return s },
			rateagreementservice.New,
			ratezoneservice.New,
			ratematrixservice.New,
			ratequoteservice.New,
			ratesimulationservice.New,
			rateimportservice.New,
			func(s *ratesimulationservice.Service) ratesimulationports.RateSimulationRunner { return s },
			fuelsurchargeservice.New,
			func(s *fuelsurchargeservice.Service) fuelsurchargeports.FuelSurchargeResolver { return s },
		),
		formula.Module,
		formulatemplateservice.Module,
		rateengine.Module,
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
