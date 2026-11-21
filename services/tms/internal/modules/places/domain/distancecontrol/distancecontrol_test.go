package distancecontrol

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultMapsPurposes(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	profileID := pulid.MustNew("dp_")

	control := NewDefault(orgID, buID, profileID)

	require.True(t, control.StoreMileage)
	require.True(t, control.AutoCreateStoredMileage)
	require.True(t, control.PostalCodeFallbackToCity)
	require.Equal(t, distanceprofile.DefaultDistanceUnits, control.RatingDistanceUnit)
	require.False(t, control.CaptureJurisdictionMiles)
	require.Equal(t, profileID, control.ProfileIDForPurpose(PurposeLoadedMove))
	require.Equal(t, profileID, control.ProfileIDForPurpose(PurposeEmptyMove))
}

func TestValidateRejectsInvalidRatingDistanceUnit(t *testing.T) {
	control := NewDefault(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("dp_"))
	control.RatingDistanceUnit = "NauticalMiles"
	multiErr := errortypes.NewMultiError()

	control.Validate(multiErr)

	require.True(t, multiErr.HasErrorAt("ratingDistanceUnit"))
}

func TestProfileIDForPurposeNamesEachPurposesProfile(t *testing.T) {
	control := &DistanceControl{
		LoadedMoveDistanceProfileID:    pulid.MustNew("dp_"),
		EmptyMoveDistanceProfileID:     pulid.MustNew("dp_"),
		PayDistanceProfileID:           pulid.MustNew("dp_"),
		BillingDistanceProfileID:       pulid.MustNew("dp_"),
		FuelDistanceProfileID:          pulid.MustNew("dp_"),
		EtaOutOfRouteDistanceProfileID: pulid.MustNew("dp_"),
	}

	require.Equal(
		t,
		control.LoadedMoveDistanceProfileID,
		control.ProfileIDForPurpose(PurposeLoadedMove),
	)
	require.Equal(
		t,
		control.EmptyMoveDistanceProfileID,
		control.ProfileIDForPurpose(PurposeEmptyMove),
	)
	require.Equal(t, control.PayDistanceProfileID, control.ProfileIDForPurpose(PurposePay))
	require.Equal(t, control.BillingDistanceProfileID, control.ProfileIDForPurpose(PurposeBilling))
	require.Equal(t, control.FuelDistanceProfileID, control.ProfileIDForPurpose(PurposeFuel))
	require.Equal(
		t,
		control.EtaOutOfRouteDistanceProfileID,
		control.ProfileIDForPurpose(PurposeEtaOutOfRoute),
	)
}

func TestValidateAcceptsKilometersAsTheRatingDistanceUnit(t *testing.T) {
	control := NewDefault(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("dp_"))
	control.RatingDistanceUnit = RatingDistanceUnitKilometers
	multiErr := errortypes.NewMultiError()

	control.Validate(multiErr)

	require.False(t, multiErr.HasErrors())
}
