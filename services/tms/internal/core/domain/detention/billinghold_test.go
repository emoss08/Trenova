package detention_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/detention"
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
