package conversationscheduleservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// fakeSchedules keeps schedules in memory, scoped the way the repository
// scopes them.
type fakeSchedules struct {
	repositories.ConversationScheduleRepository

	rows     map[pulid.ID]*conversationschedule.Schedule
	messages []conversation.Message
	runs     []repositories.RecordConversationScheduleRunRequest
}

func newFakeSchedules() *fakeSchedules {
	return &fakeSchedules{rows: map[pulid.ID]*conversationschedule.Schedule{}}
}

func (f *fakeSchedules) Create(
	_ context.Context,
	req repositories.CreateConversationScheduleRequest,
) (*conversationschedule.Schedule, *conversation.Message, error) {
	saved := *req.Schedule
	saved.ID = pulid.MustNew("csch_")
	f.rows[saved.ID] = &saved
	message := req.Message
	message.ID = pulid.MustNew("amsg_")
	message.ThreadID = saved.ThreadID
	message.ScheduleID = saved.ID
	f.messages = append(f.messages, message)

	return &saved, &message, nil
}

func (f *fakeSchedules) Get(
	_ context.Context,
	req repositories.GetConversationScheduleRequest,
) (*conversationschedule.Schedule, error) {
	row, ok := f.rows[req.ID]
	if !ok || (req.UserID.IsNotNil() && row.UserID != req.UserID) {
		return nil, errortypes.NewNotFoundError("Schedule not found within your organization")
	}
	clone := *row

	return &clone, nil
}

func (f *fakeSchedules) Count(
	_ context.Context,
	req repositories.CountConversationSchedulesRequest,
) (int, error) {
	count := 0
	for _, row := range f.rows {
		if row.UserID == req.UserID && (req.ThreadID.IsNil() || row.ThreadID == req.ThreadID) {
			count++
		}
	}

	return count, nil
}

func (f *fakeSchedules) UpdateState(
	_ context.Context,
	schedule *conversationschedule.Schedule,
) (*conversationschedule.Schedule, error) {
	schedule.Version++
	clone := *schedule
	f.rows[schedule.ID] = &clone

	return schedule, nil
}

func (f *fakeSchedules) RecordRun(
	_ context.Context,
	req repositories.RecordConversationScheduleRunRequest,
) error {
	f.runs = append(f.runs, req)
	if row, ok := f.rows[req.ID]; ok {
		row.LastRunAt = &req.RunAt
		row.NextRunAt = req.NextRunAt
		row.LastTurnID = req.TurnID
	}

	return nil
}

func (f *fakeSchedules) Delete(
	_ context.Context,
	req repositories.GetConversationScheduleRequest,
) error {
	delete(f.rows, req.ID)

	return nil
}

type fakeThreads struct {
	thread *conversation.Thread
}

func (f *fakeThreads) GetThread(
	_ context.Context,
	req repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	if req.ID != f.thread.ID || req.UserID != f.thread.UserID {
		return nil, errortypes.NewNotFoundError("Thread not found within your organization")
	}

	return f.thread, nil
}

func (f *fakeThreads) GetThreadOwned(
	_ context.Context,
	req repositories.GetThreadOwnedRequest,
) (*conversation.Thread, error) {
	if req.ID != f.thread.ID {
		return nil, errortypes.NewNotFoundError("Thread not found within your organization")
	}

	return f.thread, nil
}

type fakeDefinitions struct {
	repositories.AgentDefinitionRepository
	definition *agentdefinition.Definition
}

func (f *fakeDefinitions) GetByID(
	_ context.Context,
	_ repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return f.definition, nil
}

type fakePermissions struct {
	serviceports.PermissionEngine
	denied bool
	asked  []serviceports.RequestActor
}

func (f *fakePermissions) MayUseAgent(
	_ context.Context,
	actor *serviceports.RequestActor,
	_ *agentdefinition.Definition,
) (bool, error) {
	f.asked = append(f.asked, *actor)

	return !f.denied, nil
}

type fakeTurns struct {
	active  *conversation.AssistantTurn
	started []assistantturnservice.StartRequest
}

