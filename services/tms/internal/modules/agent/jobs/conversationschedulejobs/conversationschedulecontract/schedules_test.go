package conversationschedulecontract

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// fakeHandle is one Temporal schedule as the fake server holds it.
type fakeHandle struct {
	client.ScheduleHandle

	server *fakeScheduleClient
	id     string
}

func (h *fakeHandle) GetID() string { return h.id }

func (h *fakeHandle) Update(_ context.Context, opts client.ScheduleUpdateOptions) error {
	current, ok := h.server.schedules[h.id]
	if !ok {
		return serviceerror.NewNotFound("schedule not found")
	}
	update, err := opts.DoUpdate(client.ScheduleUpdateInput{
		Description: client.ScheduleDescription{Schedule: current},
	})
	if err != nil {
		return err
	}
	h.server.schedules[h.id] = *update.Schedule
	h.server.updates++

	return nil
}

func (h *fakeHandle) Delete(context.Context) error {
	if _, ok := h.server.schedules[h.id]; !ok {
		return serviceerror.NewNotFound("schedule not found")
	}
	delete(h.server.schedules, h.id)

	return nil
}

type fakeIterator struct {
	entries []*client.ScheduleListEntry
}

func (i *fakeIterator) HasNext() bool { return len(i.entries) > 0 }

func (i *fakeIterator) Next() (*client.ScheduleListEntry, error) {
	entry := i.entries[0]
	i.entries = i.entries[1:]

	return entry, nil
}

// fakeScheduleClient holds schedules the way the Temporal server does: by
// id, refusing a second create of one.
type fakeScheduleClient struct {
	client.ScheduleClient

	schedules map[string]client.Schedule
	created   []client.ScheduleOptions
	updates   int
}

func (c *fakeScheduleClient) Create(
	_ context.Context,
	options client.ScheduleOptions,
) (client.ScheduleHandle, error) {
	if _, exists := c.schedules[options.ID]; exists {
		return nil, serviceerror.NewAlreadyExists("schedule already exists")
	}
	c.created = append(c.created, options)
	c.schedules[options.ID] = client.Schedule{
		Action: options.Action,
		Spec:   &options.Spec,
		Policy: &client.SchedulePolicies{
			Overlap:       options.Overlap,
			CatchupWindow: options.CatchupWindow,
		},
		State: &client.ScheduleState{Paused: options.Paused, Note: options.Note},
	}

	return &fakeHandle{server: c, id: options.ID}, nil
}

func (c *fakeScheduleClient) GetHandle(_ context.Context, id string) client.ScheduleHandle {
	return &fakeHandle{server: c, id: id}
}

func (c *fakeScheduleClient) List(
	context.Context,
	client.ScheduleListOptions,
) (client.ScheduleListIterator, error) {
	entries := make([]*client.ScheduleListEntry, 0, len(c.schedules))
	for id := range c.schedules {
		entries = append(entries, &client.ScheduleListEntry{ID: id})
	}

	return &fakeIterator{entries: entries}, nil
}

type fakeTemporal struct {
	client.Client
	schedules *fakeScheduleClient
}

func (c *fakeTemporal) ScheduleClient() client.ScheduleClient { return c.schedules }

type fakeRows struct {
	conversationscheduleports.ConversationScheduleRepository
	rows []*conversationschedule.Schedule
}

func (f *fakeRows) ListAcrossTenants(
	_ context.Context,
	req conversationscheduleports.ListConversationSchedulesAcrossTenantsRequest,
) ([]*conversationschedule.Schedule, error) {
	if req.AfterID.IsNotNil() {
		return nil, nil
	}

	return f.rows, nil
}

func newSchedules(rows ...*conversationschedule.Schedule) (*Schedules, *fakeScheduleClient) {
	server := &fakeScheduleClient{schedules: map[string]client.Schedule{}}

	return &Schedules{
		client:    &fakeTemporal{schedules: server},
		schedules: &fakeRows{rows: rows},
		l:         zap.NewNop(),
	}, server
}

