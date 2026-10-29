package placesselectoptions

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestPlacesSelectOptionMappers(t *testing.T) {
	t.Parallel()

	locationOption := locationSelectOption(&location.Location{
		ID:   pulid.MustNew("loc_"),
		Code: "DAL01",
		Name: "Dallas DC",
	})
	assert.Equal(t, "Dallas DC", locationOption.Label)
	assert.Equal(t, "DAL01", *locationOption.Description)
	assert.Equal(t, "DAL01", locationOption.Meta["code"])

	zoneOption := rateZoneSelectOption(&ratezone.RateZone{
		ID:     pulid.MustNew("rzn_"),
		Code:   "SW",
		Name:   "Southwest",
		Status: domaintypes.StatusActive,
	})
	assert.Equal(t, "Southwest", zoneOption.Label)
	assert.Equal(t, "SW", *zoneOption.Description)
	assert.Equal(t, "Active", zoneOption.Meta["status"])

	stateOption := usStateSelectOption(&usstate.UsState{
		ID:           pulid.MustNew("us_"),
		Name:         "Illinois",
		Abbreviation: "IL",
		CountryIso3:  "USA",
	})
	assert.Equal(t, "Illinois", stateOption.Label)
	assert.Equal(t, "IL", stateOption.Meta["abbreviation"])
	assert.Equal(t, "USA", stateOption.Meta["countryIso3"])
}
