package podchase

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/modules/agent/app/drivermessage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerplay"
	"github.com/emoss08/trenova/internal/modules/agent/ports/podchaseports"
	"github.com/emoss08/trenova/shared/pulid"
)

func podFact() *podchaseports.Facts {
	return &podchaseports.Facts{
		ShipmentID:   pulid.MustNew("shp_"),
		ProNumber:    "S26080375057552",
		BOL:          "7557621815",
		CustomerName: "Simpson & Jones Industries",
		DeliveredAt:  time.Date(2026, 8, 15, 18, 0, 0, 0, time.UTC).Unix(),
	}
}

var everything = reach{email: true, texts: true}

// A driver on Dash is asked there, in a message that names the load and fits
// the message's bound, whatever else could reach them.
func TestPlanAskAsksADriverOnDash(t *testing.T) {
	t.Parallel()

	fact := podFact()
	fact.Driver = &podchaseports.Driver{
		WorkerID: pulid.MustNew("wrk_"), Name: "Laura Wright", OnDash: true,
		Email: "laura@drivers.test", HasPhone: true,
	}
	fact.Carrier = &podchaseports.Carrier{Name: "Acme", Email: "ops@acme.test"}

	planned := planAsk(fact, everything, "America/New_York")

	require.NotNil(t, planned)
	assert.Equal(t, watchtowerplay.ChannelDash, planned.channel)
	assert.Equal(t, fact.Driver.WorkerID, planned.workerID)
	assert.True(t, strings.HasPrefix(planned.message, "Hi Laura,"))
	assert.Contains(t, planned.message, "PRO S26080375057552 (BOL 7557621815)")
	assert.Contains(t, planned.message, "on Aug 15, 2026")
	assert.LessOrEqual(t, len(planned.message), drivermessage.MaxChars)
	assert.Equal(t, "Asked Laura Wright for the signed proof of delivery in Dash", planned.asked())
}

// A driver with no Dash login and an email on file is emailed before the
// carrier is: they hold the paperwork. The address stays out of every
// sentence a person or a model reads.
func TestPlanAskEmailsADriverWithoutDashBeforeTheCarrier(t *testing.T) {
	t.Parallel()

	fact := podFact()
	fact.Driver = &podchaseports.Driver{
		WorkerID: pulid.MustNew("wrk_"), Name: "Theresa Aguilar", Email: "theresa@drivers.test",
	}
	fact.Carrier = &podchaseports.Carrier{Name: "Acme Freight", Email: "ops@acme.test"}

	planned := planAsk(fact, everything, "UTC")

	require.NotNil(t, planned)
	assert.Equal(t, watchtowerplay.ChannelEmail, planned.channel)
	assert.Equal(t, "theresa@drivers.test", planned.email)
	assert.Equal(t, "Theresa Aguilar", planned.name)
	assert.Equal(t, "Signed proof of delivery for PRO S26080375057552", planned.subject)
	assert.NotContains(t, planned.message, "theresa@drivers.test")
	assert.NotContains(t, planned.asked(), "@")
	assert.Equal(t, "Asked Theresa Aguilar for the signed proof of delivery by email", planned.asked())

	call := planned.call(fact, "UTC")
	assert.Equal(t, toolDocumentRequest, call.tool)
	assert.Equal(t, []any{"theresa@drivers.test"}, call.arguments["to"])
	assert.NotContains(t, call.rationale, "@")
	assert.NotContains(t, call.arguments, "profileId", "the mailer picks the profile to send from")
}

// With no driver to ask, the carrier that hauled it is emailed.
func TestPlanAskEmailsTheCarrierWhenNoDriverCanBeAsked(t *testing.T) {
	t.Parallel()

	fact := podFact()
	fact.Driver = &podchaseports.Driver{WorkerID: pulid.MustNew("wrk_"), Name: "Theresa Aguilar"}
	fact.Carrier = &podchaseports.Carrier{Name: "Acme Freight", Email: "ops@acme.test"}

	planned := planAsk(fact, everything, "UTC")

	require.NotNil(t, planned)
	assert.Equal(t, watchtowerplay.ChannelEmail, planned.channel)
	assert.Equal(t, "Acme Freight", planned.name)
	assert.Equal(t, "ops@acme.test", planned.email)
}

