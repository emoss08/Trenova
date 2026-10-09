package agentwait_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func base(kind agentwait.Kind, condition agentwait.Condition) *agentwait.Wait {
	return &agentwait.Wait{
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		AgentDefinitionID: pulid.MustNew("agdef_"),
		ThreadID:          pulid.MustNew("athr_"),
		Kind:              kind,
		Condition:         &condition,
		Description:       "  Truck 2214 to reach Kroger DC  ",
		Status:            agentwait.StatusWaiting,
	}
}

func problems(wait *agentwait.Wait) []string {
	wait.Normalize()
	multiErr := errortypes.NewMultiError()
	wait.Validate(multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}

	return fields
}

func TestValidate_EachKindNamesWhatItNeeds(t *testing.T) {
	t.Parallel()

	move, stop := pulid.MustNew("smv_"), pulid.MustNew("stp_")
	cases := []struct {
		name      string
		kind      agentwait.Kind
		condition agentwait.Condition
		want      []string
	}{
		{"a time needs when", agentwait.KindTime, agentwait.Condition{}, []string{"at"}},
		{"an arrival needs the move", agentwait.KindStopArrival, agentwait.Condition{}, []string{"shipmentMoveId"}},
		{"an arrival may leave the stop out", agentwait.KindStopArrival, agentwait.Condition{ShipmentMoveID: move}, []string{}},
		{"a reply names one party", agentwait.KindReply, agentwait.Condition{
			CarrierID: pulid.MustNew("car_"), CustomerID: pulid.MustNew("cus_"),
		}, []string{"shipmentId"}},
		{"an appointment needs move and stop", agentwait.KindAppointmentNear, agentwait.Condition{
			MinutesBefore: 30,
		}, []string{"shipmentMoveId", "stopId"}},
		{"minutes before stay within a day", agentwait.KindAppointmentNear, agentwait.Condition{
			ShipmentMoveID: move, StopID: stop, MinutesBefore: agentwait.MaxMinutesBefore + 1,
		}, []string{"minutesBefore"}},
		{"drive time needs a threshold", agentwait.KindHOSDriveBelow, agentwait.Condition{
			WorkerID: pulid.MustNew("wrk_"),
		}, []string{"driveHoursBelow"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ElementsMatch(t, tc.want, problems(base(tc.kind, tc.condition)))
		})
	}
}

func TestValidate_AWaitBelongsToAConversationOrARunsRecord(t *testing.T) {
	t.Parallel()

	wait := base(agentwait.KindTime, agentwait.Condition{At: 1})
	wait.ThreadID = pulid.Nil
	assert.Contains(t, problems(wait), "owner")

	wait.RunID = pulid.MustNew("ar_")
	wait.SubjectID = pulid.MustNew("smv_")
	assert.NotContains(t, problems(wait), "owner")
}

func TestNormalize_WatchesTheRecordWhoseChangeEndsTheWait(t *testing.T) {
	t.Parallel()

	move, stop := pulid.MustNew("smv_"), pulid.MustNew("stp_")
	carrier, worker := pulid.MustNew("car_"), pulid.MustNew("wrk_")
	assert.Equal(t, move, watchOf(agentwait.KindStopDeparture, agentwait.Condition{
		ShipmentMoveID: move, StopID: stop,
	}), "a stop visit is watched on its move, which the event names")
	assert.Equal(t, stop, watchOf(agentwait.KindAppointmentNear, agentwait.Condition{
		ShipmentMoveID: move, StopID: stop,
	}))
	assert.Equal(t, carrier, watchOf(agentwait.KindReply, agentwait.Condition{CarrierID: carrier}))
	assert.Equal(t, worker, watchOf(agentwait.KindHOSDriveBelow, agentwait.Condition{WorkerID: worker}))
	assert.True(t, watchOf(agentwait.KindTime, agentwait.Condition{At: 10}).IsNil())
}

func watchOf(kind agentwait.Kind, condition agentwait.Condition) pulid.ID {
	wait := base(kind, condition)
	wait.Normalize()

	return wait.WatchID
}

func TestResumeNote_SaysWhatHappenedAndWhatToDoNext(t *testing.T) {
	t.Parallel()

	wait := base(agentwait.KindStopArrival, agentwait.Condition{ShipmentMoveID: pulid.MustNew("smv_")})
	wait.ID = pulid.MustNew(agentwait.IDPrefix)
	wait.Normalize()
	wait.Status = agentwait.StatusMet
	wait.Outcome = "Arrived at stop stp_1 at Thu Oct 8 15:40 UTC."
	wait.Then = "Tell Acme the truck is at the dock."

	note := wait.ResumeNote()
	assert.Contains(t, note, "not from the person")
	assert.Contains(t, note, "You waited for: Truck 2214 to reach Kroger DC\n")
	assert.Contains(t, note, "It happened. Arrived at stop stp_1")
	assert.Contains(t, note, "you planned to: Tell Acme the truck is at the dock.")
	assert.Contains(t, note, "Wait id: "+wait.ID.String())

	wait.Status = agentwait.StatusTimedOut
	wait.Outcome = ""
	require.Contains(t, wait.ResumeNote(), "It did not happen before the wait ran out.")
}
