package detention_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOccurrence_HoldsBillingOnlyWhenPendingAndAwaitingApproval(t *testing.T) {
	t.Parallel()

	statuses := []detention.OccurrenceStatus{
		detention.OccurrenceStatusAccruing,
		detention.OccurrenceStatusPending,
		detention.OccurrenceStatusApproved,
		detention.OccurrenceStatusBilled,
		detention.OccurrenceStatusWaived,
		detention.OccurrenceStatusDisputed,
		detention.OccurrenceStatusNotBillable,
	}

	for _, status := range statuses {
		for _, requiresApproval := range []bool{false, true} {
			occurrence := billableOccurrence()
			occurrence.Status = status
			occurrence.RequiresApproval = requiresApproval

			want := status == detention.OccurrenceStatusPending && requiresApproval
			assert.Equal(t, want, occurrence.HoldsBilling(),
				"status %s, requires approval %t", status, requiresApproval)
		}
	}
}

func TestOccurrence_PendingWithoutApprovalKeepsBilling(t *testing.T) {
	t.Parallel()

	occurrence := billableOccurrence()
	occurrence.RequiresApproval = false

	assert.False(t, occurrence.HoldsBilling())
}

func TestOccurrence_ApprovingReleasesTheHold(t *testing.T) {
	t.Parallel()

	occurrence := billableOccurrence()
	occurrence.RequiresApproval = true
	require.True(t, occurrence.HoldsBilling())

	require.NoError(t, occurrence.Approve(pulid.MustNew("usr_"), 2000))
	assert.False(t, occurrence.HoldsBilling())
}

func TestOccurrence_BillingHoldReason(t *testing.T) {
	t.Parallel()

	threshold := billableOccurrence()
	threshold.RequiresApproval = true
	threshold.PolicySnapshot = &detention.PolicySnapshot{
		RequireApprovalOverAmount: decimal.NewNullDecimal(decimal.NewFromInt(200)),
	}
	assert.Equal(
		t,
		detention.BillingHoldReasonOverApprovalThreshold,
		threshold.BillingHoldReason(),
	)

	notice := billableOccurrence()
	notice.RequiresApproval = true
	notice.NotificationStatus = detention.NotificationStatusMissed
	notice.PolicySnapshot = &detention.PolicySnapshot{
		NotificationRequirement:   detention.NotificationRequirementRequired,
		UnnotifiedBehavior:        detention.UnnotifiedBehaviorFlag,
		RequireApprovalOverAmount: decimal.NewNullDecimal(decimal.NewFromInt(500)),
	}
	assert.Equal(t, detention.BillingHoldReasonNoticeNotSent, notice.BillingHoldReason())

	escalated := billableOccurrence()
	escalated.RequiresApproval = true
	escalated.NotificationStatus = detention.NotificationStatusSent
	escalated.PolicySnapshot = &detention.PolicySnapshot{
		NotificationRequirement: detention.NotificationRequirementRequired,
		UnnotifiedBehavior:      detention.UnnotifiedBehaviorFlag,
	}
	assert.Equal(t, detention.BillingHoldReasonEscalated, escalated.BillingHoldReason())

	assert.Equal(
		t,
		detention.BillingHoldReasonEscalated,
		billableOccurrence().BillingHoldReason(),
	)
}

func TestOccurrence_MarkBilledFromPendingOrApproved(t *testing.T) {
	t.Parallel()

	for _, status := range []detention.OccurrenceStatus{
		detention.OccurrenceStatusPending,
		detention.OccurrenceStatusApproved,
	} {
		occurrence := billableOccurrence()
		occurrence.Status = status

		require.NoError(t, occurrence.MarkBilled(), status)
		assert.Equal(t, detention.OccurrenceStatusBilled, occurrence.Status)
		assert.True(t, occurrence.IsFrozen())
	}

	for _, status := range []detention.OccurrenceStatus{
		detention.OccurrenceStatusAccruing,
		detention.OccurrenceStatusBilled,
		detention.OccurrenceStatusWaived,
		detention.OccurrenceStatusDisputed,
		detention.OccurrenceStatusNotBillable,
	} {
		occurrence := billableOccurrence()
		occurrence.Status = status

		require.Error(t, occurrence.MarkBilled(), status)
		assert.Equal(t, status, occurrence.Status)
	}
}

func TestOccurrence_ReleaseBillingReturnsToApprovedAndKeepsTheApprover(t *testing.T) {
	t.Parallel()

	approver := pulid.MustNew("usr_")
	approvedAt := int64(1500)
	occurrence := billableOccurrence()
	occurrence.Status = detention.OccurrenceStatusBilled
	occurrence.ApprovedByID = &approver
	occurrence.ApprovedAt = &approvedAt

	require.NoError(t, occurrence.ReleaseBilling())
	assert.Equal(t, detention.OccurrenceStatusApproved, occurrence.Status)
	require.NotNil(t, occurrence.ApprovedByID)
	assert.Equal(t, approver, *occurrence.ApprovedByID)
	assert.Equal(t, &approvedAt, occurrence.ApprovedAt)
	assert.False(t, occurrence.IsFrozen())

	require.Error(t, billableOccurrence().ReleaseBilling())
}

