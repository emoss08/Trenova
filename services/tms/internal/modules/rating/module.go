package rating

import (
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulalibraryrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulalibraryreviewrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulalibraryversionrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulatemplaterepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulatemplatereviewrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulatemplateversionrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/fuelsurchargerepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/rateagreementrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/rateconfirmationrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/rateimportrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/ratematrixrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/ratematrixreviewrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/ratematrixversionrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/ratequoterepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/ratesimulationrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/ratezonerepository"
	"github.com/emoss08/trenova/internal/modules/rating/app/carrierassignmentservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulalibraryservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateimportservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatereviewservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatestudioservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/fuelsurchargeservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateagreementservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateconfirmationservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateengine"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateimportservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratematrixservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratequoteservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratesimulationservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratezoneservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratinghydration"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/app/tarifftables"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateconfirmationports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratesimulationports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			formulatemplaterepository.New,
			formulatemplatereviewrepository.New,
			formulatemplateversionrepository.New,
			formulalibraryrepository.New,
			formulalibraryversionrepository.New,
			formulalibraryreviewrepository.New,
			rateagreementrepository.NewTemplateInputsLister,
			rateconfirmationrepository.New,
			rateagreementrepository.New,
			ratezonerepository.New,
			ratematrixrepository.New,
			ratematrixversionrepository.New,
			ratematrixreviewrepository.New,
			ratesimulationrepository.New,
			rateimportrepository.New,
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
			ratinghydration.New,
			func(h *ratinghydration.Hydrator) ratequoteports.ShipmentRatingHydrator { return h },
			tarifftables.NewTables,
			tariff.NewCodes,
			tariff.NewSettings,
			tariff.NewContexts,
			func(c *tariff.Contexts) tariffports.ContextSource { return c },
			tariff.NewRater,
			func(r *tariff.Rater) tariffports.Rater { return r },
			tariff.NewRecordSource,
			func(r *tariff.RecordSource) tariffports.RecordBuilder { return r },
		),
		tariff.AuthoringModule,
		formulatemplateservice.Module,
		fx.Provide(
			formulatemplatereviewservice.New,
			formulatemplatestudioservice.New,
			formulatemplateimportservice.New,
		),
		formulalibraryservice.Module,
		rateengine.Module,
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