func weekdayMornings() *conversationschedule.Schedule {
	return &conversationschedule.Schedule{
		ID:             pulid.MustNew("csch_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ThreadID:       pulid.MustNew("athr_"),
		UserID:         pulid.MustNew("usr_"),
		Prompt:         "What's blocking the billing queue?",
		Cadence:        "Every weekday · 7:30 AM",
		CronExpression: "30 7 * * 1-5",
		Timezone:       "America/Chicago",
		Enabled:        true,
	}
}

/*
A schedule is a Temporal schedule of its own: its cron on its owner's clock,
an overlapping slot skipped, a short catch-up window, marked as not the static
registry's, and a slot starts the run workflow on the system queue.
*/
func TestSync_CreatesTheTemporalSchedule(t *testing.T) {
	t.Parallel()

	sched := weekdayMornings()
	schedules, server := newSchedules()
	schedules.Sync(t.Context(), sched)

	require.Len(t, server.created, 1)
	options := server.created[0]
	assert.Equal(t, "conversation-schedule/"+sched.ID.String(), options.ID)
	assert.Equal(t, []string{"30 7 * * 1-5"}, options.Spec.CronExpressions)
	assert.Equal(t, "America/Chicago", options.Spec.TimeZoneName)
	assert.Equal(t, enums.SCHEDULE_OVERLAP_POLICY_SKIP, options.Overlap)
	assert.Equal(t, catchupWindow, options.CatchupWindow)
	assert.False(t, options.Paused)
	assert.Equal(t, scheduleOwner, options.Memo[schedule.ManagedByMemoKey])

	action, ok := options.Action.(*client.ScheduleWorkflowAction)
	require.True(t, ok)
	assert.Equal(t, RunWorkflowName, action.Workflow)
	assert.Equal(t, temporaltype.TaskQueueSystem.String(), action.TaskQueue)
	payload, ok := action.Args[0].(*RunPayload)
	require.True(t, ok)
	assert.Equal(t, sched.ID, payload.ScheduleID)
	assert.Equal(t, sched.OrganizationID, payload.OrganizationID)
	assert.Equal(t, sched.BusinessUnitID, payload.BusinessUnitID)
}

/*
Pausing and resuming update the schedule that exists rather than making a
second one, and deleting removes it. Removing one already gone is not an
error: a deleted conversation's schedule may have gone first.
*/
func TestPauseResumeAndDeleteSyncTheTemporalSchedule(t *testing.T) {
	t.Parallel()

	sched := weekdayMornings()
	schedules, server := newSchedules()
	id := ScheduleID(sched.ID)
	schedules.Sync(t.Context(), sched)

	sched.Enabled = false
	schedules.Sync(t.Context(), sched)
	require.Len(t, server.created, 1, "the existing schedule is updated")
	assert.True(t, server.schedules[id].State.Paused)

	sched.Enabled = true
	schedules.Sync(t.Context(), sched)
	assert.False(t, server.schedules[id].State.Paused)
	assert.Equal(t, 2, server.updates)

	schedules.Remove(t.Context(), sched.ID)
	assert.NotContains(t, server.schedules, id)
	schedules.Remove(t.Context(), sched.ID)
}

/*
The reconcile gives every row its schedule and removes the schedules whose
rows are gone, leaving every other kind of schedule alone.
*/
func TestReconcile_MakesTheSchedulesMatchTheRows(t *testing.T) {
	t.Parallel()

	kept := weekdayMornings()
	schedules, server := newSchedules(kept)
	orphan := ScheduleID(pulid.MustNew("csch_"))
	server.schedules[orphan] = client.Schedule{}
	server.schedules["agent-definition/agdef_1"] = client.Schedule{}

	result, err := schedules.Reconcile(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, result.Synced)
	assert.Equal(t, 1, result.Removed)
	assert.Contains(t, server.schedules, ScheduleID(kept.ID))
	assert.NotContains(t, server.schedules, orphan)
	assert.Contains(t, server.schedules, "agent-definition/agdef_1")
}
