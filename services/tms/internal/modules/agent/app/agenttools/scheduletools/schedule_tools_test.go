package scheduletools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePlanner answers Plan and CreatePlanned with the schedule an edit makes,
// or with existing when it is set, and records each request.
type fakePlanner struct {
	existing *conversationschedule.Schedule
	err      error
	planned  []conversationscheduleports.PlanConversationScheduleRequest
	created  []conversationscheduleports.PlanConversationScheduleRequest
}

func (f *fakePlanner) answer(
	req conversationscheduleports.PlanConversationScheduleRequest,
) (*conversationscheduleports.PlannedConversationSchedule, error) {
	if f.err != nil {
		return nil, f.err
	}
	zone := "America/Chicago"
	if req.Timezone != "" {
		zone, _ = conversationschedule.ResolveZone(req.Timezone)
	}
	schedule := &conversationschedule.Schedule{
		ID:             pulid.MustNew("csch_"),
		OrganizationID: req.Actor.OrganizationID,
		BusinessUnitID: req.Actor.BusinessUnitID,
		ThreadID:       req.ThreadID,
		UserID:         req.Actor.UserID,
		Timezone:       zone,
		Enabled:        true,
	}
	multiErr := errortypes.NewMultiError()
	schedule.Change(req.Edit, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return &conversationscheduleports.PlannedConversationSchedule{
		Schedule: schedule,
		Existing: f.existing,
	}, nil
}

func (f *fakePlanner) Plan(
	_ context.Context,
	req conversationscheduleports.PlanConversationScheduleRequest,
) (*conversationscheduleports.PlannedConversationSchedule, error) {
	f.planned = append(f.planned, req)

	return f.answer(req)
}

func (f *fakePlanner) CreatePlanned(
	_ context.Context,
	req conversationscheduleports.PlanConversationScheduleRequest,
) (*conversationscheduleports.PlannedConversationSchedule, error) {
	f.created = append(f.created, req)

	return f.answer(req)
}

func scheduleParams(args map[string]any) *agenttoolports.ToolExecuteParams {
	person := pulid.MustNew("usr_")
	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    person,
		UserID:         person,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	return &agenttoolports.ToolExecuteParams{
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		Actor:          actor,
		Params:         args,
		IdempotencyKey: "aprp_1",
		ThreadID:       pulid.MustNew("athr_"),
	}
}

func boardArgs() map[string]any {
	return map[string]any{
		"request":  "Give me today's operations board: active shipments, on-time %, at-risk",
		"repeat":   "day",
		"time":     "09:30",
		"timezone": "America/New_York",
	}
}

/*
"Schedule this to run everyday at 9:30 AM EST" is a request the agent writes
out in full; the schedule is kept in the conversation the runtime names, for
the person in it, on the clock they named.
*/
func TestCreateSchedule_KeepsTheRequestInThisConversationForThePerson(t *testing.T) {
	t.Parallel()

	planner := &fakePlanner{}
	tool := NewCreateScheduleTool(planner)
	params := scheduleParams(boardArgs())
	params.ProposalID = pulid.MustNew("aprp_")

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)

	require.Len(t, planner.created, 1)
	req := planner.created[0]
	assert.Equal(
		t,
		params.ThreadID,
		req.ThreadID,
		"the conversation is the runtime's, not the model's",
	)
	assert.Equal(t, params.Actor.UserID, req.Actor.UserID)
	assert.Equal(t, "day", req.Edit.Repeat)
	assert.Equal(t, "09:30", req.Edit.Time)
	assert.Equal(t, "America/New_York", req.Timezone)
	assert.Equal(t, params.ProposalID, req.ProposalID, "the card names the approved proposal")

	assert.Equal(t, "created", result.Action)
	assert.Equal(t, "schedule", result.Kind)
	assert.Equal(
		t,
		"Every day · 9:30 AM · Give me today's operations board: active shipments, on-time %, at-risk",
		result.Name,
	)
	assert.NotEmpty(t, result.IDs["scheduleId"])
	assert.Equal(t, &agentrecord.RecordRef{
		EntityType: "assistant_thread",
		ID:         params.ThreadID.String(),
	}, result.Record, "the result links to the conversation its answers post into")
}

