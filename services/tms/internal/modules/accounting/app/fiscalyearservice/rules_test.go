package fiscalyearservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:paralleltest // mutates the process-wide time.Local
func TestDateValidationRuleReadsCalendarBoundsInUTC(t *testing.T) {
	zones := []*time.Location{
		time.FixedZone("CST", -6*60*60),
		time.FixedZone("JST", 9*60*60),
	}

	for _, zone := range zones {
		t.Run(zone.String(), func(t *testing.T) {
			previous := time.Local
			time.Local = zone
			t.Cleanup(func() { time.Local = previous })

			entity := openYear()
			multiErr := errortypes.NewMultiError()

			err := createDateValidationRule().Validate(
				t.Context(),
				entity,
				&validationframework.TenantedValidationContext{
					Mode:           validationframework.ModeCreate,
					OrganizationID: entity.OrganizationID,
					BusinessUnitID: entity.BusinessUnitID,
				},
				multiErr,
			)

			require.NoError(t, err)
			assert.False(t, multiErr.HasErrors(), fieldMessages(t, multiErr))
		})
	}
}

//nolint:paralleltest // mutates the process-wide time.Local
func TestDateValidationRuleRejectsNonCalendarBoundsInUTC(t *testing.T) {
	previous := time.Local
	time.Local = time.FixedZone("CST", -6*60*60)
	t.Cleanup(func() { time.Local = previous })

	entity := openYear()
	entity.StartDate = time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC).Unix()
	entity.EndDate = time.Date(2027, time.January, 1, 23, 59, 59, 0, time.UTC).Unix()
	multiErr := errortypes.NewMultiError()

	err := createDateValidationRule().Validate(
		t.Context(),
		entity,
		&validationframework.TenantedValidationContext{Mode: validationframework.ModeCreate},
		multiErr,
	)

	require.NoError(t, err)
	messages := fieldMessages(t, multiErr)
	assert.Contains(t, messages["startDate"], "Calendar year must start on January 1st")
	assert.Contains(t, messages["endDate"], "Calendar year must end on December 31st")
}
