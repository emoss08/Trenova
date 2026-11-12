package agenteventjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agenteventjobs/agenteventcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type capDefinitions struct {
	agentdefinitionports.AgentDefinitionRepository
	maxRuns int
}

func (d *capDefinitions) GetByID(
	_ context.Context,
	req agentdefinitionports.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return &agentdefinition.Definition{ID: req.ID, MaxConcurrentRuns: d.maxRuns}, nil
}

type workingRuns struct {
	agentruntimeports.AgentRunRepository
	working int
	asked   *agentruntimeports.CountOpenAgentRunsRequest
}

func (r *workingRuns) CountOpen(_ context.Context, req *agentruntimeports.CountOpenAgentRunsRequest) (int, error) {
	r.asked = req

	return r.working, nil
}

type startedRuns struct {
	agentruntimeports.AgentRunService
	started []*agentruntimeports.StartAgentRunForDefinitionRequest
	errs    map[pulid.ID]error
}

func (r *startedRuns) StartForDefinition(
	_ context.Context,
	req *agentruntimeports.StartAgentRunForDefinitionRequest,
	_ *permissionports.RequestActor,
) (*agent.AgentRun, error) {
	r.started = append(r.started, req)
	if err := r.errs[req.SubjectID]; err != nil {
		return nil, err
	}

	return &agent.AgentRun{ID: pulid.MustNew("ar_")}, nil
}

// memoryQueue holds one kind's queued subjects in order.
type memoryQueue struct {
	workqueueports.Queue
	due     []workqueueports.Due
	listed  int
	settled map[pulid.ID]*int64
}

func (q *memoryQueue) ListDue(_ context.Context, req *workqueueports.ListDueRequest) ([]workqueueports.Due, error) {
	q.listed++

	return q.due[:min(len(q.due), req.Limit)], nil
}

func (q *memoryQueue) SettleMany(_ context.Context, req *workqueueports.SettleManyRequest) error {
	for _, row := range req.Done {
		q.settled[row.SubjectID] = nil
	}
	for _, row := range req.Retry {
		retryAt := req.RetryAt
		q.settled[row.SubjectID] = &retryAt
	}

	return nil
}

func queued(n int) []workqueueports.Due {
	due := make([]workqueueports.Due, 0, n)
	for i := range n {
		due = append(due, workqueueports.Due{
			SubjectID: pulid.MustNew("bqi_"),
			DueAt:     int64(100 + i),
			Detail:    string(agenttrigger.EventBillingQueueItemException),
		})
	}

	return due
}

func drainPayload(definitionID pulid.ID) *temporaljobs.DrainPayload {
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	return &temporaljobs.DrainPayload{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Kind:           agenteventcontract.Kind(definitionID),
	}
}

// An agent starts as many queued runs as its cap less the runs still working,
// counting only working runs, each on the event it was queued with.
func TestStartEventRunsStartsUpToTheAgentsRoom(t *testing.T) {
	t.Parallel()

	definitionID := pulid.MustNew("agd_")
	runRepo := &workingRuns{working: 1}
	runs := &startedRuns{}
	queue := &memoryQueue{due: queued(3), settled: map[pulid.ID]*int64{}}
	a := NewActivities(ActivitiesParams{
		Queue:       queue,
		Definitions: &capDefinitions{maxRuns: 3},
		Runs:        runs,
		RunRepo:     runRepo,
		Logger:      zap.NewNop(),
	})

	page, err := a.StartEventRunsActivity(t.Context(), drainPayload(definitionID))

	require.NoError(t, err)
	require.NotNil(t, runRepo.asked)
	assert.True(t, runRepo.asked.WorkingOnly, "a run waiting on a decision takes no room")
	require.Len(t, runs.started, 2)
	for _, req := range runs.started {
		assert.Equal(t, definitionID, req.DefinitionID)
		assert.Equal(t, agent.RunTriggerEvent, req.Trigger)
		assert.Equal(t, agenttrigger.EventBillingQueueItemException, req.EventKind)
	}
	assert.Equal(t, 2, page.Done)
	assert.True(t, page.Full, "a full page sends the drain back for the rest")
}

// A full agent claims nothing and says so, so its drain looks again later
// rather than ending with the events still queued.
func TestStartEventRunsIsBlockedWhenTheAgentIsFull(t *testing.T) {
	t.Parallel()

	queue := &memoryQueue{due: queued(2), settled: map[pulid.ID]*int64{}}
	a := NewActivities(ActivitiesParams{
		Queue:       queue,
		Definitions: &capDefinitions{maxRuns: 1},
		Runs:        &startedRuns{},
		RunRepo:     &workingRuns{working: 1},
		Logger:      zap.NewNop(),
	})

	page, err := a.StartEventRunsActivity(t.Context(), drainPayload(pulid.MustNew("agd_")))

	require.NoError(t, err)
	assert.True(t, page.Blocked)
	assert.Zero(t, queue.listed, "nothing is claimed while there is no room")
}

// A subject whose run is still open spends its event, as it did when events
// started runs directly; a start that fails for a passing reason comes back.
func TestStartEventRunsSettlesOpenSubjectsAndRetriesFailures(t *testing.T) {
	t.Parallel()

	due := queued(2)
	open, failing := due[0].SubjectID, due[1].SubjectID
	queue := &memoryQueue{due: due, settled: map[pulid.ID]*int64{}}
	a := NewActivities(ActivitiesParams{
		Queue:       queue,
		Definitions: &capDefinitions{maxRuns: 5},
		Runs: &startedRuns{errs: map[pulid.ID]error{
			open:    agentruntimeports.ErrAgentRunAlreadyOpen,
			failing: errors.New("temporal unavailable"),
		}},
		RunRepo: &workingRuns{},
		Logger:  zap.NewNop(),
	})

	page, err := a.StartEventRunsActivity(t.Context(), drainPayload(pulid.MustNew("agd_")))

	require.NoError(t, err)
	assert.Equal(t, 1, page.Done)
	assert.Equal(t, 1, page.Failed)
	retryAt, settled := queue.settled[open]
	assert.True(t, settled, "the open subject's event is settled")
	assert.Nil(t, retryAt, "and not queued again")
	assert.NotNil(t, queue.settled[failing], "the failed start is due again")
}
