package tariff

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/places/ports/distancecontrolports"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/record"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"go.uber.org/fx"
)

const storedKilometers = "Kilometers"

type SettingsParams struct {
	fx.In

	DistanceControls distancecontrolports.DistanceControlRepository
}

type Settings struct {
	distanceControls distancecontrolports.DistanceControlRepository
}

func NewSettings(p SettingsParams) *Settings {
	return &Settings{distanceControls: p.DistanceControls}
}

func (s *Settings) For(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (rules.TariffSettings, error) {
	control, err := s.distanceControls.Get(ctx, tenant)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return rules.TariffSettings{DistanceUnit: record.DistanceMiles}, nil
		}
		return rules.TariffSettings{}, err
	}
	if control.StoredDistanceUnits == storedKilometers {
		return rules.TariffSettings{DistanceUnit: record.DistanceKilometers}, nil
	}
	return rules.TariffSettings{DistanceUnit: record.DistanceMiles}, nil
}
