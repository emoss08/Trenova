package aiauditprojector

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/palettequestion"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func paletteCall(outcome palettequestion.ToolCallOutcome) *palettequestion.ToolCall {
	return &palettequestion.ToolCall{
		ID:             pulid.MustNew(palettequestion.ToolCallIDPrefix),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		QuestionID:     pulid.MustNew(palettequestion.IDPrefix),
		UserID:         pulid.MustNew("usr_"),
		CallID:         "toolu_1",
		Sequence:       1,
		Model:          "claude-haiku-5-5",
		Tool:           "get_record",
		Arguments:      map[string]any{"id": "cus_1"},
		Read:           "customer_get",
		Params:         map[string]any{"customerId": "cus_1"},
		Outcome:        outcome,
		StartedAtMs:    1_793_790_509_250,
		DurationMs:     1_800,
		CreatedAt:      1_793_790_511,
	}
}

// A palette question's read is on the trail as its tool call: owned by the
// question, acted as the person who asked, named for the read it went to.
func TestPaletteCallEvent_IsTheQuestionsToolCallAsThePerson(t *testing.T) {
	t.Parallel()

	call := paletteCall(palettequestion.ToolCallFailed)
	call.Reason = "customer_get failed: an agent read needs the tool call's read scope"
	d := &deriver{redactor: testRedactor()}

	event, err := d.paletteCallEvent(emptyLookups(), call)

	require.NoError(t, err)
	assert.Equal(t, aiaudit.KindToolCall, event.Kind)
	assert.Equal(t, aiaudit.OutcomeFailed, event.Outcome)
	assert.Equal(t, string(agent.RunOwnerPaletteQuestion), event.OwnerKind)
	assert.Equal(t, call.QuestionID, event.OwnerID)
	assert.Equal(t, aiaudit.PrincipalUser, event.PrincipalType)
	assert.Equal(t, call.UserID.String(), event.PrincipalID)
	assert.Equal(t, call.UserID, event.OnBehalfOfUserID)
	assert.Equal(t, aiaudit.PurposeLive, event.Purpose)
	assert.Equal(t, "customer_get", event.ToolName)
	assert.Equal(t, "toolu_1", event.CallID)
	assert.Equal(t, "claude-haiku-5-5", event.Model)
	assert.Equal(t, call.Reason, event.Reason)
	assert.Equal(t, int64(1_793_790_509), event.OccurredAt)
	assert.Equal(t, int64(1_793_790_511), event.WindowEnd)
	require.NotNil(t, event.LatencyMs)
	assert.Equal(t, int64(1_800), *event.LatencyMs)
	assert.Equal(t, paletteCallKey(call), event.SourceKey)
}

func TestPaletteCallEvent_FollowsTheReadsOutcome(t *testing.T) {
	t.Parallel()

	for outcome, want := range map[palettequestion.ToolCallOutcome]aiaudit.Outcome{
		palettequestion.ToolCallRan:     aiaudit.OutcomeRan,
		palettequestion.ToolCallRefused: aiaudit.OutcomeRefused,
		palettequestion.ToolCallFailed:  aiaudit.OutcomeFailed,
	} {
		event, err := (&deriver{redactor: testRedactor()}).paletteCallEvent(
			emptyLookups(), paletteCall(outcome),
		)
		require.NoError(t, err)
		assert.Equal(t, want, event.Outcome, outcome)
	}
}

// A search goes to no Desk read, so the trail names the palette tool and keeps
// the model's arguments.
func TestPaletteCallEvent_ASearchKeepsThePaletteTool(t *testing.T) {
	t.Parallel()

	call := paletteCall(palettequestion.ToolCallRan)
	call.Tool, call.Read, call.Params = "search_records", "", map[string]any{}
	call.Arguments = map[string]any{"query": "Gordon"}
	call.Summary = "16 records"

	event, err := (&deriver{redactor: testRedactor()}).paletteCallEvent(emptyLookups(), call)

	require.NoError(t, err)
	assert.Equal(t, "search_records", event.ToolName)
	assert.Equal(t, "16 records", event.ResultSummary)
}

// A palette question's model calls name the person who asked, as the reads do.
func TestUsageEvent_APaletteQuestionsModelCallIsThePersons(t *testing.T) {
	t.Parallel()

	user := pulid.MustNew("usr_")
	question := pulid.MustNew(palettequestion.IDPrefix)
	record := &aiusage.AIUsageRecord{
		ID:        pulid.MustNew("aiu_"),
		UserID:    user,
		OwnerKind: agent.RunOwnerPaletteQuestion,
		OwnerID:   question,
		Succeeded: true,
		CreatedAt: 1_793_790_509,
	}

	event := (&deriver{redactor: testRedactor()}).usageEvent(emptyLookups(), record)

	assert.Equal(t, aiaudit.KindModelCall, event.Kind)
	assert.Equal(t, string(agent.RunOwnerPaletteQuestion), event.OwnerKind)
	assert.Equal(t, question, event.OwnerID)
	assert.Equal(t, aiaudit.PrincipalUser, event.PrincipalType)
	assert.Equal(t, user.String(), event.PrincipalID)
	assert.False(t, event.Reconstructed)
}
