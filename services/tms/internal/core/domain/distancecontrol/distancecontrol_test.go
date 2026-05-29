package distancecontrol

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultMapsPurposes(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	practicalID := pulid.MustNew("dp_")
	shortestID := pulid.MustNew("dp_")

	control := NewDefault(orgID, buID, practicalID, shortestID)

	require.True(t, control.StoreMileage)
	require.True(t, control.AutoCreateStoredMileage)
	require.True(t, control.PostalCodeFallbackToCity)
	require.Equal(t, distanceprofile.DefaultDistanceUnits, control.StoredDistanceUnits)
	require.Equal(t, practicalID, control.ProfileIDForPurpose(PurposeLoadedMove))
	require.Equal(t, practicalID, control.ProfileIDForPurpose(PurposeEmptyMove))
	require.Equal(t, practicalID, control.ProfileIDForPurpose(PurposeDistanceCalculatorPractical))
	require.Equal(t, shortestID, control.ProfileIDForPurpose(PurposeDistanceCalculatorShortest))
}

func TestValidateRejectsInvalidStoredUnits(t *testing.T) {
	control := NewDefault(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("dp_"), "")
	control.StoredDistanceUnits = "NauticalMiles"
	multiErr := errortypes.NewMultiError()

	control.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
}
