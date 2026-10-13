package customertrackingservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pickedUpShipment(h *harness) *shipment.Shipment {
	return h.addShipment(shipment.StatusInTransit,
		stopSpec{stopType: shipment.StopTypePickup, city: "Dallas", state: "TX",
			arrival: at(-7200), departure: at(-3600)},
		stopSpec{stopType: shipment.StopTypeDelivery, city: "Memphis", state: "TN"},
	)
}

func TestEvaluate_SendsAnEventExactlyOnceHoweverOftenItIsSeen(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)
	h.notify(sh.CustomerID, func(p *customertracking.Profile) { p.NotifyPickedUp = true })

	first, err := h.service.Evaluate(t.Context(), testTenant, sh.ID)
	require.NoError(t, err)
	second, err := h.service.Evaluate(t.Context(), testTenant, sh.ID)
	require.NoError(t, err)

	assert.Equal(t, 1, first.Recorded)
	assert.Equal(t, 1, first.Delivered)
	assert.Equal(t, 1, second.Detected)
	assert.Equal(t, 0, second.Recorded)
	require.Len(t, h.email.sent, 1)
	sent := h.email.sent[0]
	assert.Equal(t, []string{"ops@halstead.example"}, sent.To)

	row := h.notifications.byEvent(customertracking.EventPickedUp)
	require.NotNil(t, row)
	assert.Equal(t, customertracking.NotificationStatusSent, row.Status)
	assert.Equal(t, row.IdempotencyKey(), sent.IdempotencyKey)
	assert.Equal(t, testNow-3600, row.OccurredAt)
	assert.False(t, row.EmailMessageID.IsNil())
}

func TestEvaluate_TheEmailCarriesItsOwnTrackingLink(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)
	h.notify(sh.CustomerID, func(p *customertracking.Profile) { p.NotifyPickedUp = true })

	_, err := h.service.Evaluate(t.Context(), testTenant, sh.ID)
	require.NoError(t, err)

	require.Len(t, h.links.rows, 1)
	link := h.links.rows[0]
	assert.Equal(t, customertracking.LinkPurposeNotification, link.Purpose)
	assert.Equal(t, testNow+int64(customertracking.DefaultLinkTTLDays)*customertracking.SecondsPerDay,
		link.ExpiresAt)
	require.Len(t, h.templates.calls, 1)
	data, ok := h.templates.calls[0].data.(documenttemplate.CustomerStatusEmailContext)
	require.True(t, ok)
	assert.True(t, data.IsPickedUp)
	assert.True(t, strings.HasPrefix(string(data.TrackingURL),
		"https://app.example.com/shipment-tracking/"))
	assert.Equal(t, "Dallas, TX", data.OriginSummary)
	assert.Equal(t, "Fri 15 Jan 2027 01:00 CST", data.OccurredAt)
	assert.Equal(t, link.ID, h.notifications.byEvent(customertracking.EventPickedUp).LinkID)
}

func TestEvaluate_NoLinkWhenTheCustomerTurnedLinksOff(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)
	h.notify(sh.CustomerID, func(p *customertracking.Profile) {
		p.NotifyPickedUp = true
		p.LinkEnabled = false
	})

	_, err := h.service.Evaluate(t.Context(), testTenant, sh.ID)
	require.NoError(t, err)

	assert.Empty(t, h.links.rows)
	data := h.templates.calls[0].data.(documenttemplate.CustomerStatusEmailContext)
	assert.Empty(t, data.TrackingURL)
	require.Len(t, h.email.sent, 1)
}

func TestEvaluate_RecipientsFallBackToTheCustomer(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)
	h.notify(sh.CustomerID, func(p *customertracking.Profile) {
		p.NotifyPickedUp = true
		p.Recipients = ""
	})
	h.customers.rows[sh.CustomerID].StatusUpdateRecipients = "Track@Halstead.example; desk@halstead.example"

	_, err := h.service.Evaluate(t.Context(), testTenant, sh.ID)
	require.NoError(t, err)
	require.Len(t, h.email.sent, 1)
	assert.Equal(t, []string{"track@halstead.example", "desk@halstead.example"}, h.email.sent[0].To)

	other := pickedUpShipment(h)
	h.notify(other.CustomerID, func(p *customertracking.Profile) {
		p.NotifyPickedUp = true
		p.Recipients = ""
	})
	h.customers.rows[other.CustomerID].EmailProfile = &customer.CustomerEmailProfile{
		ToRecipients: "billing@halstead.example",
	}
	_, err = h.service.Evaluate(t.Context(), testTenant, other.ID)
	require.NoError(t, err)
	require.Len(t, h.email.sent, 2)
	assert.Equal(t, []string{"billing@halstead.example"}, h.email.sent[1].To)
}

func TestEvaluate_SkipsWithoutRecipientsAndSaysWhy(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)
	h.notify(sh.CustomerID, func(p *customertracking.Profile) {
		p.NotifyPickedUp = true
		p.Recipients = ""
	})

	result, err := h.service.Evaluate(t.Context(), testTenant, sh.ID)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Skipped)
	assert.Empty(t, h.email.sent)
	row := h.notifications.byEvent(customertracking.EventPickedUp)
	assert.Equal(t, customertracking.NotificationStatusSkipped, row.Status)
	assert.Equal(t, customertracking.SkipReasonNoRecipients, row.SkipReason)
}

