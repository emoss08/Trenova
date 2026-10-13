package customertracking

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func validProfile() *Profile {
	p := DefaultProfile(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("cus_"))
	p.ID = NewProfileID()
	return p
}

func TestProfile_NormalizeOrdersAndDeduplicatesFields(t *testing.T) {
	p := validProfile()
	p.VisibleFields = []Field{FieldEta, FieldStatus, FieldEta, Field("Bogus")}

	p.Normalize(nil, now)

	assert.Equal(t, []Field{FieldStatus, FieldEta}, p.VisibleFields)
}

func TestProfile_NormalizeCleansRecipients(t *testing.T) {
	p := validProfile()
	p.Recipients = " Ops@Example.com; ops@example.com,\nlane@example.com "

	p.Normalize(nil, now)

	assert.Equal(t, "ops@example.com, lane@example.com", p.Recipients)
	assert.Equal(t, []string{"ops@example.com", "lane@example.com"}, p.RecipientList())
}

func TestProfile_NormalizeStampsWhenARuleIsSwitchedOn(t *testing.T) {
	earlier := now - 3600
	previous := validProfile()
	previous.NotifyDelivered = true
	previous.NotificationsEnabledAt = &earlier

	unchanged := validProfile()
	unchanged.NotifyDelivered = true
	unchanged.Normalize(previous, now)
	assert.Equal(t, &earlier, unchanged.NotificationsEnabledAt)

	added := validProfile()
	added.NotifyDelivered = true
	added.NotifyPickedUp = true
	added.Normalize(previous, now)
	assert.Equal(t, now, *added.NotificationsEnabledAt)

	fresh := validProfile()
	fresh.NotifyDelayed = true
	fresh.Normalize(nil, now)
	assert.Equal(t, now, *fresh.NotificationsEnabledAt)

	off := validProfile()
	off.Normalize(previous, now)
	assert.Nil(t, off.NotificationsEnabledAt)
}

func TestProfile_ValidateRejectsBadRecipientsAndTTL(t *testing.T) {
	p := validProfile()
	p.Recipients = "not-an-email"
	p.LinkTTLDays = 181

	multiErr := errortypes.NewMultiError()
	p.Validate(multiErr)

	assert.True(t, multiErr.HasErrors())
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	assert.ElementsMatch(t, []string{"recipients", "linkTtlDays"}, fields)
}

func TestProfile_LinkTTLFallsBackToTheDefault(t *testing.T) {
	p := validProfile()
	p.LinkTTLDays = 0

	assert.Equal(t, int64(DefaultLinkTTLDays)*SecondsPerDay, p.LinkTTLSeconds())
}

func TestRetryDelaySecondsBacksOffToAnHour(t *testing.T) {
	assert.Equal(t, int64(60), RetryDelaySeconds(1))
	assert.Equal(t, int64(120), RetryDelaySeconds(2))
	assert.Equal(t, int64(3600), RetryDelaySeconds(8))
}
