package trackingprofile

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
)

const now = int64(1_800_000_000)

func validProfile() *Profile {
	return NewDefault(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("cus_"))
}

func TestProfile_NormalizeOrdersAndDeduplicatesFields(t *testing.T) {
	p := validProfile()
	p.VisibleFields = []Field{FieldEta, FieldStatus, FieldEta, Field("Bogus")}

	p.Normalize()

	assert.Equal(t, []Field{FieldStatus, FieldEta}, p.VisibleFields)
}

func TestProfile_NormalizeCleansRecipients(t *testing.T) {
	p := validProfile()
	p.Recipients = " Ops@Example.com; ops@example.com,\nlane@example.com "

	p.Normalize()

	assert.Equal(t, "ops@example.com, lane@example.com", p.Recipients)
	assert.Equal(t, []string{"ops@example.com", "lane@example.com"}, p.RecipientList())
}

func TestProfile_StampNotificationsWhenARuleIsSwitchedOn(t *testing.T) {
	earlier := now - 3600
	previous := validProfile()
	previous.NotifyDelivered = true
	previous.NotificationsEnabledAt = &earlier

	unchanged := validProfile()
	unchanged.NotifyDelivered = true
	unchanged.StampNotifications(previous, now)
	assert.Equal(t, earlier, *unchanged.NotificationsEnabledAt)

	added := validProfile()
	added.NotifyDelivered = true
	added.NotifyPickedUp = true
	added.StampNotifications(previous, now)
	assert.Equal(t, now, *added.NotificationsEnabledAt)

	fresh := validProfile()
	fresh.NotifyDelayed = true
	fresh.StampNotifications(nil, now)
	assert.Equal(t, now, *fresh.NotificationsEnabledAt)

	neverSaved := validProfile()
	neverSaved.NotifyPodReady = true
	neverSaved.StampNotifications(validProfile(), now)
	assert.Equal(t, now, *neverSaved.NotificationsEnabledAt)

	off := validProfile()
	off.StampNotifications(previous, now)
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

func TestProfile_ValidateAcceptsTheDefaults(t *testing.T) {
	multiErr := errortypes.NewMultiError()
	validProfile().Validate(multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestProfile_LinkTTLFallsBackToTheDefault(t *testing.T) {
	p := validProfile()
	p.LinkTTLDays = 0

	assert.Equal(t, int64(DefaultLinkTTLDays)*timeutils.SecondsPerDay, p.LinkTTLSeconds())
}
