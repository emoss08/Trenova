package equipment

import (
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmentcontinuityrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmentmanufacturerrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmenttyperepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/fleetcoderepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/recordlabelrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/tractorrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/trailerrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/app/equipmentmanufacturerservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/equipmenttypeservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/fleetcodeservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/trailerservice"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			equipmentmanufacturerrepository.New,
			equipmenttyperepository.New,
			equipmentcontinuityrepository.New,
			fleetcoderepository.New,
			tractorrepository.New,
			trailerrepository.New,
			recordlabelrepository.New,
			equipmentmanufacturerservice.NewValidator,
			equipmenttypeservice.NewValidator,
			fleetcodeservice.NewValidator,
			tractorservice.NewValidator,
			trailerservice.NewValidator,
			equipmentmanufacturerservice.New,
			equipmenttypeservice.New,
			fleetcodeservice.New,
			tractorservice.New,
			trailerservice.New,
		),
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
