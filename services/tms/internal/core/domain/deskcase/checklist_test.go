package deskcase_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func item(t *testing.T, checklist *deskcase.Checklist, key deskcase.ItemKey) *deskcase.Item {
	t.Helper()

	for _, candidate := range checklist.Items {
		if candidate.Key == key {
			return candidate
		}
	}
	require.Failf(t, "missing item", "%s is not on the checklist", key)

	return nil
}

func readyShipment() *deskcase.ShipmentFacts {
	return &deskcase.ShipmentFacts{
		Status:      shipment.StatusCompleted,
		DeliveredAt: ptr(now - 3600),
		PODCode:     "POD",
		Requirements: []deskcase.Requirement{
			{Code: "POD", Name: "Proof of delivery", Satisfied: true},
			{Code: "BOL", Name: "Bill of lading", Satisfied: true},
		},
		RateCons:           []deskcase.RateCon{{CarrierName: "Swift", Confirmed: true}},
		CustomerNotifiedAt: ptr(now - 600),
	}
}

func TestReadyToBill_AllClearOffersToMarkReady(t *testing.T) {
	t.Parallel()

	got := deskcase.ReadyToBill(readyShipment())

	assert.Equal(t, deskcase.ChecklistReadyToBill, got.Kind)
	assert.True(t, got.Ready)
	assert.Equal(t, deskcase.StepMarkReady, got.Next)
	for _, entry := range got.Items {
		assert.Contains(t, []deskcase.ItemState{deskcase.ItemDone, deskcase.ItemNotNeeded}, entry.State, entry.Key)
	}
}

func TestReadyToBill_WithBillingTheNextStepIsToSend(t *testing.T) {
	t.Parallel()

	facts := readyShipment()
	facts.InBillingQueue = true

	assert.Equal(t, deskcase.StepSendInvoice, deskcase.ReadyToBill(facts).Next)
}

func TestReadyToBill_MissingPODBlocksAndAsksForIt(t *testing.T) {
	t.Parallel()

	facts := readyShipment()
	facts.Requirements[0].Satisfied = false

	got := deskcase.ReadyToBill(facts)

	assert.False(t, got.Ready)
	assert.Equal(t, deskcase.StepRequestPOD, got.Next)
	assert.Equal(t, deskcase.ItemBlocked, item(t, got, deskcase.ItemPOD).State)
	assert.Equal(t, deskcase.ItemDone, item(t, got, deskcase.ItemPaperwork).State)
}

func TestReadyToBill_PODNotRequiredButOnFileCounts(t *testing.T) {
	t.Parallel()

	facts := readyShipment()
	facts.Requirements = nil
	assert.Equal(t, deskcase.ItemNotNeeded, item(t, deskcase.ReadyToBill(facts), deskcase.ItemPOD).State)

	facts.PODOnFile = true
	assert.Equal(t, deskcase.ItemDone, item(t, deskcase.ReadyToBill(facts), deskcase.ItemPOD).State)
}

func TestReadyToBill_BeforeDeliveryItemsWaitOnTheWorld(t *testing.T) {
	t.Parallel()

	facts := readyShipment()
	facts.Status = shipment.StatusInTransit
	facts.DeliveredAt = nil
	facts.Requirements[0].Satisfied = false
	facts.CustomerNotifiedAt = nil

	got := deskcase.ReadyToBill(facts)

	assert.False(t, got.Ready)
	assert.Equal(t, deskcase.ItemPending, item(t, got, deskcase.ItemDelivered).State)
	assert.Equal(t, deskcase.ItemPending, item(t, got, deskcase.ItemPOD).State)
	assert.Equal(t, deskcase.ItemPending, item(t, got, deskcase.ItemCustomerNotified).State)
	assert.Equal(t, deskcase.StepTrackDelivery, got.Next,
		"with nothing blocked, the first step on a pending item is offered")
}

