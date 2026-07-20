package report

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func validSchedule() *ReportSchedule {
	return &ReportSchedule{
		CronExpression: "0 8 * * 1",
		Timezone:       "UTC",
		Formats:        []string{"xlsx"},
		RunAsID:        pulid.ID("usr_test"),
	}
}

func validateSchedule(rs *ReportSchedule) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	rs.Validate(multiErr)
	return multiErr
}

func TestScheduleDeliveryValidate(t *testing.T) {
	t.Run("nil delivery is valid", func(t *testing.T) {
		assert.False(t, validateSchedule(validSchedule()).HasErrors())
	})

	t.Run("valid delivery", func(t *testing.T) {
		rs := validSchedule()
		rs.Delivery = &ScheduleDelivery{
			EmailRecipients: []string{"ops@example.com", "billing@example.com"},
			EmailAttach:     true,
			NotifyUserIDs:   []pulid.ID{pulid.ID("usr_a")},
		}
		assert.False(t, validateSchedule(rs).HasErrors())
	})

	t.Run("invalid email", func(t *testing.T) {
		rs := validSchedule()
		rs.Delivery = &ScheduleDelivery{EmailRecipients: []string{"not-an-email"}}
		multiErr := validateSchedule(rs)
		assert.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "is not a valid email address")
	})

	t.Run("too many recipients", func(t *testing.T) {
		rs := validSchedule()
		recipients := make([]string, 0, MaxScheduleEmailRecipients+1)
		for range MaxScheduleEmailRecipients + 1 {
			recipients = append(recipients, "ops@example.com")
		}
		rs.Delivery = &ScheduleDelivery{EmailRecipients: recipients}
		assert.True(t, validateSchedule(rs).HasErrors())
	})

	t.Run("too many notify users", func(t *testing.T) {
		rs := validSchedule()
		users := make([]pulid.ID, 0, MaxScheduleNotifyUsers+1)
		for range MaxScheduleNotifyUsers + 1 {
			users = append(users, pulid.ID("usr_a"))
		}
		rs.Delivery = &ScheduleDelivery{NotifyUserIDs: users}
		assert.True(t, validateSchedule(rs).HasErrors())
	})

	t.Run("nil notify user id", func(t *testing.T) {
		rs := validSchedule()
		rs.Delivery = &ScheduleDelivery{NotifyUserIDs: []pulid.ID{pulid.Nil}}
		multiErr := validateSchedule(rs)
		assert.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "In-app recipient is invalid")
	})
}

func TestScheduleDeliveryHelpers(t *testing.T) {
	var nilDelivery *ScheduleDelivery
	assert.False(t, nilDelivery.HasEmail())
	assert.False(t, nilDelivery.HasNotify())

	delivery := &ScheduleDelivery{
		EmailRecipients: []string{"ops@example.com"},
		NotifyUserIDs:   []pulid.ID{pulid.ID("usr_a")},
	}
	assert.True(t, delivery.HasEmail())
	assert.True(t, delivery.HasNotify())
	assert.True(t, strings.Contains(delivery.EmailRecipients[0], "@"))
}