func (f *fakeTurns) StartTurn(
	_ context.Context,
	req assistantturnservice.StartRequest,
	start func(turn *conversation.AssistantTurn) (string, error),
) (*conversation.AssistantTurn, error) {
	f.started = append(f.started, req)
	turn := &conversation.AssistantTurn{
		ID:       pulid.MustNew("atrn_"),
		ThreadID: req.ThreadID,
		UserID:   req.UserID,
		Origin:   req.Origin,
		Status:   conversation.AssistantTurnStatusRunning,
	}
	if _, err := start(turn); err != nil {
		return nil, err
	}

	return turn, nil
}

func (f *fakeTurns) Active(
	_ context.Context,
	_ repositories.ActiveAssistantTurnRequest,
) (*conversation.AssistantTurn, error) {
	return f.active, nil
}

type fakeWorkflows struct {
	serviceports.WorkflowStarter
	payloads []*assistantjobs.AssistantTurnPayload
}

func (f *fakeWorkflows) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	_ any,
	args ...any,
) (client.WorkflowRun, error) {
	f.payloads = append(f.payloads, args[0].(*assistantjobs.AssistantTurnPayload))

	return startedRun{id: options.ID}, nil
}

type startedRun struct {
	client.WorkflowRun
	id string
}

func (r startedRun) GetID() string { return r.id }

type fakeSyncer struct {
	synced  []conversationschedule.Schedule
	removed []pulid.ID
}

func (f *fakeSyncer) Sync(_ context.Context, schedule *conversationschedule.Schedule) {
	f.synced = append(f.synced, *schedule)
}

func (f *fakeSyncer) Remove(_ context.Context, scheduleID pulid.ID) {
	f.removed = append(f.removed, scheduleID)
}

type fakeRealtime struct {
	published []*serviceports.PublishResourceInvalidationRequest
}

func (f *fakeRealtime) PublishResourceInvalidation(
	_ context.Context,
	req *serviceports.PublishResourceInvalidationRequest,
) error {
	f.published = append(f.published, req)

	return nil
}

type fixedClocks struct {
	user, organization string
}

func (c fixedClocks) userTimezone(context.Context, pagination.TenantInfo, pulid.ID) string {
	return c.user
}

func (c fixedClocks) organizationTimezone(context.Context, pagination.TenantInfo) string {
	return c.organization
}

type fixture struct {
	service     *Service
	schedules   *fakeSchedules
	turns       *fakeTurns
	workflows   *fakeWorkflows
	syncer      *fakeSyncer
	realtime    *fakeRealtime
	permissions *fakePermissions
	thread      *conversation.Thread
	owner       serviceports.RequestActor
	now         int64
}

// Saturday 3 October 2026, 12:00 UTC.
const saturdayNoon = int64(1791028800)

func newFixture() *fixture {
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	definition := &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Enabled: true}
	thread := &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		UserID:            pulid.MustNew("usr_"),
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		AgentDefinitionID: definition.ID,
	}
	f := &fixture{
		schedules:   newFakeSchedules(),
		turns:       &fakeTurns{},
		workflows:   &fakeWorkflows{},
		syncer:      &fakeSyncer{},
		realtime:    &fakeRealtime{},
		permissions: &fakePermissions{},
		thread:      thread,
		owner: serviceports.RequestActor{
			PrincipalType:  serviceports.PrincipalTypeUser,
			PrincipalID:    thread.UserID,
			UserID:         thread.UserID,
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
		},
		now: saturdayNoon,
	}
	f.service = &Service{
		l:           zap.NewNop(),
		schedules:   f.schedules,
		threads:     &fakeThreads{thread: thread},
		definitions: &fakeDefinitions{definition: definition},
		permissions: f.permissions,
		turns:       f.turns,
		workflows:   f.workflows,
		syncer:      f.syncer,
		realtime:    f.realtime,
		clocks:      fixedClocks{user: "America/Chicago", organization: "America/New_York"},
		now:         func() int64 { return f.now },
	}

	return f
}

