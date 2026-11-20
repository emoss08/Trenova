package dashcontrol

import (
	"fmt"
	"math"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const thresholdMessage = "Detention alert threshold must be between 15 minutes and 24 hours"

func validControl() *DashControl {
	return &DashControl{
		ID:                             pulid.MustNew("dashc_"),
		OrganizationID:                 pulid.MustNew("org_"),
		BusinessUnitID:                 pulid.MustNew("bu_"),
		SendCredentialReminders:        true,
		DriverDigestCadence:            DigestImmediate,
		DriverDigestWeekday:            DefaultDriverDigestWeekday,
		EnableDetentionAlerts:          true,
		DetentionAlertThresholdMinutes: DefaultDetentionAlertMinutes,
	}
}

func validate(dc *DashControl) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	dc.Validate(multiErr)
	return multiErr
}

func errorsAt(multiErr *errortypes.MultiError, field string) []string {
	var messages []string
	for _, err := range multiErr.Errors {
		if err.Path.String() == field {
			messages = append(messages, err.Message)
		}
	}
	return messages
}

func TestValidate_DetentionThresholdIsBoundedWhetherAlertsAreOnOrOff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		minutes int16
		wantErr bool
	}{
		{name: "zero", minutes: 0, wantErr: true},
		{name: "below the minimum", minutes: 14, wantErr: true},
		{name: "the minimum", minutes: 15, wantErr: false},
		{name: "the default the client sends while alerts are off", minutes: 120, wantErr: false},
		{name: "the maximum", minutes: 1440, wantErr: false},
		{name: "above the maximum", minutes: 1441, wantErr: true},
		{name: "the largest stored value", minutes: math.MaxInt16, wantErr: true},
		{name: "negative", minutes: -120, wantErr: true},
	}

	for _, alerts := range []bool{true, false} {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s, alerts on %v", tt.name, alerts), func(t *testing.T) {
				t.Parallel()

				dc := validControl()
				dc.EnableDetentionAlerts = alerts
				dc.DetentionAlertThresholdMinutes = tt.minutes

				multiErr := validate(dc)

				if tt.wantErr {
					assert.Equal(t,
						[]string{thresholdMessage},
						errorsAt(multiErr, "detentionAlertThresholdMinutes"),
					)
					return
				}
				assert.False(t, multiErr.HasErrors(), "%v", multiErr)
			})
		}
	}
}

func TestValidate_RefusesAThresholdThatWouldHaveWrappedIntoRange(t *testing.T) {
	t.Parallel()

	const sent = 65536 + 120

	dc := validControl()
	dc.EnableDetentionAlerts = false
	dc.DetentionAlertThresholdMinutes = intutils.SafeToInt16(sent)

	assert.Equal(t, int16(math.MaxInt16), dc.DetentionAlertThresholdMinutes)
	assert.Equal(t,
		[]string{thresholdMessage},
		errorsAt(validate(dc), "detentionAlertThresholdMinutes"),
	)
}

func TestValidate_DigestCadenceIsRequired(t *testing.T) {
	t.Parallel()

	dc := validControl()
	dc.DriverDigestCadence = ""

	multiErr := validate(dc)

	assert.Equal(t,
		[]string{"Digest cadence is required"},
		errorsAt(multiErr, "driverDigestCadence"),
	)
}

func TestValidate_DigestWeekdayMustBeADayOfTheWeek(t *testing.T) {
	t.Parallel()

	for _, day := range []int16{-1, 7, math.MaxInt16} {
		dc := validControl()
		dc.DriverDigestWeekday = day

		assert.Equal(t,
			[]string{"Digest day must be a day of the week"},
			errorsAt(validate(dc), "driverDigestWeekday"),
			"day %d", day,
		)
	}

	for day := int16(0); day <= 6; day++ {
		dc := validControl()
		dc.DriverDigestWeekday = day

		assert.False(t, validate(dc).HasErrors(), "day %d", day)
	}
}

func TestBeforeCreate_KeepsSundayAsTheDigestDay(t *testing.T) {
	t.Parallel()

	dc := &DashControl{DriverDigestWeekday: 0}

	require.NoError(t, dc.BeforeCreate(t.Context()))

	assert.Equal(t, int16(0), dc.DriverDigestWeekday)
	assert.Equal(t, DigestImmediate, dc.DriverDigestCadence)
	assert.Equal(t, DefaultDetentionAlertMinutes, dc.DetentionAlertThresholdMinutes)
}