/*
The proposal says what would run, when and on whose clock, and when it first
runs, before anything is kept.
*/
func TestCreateSchedule_PreviewsWhatWouldRun(t *testing.T) {
	t.Parallel()

	planner := &fakePlanner{}
	preview, err := NewCreateScheduleTool(planner).(agenttoolports.ToolPreviewer).
		Preview(t.Context(), scheduleParams(boardArgs()))
	require.NoError(t, err)

	assert.Contains(t, preview.Summary,
		"Would ask \"Give me today's operations board: active shipments, on-time %, at-risk\" "+
			"here every day at 9:30 AM, on America/New_York time")
	assert.Empty(t, preview.Warnings)
	require.Len(t, preview.Changes, 1)
	assert.Equal(t, agentpreview.PreviewOperationCreate, preview.Changes[0].Operation)
	assert.Empty(t, planner.created, "a preview keeps nothing")
}

/*
A request the conversation already repeats is refused while the agent can
still say so; approving the same proposal again answers with the schedule it
kept rather than failing.
*/
func TestCreateSchedule_RefusesARepeatButNotARetriedApproval(t *testing.T) {
	t.Parallel()

	existing := &conversationschedule.Schedule{
		ID:      pulid.MustNew("csch_"),
		Prompt:  "Unbilled loads",
		Cadence: "Every weekday · 7:30 AM",
	}
	planner := &fakePlanner{existing: existing}
	tool := NewCreateScheduleTool(planner)
	params := scheduleParams(map[string]any{
		"request": "Unbilled loads", "repeat": "weekday", "time": "07:30",
	})

	err := tool.(agenttoolports.ToolValidator).Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already asks \"Unbilled loads\" every weekday at 7:30 AM")

	params.ProposalID = pulid.MustNew("aprp_")
	require.NoError(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "found", result.Action)
	assert.Equal(t, existing.ID.String(), result.IDs["scheduleId"])
}

func TestCreateSchedule_RefusesWhatItCannotKeep(t *testing.T) {
	t.Parallel()

	t.Run("a run with no conversation", func(t *testing.T) {
		t.Parallel()
		params := scheduleParams(boardArgs())
		params.ThreadID = pulid.Nil
		err := NewCreateScheduleTool(&fakePlanner{}).(agenttoolports.ToolValidator).
			Validate(t.Context(), params)
		require.ErrorIs(t, err, errNeedsConversation)
	})

	t.Run("a run with no person", func(t *testing.T) {
		t.Parallel()
		params := scheduleParams(boardArgs())
		params.Actor.PrincipalType = permissionports.PrincipalTypeAgent
		err := NewCreateScheduleTool(&fakePlanner{}).(agenttoolports.ToolValidator).
			Validate(t.Context(), params)
		require.ErrorIs(t, err, errNeedsPerson)
	})

	t.Run("a cadence it cannot run", func(t *testing.T) {
		t.Parallel()
		args := boardArgs()
		args["repeat"] = "hourly"
		err := NewCreateScheduleTool(&fakePlanner{}).(agenttoolports.ToolValidator).
			Validate(t.Context(), scheduleParams(args))
		require.Error(t, err)
	})

	t.Run("an approver whose conversation it is not", func(t *testing.T) {
		t.Parallel()
		planner := &fakePlanner{err: errortypes.NewNotFoundError("Thread not found")}
		_, err := NewCreateScheduleTool(planner).(agenttoolports.ToolResultReporter).
			ExecuteWithResult(t.Context(), scheduleParams(boardArgs()))
		require.Error(t, err)
		assert.True(t, errortypes.IsBusinessError(err))
		assert.Contains(t, err.Error(), "Only the person whose conversation this is")
	})
}

/*
An agent proposes a schedule and a person approves it: it never runs on its
own, and it touches only the person's own conversation.
*/
func TestCreateSchedule_AlwaysWaitsForAPerson(t *testing.T) {
	t.Parallel()

	policy := NewCreateScheduleTool(nil).Policy()
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
	assert.Equal(t, agenttooltrust.ToolScopeSelf, policy.Scope)
	assert.Equal(t, []agent.EgressClass{agent.EgressPersonal}, policy.Egress)
}