func (f *fixture) create(t *testing.T, text string) *CreateResult {
	t.Helper()

	result, err := f.service.Create(t.Context(), CreateRequest{
		ThreadID: f.thread.ID,
		Text:     text,
		Actor:    f.owner,
	})
	require.NoError(t, err)

	return result
}

func (f *fixture) request(id pulid.ID) ScheduleRequest {
	return ScheduleRequest{ScheduleID: id, Actor: f.owner}
}

/*
Sending "every weekday at 7:30am, …" keeps a schedule and its card, on the
person's own clock, and hands it to Temporal. Nothing is asked yet: the first
answer comes on the first slot.
*/
func TestCreate_KeepsTheScheduleItsCardAndItsTemporalSchedule(t *testing.T) {
	t.Parallel()

	f := newFixture()
	text := "every weekday at 7:30am, what's blocking the billing queue?"
	result := f.create(t, text)

	schedule := result.Schedule
	assert.Equal(t, "What's blocking the billing queue?", schedule.Prompt)
	assert.Equal(t, "Every weekday · 7:30 AM", schedule.Cadence)
	assert.Equal(t, "30 7 * * 1-5", schedule.CronExpression)
	assert.Equal(t, "America/Chicago", schedule.Timezone, "the person's clock comes first")
	assert.True(t, schedule.Enabled)
	require.NotNil(t, schedule.NextRunAt)
	// Monday 5 October, 7:30 in Chicago is 12:30 UTC.
	assert.Equal(t, int64(1791203400), *schedule.NextRunAt)

	require.Len(t, f.schedules.messages, 1)
	message := f.schedules.messages[0]
	assert.Equal(t, conversation.MessageKindSchedule, message.Kind)
	assert.Equal(t, conversation.RoleUser, message.Role)
	assert.Equal(t, text, message.Content, "the card keeps the person's words")
	assert.Equal(t, schedule.ID, message.ScheduleID)

	require.Len(t, f.syncer.synced, 1)
	assert.Equal(t, schedule.ID, f.syncer.synced[0].ID)
	assert.Empty(t, f.turns.started, "creating a schedule asks nothing yet")
	require.Len(t, f.realtime.published, 1)
	assert.Equal(t, serviceports.ConversationSchedulesResource, f.realtime.published[0].Resource)
	assert.Equal(t, f.thread.UserID, f.realtime.published[0].AudienceUserID)
}

func TestCreate_FallsBackToTheOrganizationsClockThenUTC(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.service.clocks = fixedClocks{user: "", organization: "America/New_York"}
	assert.Equal(t, "America/New_York", f.create(t, "every day check loads").Schedule.Timezone)

	f.service.clocks = fixedClocks{user: "Not/AZone", organization: ""}
	assert.Equal(t, "UTC", f.create(t, "every day check loads").Schedule.Timezone)
}

func TestCreate_RefusesWhatItCannotKeep(t *testing.T) {
	t.Parallel()

	t.Run("a cadence it cannot read", func(t *testing.T) {
		t.Parallel()
		f := newFixture()
		_, err := f.service.Create(t.Context(), CreateRequest{
			ThreadID: f.thread.ID, Text: "/schedule hourly, check", Actor: f.owner,
		})
		require.Error(t, err)
		assert.Empty(t, f.schedules.rows)
	})

	t.Run("someone else's conversation", func(t *testing.T) {
		t.Parallel()
		f := newFixture()
		stranger := f.owner
		stranger.UserID = pulid.MustNew("usr_")
		_, err := f.service.Create(t.Context(), CreateRequest{
			ThreadID: f.thread.ID, Text: "every day check loads", Actor: stranger,
		})
		require.True(t, errortypes.IsNotFoundError(err))
		assert.Empty(t, f.schedules.rows)
	})

	t.Run("an agent the person lost", func(t *testing.T) {
		t.Parallel()
		f := newFixture()
		f.permissions.denied = true
		_, err := f.service.Create(t.Context(), CreateRequest{
			ThreadID: f.thread.ID, Text: "every day check loads", Actor: f.owner,
		})
		require.Error(t, err)
		assert.Empty(t, f.schedules.rows)
	})

	t.Run("a conversation already full of schedules", func(t *testing.T) {
		t.Parallel()
		f := newFixture()
		for range conversationschedule.MaxPerThread {
			f.create(t, "every day check loads")
		}
		_, err := f.service.Create(t.Context(), CreateRequest{
			ThreadID: f.thread.ID, Text: "every day check loads", Actor: f.owner,
		})
		require.Error(t, err)
		assert.Len(t, f.schedules.rows, conversationschedule.MaxPerThread)
	})
}