func TestReadyToBill_RateIssuesAndUnconfirmedRateCons(t *testing.T) {
	t.Parallel()

	facts := readyShipment()
	facts.Validations = []deskcase.Validation{{Code: "rate_variance_requires_action"}}
	facts.OpenChargeIssues = []string{"ChargeOverRateCon"}

	rate := item(t, deskcase.ReadyToBill(facts), deskcase.ItemRateConfirmation)
	assert.Equal(t, deskcase.ItemBlocked, rate.State)
	assert.Equal(t, deskcase.StepReviewRate, rate.Step)
	assert.Equal(t, []string{"rate_variance_requires_action", "ChargeOverRateCon"}, rate.Codes)

	facts = readyShipment()
	facts.RateCons = append(facts.RateCons, deskcase.RateCon{CarrierName: "Knight"})
	checklist := deskcase.ReadyToBill(facts)
	assert.Equal(t, deskcase.ItemDone, item(t, checklist, deskcase.ItemRateConfirmation).State)
	carrier := item(t, checklist, deskcase.ItemCarrierRateCon)
	assert.Equal(t, deskcase.StepConfirmRate, carrier.Step)
	assert.Equal(t, []string{"Knight"}, carrier.Names)

	facts.RateCons = nil
	assert.Equal(t, deskcase.ItemNotNeeded,
		item(t, deskcase.ReadyToBill(facts), deskcase.ItemCarrierRateCon).State,
		"a shipment no carrier hauls needs no carrier rate confirmation")
}

func TestReadyToBill_HoldsAndPaperworkSplitTheValidations(t *testing.T) {
	t.Parallel()

	facts := readyShipment()
	facts.Validations = []deskcase.Validation{{Code: "missing_bol"}, {Code: "credit_hold"}}

	got := deskcase.ReadyToBill(facts)

	assert.Equal(t, []string{"missing_bol"}, item(t, got, deskcase.ItemPaperwork).Codes)
	assert.Equal(t, []string{"credit_hold"}, item(t, got, deskcase.ItemBillingHolds).Codes)
	assert.Equal(t, deskcase.StepRequestPaperwork, got.Next)
}

func TestReadyToBill_Accessorials(t *testing.T) {
	t.Parallel()

	facts := readyShipment()
	facts.DetentionAccruing = 1
	assert.Equal(t, deskcase.ItemPending, item(t, deskcase.ReadyToBill(facts), deskcase.ItemAccessorials).State)

	facts.DetentionUnapproved = 2
	accessorials := item(t, deskcase.ReadyToBill(facts), deskcase.ItemAccessorials)
	assert.Equal(t, deskcase.ItemBlocked, accessorials.State)
	assert.Equal(t, 2, accessorials.Count)
}

func TestReadyToBill_AnUpdateBeforeDeliveryDidNotNotify(t *testing.T) {
	t.Parallel()

	facts := readyShipment()
	facts.CustomerNotifiedAt = ptr(*facts.DeliveredAt - 1)

	notified := item(t, deskcase.ReadyToBill(facts), deskcase.ItemCustomerNotified)
	assert.Equal(t, deskcase.ItemBlocked, notified.State)
	assert.Equal(t, deskcase.StepNotifyCustomer, notified.Step)
}

func TestReadyToClose(t *testing.T) {
	t.Parallel()

	facts := &deskcase.InvoiceFacts{
		Status:           invoice.StatusDraft,
		SendStatus:       invoice.SendStatusNotSent,
		DisputeStatus:    invoice.DisputeStatusNone,
		SettlementStatus: invoice.SettlementStatusUnpaid,
		DueDate:          ptr(now + 86400),
		Now:              now,
	}
	assert.Equal(t, deskcase.StepPostInvoice, deskcase.ReadyToClose(facts).Next)

	facts.Status = invoice.StatusPosted
	assert.Equal(t, deskcase.StepSendInvoice, deskcase.ReadyToClose(facts).Next)

	facts.SendStatus = invoice.SendStatusSent
	facts.DisputeStatus = invoice.DisputeStatusDisputed
	assert.Equal(t, deskcase.StepWorkDispute, deskcase.ReadyToClose(facts).Next)

	facts.DisputeStatus = invoice.DisputeStatusNone
	waiting := deskcase.ReadyToClose(facts)
	assert.False(t, waiting.Ready)
	assert.Empty(t, waiting.Next, "payment not yet due is nothing to do")
	assert.Equal(t, deskcase.ItemPending, item(t, waiting, deskcase.ItemPaid).State)

	facts.Now = now + 2*86400
	assert.Equal(t, deskcase.StepFollowUpPayment, deskcase.ReadyToClose(facts).Next)

	facts.SettlementStatus = invoice.SettlementStatusPaid
	paid := deskcase.ReadyToClose(facts)
	assert.True(t, paid.Ready)
	assert.Empty(t, paid.Next)
}