// A text is the last resort: it cannot carry the paperwork back, so it is
// used only when a number is all there is and the plan includes texts.
func TestPlanAskTextsADriverOnlyWhenNothingElseReaches(t *testing.T) {
	t.Parallel()

	fact := podFact()
	fact.Driver = &podchaseports.Driver{
		WorkerID: pulid.MustNew("wrk_"), Name: "Theresa Aguilar", HasPhone: true,
	}

	planned := planAsk(fact, everything, "UTC")
	require.NotNil(t, planned)
	assert.Equal(t, watchtowerplay.ChannelText, planned.channel)
	assert.LessOrEqual(t, len(planned.message), drivermessage.MaxChars)
	assert.Equal(t, "Asked Theresa Aguilar for the signed proof of delivery by text", planned.asked())

	assert.Nil(t, planAsk(fact, reach{email: true}, "UTC"), "a plan without text messages")
}

// A load with no PRO is named as a person names it; its record id is never
// put in a message.
func TestPlanAskNeverNamesALoadByItsRecordID(t *testing.T) {
	t.Parallel()

	fact := podFact()
	fact.ProNumber, fact.BOL = "", ""
	fact.Driver = &podchaseports.Driver{
		WorkerID: pulid.MustNew("wrk_"), Name: "Laura Wright", OnDash: true,
	}

	planned := planAsk(fact, everything, "UTC")

	require.NotNil(t, planned)
	assert.NotContains(t, planned.message, fact.ShipmentID.String())
	assert.NotContains(t, planned.title, fact.ShipmentID.String())
	assert.Contains(t, planned.message, "the load")
}

// Nobody reachable: nobody is asked, and the reason names who is missing
// what without giving an address.
func TestPlanAskAsksNobodyWhenNobodyCanBeReached(t *testing.T) {
	t.Parallel()

	offDash := podFact()
	offDash.Driver = &podchaseports.Driver{WorkerID: pulid.MustNew("wrk_"), Name: "A Driver"}
	assert.Nil(t, planAsk(offDash, everything, "UTC"))
	refusal := unreachable(offDash, everything)
	assert.Equal(t,
		"{0} has no Dash login, email or phone number on file, so they cannot be asked for "+
			"the proof of delivery. Upload it here, or add a way to reach them.", refusal.Message)
	assert.Equal(t, []any{"A Driver"}, refusal.Args)

	noEmail := podFact()
	noEmail.Carrier = &podchaseports.Carrier{Name: "Acme"}
	assert.Nil(t, planAsk(noEmail, everything, "UTC"))
	refusal = unreachable(noEmail, everything)
	assert.Equal(t,
		"{0} has no email on file, so they cannot be asked for the proof of delivery. "+
			"Upload it here, or add their email.", refusal.Message)
	assert.Equal(t, []any{"Acme"}, refusal.Args)

	noProfile := podFact()
	noProfile.Carrier = &podchaseports.Carrier{Name: "Acme", Email: "ops@acme.test"}
	assert.Nil(t, planAsk(noProfile, reach{texts: true}, "UTC"), "no profile to email from")
	refusal = unreachable(noProfile, reach{texts: true})
	assert.Contains(t, refusal.Message, "No email profile is set up to send from")
	assert.NotContains(t, refusal.Message, "ops@acme.test")

	textOnly := podFact()
	textOnly.Driver = &podchaseports.Driver{
		WorkerID: pulid.MustNew("wrk_"), Name: "A Driver", HasPhone: true,
	}
	refusal = unreachable(textOnly, reach{email: true})
	assert.Contains(t, refusal.Message, "can only be reached by text")

	nobody := podFact()
	assert.Nil(t, planAsk(nobody, everything, "UTC"))
	assert.Contains(t, unreachable(nobody, everything).Message, "Nobody is assigned")
}

func TestClipBytesKeepsWholeCharacters(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "abc", clipBytes("abc", 10))
	clipped := clipBytes(strings.Repeat("é", 10), 5)
	assert.LessOrEqual(t, len(clipped), 5)
	assert.True(t, strings.HasPrefix(strings.Repeat("é", 10), clipped))
	for _, r := range clipped {
		assert.Equal(t, 'é', r)
	}
}