/*
Pausing pauses the Temporal schedule, resuming unpauses it and reads the next
slot from now, and deleting removes it. The card's message stays behind.
*/
func TestPauseResumeAndDeleteFollowTheTemporalSchedule(t *testing.T) {
	t.Parallel()

	f := newFixture()
	created := f.create(t, "every monday at 8am, summarize detention").Schedule

	paused, err := f.service.SetEnabled(t.Context(), f.request(created.ID), false)
	require.NoError(t, err)
	assert.False(t, paused.Enabled)
	require.Len(t, f.syncer.synced, 2)
	assert.False(t, f.syncer.synced[1].Enabled, "the Temporal schedule is paused with it")

	// A week passes while it is paused.
	f.now = saturdayNoon + 7*24*60*60
	resumed, err := f.service.SetEnabled(t.Context(), f.request(created.ID), true)
	require.NoError(t, err)
	assert.True(t, resumed.Enabled)
	require.Len(t, f.syncer.synced, 3)
	assert.True(t, f.syncer.synced[2].Enabled)
	require.NotNil(t, resumed.NextRunAt)
	assert.Greater(t, *resumed.NextRunAt, f.now, "the slot that passed while paused is not run late")

	require.NoError(t, f.service.Delete(t.Context(), f.request(created.ID)))
	assert.Equal(t, []pulid.ID{created.ID}, f.syncer.removed)
	assert.Empty(t, f.schedules.rows)
	assert.Len(t, f.schedules.messages, 1, "the card stays and says it was deleted")
}

func TestScheduleRoutesAreTheOwnersOnly(t *testing.T) {
	t.Parallel()

	f := newFixture()
	created := f.create(t, "every day check loads").Schedule
	stranger := f.owner
	stranger.UserID = pulid.MustNew("usr_")
	request := ScheduleRequest{ScheduleID: created.ID, Actor: stranger}

	_, err := f.service.SetEnabled(t.Context(), request, false)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.True(t, errortypes.IsNotFoundError(f.service.Delete(t.Context(), request)))
	_, err = f.service.RunNow(t.Context(), request)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Len(t, f.schedules.rows, 1)
	assert.Empty(t, f.turns.started)
}

