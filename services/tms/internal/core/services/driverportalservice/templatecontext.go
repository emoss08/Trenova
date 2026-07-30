package driverportalservice

import (
	"html/template"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/timeutils"
)

// secondsPerDay converts the invitation's stored expiry into the "expires in N
// days" the message quotes.
const secondsPerDay = int64(24 * 60 * 60)

// defaultInviterName is what an invitation calls the carrier when the
// organization record has no name to use.
const defaultInviterName = "your carrier"

// invitationContext maps a driver and their pending invitation onto the
// variables the invitation template declares.
//
// There is no registered ContextBuilder for this kind, deliberately: an
// invitation cannot be rendered from a worker id alone, because its single-use
// link exists only in the request that minted it. Rendering one without that link
// would produce an email a driver cannot act on, so the send path is the only
// caller and it supplies the context directly.
func invitationContext(
	wrk *worker.Worker,
	inviteURL string,
	expiresAt, now int64,
) documenttemplate.DriverPortalInvitationContext {
	companyName := defaultInviterName
	timezone := ""
	if wrk.Organization != nil {
		if wrk.Organization.Name != "" {
			companyName = wrk.Organization.Name
		}
		timezone = wrk.Organization.Timezone
	}

	return documenttemplate.DriverPortalInvitationContext{
		FirstName:   wrk.FirstName,
		LastName:    wrk.LastName,
		FullName:    strings.TrimSpace(wrk.FirstName + " " + wrk.LastName),
		CompanyName: companyName,
		// The link is built from the configured portal base URL and a token this
		// system generated, which is why the context types it as a template.URL:
		// html/template refuses to interpolate an unrecognized scheme, and every
		// other field here is driver-entered text that must stay escaped.
		//
		//nolint:gosec // G203: system-built URL, never user input. See the field's doc.
		InviteURL:     template.URL(inviteURL),
		ExpiresInDays: expiresInDays(expiresAt, now),
		ExpiresAt:     timeutils.FormatUnixDateTimeIn(expiresAt, timezone),
	}
}

// expiresInDays rounds the remaining life of an invitation up to whole days.
//
// It reads the invitation's own expiry rather than the TTL constant, so an
// invitation that was issued earlier or extended later quotes the deadline it
// actually has. A lapsed invitation reports zero and the template drops the
// sentence rather than promising a link that is already dead.
func expiresInDays(expiresAt, now int64) int {
	remaining := expiresAt - now
	if remaining <= 0 {
		return 0
	}

	return int((remaining + secondsPerDay - 1) / secondsPerDay)
}
