package billingqueue

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func nullMoney(v string) decimal.NullDecimal { return decimal.NewNullDecimal(dec(v)) }

func reviewItem(status Status, withBiller bool) *BillingQueueItem {
	item := &BillingQueueItem{ID: pulid.MustNew("bqi_"), Status: status}
	if withBiller {
		biller := pulid.MustNew("usr_")
		item.AssignedBillerID = &biller
	}

	return item
}

func checkKeys(review *Review) []CheckKey {
	keys := make([]CheckKey, 0, len(review.Checks))
	for _, check := range review.Checks {
		keys = append(keys, check.Key)
	}

	return keys
}

func TestBuildReadinessAlwaysReadsTheFiveChecksInOrder(t *testing.T) {
	t.Parallel()

	review := BuildReadiness(reviewItem(StatusReadyForReview, true), nil, CheckFacts{BillerName: "Avery Lane"})

	assert.Equal(t, AllCheckKeys(), checkKeys(review))
	assert.Equal(t, 0, review.NeedsCount)
	assert.True(t, review.Ready)
	assert.Equal(t, "Avery Lane", review.Checks[0].Detail)
}

func TestBuildReadinessNeedsABillerBeforeApproval(t *testing.T) {
	t.Parallel()

	review := BuildReadiness(reviewItem(StatusReadyForReview, false), nil, CheckFacts{})

	assert.Equal(t, CheckStateFail, review.Checks[0].State)
	assert.Equal(t, CheckCodeUnassigned, review.Checks[0].Code)
	assert.Equal(t, 1, review.NeedsCount)
	assert.Equal(t, BlockerBiller, review.Blocker)
	assert.False(t, review.Ready)
}

// An open issue holds its check, a settled one says how it was settled, and
// one the checks themselves cleared leaves the check reading as if it never
// was raised.
func TestBuildReadinessReadsTheChecksFromTheirIssues(t *testing.T) {
	t.Parallel()

	keep := "keep"
	cleared := ResolutionCleared
	settledAt := int64(200)
	open := &Issue{
		ID: pulid.MustNew("bqis_"), CheckKey: CheckCharges, Summary: "$110.00 lumper fee isn't on the rate con",
		Options: []IssueOption{{Key: "keep"}, {Key: "drop"}},
	}
	settled := &Issue{
		ID: pulid.MustNew("bqis_"), CheckKey: CheckPOD, ResolutionKey: &keep, ResolutionText: "Billing without a signed POD",
		ResolvedAt: &settledAt, Options: []IssueOption{{Key: "keep"}},
	}
	clearedIssue := &Issue{
		ID: pulid.MustNew("bqis_"), CheckKey: CheckDuplicate, ResolutionKey: &cleared, ResolvedAt: &settledAt,
	}
	creditHold := &Issue{ID: pulid.MustNew("bqis_"), CheckKey: CheckTerms, Summary: "Acme is on credit hold"}

	review := BuildReadiness(
		reviewItem(StatusInReview, true),
		[]*Issue{open, settled, clearedIssue, creditHold},
		CheckFacts{ShipmentRef: "S-1001"},
	)

	byKey := map[CheckKey]*Check{}
	for _, check := range review.Checks {
		byKey[check.Key] = check
	}
	assert.Equal(t, CheckStateWarn, byKey[CheckCharges].State)
	assert.Equal(t, open.ID, byKey[CheckCharges].IssueID)
	assert.Equal(t, CheckStateOK, byKey[CheckPOD].State)
	assert.Equal(t, CheckCodeResolved, byKey[CheckPOD].Code)
	assert.Equal(t, "Billing without a signed POD", byKey[CheckPOD].Detail)
	assert.Equal(t, CheckCodeUnique, byKey[CheckDuplicate].Code)
	assert.Equal(t, CheckStateFail, byKey[CheckTerms].State, "an issue with nothing to choose is a fail")

	assert.Equal(t, 2, review.NeedsCount)
	assert.Equal(t, BlockerIssue, review.Blocker)
	assert.False(t, review.Ready)
}

func TestBuildReadinessNeverCallsAHeldOrSettledItemReady(t *testing.T) {
	t.Parallel()

	held := BuildReadiness(reviewItem(StatusOnHold, true), nil, CheckFacts{})
	assert.Equal(t, BlockerHold, held.Blocker)
	assert.False(t, held.Ready)

	approved := BuildReadiness(reviewItem(StatusApproved, true), nil, CheckFacts{})
	assert.Equal(t, BlockerStatus, approved.Blocker)
	assert.False(t, approved.Ready)
}

func chargesWith(lines ...*ChargeLine) *ChargeReview {
	review := &ChargeReview{Lines: lines}
	review.Totals()

	return review
}

