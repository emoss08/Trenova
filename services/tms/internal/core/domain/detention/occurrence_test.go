package detention_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func billableOccurrence() *detention.DetentionOccurrence {
	return &detention.DetentionOccurrence{
		ID:              pulid.MustNew("dto_"),
		ShipmentID:      pulid.MustNew("shp_"),
		StopID:          pulid.MustNew("stp_"),
		Status:          detention.OccurrenceStatusPending,
		BillableAmount:  decimal.RequireFromString("225.00"),
		DriverPayAmount: decimal.RequireFromString("75.00"),
		NetMargin:       decimal.RequireFromString("150.00"),
		Currency:        "USD",
		ClockStartAt:    1000,
	}
}

func TestOccurrenceStatus_Terminal(t *testing.T) {
	t.Parallel()

	terminal := []detention.OccurrenceStatus{
		detention.OccurrenceStatusBilled,
		detention.OccurrenceStatusWaived,
		detention.OccurrenceStatusDisputed,
	}
	nonTerminal := []detention.OccurrenceStatus{
		detention.OccurrenceStatusAccruing,
		detention.OccurrenceStatusPending,
		detention.OccurrenceStatusApproved,
		detention.OccurrenceStatusNotBillable,
	}

	for _, status := range terminal {
		assert.True(t, status.IsTerminal(), "%s must be terminal", status)
	}
	for _, status := range nonTerminal {
		assert.False(t, status.IsTerminal(), "%s must not be terminal", status)
	}
}

func TestOccurrenceStatus_Billable(t *testing.T) {
	t.Parallel()

	billable := []detention.OccurrenceStatus{
		detention.OccurrenceStatusPending,
		detention.OccurrenceStatusApproved,
		detention.OccurrenceStatusBilled,
	}
	notBillable := []detention.OccurrenceStatus{
		detention.OccurrenceStatusAccruing,
		detention.OccurrenceStatusWaived,
		detention.OccurrenceStatusDisputed,
		detention.OccurrenceStatusNotBillable,
	}

	for _, status := range billable {
		assert.True(t, status.IsBillable(), "%s must be billable", status)
	}
	for _, status := range notBillable {
		assert.False(t, status.IsBillable(), "%s must not be billable", status)
	}
}

func TestOccurrence_Waive(t *testing.T) {
	t.Parallel()

	occ := billableOccurrence()

	userID := pulid.MustNew("usr_")
	require.NoError(t, occ.Waive(detention.WaiverReasonWeather, "ice storm", userID, 5000))

	assert.Equal(t, detention.OccurrenceStatusWaived, occ.Status)
	assert.Equal(t, "225.00", occ.WaivedAmount.StringFixed(2),
		"the forgiven amount is preserved so leakage stays measurable")
	assert.True(t, occ.BillableAmount.IsZero())
	assert.Equal(t, "-75.00", occ.NetMargin.StringFixed(2),
		"waiving revenue does not unwind the driver pay already owed")
	require.NotNil(t, occ.WaiverReason)
	assert.Equal(t, detention.WaiverReasonWeather, *occ.WaiverReason)
	assert.Equal(t, &userID, occ.WaivedByID)
}

func TestOccurrence_CannotWaiveBilledCharge(t *testing.T) {
	t.Parallel()

	occ := billableOccurrence()
	occ.Status = detention.OccurrenceStatusBilled

	err := occ.Waive(detention.WaiverReasonOther, "", pulid.MustNew("usr_"), 5000)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "adjustment")
	assert.Equal(t, detention.OccurrenceStatusBilled, occ.Status)
}

