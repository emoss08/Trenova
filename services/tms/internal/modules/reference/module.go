package reference

import (
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/accessorialchargerepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/commodityrepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/documenttyperepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/dothazmatreferencerepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/exchangeraterepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/hazardousmaterialrepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/hazmatsegregationrulerepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/holdreasonrepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/modeprofilerepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/servicetyperepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/shipmenttyperepository"
	"github.com/emoss08/trenova/internal/modules/reference/app/accessorialchargeservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/commodityservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/datatransformer"
	"github.com/emoss08/trenova/internal/modules/reference/app/documenttypeservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/dothazmatreferenceservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/exchangerateservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/hazardousmaterialservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/hazmatsegregationruleservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/holdreasonservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/modeprofileservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/servicetypeservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/shipmenttypeservice"
	"github.com/emoss08/trenova/internal/modules/reference/ports/exchangerateports"
	"go.uber.org/fx"
)

//nolint:funlen // the module's providers are its part of the application graph
func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			accessorialchargerepository.New,
			servicetyperepository.New,
			shipmenttyperepository.New,
			hazardousmaterialrepository.New,
			hazmatsegregationrulerepository.New,
			dothazmatreferencerepository.New,
			commodityrepository.New,
			documenttyperepository.New,
			holdreasonrepository.New,
			exchangeraterepository.New,
			modeprofilerepository.NewProfileRepository,
			modeprofilerepository.NewDeviationRepository,
			accessorialchargeservice.NewValidator,
			servicetypeservice.NewValidator,
			shipmenttypeservice.NewValidator,
			hazardousmaterialservice.NewValidator,
			hazmatsegregationruleservice.NewValidator,
			commodityservice.NewValidator,
			documenttypeservice.NewValidator,
			holdreasonservice.NewValidator,
			datatransformer.New,
			accessorialchargeservice.New,
			servicetypeservice.New,
			shipmenttypeservice.New,
			hazardousmaterialservice.New,
			hazmatsegregationruleservice.New,
			dothazmatreferenceservice.New,
			commodityservice.New,
			documenttypeservice.New,
			holdreasonservice.New,
			modeprofileservice.NewService,
			exchangerateservice.New,
			func(s *exchangerateservice.Service) exchangerateports.ExchangeRateService { return s },
		),
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