func TestFindingsFlagAChargeTheRateConDoesNotPrice(t *testing.T) {
	t.Parallel()

	lumper := pulid.MustNew("ac_")
	charges := chargesWith(
		&ChargeLine{Key: "freight", Label: "Linehaul", Billed: dec("1945.65"), Expected: nullMoney("1945.65")},
		&ChargeLine{Key: lumper.String(), AdditionalChargeID: lumper, Label: "Lumper fee", Billed: dec("110")},
	)

	findings := Findings(FindingInput{CustomerName: "Acme", Charges: charges})

	require.Len(t, findings, 1)
	issue := findings[0]
	assert.Equal(t, IssueAccessorialNotOnRateCon, issue.Code)
	assert.Equal(t, CheckCharges, issue.CheckKey)
	assert.Equal(t, "$110.00 lumper fee isn't on the rate con", issue.Summary)
	assert.Equal(t, lumper.String(), issue.SubjectKey)
	require.Len(t, issue.Options, 2)
	assert.Equal(t, "Keep it · bill $110.00", issue.Options[0].Label)
	assert.Equal(t, EffectKeep, issue.Options[0].Effect.Kind)
	assert.Equal(t, "Remove it", issue.Options[1].Label)
	assert.Equal(t, EffectDrop, issue.Options[1].Effect.Kind)
	assert.Equal(t, lumper, issue.Options[1].Effect.ChargeID)

	assert.True(t, dec("110").Equal(charges.Difference), "the off-contract charge is the overage")
}

// Without a rate con there is nothing to hold a charge against, so nothing is
// flagged; flagging every charge would only teach people to ignore the check.
func TestFindingsFlagNothingWhenNothingPricedTheLoad(t *testing.T) {
	t.Parallel()

	charge := pulid.MustNew("ac_")
	findings := Findings(FindingInput{Charges: chargesWith(
		&ChargeLine{Key: "freight", Label: "Freight", Billed: dec("900")},
		&ChargeLine{Key: charge.String(), AdditionalChargeID: charge, Label: "Tarp", Billed: dec("75")},
	)})

	assert.Empty(t, findings)
}

func TestFindingsOfferTheRateConPriceForAnOvercharge(t *testing.T) {
	t.Parallel()

	stop := pulid.MustNew("ac_")
	findings := Findings(FindingInput{Charges: chargesWith(
		&ChargeLine{Key: stop.String(), AdditionalChargeID: stop, Label: "Stop-off", Billed: dec("95"), Expected: nullMoney("75")},
	)})

	require.Len(t, findings, 1)
	assert.Equal(t, IssueChargeOverRateCon, findings[0].Code)
	assert.Equal(t, "Stop-off is $20.00 over the rate con", findings[0].Summary)
	assert.Equal(t, EffectSet, findings[0].Options[0].Effect.Kind)
	assert.True(t, dec("75").Equal(findings[0].Options[0].Effect.Amount.Decimal))
}

func TestFindingsAskForASignedPODOnlyWhenOneIsKnownMissingOrUnsigned(t *testing.T) {
	t.Parallel()

	driver := pulid.MustNew("wrk_")
	doc := pulid.MustNew("doc_")
	cases := []struct {
		name string
		pod  *PODState
		code IssueCode
	}{
		{"not required", &PODState{Required: false}, ""},
		{"signed", &PODState{Required: true, Present: true, Signed: true}, ""},
		{"signature not read yet", &PODState{Required: true, Present: true}, ""},
		{"missing", &PODState{Required: true, DriverID: driver, DriverName: "Dana Ortiz"}, IssuePODMissing},
		{"unsigned", &PODState{Required: true, Present: true, Unsigned: true, DocumentID: doc, DriverID: driver, DriverName: "Dana Ortiz"}, IssuePODUnsigned},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			findings := Findings(FindingInput{CustomerName: "Acme", POD: tc.pod})
			if tc.code == "" {
				assert.Empty(t, findings)
				return
			}
			require.Len(t, findings, 1)
			assert.Equal(t, tc.code, findings[0].Code)
			assert.Equal(t, CheckPOD, findings[0].CheckKey)
			assert.Equal(t, "Ask the driver for it", findings[0].Options[0].Label)
			assert.Equal(t, EffectRequest, findings[0].Options[0].Effect.Kind)
			assert.Equal(t, "Bill without it", findings[0].Options[1].Label)
			assert.Contains(t, findings[0].Reasoning, "Dana Ortiz can re-send it from the driver app.")
		})
	}
}

func TestFindingsOnlyOfferToAskWhenThereIsADriverToAsk(t *testing.T) {
	t.Parallel()

	findings := Findings(FindingInput{POD: &PODState{Required: true}})

	require.Len(t, findings, 1)
	require.Len(t, findings[0].Options, 1)
	assert.Equal(t, EffectAccept, findings[0].Options[0].Effect.Kind)
}

