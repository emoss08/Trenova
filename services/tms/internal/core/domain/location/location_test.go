package location_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func timezoneErrors(multiErr *errortypes.MultiError) []string {
	messages := make([]string, 0, 1)
	for _, err := range multiErr.Errors {
		if err.Field == "timezone" {
			messages = append(messages, err.Message)
		}
	}
	return messages
}

func TestLocationValidate_TimezoneMustBeIANA(t *testing.T) {
	t.Parallel()

	invalid := &location.Location{Timezone: "Mars/Olympus_Mons"}
	multiErr := errortypes.NewMultiError()
	invalid.Validate(multiErr)
	assert.NotEmpty(t, timezoneErrors(multiErr), "an unknown zone is rejected")

	for _, tz := range []string{"", "America/Chicago", "UTC"} {
		valid := &location.Location{Timezone: tz}
		multiErr = errortypes.NewMultiError()
		valid.Validate(multiErr)
		assert.Empty(t, timezoneErrors(multiErr), "%q is a valid timezone", tz)
	}
}
