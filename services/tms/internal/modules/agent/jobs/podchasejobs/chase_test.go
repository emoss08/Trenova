package podchasejobs

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/podchaseports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// A driver on Dash is asked there, in a message that names the load and fits
// the message tool's bound.
func TestPlanCallAsksADriverOnDash(t *testing.T) {
	t.Parallel()

	fact := podFact()
	fact.Driver = &podchaseports.Driver{WorkerID: pulid.MustNew("wrk_"), Name: "Laura Wright", OnDash: true}
	fact.Carrier = &podchaseports.Carrier{CarrierID: pulid.MustNew("car_"), Name: "Acme", Email: "ops@acme.test"}

	call := planCall(fact, "emp_billing", "America/New_York")

	require.NotNil(t, call)
	assert.Equal(t, toolDriverMessage, call.tool)
	assert.Equal(t, fact.Driver.WorkerID.String(), call.arguments["workerId"])
	assert.Equal(t, fact.ShipmentID.String(), call.arguments["shipmentId"])
	message, ok := call.arguments["message"].(string)
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(message, "Hi Laura,"))
	assert.Contains(t, message, "PRO S26080375057552 (BOL 7557621815)")
	assert.Contains(t, message, "on Aug 15, 2026")
	assert.LessOrEqual(t, len(message), maxDriverMessageBytes)
	assert.Contains(t, call.rationale, "Laura Wright drove the delivery")
}

// A driver without Dash cannot be reached that way, so the carrier is emailed
// from the billing sender; the request is one, never a message and an email.
func TestPlanCallEmailsTheCarrierWhenTheDriverIsNotOnDash(t *testing.T) {
	t.Parallel()

	fact := podFact()
	fact.Driver = &podchaseports.Driver{WorkerID: pulid.MustNew("wrk_"), Name: "Theresa Aguilar"}
	fact.Carrier = &podchaseports.Carrier{CarrierID: pulid.MustNew("car_"), Name: "Acme Freight", Email: "ops@acme.test"}

	call := planCall(fact, "emp_billing", "UTC")

	require.NotNil(t, call)
	assert.Equal(t, toolDocumentRequest, call.tool)
	assert.Equal(t, "emp_billing", call.arguments["profileId"])
	assert.Equal(t, []any{"ops@acme.test"}, call.arguments["to"])
	assert.Equal(t, "Acme Freight", call.arguments["customerName"])
	assert.Equal(t, []any{requestedDocument}, call.arguments["requestedDocuments"])
}

// Nobody reachable, no carrier email, or no billing sender: the routine asks
// nobody and leaves the shipment to the agent's model.
func TestPlanCallLeavesAnUnreachableShipmentToTheModel(t *testing.T) {
	t.Parallel()

	offDash := podFact()
	offDash.Driver = &podchaseports.Driver{WorkerID: pulid.MustNew("wrk_"), Name: "A Driver"}
	assert.Nil(t, planCall(offDash, "emp_billing", "UTC"), "a driver off Dash and no carrier")

	noEmail := podFact()
	noEmail.Carrier = &podchaseports.Carrier{CarrierID: pulid.MustNew("car_"), Name: "Acme"}
	assert.Nil(t, planCall(noEmail, "emp_billing", "UTC"), "a carrier with no email")

	noSender := podFact()
	noSender.Carrier = &podchaseports.Carrier{CarrierID: pulid.MustNew("car_"), Name: "Acme", Email: "ops@acme.test"}
	assert.Nil(t, planCall(noSender, "", "UTC"), "no billing sender to email from")
}

func TestClipBytesKeepsWholeCharacters(t *testing.T) {
	t.Parallel()

	text := strings.Repeat("é", 300)
	clipped := clipBytes(text, maxDriverMessageBytes)

	assert.LessOrEqual(t, len(clipped), maxDriverMessageBytes)
	assert.True(t, strings.HasPrefix(text, clipped))
	assert.Equal(t, "short", clipBytes("short", maxDriverMessageBytes))
}