func TestFindingsSetDetentionAgainstTheELD(t *testing.T) {
	t.Parallel()

	charge := pulid.MustNew("ac_")
	occurrence := pulid.MustNew("dto_")
	dwell := &DetentionDwell{
		OccurrenceID:  occurrence,
		ChargeID:      charge,
		LoggedMinutes: 210,
		ELDMinutes:    150,
		FreeMinutes:   120,
		HourlyRate:    dec("65"),
		BilledAmount:  dec("97.50"),
		ELDAmount:     dec("32.50"),
	}

	findings := Findings(FindingInput{Detention: []*DetentionDwell{dwell}})

	require.Len(t, findings, 1)
	issue := findings[0]
	assert.Equal(t, IssueDetentionELDMismatch, issue.Code)
	assert.Equal(t, "Detention doesn't match the ELD", issue.Summary)
	assert.Equal(t,
		"The driver logged 3.5 h at the dock, but the ELD shows 2 h 30 m — that's 0.5 h billable, not 1.5 h.",
		issue.Reasoning)
	assert.Equal(t, "Bill ELD time · $32.50", issue.Options[0].Label)
	assert.Equal(t, "0.5 h × $65.00 · from ELD", issue.Options[0].Effect.Basis)
	assert.Equal(t, "Remove detention", issue.Options[1].Label)
}

func TestFindingsLeaveDetentionAloneWithinTheTolerance(t *testing.T) {
	t.Parallel()

	dwell := &DetentionDwell{
		OccurrenceID: pulid.MustNew("dto_"), ChargeID: pulid.MustNew("ac_"),
		LoggedMinutes: 180, ELDMinutes: 172, FreeMinutes: 120,
		HourlyRate: dec("65"), BilledAmount: dec("65"), ELDAmount: dec("56.33"),
	}

	assert.Empty(t, Findings(FindingInput{Detention: []*DetentionDwell{dwell}}))
}

func TestFindingsNameTheOtherBillForAShipment(t *testing.T) {
	t.Parallel()

	dup := &DuplicateRef{Kind: "invoice", ID: pulid.MustNew("inv_"), Number: "INV-24090"}
	findings := Findings(FindingInput{CustomerName: "Acme", ShipmentRef: "S-1001", Duplicates: []*DuplicateRef{dup}})

	require.Len(t, findings, 1)
	assert.Equal(t, CheckDuplicate, findings[0].CheckKey)
	assert.Equal(t, "Invoice INV-24090 already bills this shipment", findings[0].Summary)
	assert.Equal(t, dup.ID.String(), findings[0].SubjectKey)
}

// The same input finds the same issues under the same keys, which is what
// lets the sync run on every read without raising anything twice.
func TestFindingsAreDeterministic(t *testing.T) {
	t.Parallel()

	lumper := pulid.MustNew("ac_")
	in := FindingInput{
		CustomerName: "Acme",
		Charges: chargesWith(
			&ChargeLine{Key: "freight", Billed: dec("100"), Expected: nullMoney("100")},
			&ChargeLine{Key: lumper.String(), AdditionalChargeID: lumper, Label: "Lumper fee", Billed: dec("110")},
		),
		POD: &PODState{Required: true},
	}

	first, second := Findings(in), Findings(in)
	require.Len(t, second, len(first))
	for i := range first {
		assert.Equal(t, first[i].FindingKey(), second[i].FindingKey())
		assert.Equal(t, first[i].Summary, second[i].Summary)
	}
}

func TestDraftInvoiceNumber(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "INV-D-24101", DraftInvoiceNumber("INV-24101"))
	assert.Equal(t, "INV-D-2026-0042", DraftInvoiceNumber("INV-2026-0042"))
	assert.Equal(t, "D-24101", DraftInvoiceNumber("24101"))
	assert.Empty(t, DraftInvoiceNumber(""))
}

func TestFormatMoney(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "$1,945.65", FormatMoney(dec("1945.65")))
	assert.Equal(t, "$110.00", FormatMoney(dec("110")))
	assert.Equal(t, "$1,234,567.80", FormatMoney(dec("1234567.8")))
	assert.Equal(t, "−$20.00", FormatMoney(dec("-20")))
}

func TestHoldReasonCodes(t *testing.T) {
	t.Parallel()

	for _, code := range AllHoldReasonCodes() {
		assert.True(t, code.IsValid())
	}
	assert.False(t, HoldReasonCode("Lunch").IsValid())
	assert.Equal(t, "waiting on paperwork", HoldReasonWaitingOnPaperwork.Phrase())
}
