package detentionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func billableOccurrence(amount string) *detention.DetentionOccurrence {
	return &detention.DetentionOccurrence{
		Status:             detention.OccurrenceStatusPending,
		BillableAmount:     decimal.RequireFromString(amount),
		Currency:           "USD",
		StopType:           shipment.StopTypeDelivery,
		RawDwellMinutes:    270,
		FreeMinutesGranted: 120,
		RoundedMinutes:     150,
	}
}

func TestChargeCommentKindFor_OnlyNarratesRealMovement(t *testing.T) {
	t.Parallel()

	_, ok := chargeCommentKindFor(nil, nil)
	require.False(t, ok, "nothing to say about an occurrence that does not exist")

	accruing := &detention.DetentionOccurrence{
		Status:         detention.OccurrenceStatusAccruing,
		BillableAmount: decimal.NewFromInt(75),
	}
	_, ok = chargeCommentKindFor(accruing, nil)
	require.False(t, ok, "an accruing occurrence carries no charge yet")

	kind, ok := chargeCommentKindFor(billableOccurrence("150"), nil)
	require.True(t, ok)
	require.Equal(t, chargeCommentAdded, kind)

	kind, ok = chargeCommentKindFor(billableOccurrence("180"), billableOccurrence("150"))
	require.True(t, ok)
	require.Equal(t, chargeCommentRevised, kind)

	_, ok = chargeCommentKindFor(billableOccurrence("150"), billableOccurrence("150"))
	require.False(t, ok, "a recalculation landing on the same number is noise")

	waived := billableOccurrence("0")
	waived.Status = detention.OccurrenceStatusWaived
	kind, ok = chargeCommentKindFor(waived, billableOccurrence("150"))
	require.True(t, ok)
	require.Equal(t, chargeCommentRemoved, kind)
}

func TestBuildChargeComment_StatesTheDerivationAndTheRisk(t *testing.T) {
	t.Parallel()

	occurrence := billableOccurrence("150")
	occurrence.SuppressedByGate = true
	occurrence.ArrivedLate = true
	occurrence.LateByMinutes = 45
	occurrence.CapApplied = detention.CapKindPerStopAmount
	occurrence.PolicySnapshot = &detention.PolicySnapshot{
		PolicyName:   "Grocery DC Detention",
		PolicyCode:   "GRC-DET",
		RateSource:   detention.RateSourceAccessorial,
		FlatRate:     decimal.NewFromInt(60),
		FlatRateUnit: detention.TierRateUnitHour,
	}

	body := buildChargeComment(chargeCommentAdded, chargeCommentParams{
		occurrence: occurrence,
		stop:       &shipment.Stop{AddressLine: "1200 W Fulton St, Chicago IL"},
	})

	require.Contains(t, body, "Detention charge added — 150.00 USD")
	require.Contains(t, body, "Delivery at 1200 W Fulton St, Chicago IL")
	require.Contains(t, body, "4h 30m on site · 2h free · 2h 30m billable")
	require.Contains(t, body, "Grocery DC Detention (GRC-DET)")
	require.Contains(t, body, "Driver arrived 45 min late")
	require.Contains(t, body, "per-stop ceiling")
	require.Contains(t, body, "the customer can refuse this charge")
}

func TestBuildChargeComment_RemovalExplainsItself(t *testing.T) {
	t.Parallel()

	removed := billableOccurrence("0")
	removed.Status = detention.OccurrenceStatusNotBillable

	body := buildChargeComment(chargeCommentRemoved, chargeCommentParams{
		occurrence: removed,
		previous:   billableOccurrence("150"),
	})

	require.Contains(t, body, "Detention charge removed (was 150.00 USD)")
	require.Contains(t, body, "no longer carries a detention charge")
}

func TestChargeCommentPriority_RaisesOnlyWhatNeedsAction(t *testing.T) {
	t.Parallel()

	clean := billableOccurrence("150")
	require.Equal(
		t,
		shipment.CommentPriorityNormal,
		chargeCommentPriority(chargeCommentAdded, clean),
	)

	atRisk := billableOccurrence("150")
	atRisk.SuppressedByGate = true
	require.Equal(
		t,
		shipment.CommentPriorityHigh,
		chargeCommentPriority(chargeCommentAdded, atRisk),
	)

	require.Equal(
		t,
		shipment.CommentPriorityNormal,
		chargeCommentPriority(chargeCommentRemoved, atRisk),
	)
}
