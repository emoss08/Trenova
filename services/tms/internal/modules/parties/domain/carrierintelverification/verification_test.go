package carrierintelverification_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelverification"
	"github.com/emoss08/trenova/internal/modules/parties/partiestest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEquipmentVerificationValidateAndOverride(t *testing.T) {
	t.Parallel()

	v := &carrierintelverification.CarrierEquipmentVerification{
		CarrierAssignmentID: pulid.MustNew("ca_"),
		CarrierID:           pulid.MustNew("car_"),
		UnitType:            carrierintel.UnitTypeTractor,
		VIN:                 " 1ftfw1et5dfc10312 ",
		PlateNumber:         "abc123",
	}
	multiErr := errortypes.NewMultiError()
	v.Validate(multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	assert.Equal(t, "1FTFW1ET5DFC10312", v.VIN)
	assert.Contains(t, fields, "plateState")

	v.Result = carrierintelverification.VerificationResultMismatch
	require.Error(t, v.Override(pulid.MustNew("usr_"), " ", partiestest.CarrierIntelNow))
	require.NoError(
		t,
		v.Override(
			pulid.MustNew("usr_"),
			"Swapped tractor, confirmed by phone",
			partiestest.CarrierIntelNow,
		),
	)
	assert.True(t, v.IsCleared())
	require.Error(t, v.Override(pulid.MustNew("usr_"), "again", partiestest.CarrierIntelNow))
}