func TestDelivery_RetriesWithBackoffThenGivesUp(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)
	h.notify(sh.CustomerID, func(p *customertracking.Profile) {
		p.NotifyPickedUp = true
		p.IncludeLink = false
	})
	h.email.failures = 1

	_, err := h.service.Evaluate(t.Context(), testTenant, sh.ID)
	require.NoError(t, err)
	row := h.notifications.byEvent(customertracking.EventPickedUp)
	assert.Equal(t, customertracking.NotificationStatusPending, row.Status)
	assert.Equal(t, int16(1), row.Attempts)
	assert.Equal(t, testNow+60, row.NextAttemptAt)
	assert.Contains(t, row.LastError, "provider unavailable")

	early, err := h.service.DeliverDue(t.Context(), testTenant)
	require.NoError(t, err)
	assert.Zero(t, early.Delivered)

	h.now = testNow + 60
	due, err := h.service.DeliverDue(t.Context(), testTenant)
	require.NoError(t, err)
	assert.Equal(t, 1, due.Delivered)
	assert.Equal(t, customertracking.NotificationStatusSent, row.Status)
	require.Len(t, h.email.sent, 1)

	h2 := newHarness()
	sh2 := pickedUpShipment(h2)
	h2.notify(sh2.CustomerID, func(p *customertracking.Profile) {
		p.NotifyPickedUp = true
		p.IncludeLink = false
	})
	h2.email.failures = customertracking.MaxDeliveryAttempts
	_, err = h2.service.Evaluate(t.Context(), testTenant, sh2.ID)
	require.NoError(t, err)
	failed := h2.notifications.byEvent(customertracking.EventPickedUp)
	for failed.Status == customertracking.NotificationStatusPending {
		h2.now = failed.NextAttemptAt
		_, err = h2.service.DeliverDue(t.Context(), testTenant)
		require.NoError(t, err)
	}
	assert.Equal(t, customertracking.NotificationStatusFailed, failed.Status)
	assert.Equal(t, int16(customertracking.MaxDeliveryAttempts), failed.Attempts)
	assert.Empty(t, h2.email.sent)
}

func TestDelivery_ARuleTurnedOffBeforeTheRetrySkipsTheEmail(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)
	h.notify(sh.CustomerID, func(p *customertracking.Profile) {
		p.NotifyPickedUp = true
		p.IncludeLink = false
	})
	h.email.failures = 1
	_, err := h.service.Evaluate(t.Context(), testTenant, sh.ID)
	require.NoError(t, err)

	h.profiles.rows[sh.CustomerID].NotifyPickedUp = false
	h.now = testNow + 60
	_, err = h.service.DeliverDue(t.Context(), testTenant)
	require.NoError(t, err)

	row := h.notifications.byEvent(customertracking.EventPickedUp)
	assert.Equal(t, customertracking.NotificationStatusSkipped, row.Status)
	assert.Equal(t, customertracking.SkipReasonRuleOff, row.SkipReason)
	assert.Empty(t, h.email.sent)
}

func TestEvaluate_DelayedNamesTheStopAndItsNewArrival(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)
	h.notify(sh.CustomerID, func(p *customertracking.Profile) { p.NotifyDelayed = true })
	delivery := sh.Moves[0].Stops[1]
	estimated := testNow + 5*3600
	h.etas.rows = []*stopeta.StopETA{{
		StopID:           delivery.ID,
		Verdict:          stopeta.VerdictLate,
		EstimatedArrival: &estimated,
		ComputedAt:       testNow - 60,
	}}

	_, err := h.service.Evaluate(t.Context(), testTenant, sh.ID)
	require.NoError(t, err)

	row := h.notifications.byEvent(customertracking.EventDelayed)
	require.NotNil(t, row)
	assert.Equal(t, delivery.ID.String(), row.SubjectKey)
	data := h.templates.calls[0].data.(documenttemplate.CustomerStatusEmailContext)
	assert.True(t, data.IsDelayed)
	assert.Equal(t, "Memphis, TN", data.StopSummary)
	assert.Equal(t, "Fri 15 Jan 2027 07:00 CST", data.EstimatedArrival)
}

func TestEvaluate_PodReadyFromTheDocumentOnFile(t *testing.T) {
	h := newHarness()
	sh := h.addShipment(shipment.StatusCompleted,
		stopSpec{stopType: shipment.StopTypePickup, city: "Dallas", state: "TX",
			departure: at(-90000)},
		stopSpec{stopType: shipment.StopTypeDelivery, city: "Memphis", state: "TN",
			arrival: at(-7200), departure: at(-3600)},
	)
	h.notify(sh.CustomerID, func(p *customertracking.Profile) {
		p.NotifyDelivered = true
		p.NotifyPodReady = true
	})
	h.facts.pods = map[pulid.ID]*customertrackingports.PodDocument{
		sh.ID: {ShipmentID: sh.ID, DocumentID: pulid.MustNew("doc_"), FiledAt: testNow - 600},
	}

	result, err := h.service.Evaluate(t.Context(), testTenant, sh.ID)
	require.NoError(t, err)

	assert.Equal(t, 2, result.Delivered)
	assert.NotNil(t, h.notifications.byEvent(customertracking.EventDelivered))
	assert.NotNil(t, h.notifications.byEvent(customertracking.EventPodReady))
	assert.Len(t, h.email.sent, 2)
}

func TestEvaluate_NothingForACustomerWithoutRules(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)

	result, err := h.service.Evaluate(t.Context(), testTenant, sh.ID)
	require.NoError(t, err)

	assert.Zero(t, result.Detected)
	assert.Empty(t, h.notifications.rows)
}

func TestEvaluate_AShipmentThatIsGoneIsNotAnError(t *testing.T) {
	h := newHarness()

	result, err := h.service.Evaluate(t.Context(), testTenant, pulid.MustNew("shp_"))

	require.NoError(t, err)
	assert.Zero(t, result.Detected)
}
