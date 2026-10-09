package deskcase_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const now = int64(1_800_000_000)

func ptr(v int64) *int64 { return &v }

func openShipment() *deskcase.Record {
	return &deskcase.Record{Type: agent.SubjectShipment, ID: pulid.MustNew("shp_"), Status: "InTransit"}
}

func TestResolve_SettledWinsOverEverything(t *testing.T) {
	t.Parallel()

	record := openShipment()
	record.Closed = true
	record.ClosedAs = "Invoiced"

	got := deskcase.Resolve(&deskcase.Inputs{
		Record: record,
		Waits:  []deskcase.OpenWait{{Party: deskcase.WaitingOnCarrier}},
		Snooze: deskcase.Snooze{Until: ptr(now + 3600), Anchor: deskcase.SnoozeTime},
		Now:    now,
	})

	assert.Equal(t, deskcase.StateSettled, got.State)
	assert.Nil(t, got.SnoozedUntil)
	assert.Equal(t, 1, got.OpenWaits)
}

func TestResolve_SnoozeHoldsUntilItEnds(t *testing.T) {
	t.Parallel()

	in := &deskcase.Inputs{
		Record: openShipment(),
		Waits:  []deskcase.OpenWait{{Party: deskcase.WaitingOnCustomer}},
		Snooze: deskcase.Snooze{Until: ptr(now + 60), Anchor: deskcase.SnoozeTime},
		Now:    now,
	}
	snoozed := deskcase.Resolve(in)
	require.Equal(t, deskcase.StateSnoozed, snoozed.State)
	assert.Equal(t, now+60, *snoozed.SnoozedUntil)

	in.Now = now + 60
	assert.Equal(t, deskcase.StateWaiting, deskcase.Resolve(in).State)
}

func TestResolve_WorkingWithNothingOpen(t *testing.T) {
	t.Parallel()

	got := deskcase.Resolve(&deskcase.Inputs{Record: openShipment(), Now: now})

	assert.Equal(t, deskcase.StateWorking, got.State)
	assert.Empty(t, got.WaitingOn)
}

func TestResolve_WaitingNamesTheCustomerOrCarrierBeforeAnEvent(t *testing.T) {
	t.Parallel()

	got := deskcase.Resolve(&deskcase.Inputs{
		Record: openShipment(),
		Waits: []deskcase.OpenWait{
			{Party: deskcase.WaitingOnEvent, DueAt: ptr(now + 900)},
			{Party: deskcase.WaitingOnCarrier},
			{Party: deskcase.WaitingOnReply, DueAt: ptr(now + 300)},
		},
		Now: now,
	})

	assert.Equal(t, deskcase.StateWaiting, got.State)
	assert.Equal(t, deskcase.WaitingOnCarrier, got.WaitingOn)
	assert.Equal(t, now+300, *got.NextWaitDue)
	assert.Equal(t, 3, got.OpenWaits)
}

func TestResolve_AppointmentSnoozeFollowsTheStop(t *testing.T) {
	t.Parallel()

	stop := pulid.MustNew("stp_")
	record := openShipment()
	record.NextStopID = stop
	record.NextAppointmentAt = ptr(now + 7200)
	in := &deskcase.Inputs{
		Record: record,
		Snooze: deskcase.Snooze{Until: ptr(now + 3600), Anchor: deskcase.SnoozeAppointment, StopID: stop},
		Now:    now,
	}

	moved := deskcase.Resolve(in)
	require.Equal(t, deskcase.StateSnoozed, moved.State)
	assert.Equal(t, now+7200, *moved.SnoozedUntil, "a moved window moves the snooze")

	record.NextStopID = pulid.MustNew("stp_")
	assert.Equal(t, deskcase.StateWorking, deskcase.Resolve(in).State,
		"the stop was reached, so the snooze is over")
}

func TestResolve_ETASnoozeFollowsTheEstimateAndKeepsItsTimeWithoutOne(t *testing.T) {
	t.Parallel()

	in := &deskcase.Inputs{
		Record: openShipment(),
		Snooze: deskcase.Snooze{Until: ptr(now + 600), Anchor: deskcase.SnoozeETA},
		ETA:    ptr(now + 5400),
		Now:    now,
	}
	assert.Equal(t, now+5400, *deskcase.Resolve(in).SnoozedUntil)

	in.ETA = nil
	assert.Equal(t, now+600, *deskcase.Resolve(in).SnoozedUntil)
}

func TestMissingRecordIsGoneAndNotSettled(t *testing.T) {
	t.Parallel()

	missing := deskcase.Missing(deskcase.Ref{Type: agent.SubjectInvoice, ID: pulid.MustNew("inv_")})

	assert.True(t, missing.Gone())
	assert.False(t, openShipment().Gone())
	assert.Equal(t, deskcase.StateWorking, deskcase.Resolve(&deskcase.Inputs{Record: missing, Now: now}).State)
}

func TestSnoozeAnchorStoresNoneAsNull(t *testing.T) {
	t.Parallel()

	empty, err := deskcase.SnoozeAnchor("").Value()
	require.NoError(t, err)
	assert.Nil(t, empty)

	set, err := deskcase.SnoozeETA.Value()
	require.NoError(t, err)
	assert.Equal(t, "ETA", set)
}

func TestIsSubject(t *testing.T) {
	t.Parallel()

	for _, subject := range deskcase.Subjects() {
		assert.True(t, deskcase.IsSubject(subject), subject)
		_, named := subject.Resource()
		assert.True(t, named, "%s must name the permission that reads it", subject)
	}
	assert.False(t, deskcase.IsSubject(agent.SubjectWorker))
}
