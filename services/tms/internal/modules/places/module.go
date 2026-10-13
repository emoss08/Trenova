package places

import (
	"github.com/emoss08/trenova/internal/modules/places/adapters/postgres/distancecalculationrepository"
	"github.com/emoss08/trenova/internal/modules/places/adapters/postgres/distancecontrolrepository"
	"github.com/emoss08/trenova/internal/modules/places/adapters/postgres/distanceoverriderepository"
	"github.com/emoss08/trenova/internal/modules/places/adapters/postgres/distanceprofilerepository"
	"github.com/emoss08/trenova/internal/modules/places/adapters/postgres/locationcategoryrepository"
	"github.com/emoss08/trenova/internal/modules/places/adapters/postgres/locationrepository"
	"github.com/emoss08/trenova/internal/modules/places/adapters/postgres/storedmileagerepository"
	"github.com/emoss08/trenova/internal/modules/places/app/distancecalculationservice"
	"github.com/emoss08/trenova/internal/modules/places/app/routeestimateservice"
	"github.com/emoss08/trenova/internal/modules/places/app/distancecontrolservice"
	"github.com/emoss08/trenova/internal/modules/places/app/distanceoverrideservice"
	"github.com/emoss08/trenova/internal/modules/places/app/distanceprofileservice"
	"github.com/emoss08/trenova/internal/modules/places/app/jurisdictionruleservice"
	"github.com/emoss08/trenova/internal/modules/places/app/locationcategoryservice"
	"github.com/emoss08/trenova/internal/modules/places/app/locationservice"
	"github.com/emoss08/trenova/internal/modules/places/app/storedmileageservice"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			locationcategoryrepository.New,
			locationrepository.New,
			distancecalculationrepository.New,
			distancecontrolrepository.New,
			distanceoverriderepository.New,
			distanceprofilerepository.New,
			storedmileagerepository.New,
			locationcategoryservice.NewValidator,
			locationservice.NewValidator,
			distanceoverrideservice.NewValidator,
			distanceprofileservice.NewValidator,
			locationcategoryservice.New,
			locationservice.New,
			jurisdictionruleservice.NewService,
			distancecalculationservice.New,
			routeestimateservice.New,
			distancecontrolservice.New,
			distanceoverrideservice.New,
			distanceprofileservice.New,
			storedmileageservice.New,
		),
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