func TestOccurrence_CannotWaiveTwice(t *testing.T) {
	t.Parallel()

	occ := billableOccurrence()
	userID := pulid.MustNew("usr_")

	require.NoError(t, occ.Waive(detention.WaiverReasonOther, "", userID, 5000))

	err := occ.Waive(detention.WaiverReasonOther, "", userID, 6000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already waived")
}

func TestOccurrence_Approve(t *testing.T) {
	t.Parallel()

	occ := billableOccurrence()
	occ.RequiresApproval = true

	userID := pulid.MustNew("usr_")
	require.NoError(t, occ.Approve(userID, 5000))

	assert.Equal(t, detention.OccurrenceStatusApproved, occ.Status)
	assert.False(t, occ.RequiresApproval)
	assert.Equal(t, &userID, occ.ApprovedByID)
}

func TestOccurrence_ApproveOnlyFromPending(t *testing.T) {
	t.Parallel()

	for _, status := range []detention.OccurrenceStatus{
		detention.OccurrenceStatusAccruing,
		detention.OccurrenceStatusApproved,
		detention.OccurrenceStatusBilled,
		detention.OccurrenceStatusWaived,
		detention.OccurrenceStatusNotBillable,
	} {
		occ := billableOccurrence()
		occ.Status = status

		assert.Error(t, occ.Approve(pulid.MustNew("usr_"), 5000),
			"approving from %s must be rejected", status)
	}
}

func TestOccurrence_Dispute(t *testing.T) {
	t.Parallel()

	occ := billableOccurrence()

	require.NoError(t, occ.Dispute("customer says driver arrived at 09:00", 5000))

	assert.Equal(t, detention.OccurrenceStatusDisputed, occ.Status)
	assert.Contains(t, occ.DisputeNote, "09:00")
	assert.Equal(t, "225.00", occ.BillableAmount.StringFixed(2),
		"the original computation survives the dispute so the claim can be worked")
}

func TestOccurrence_CannotDisputeWaivedCharge(t *testing.T) {
	t.Parallel()

	occ := billableOccurrence()
	require.NoError(t, occ.Waive(detention.WaiverReasonOther, "", pulid.MustNew("usr_"), 5000))

	assert.Error(t, occ.Dispute("too late", 6000))
}

func TestOccurrence_IsFrozen(t *testing.T) {
	t.Parallel()

	occ := billableOccurrence()
	assert.False(t, occ.IsFrozen())

	occ.Status = detention.OccurrenceStatusBilled
	assert.True(t, occ.IsFrozen(),
		"a billed occurrence must never be silently recomputed")
}

func TestOccurrence_NoticeCountdown(t *testing.T) {
	t.Parallel()

	occ := billableOccurrence()
	assert.Nil(t, occ.MinutesUntilNoticeDue(1000))

	occ.NoticeDueAt = ptr(int64(7000))
	remaining := occ.MinutesUntilNoticeDue(4000)
	require.NotNil(t, remaining)
	assert.Equal(t, int32(50), *remaining)

	overdue := occ.MinutesUntilNoticeDue(10000)
	require.NotNil(t, overdue)
	assert.Negative(t, *overdue)
}

func TestOccurrence_NoticeWindowOpen(t *testing.T) {
	t.Parallel()

	occ := billableOccurrence()
	occ.NotificationStatus = detention.NotificationStatusPending
	occ.NoticeDeadlineAt = ptr(int64(9000))

	assert.True(t, occ.NoticeWindowOpen(8000))
	assert.False(t, occ.NoticeWindowOpen(9001))

	occ.NotificationStatus = detention.NotificationStatusSent
	assert.False(t, occ.NoticeWindowOpen(8000),
		"an already-sent notice does not leave the window open")
}

func TestOccurrence_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(*detention.DetentionOccurrence)
		wantField string
	}{
		{
			name:      "missing shipment",
			mutate:    func(o *detention.DetentionOccurrence) { o.ShipmentID = pulid.Nil },
			wantField: "shipmentId",
		},
		{
			name:      "missing stop",
			mutate:    func(o *detention.DetentionOccurrence) { o.StopID = pulid.Nil },
			wantField: "stopId",
		},
		{
			name: "clock stop precedes clock start",
			mutate: func(o *detention.DetentionOccurrence) {
				o.ClockStartAt = 5000
				o.ClockStopAt = ptr(int64(1000))
			},
			wantField: "clockStopAt",
		},
		{
			name:      "negative billable minutes",
			mutate:    func(o *detention.DetentionOccurrence) { o.BillableMinutes = -1 },
			wantField: "billableMinutes",
		},
		{
			name: "waived without a coded reason",
			mutate: func(o *detention.DetentionOccurrence) {
				o.Status = detention.OccurrenceStatusWaived
			},
			wantField: "waiverReason",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			occ := billableOccurrence()
			tt.mutate(occ)

			multiErr := errortypes.NewMultiError()
			occ.Validate(multiErr)

			require.True(t, multiErr.HasErrors())
			assert.Contains(t, errorFields(multiErr), tt.wantField)
		})
	}
}

func TestNotice_DeliveryTransitions(t *testing.T) {
	t.Parallel()

	deadline := int64(9000)

	t.Run("sent inside the window satisfies the requirement", func(t *testing.T) {
		t.Parallel()
		n := &detention.DetentionNotice{}
		n.MarkSent(8000, &deadline, "msg-1")

		assert.Equal(t, detention.NoticeDeliveryStatusSent, n.DeliveryStatus)
		assert.True(t, n.SatisfiesRequirement)
		assert.Equal(t, "msg-1", n.ProviderMessageID)
	})

	t.Run("sent after the deadline does not", func(t *testing.T) {
		t.Parallel()
		n := &detention.DetentionNotice{}
		n.MarkSent(9001, &deadline, "msg-2")

		assert.False(t, n.SatisfiesRequirement)
	})

	t.Run("no deadline means any send counts", func(t *testing.T) {
		t.Parallel()
		n := &detention.DetentionNotice{}
		n.MarkSent(99999, nil, "msg-3")

		assert.True(t, n.SatisfiesRequirement)
	})

	t.Run("a bounce revokes the satisfaction", func(t *testing.T) {
		t.Parallel()
		n := &detention.DetentionNotice{}
		n.MarkSent(8000, &deadline, "msg-4")
		require.True(t, n.SatisfiesRequirement)

		n.MarkBounced(8100, "mailbox full")

		assert.Equal(t, detention.NoticeDeliveryStatusBounced, n.DeliveryStatus)
		assert.False(t, n.SatisfiesRequirement,
			"a notice the customer never received cannot support the claim")
		assert.Contains(t, n.FailureReason, "mailbox full")
	})

	t.Run("delivery and open are recorded", func(t *testing.T) {
		t.Parallel()
		n := &detention.DetentionNotice{}
		n.MarkSent(8000, &deadline, "msg-5")
		n.MarkDelivered(8010)
		assert.True(t, n.DeliveryStatus.IsDelivered())

		n.MarkOpened(8020)
		assert.Equal(t, detention.NoticeDeliveryStatusOpened, n.DeliveryStatus)
		assert.True(t, n.DeliveryStatus.IsDelivered())
	})
}

func TestNotificationStatus_SatisfiesRequirement(t *testing.T) {
	t.Parallel()

	satisfying := []detention.NotificationStatus{
		detention.NotificationStatusNotRequired,
		detention.NotificationStatusSent,
	}
	failing := []detention.NotificationStatus{
		detention.NotificationStatusPending,
		detention.NotificationStatusLate,
		detention.NotificationStatusMissed,
		detention.NotificationStatusFailed,
	}

	for _, status := range satisfying {
		assert.True(t, status.SatisfiesRequirement(), "%s", status)
	}
	for _, status := range failing {
		assert.False(t, status.SatisfiesRequirement(), "%s", status)
	}
}