func TestOccurrence_FollowInvoicesMovesOnlyABillableCharge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status   detention.OccurrenceStatus
		invoiced bool
		want     detention.OccurrenceStatus
		change   detention.InvoiceBillingChange
	}{
		{
			detention.OccurrenceStatusAccruing, true,
			detention.OccurrenceStatusAccruing, detention.InvoiceBillingUnchanged,
		},
		{
			detention.OccurrenceStatusAccruing, false,
			detention.OccurrenceStatusAccruing, detention.InvoiceBillingUnchanged,
		},
		{
			detention.OccurrenceStatusPending, true,
			detention.OccurrenceStatusBilled, detention.InvoiceBillingMarked,
		},
		{
			detention.OccurrenceStatusPending, false,
			detention.OccurrenceStatusPending, detention.InvoiceBillingUnchanged,
		},
		{
			detention.OccurrenceStatusApproved, true,
			detention.OccurrenceStatusBilled, detention.InvoiceBillingMarked,
		},
		{
			detention.OccurrenceStatusApproved, false,
			detention.OccurrenceStatusApproved, detention.InvoiceBillingUnchanged,
		},
		{
			detention.OccurrenceStatusBilled, true,
			detention.OccurrenceStatusBilled, detention.InvoiceBillingUnchanged,
		},
		{
			detention.OccurrenceStatusBilled, false,
			detention.OccurrenceStatusApproved, detention.InvoiceBillingReleased,
		},
		{
			detention.OccurrenceStatusWaived, true,
			detention.OccurrenceStatusWaived, detention.InvoiceBillingUnchanged,
		},
		{
			detention.OccurrenceStatusWaived, false,
			detention.OccurrenceStatusWaived, detention.InvoiceBillingUnchanged,
		},
		{
			detention.OccurrenceStatusDisputed, true,
			detention.OccurrenceStatusDisputed, detention.InvoiceBillingUnchanged,
		},
		{
			detention.OccurrenceStatusDisputed, false,
			detention.OccurrenceStatusDisputed, detention.InvoiceBillingUnchanged,
		},
		{
			detention.OccurrenceStatusNotBillable, true,
			detention.OccurrenceStatusNotBillable, detention.InvoiceBillingUnchanged,
		},
		{
			detention.OccurrenceStatusNotBillable, false,
			detention.OccurrenceStatusNotBillable, detention.InvoiceBillingUnchanged,
		},
	}

	for _, tc := range cases {
		occurrence := billableOccurrence()
		occurrence.Status = tc.status
		before := *occurrence

		change, err := occurrence.FollowInvoices(tc.invoiced)
		require.NoError(t, err, "%s invoiced %t", tc.status, tc.invoiced)
		assert.Equal(t, tc.change, change, "%s invoiced %t", tc.status, tc.invoiced)
		assert.Equal(t, tc.want, occurrence.Status, "%s invoiced %t", tc.status, tc.invoiced)
		assert.True(t, before.BillableAmount.Equal(occurrence.BillableAmount))

		again, err := occurrence.FollowInvoices(tc.invoiced)
		require.NoError(t, err)
		assert.Equal(t, detention.InvoiceBillingUnchanged, again,
			"%s invoiced %t must settle in one step", tc.status, tc.invoiced)
		assert.Equal(t, tc.want, occurrence.Status)
	}
}

func TestOccurrence_StandingByABilledChargeKeepsTheDisputeUntilItIsWaived(t *testing.T) {
	t.Parallel()

	occurrence := billableOccurrence()
	require.NoError(t, occurrence.MarkBilled())
	require.NoError(t, occurrence.Dispute("Standing by the gate record", 2000))
	assert.True(t, occurrence.IsFrozen())

	for _, invoiced := range []bool{true, false, true} {
		change, err := occurrence.FollowInvoices(invoiced)
		require.NoError(t, err)
		assert.Equal(t, detention.InvoiceBillingUnchanged, change)
		assert.Equal(t, detention.OccurrenceStatusDisputed, occurrence.Status)
		assert.Equal(t, "Standing by the gate record", occurrence.DisputeNote)
	}

	require.NoError(t, occurrence.Waive(
		detention.WaiverReasonCustomerGoodwill, "Credited", pulid.MustNew("usr_"), 3000,
	))
	assert.Equal(t, detention.OccurrenceStatusWaived, occurrence.Status)
	assert.True(t, occurrence.WaivedAmount.Equal(decimal.RequireFromString("225.00")))

	change, err := occurrence.FollowInvoices(true)
	require.NoError(t, err)
	assert.Equal(t, detention.InvoiceBillingUnchanged, change)
	assert.Equal(t, detention.OccurrenceStatusWaived, occurrence.Status)
	require.Error(t, occurrence.MarkBilled())
	require.Error(t, occurrence.Dispute("again", 4000))
}

func TestOccurrenceStatus_InvoiceSettledStatuses(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []detention.OccurrenceStatus{
		detention.OccurrenceStatusPending,
		detention.OccurrenceStatusApproved,
		detention.OccurrenceStatusBilled,
		detention.OccurrenceStatusDisputed,
	}, detention.InvoiceSettledStatuses())
}