/*
A slot coming round asks the request in the conversation it was made in, as
a scheduled turn by the conversation's owner, and records the run and the
next slot.
*/
func TestFire_AsksTheRequestInItsConversationAsTheOwner(t *testing.T) {
	t.Parallel()

	f := newFixture()
	created := f.create(t, "every weekday at 7:30am, what's blocking the billing queue?").Schedule
	f.now = 1791203400 // Monday 7:30 in Chicago.

	result, err := f.service.Fire(t.Context(), serviceports.FireConversationScheduleRequest{
		ScheduleID: created.ID,
		TenantInfo: pagination.TenantInfo{OrgID: created.OrganizationID, BuID: created.BusinessUnitID},
		FiredAt:    f.now,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Skipped)
	assert.True(t, result.TurnID.IsNotNil())

	require.Len(t, f.turns.started, 1)
	started := f.turns.started[0]
	assert.Equal(t, conversation.AssistantTurnOriginScheduled, started.Origin)
	assert.Equal(t, f.thread.ID, started.ThreadID)
	assert.Equal(t, f.thread.UserID, started.UserID)
	assert.Equal(t, "What's blocking the billing queue?", started.Input)

	require.Len(t, f.workflows.payloads, 1, "a worker answers it, like any turn")
	payload := f.workflows.payloads[0]
	assert.Equal(t, "What's blocking the billing queue?", payload.Content)
	assert.Equal(t, f.thread.UserID, payload.Actor.UserID, "it runs as the owner")
	assert.Equal(t, serviceports.PrincipalTypeUser, payload.Actor.PrincipalType)
	require.Len(t, f.permissions.asked, 2, "access is checked again when it runs")

	row := f.schedules.rows[created.ID]
	require.NotNil(t, row.LastRunAt)
	assert.Equal(t, f.now, *row.LastRunAt)
	assert.Equal(t, result.TurnID, row.LastTurnID)
	require.NotNil(t, row.NextRunAt)
	assert.Equal(t, f.now+24*60*60, *row.NextRunAt, "Tuesday at the same time")

	// A retried attempt for the same slot starts no second turn.
	again, err := f.service.Fire(t.Context(), serviceports.FireConversationScheduleRequest{
		ScheduleID: created.ID,
		TenantInfo: pagination.TenantInfo{OrgID: created.OrganizationID, BuID: created.BusinessUnitID},
		FiredAt:    f.now,
	})
	require.NoError(t, err)
	assert.Equal(t, SkipAlreadyRan, again.Skipped)
	assert.Len(t, f.turns.started, 1)
}

func TestFire_StartsNothingItShouldNot(t *testing.T) {
	t.Parallel()

	fire := func(f *fixture, id pulid.ID) (*serviceports.FireConversationScheduleResult, error) {
		return f.service.Fire(context.Background(), serviceports.FireConversationScheduleRequest{
			ScheduleID: id,
			TenantInfo: pagination.TenantInfo{OrgID: f.thread.OrganizationID, BuID: f.thread.BusinessUnitID},
			FiredAt:    f.now,
		})
	}

	t.Run("a deleted schedule is reported gone", func(t *testing.T) {
		t.Parallel()
		f := newFixture()
		result, err := fire(f, pulid.MustNew("csch_"))
		require.NoError(t, err)
		assert.True(t, result.Gone)
		assert.Empty(t, f.turns.started)
	})

	t.Run("a paused schedule", func(t *testing.T) {
		t.Parallel()
		f := newFixture()
		created := f.create(t, "every day check loads").Schedule
		_, err := f.service.SetEnabled(t.Context(), f.request(created.ID), false)
		require.NoError(t, err)
		result, err := fire(f, created.ID)
		require.NoError(t, err)
		assert.Equal(t, SkipPaused, result.Skipped)
		assert.Empty(t, f.turns.started)
	})

	t.Run("an owner who lost the agent", func(t *testing.T) {
		t.Parallel()
		f := newFixture()
		created := f.create(t, "every day check loads").Schedule
		f.permissions.denied = true
		result, err := fire(f, created.ID)
		require.NoError(t, err)
		assert.Equal(t, SkipNoAccess, result.Skipped)
		assert.Empty(t, f.turns.started)
	})

	t.Run("a conversation busy with another reply waits", func(t *testing.T) {
		t.Parallel()
		f := newFixture()
		created := f.create(t, "every day check loads").Schedule
		f.turns.active = &conversation.AssistantTurn{ID: pulid.MustNew("atrn_")}
		_, err := fire(f, created.ID)
		require.True(t, errors.Is(err, ErrConversationBusy))
		assert.Empty(t, f.turns.started)
		assert.Nil(t, f.schedules.rows[created.ID].LastRunAt)
	})
}

// Run now asks at once, paused or not, and hands back the turn to watch.
func TestRunNow_ReturnsTheTurnToWatch(t *testing.T) {
	t.Parallel()

	f := newFixture()
	created := f.create(t, "every day check loads").Schedule
	_, err := f.service.SetEnabled(t.Context(), f.request(created.ID), false)
	require.NoError(t, err)

	turn, err := f.service.RunNow(t.Context(), f.request(created.ID))
	require.NoError(t, err)
	assert.Equal(t, conversation.AssistantTurnOriginScheduled, turn.Origin)
	assert.Equal(t, turn.ID, f.schedules.rows[created.ID].LastTurnID)

	f.turns.active = turn
	_, err = f.service.RunNow(t.Context(), f.request(created.ID))
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business, "a busy conversation is said so, not retried")
}
