package worker

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
)

func homedWorker() *Worker {
	return &Worker{
		AddressLine1:  "1 Main St",
		City:          "Joliet",
		PostalCode:    "60432",
		HomeLatitude:  new(41.52),
		HomeLongitude: new(-88.08),
		HomePlaceID:   "place_1",
	}
}

func TestKeepHomeWithAddress_ClearsAHomeTheNewAddressWasNotLookedUpFor(t *testing.T) {
	t.Parallel()

	stored := homedWorker()
	edited := homedWorker()
	edited.City = "Plainfield"

	edited.KeepHomeWithAddress(stored)

	assert.Nil(t, edited.HomeLatitude)
	assert.Nil(t, edited.HomeLongitude)
	assert.Empty(t, edited.HomePlaceID)
}

func TestKeepHomeWithAddress_KeepsAHomeLookedUpForTheNewAddress(t *testing.T) {
	t.Parallel()

	stored := homedWorker()
	edited := homedWorker()
	edited.City = "Plainfield"
	edited.HomePlaceID = "place_2"
	edited.HomeLatitude = new(41.61)

	edited.KeepHomeWithAddress(stored)

	assert.InDelta(t, 41.61, *edited.HomeLatitude, 0.0001)
	assert.Equal(t, "place_2", edited.HomePlaceID)
}

func TestKeepHomeWithAddress_LeavesTheHomeAloneWhenTheAddressIsUnchanged(t *testing.T) {
	t.Parallel()

	stored := homedWorker()
	edited := homedWorker()

	edited.KeepHomeWithAddress(stored)

	assert.NotNil(t, edited.HomeLatitude)
}

func TestAtHome_IsWithinTheHomeRadius(t *testing.T) {
	t.Parallel()

	assert.True(t, AtHome(41.52, -88.08, 41.60, -88.08))
	assert.False(t, AtHome(41.52, -88.08, 39.74, -104.99))
}

func TestValidate_RefusesAHalfSetHomeAndALastHomeInTheFuture(t *testing.T) {
	t.Parallel()

	w := validWorker()
	w.HomeLatitude = new(41.52)
	w.LastHomeAt = new(timeutils.NowUnix() + 3*24*60*60)

	multiErr := errortypes.NewMultiError()
	w.Validate(multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}

	assert.Contains(t, fields, "homeLatitude")
	assert.Contains(t, fields, "lastHomeAt")
}

func TestWithholdFields_HidesTheHomePointWithoutPersonalDetails(t *testing.T) {
	t.Parallel()

	w := homedWorker()
	w.WithholdFields(0)

	assert.Nil(t, w.HomeLatitude)
	assert.Nil(t, w.HomeLongitude)
	assert.Empty(t, w.HomePlaceID)

	shown := homedWorker()
	shown.WithholdFields(protectedfield.AllGroups())
	assert.NotNil(t, shown.HomeLatitude, "a reader with personal details sees where home is")
}

func TestKeepFields_KeepsTheStoredHomeWhenTheEditorCannotSeeIt(t *testing.T) {
	t.Parallel()

	stored := homedWorker()
	edited := homedWorker()
	edited.HomeLatitude, edited.HomeLongitude, edited.HomePlaceID = nil, nil, ""

	edited.KeepFields(stored, 0)

	assert.InDelta(t, 41.52, *edited.HomeLatitude, 0.0001,
		"a save by someone who cannot see the home point never clears it")
	assert.Equal(t, "place_1", edited.HomePlaceID)
}
