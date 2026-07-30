package driverportalservice

import (
	"html/template"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

const invitationNow = int64(1_767_225_600)

func invitedWorker(org *tenant.Organization) *worker.Worker {
	return &worker.Worker{
		ID:           pulid.MustNew("wrk_"),
		FirstName:    "Marcus",
		LastName:     "Dell",
		Organization: org,
	}
}

// The deadline the message quotes comes from the invitation row, not from the
// TTL constant. An invitation issued yesterday must not tell the driver they have
// seven days left.
func TestInvitationContextQuotesTheInvitationsOwnDeadline(t *testing.T) {
	t.Parallel()

	fresh := invitationContext(
		invitedWorker(nil),
		"https://dash.example.com/dash/accept?token=abc",
		invitationNow+7*secondsPerDay,
		invitationNow,
	)
	assert.Equal(t, 7, fresh.ExpiresInDays)

	issuedFiveDaysAgo := invitationContext(
		invitedWorker(nil),
		"https://dash.example.com/dash/accept?token=abc",
		invitationNow+2*secondsPerDay,
		invitationNow,
	)
	assert.Equal(t, 2, issuedFiveDaysAgo.ExpiresInDays)

	// Part of a day still counts as a day; "expires in 0 days" on a live link
	// would read as already dead.
	assert.Equal(t, 1, expiresInDays(invitationNow+90, invitationNow))

	// A lapsed invitation reports nothing, and the template drops the sentence
	// rather than promising a link that no longer works.
	assert.Equal(t, 0, expiresInDays(invitationNow-1, invitationNow))
	assert.Equal(t, 0, expiresInDays(invitationNow, invitationNow))
}

func TestInvitationContextNamesTheCarrierAndFormatsTheDeadlineInItsZone(t *testing.T) {
	t.Parallel()

	withOrg := invitationContext(
		invitedWorker(&tenant.Organization{Name: "Trenova Logistics", Timezone: "America/Chicago"}),
		"https://dash.example.com/dash/accept?token=abc",
		invitationNow+7*secondsPerDay,
		invitationNow,
	)
	assert.Equal(t, "Trenova Logistics", withOrg.CompanyName)
	assert.Equal(t, "Marcus Dell", withOrg.FullName)
	assert.Contains(t, withOrg.ExpiresAt, "CST",
		"a driver reads the deadline in their carrier's zone, not in UTC")

	// An organization that was not loaded, or has no name, still produces a
	// sendable invitation: the driver needs the link more than the letterhead.
	withoutOrg := invitationContext(
		invitedWorker(nil),
		"https://dash.example.com/dash/accept?token=abc",
		invitationNow+7*secondsPerDay,
		invitationNow,
	)
	assert.Equal(t, defaultInviterName, withoutOrg.CompanyName)
	assert.NotEmpty(t, withoutOrg.ExpiresAt)

	assert.Equal(
		t,
		template.URL("https://dash.example.com/dash/accept?token=abc"),
		withOrg.InviteURL,
		"the link is typed template.URL so html/template will not mangle it",
	)
}
