package benchcheck

import (
	"testing"

	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestTurnsCompleted_TakesTheStopTheBenchMadeAsTheRunsOutcome(t *testing.T) {
	t.Parallel()

	turn := &benchrecord.TurnRecord{
		Status:  conversation.AssistantTurnStatusFailed,
		Failure: "the run was stopped while its proposals waited for a decision",
	}

	stopped := turnsCompleted(&Evidence{
		Turns: []*benchrecord.TurnRecord{turn},
		Run:   &RunOutcome{Stopping: true, Stopped: true},
	})
	assert.True(t, stopped.Passed, stopped.Detail)

	failed := turnsCompleted(&Evidence{
		Turns: []*benchrecord.TurnRecord{turn},
		Run:   &RunOutcome{Stopping: true},
	})
	assert.False(t, failed.Passed, "a run the bench did not stop failed on its own")

	captured := turnsCompleted(&Evidence{
		Turns: []*benchrecord.TurnRecord{{
			Status:       conversation.AssistantTurnStatusFailed,
			CaptureError: "the stopped run did not close within 1m",
		}},
		Run: &RunOutcome{Stopping: true, Stopped: true},
	})
	assert.False(t, captured.Passed)
}

func TestRunOutcome_SubstitutesItsIDs(t *testing.T) {
	t.Parallel()

	outcome := &RunOutcome{ID: pulid.MustNew("ar_"), Subject: pulid.MustNew("bqi_")}
	got := outcome.substitute("select 1 from x where run_id = {{run}} and id = {{subject}}")
	assert.Contains(t, got, "'"+outcome.ID.String()+"'")
	assert.Contains(t, got, "'"+outcome.Subject.String()+"'")

	var none *RunOutcome
	assert.Equal(t, "select 1", none.substitute("select 1"))
}
