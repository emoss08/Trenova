package agentwait_test

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

// An agent parks on a record changing itself; a report run's wait is set by
// report_run when the run outlives the call.
func TestWhoAsksForTheStreamedKinds(t *testing.T) {
	t.Parallel()

	assert.True(t, slices.Contains(agentwait.AgentKinds(), agentwait.KindRecordChanged))
	assert.False(t, slices.Contains(agentwait.AgentKinds(), agentwait.KindReportRun))
	for _, kind := range []agentwait.Kind{agentwait.KindRecordChanged, agentwait.KindReportRun} {
		assert.True(t, kind.IsValid())
		assert.True(t, kind.Streamed(), "%s hears its record from the change stream", kind)
	}
	assert.False(t, agentwait.KindReply.Streamed())
}

// A report run is read again on a timer in case its finish was missed; a
// record change has nothing to read again and ends only from the stream.
func TestOnlyTheReportRunIsTimed(t *testing.T) {
	t.Parallel()

	assert.True(t, agentwait.KindReportRun.Timed())
	assert.False(t, agentwait.KindRecordChanged.Timed())
}

// A person fixing a record makes several edits; the wait gathers them and
// picks the work up once the record has been quiet a while.
func TestARecordChangeSettlesBeforeItPicksTheWorkUp(t *testing.T) {
	t.Parallel()

	quiet, limit := agentwait.KindRecordChanged.Settle()
	assert.Equal(t, agentwait.RecordSettleSeconds, quiet)
	assert.Equal(t, agentwait.RecordSettleMaxSeconds, limit)
	assert.Greater(t, limit, quiet)

	quiet, limit = agentwait.KindReportRun.Settle()
	assert.Zero(t, quiet)
	assert.Zero(t, limit)
}

func TestTheStreamedKindsWatchTheirRecordAndNeedIt(t *testing.T) {
	t.Parallel()

	invoice, run := pulid.MustNew("inv_"), pulid.MustNew("rrun_")
	assert.Equal(t, invoice, watchOf(agentwait.KindRecordChanged,
		agentwait.Condition{RecordID: invoice}))
	assert.Equal(t, run, watchOf(agentwait.KindReportRun, agentwait.Condition{ReportRunID: run}))

	assert.Equal(t, []string{"recordId"},
		problems(base(agentwait.KindRecordChanged, agentwait.Condition{})))
	assert.Equal(t, []string{"reportRunId"},
		problems(base(agentwait.KindReportRun, agentwait.Condition{})))
}

func TestAMetRecordChangeTellsTheAgentToCheckAndCarryOn(t *testing.T) {
	t.Parallel()

	wait := base(agentwait.KindRecordChanged, agentwait.Condition{RecordID: pulid.MustNew("shp_")})
	wait.Status = agentwait.StatusMet
	wait.Outcome = "Shipment S-1042 was changed by the person you are working for: documents."
	wait.Then = "Propose posting INV-1001."
	note := wait.ResumeNote()

	assert.Contains(t, note, wait.Outcome)
	assert.Contains(t, note, "If what held the work up is resolved, carry on")
	assert.Contains(t, note, "set the wait again")
	assert.Contains(t, note, "Propose posting INV-1001.")

	wait.Status = agentwait.StatusTimedOut
	assert.NotContains(t, wait.ResumeNote(), "If what held the work up is resolved")
}
