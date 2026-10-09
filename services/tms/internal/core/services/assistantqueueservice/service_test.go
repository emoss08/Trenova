package assistantqueueservice

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type memoryQueue struct {
	items    []*conversation.QueuedMessage
	restored []pulid.ID
}

func (q *memoryQueue) sorted() []*conversation.QueuedMessage {
	out := slices.Clone(q.items)
	slices.SortFunc(out, func(a, b *conversation.QueuedMessage) int {
		return cmp.Compare(a.Position, b.Position)
	})

	return out
}

func (q *memoryQueue) List(
	_ context.Context,
	scope *repositories.AssistantQueueScope,
) ([]*conversation.QueuedMessage, error) {
	out := make([]*conversation.QueuedMessage, 0, len(q.items))
	for _, item := range q.sorted() {
		if item.ThreadID == scope.ThreadID {
			out = append(out, item)
		}
	}

	return out, nil
}

func (q *memoryQueue) Get(
	_ context.Context,
	req *repositories.QueuedMessageRequest,
) (*conversation.QueuedMessage, error) {
	for _, item := range q.items {
		if item.ID == req.ID {
			copied := *item
			return &copied, nil
		}
	}

	return nil, errortypes.NewNotFoundError("{0} not found within your organization", "Queued message")
}

func (q *memoryQueue) Insert(
	_ context.Context,
	entity *conversation.QueuedMessage,
) (*conversation.QueuedMessage, error) {
	if len(q.items) >= conversation.MaxQueuedPerThread {
		return nil, repositories.ErrQueueFull
	}
	entity.ID = pulid.MustNew(conversation.QueuedMessageIDPrefix)
	entity.Position = int64(len(q.items) + 1)
	q.items = append(q.items, entity)

	return entity, nil
}

func (q *memoryQueue) Update(
	_ context.Context,
	entity *conversation.QueuedMessage,
) (*conversation.QueuedMessage, error) {
	for idx, item := range q.items {
		if item.ID == entity.ID {
			q.items[idx] = entity
			return entity, nil
		}
	}

	return nil, errors.New("missing")
}

func (q *memoryQueue) MarkSteer(
	ctx context.Context,
	req *repositories.QueuedMessageRequest,
) (*conversation.QueuedMessage, error) {
	for _, item := range q.items {
		if item.ID == req.ID {
			item.Steer = true
		}
	}

	return q.Get(ctx, req)
}

func (q *memoryQueue) Delete(_ context.Context, req *repositories.QueuedMessageRequest) error {
	q.items = slices.DeleteFunc(q.items, func(item *conversation.QueuedMessage) bool {
		return item.ID == req.ID
	})

	return nil
}

func (q *memoryQueue) DeleteMany(
	_ context.Context,
	req *repositories.DeleteQueuedMessagesRequest,
) error {
	q.items = slices.DeleteFunc(q.items, func(item *conversation.QueuedMessage) bool {
		return slices.Contains(req.IDs, item.ID)
	})

	return nil
}

func (q *memoryQueue) Reorder(
	_ context.Context,
	req *repositories.ReorderQueuedMessagesRequest,
) ([]*conversation.QueuedMessage, error) {
	for _, item := range q.items {
		item.Position = int64(slices.Index(req.IDs, item.ID) + 1)
	}

	return q.sorted(), nil
}

func (q *memoryQueue) ClaimNext(
	_ context.Context,
	scope *repositories.AssistantQueueScope,
) (*conversation.QueuedMessage, error) {
	for _, item := range q.sorted() {
		if item.ThreadID == scope.ThreadID {
			q.items = slices.DeleteFunc(q.items, func(known *conversation.QueuedMessage) bool {
				return known.ID == item.ID
			})
			return item, nil
		}
	}

	return nil, repositories.ErrQueueEmpty
}

func (q *memoryQueue) Claim(
	_ context.Context,
	req *repositories.QueuedMessageRequest,
) (*conversation.QueuedMessage, error) {
	for _, item := range q.items {
		if item.ID == req.ID {
			q.items = slices.DeleteFunc(q.items, func(known *conversation.QueuedMessage) bool {
				return known.ID == req.ID
			})
			return item, nil
		}
	}

	return nil, repositories.ErrQueueEmpty
}

func (q *memoryQueue) Restore(_ context.Context, entity *conversation.QueuedMessage) error {
	q.restored = append(q.restored, entity.ID)
	q.items = append(q.items, entity)

	return nil
}

type fakeTurns struct {
	active   *conversation.AssistantTurn
	startErr error
	started  []assistantturnservice.StartRequest
}

func (f *fakeTurns) Active(
	context.Context,
	repositories.ActiveAssistantTurnRequest,
) (*conversation.AssistantTurn, error) {
	return f.active, nil
}

func (f *fakeTurns) StartTurn(
	_ context.Context,
	req assistantturnservice.StartRequest,
	start func(turn *conversation.AssistantTurn) (string, error),
) (*conversation.AssistantTurn, error) {
	if f.startErr != nil {
		return nil, f.startErr
	}
	turn := &conversation.AssistantTurn{
		ID:             pulid.MustNew("atrn_"),
		ThreadID:       req.ThreadID,
		UserID:         req.UserID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Origin:         req.Origin,
		Input:          req.Input,
	}
	if _, err := start(turn); err != nil {
		return nil, err
	}
	f.started = append(f.started, req)

	return turn, nil
}

type fakeRun struct{ id string }

func (r fakeRun) GetID() string                  { return r.id }
func (r fakeRun) GetRunID() string               { return "run" }
func (r fakeRun) GetFirstExecutionRunID() string { return "run" }
func (r fakeRun) Get(context.Context, any) error { return nil }
func (r fakeRun) GetWithOptions(context.Context, any, client.WorkflowRunGetOptions) error {
	return nil
}

type signal struct {
	workflowID string
	name       string
	steer      agentruntime.Steer
}

type fakeWorkflows struct {
	signals   []signal
	signalErr error
	starts    []any
}

func (f *fakeWorkflows) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	_ any,
	args ...any,
) (client.WorkflowRun, error) {
	f.starts = append(f.starts, args...)

	return fakeRun{id: options.ID}, nil
}

func (f *fakeWorkflows) CancelWorkflow(context.Context, string, string) error { return nil }

func (f *fakeWorkflows) SignalWorkflow(
	_ context.Context,
	workflowID, _, signalName string,
	arg any,
) error {
	if f.signalErr != nil {
		return f.signalErr
	}
	steer, _ := arg.(agentruntime.Steer)
	f.signals = append(f.signals, signal{workflowID: workflowID, name: signalName, steer: steer})

	return nil
}

func (f *fakeWorkflows) Enabled() bool { return true }

type ownedThreads struct{}

func (ownedThreads) GetThread(
	_ context.Context,
	req repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	return &conversation.Thread{ID: req.ID, UserID: req.UserID}, nil
}

type harness struct {
	svc       *Service
	queue     *memoryQueue
	turns     *fakeTurns
	workflows *fakeWorkflows
	scope     *Scope
}

func newHarness() *harness {
	h := &harness{
		queue:     &memoryQueue{},
		turns:     &fakeTurns{},
		workflows: &fakeWorkflows{},
		scope: &Scope{
			ThreadID: pulid.MustNew("athr_"),
			Actor: serviceports.RequestActor{
				PrincipalType:  serviceports.PrincipalTypeUser,
				PrincipalID:    pulid.MustNew("usr_"),
				UserID:         pulid.MustNew("usr_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
			},
		},
	}
	h.scope.Actor.PrincipalID = h.scope.Actor.UserID
	h.svc = &Service{
		l:             zap.NewNop(),
		queue:         h.queue,
		conversations: ownedThreads{},
		turns:         h.turns,
		workflows:     h.workflows,
	}

	return h
}

func (h *harness) running() *conversation.AssistantTurn {
	h.turns.active = &conversation.AssistantTurn{
		ID:       pulid.MustNew("atrn_"),
		ThreadID: h.scope.ThreadID,
		Origin:   conversation.AssistantTurnOriginPerson,
	}

	return h.turns.active
}

func (h *harness) settle(read []pulid.ID, dispatch bool) *serviceports.QueuedTurn {
	return h.svc.SettleQueue(context.Background(), &serviceports.SettleQueueRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: h.scope.Actor.OrganizationID,
			BuID:  h.scope.Actor.BusinessUnitID,
		},
		ThreadID: h.scope.ThreadID,
		UserID:   h.scope.Actor.UserID,
		Read:     read,
		Dispatch: dispatch,
	})
}

func TestEnqueue_SteersTheReplyUnderWay(t *testing.T) {
	t.Parallel()

	h := newHarness()
	active := h.running()

	outcome, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{
		Scope:   h.scope,
		Content: "  Actually the carrier is Werner.  ",
		Request: conversation.QueuedRequest{
			Mentions: []agent.EntityRef{{Type: "carrier", ID: pulid.MustNew("car_").String(), Label: "Werner"}},
		},
		Steer: true,
	})

	require.NoError(t, err)
	assert.True(t, outcome.Steering)
	assert.Nil(t, outcome.Turn)
	require.Len(t, h.workflows.signals, 1)
	sent := h.workflows.signals[0]
	assert.Equal(t, active.ExecutionID(), sent.workflowID)
	assert.Equal(t, agentflow.SteerSignal, sent.name)
	assert.Equal(t, outcome.Item.ID, sent.steer.ID, "the turn reads it under the row's id")
	assert.Equal(t, "Actually the carrier is Werner.", sent.steer.Content)
	assert.Len(t, sent.steer.Mentions, 1)
	assert.Len(t, h.queue.items, 1, "the row stays until the turn that read it is saved")
	assert.Empty(t, h.turns.started)
}

func TestEnqueue_ASteerTheReplyCannotTakeWaitsInTheQueue(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.running()
	h.workflows.signalErr = serviceerror.NewNotFound("workflow not found")

	outcome, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{
		Scope:   h.scope,
		Content: "Use the later slot.",
		Steer:   true,
	})

	require.NoError(t, err)
	assert.False(t, outcome.Steering)
	require.NotNil(t, outcome.Item)
	assert.Len(t, h.queue.items, 1)
	assert.Empty(t, h.turns.started, "the reply under way still holds the conversation")
}

func TestEnqueue_AFreeConversationSendsTheMessageAtOnce(t *testing.T) {
	t.Parallel()

	h := newHarness()
	providerID := pulid.MustNew("aip_")

	outcome, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{
		Scope:   h.scope,
		Content: "Re-rate the Acme loads.",
		Request: conversation.QueuedRequest{ProviderID: providerID, ProviderChosen: true},
		Steer:   true,
	})

	require.NoError(t, err)
	require.NotNil(t, outcome.Turn)
	assert.Nil(t, outcome.Item, "the message became the turn")
	assert.Empty(t, h.queue.items)
	require.Len(t, h.turns.started, 1)
	assert.Equal(t, conversation.AssistantTurnOriginPerson, h.turns.started[0].Origin)
	assert.Equal(t, "Re-rate the Acme loads.", h.turns.started[0].Input)
	assert.Empty(t, h.workflows.signals)
}

func TestEnqueue_RefusesFilesOnASteerAndAnEmptyMessage(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.running()

	_, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{
		Scope:   h.scope,
		Content: "Read this",
		Request: conversation.QueuedRequest{AttachmentDocumentIDs: []pulid.ID{pulid.MustNew("doc_")}},
		Steer:   true,
	})
	assert.True(t, errortypes.IsMultiError(err), "files go with a queued message, not a steer")

	_, err = h.svc.Enqueue(t.Context(), &EnqueueRequest{Scope: h.scope, Content: "   "})
	assert.True(t, errortypes.IsMultiError(err))
	assert.Empty(t, h.queue.items)
}

func TestSettleQueue_ClearsWhatWasReadAndSendsTheNextAfterACompletedTurn(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.running()
	read, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{Scope: h.scope, Content: "Steered", Steer: true})
	require.NoError(t, err)
	queued, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{Scope: h.scope, Content: "Then bill it"})
	require.NoError(t, err)
	h.turns.active = nil

	next := h.settle([]pulid.ID{read.Item.ID}, true)

	require.NotNil(t, next)
	assert.Equal(t, queued.Item.ID, next.QueuedID)
	assert.Equal(t, "Then bill it", next.Input)
	assert.Empty(t, h.queue.items)
	require.Len(t, h.turns.started, 1)
	assert.Equal(t, h.scope.Actor.UserID, h.turns.started[0].UserID, "sent as the conversation's owner")
}

func TestSettleQueue_AStoppedTurnHoldsTheQueue(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.running()
	_, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{Scope: h.scope, Content: "Then bill it"})
	require.NoError(t, err)
	h.turns.active = nil

	assert.Nil(t, h.settle(nil, false))
	assert.Len(t, h.queue.items, 1)
	assert.Empty(t, h.turns.started)
}

func TestSettleQueue_AMessageThatCannotStartGoesBack(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.running()
	queued, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{Scope: h.scope, Content: "Then bill it"})
	require.NoError(t, err)
	h.turns.active = nil
	h.turns.startErr = errortypes.NewBusinessError("This conversation is already working on a reply.")

	assert.Nil(t, h.settle(nil, true))
	assert.Equal(t, []pulid.ID{queued.Item.ID}, h.queue.restored)
	assert.Len(t, h.queue.items, 1)
}

func TestReorder_MustNameEveryWaitingMessageOnce(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.running()
	first, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{Scope: h.scope, Content: "One"})
	require.NoError(t, err)
	second, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{Scope: h.scope, Content: "Two"})
	require.NoError(t, err)

	_, err = h.svc.Reorder(t.Context(), &ReorderRequest{Scope: h.scope, IDs: []pulid.ID{second.Item.ID}})
	assert.True(t, errortypes.IsError(err))
	_, err = h.svc.Reorder(t.Context(), &ReorderRequest{
		Scope: h.scope,
		IDs:   []pulid.ID{second.Item.ID, second.Item.ID},
	})
	assert.True(t, errortypes.IsError(err))

	ordered, err := h.svc.Reorder(t.Context(), &ReorderRequest{
		Scope: h.scope,
		IDs:   []pulid.ID{second.Item.ID, first.Item.ID},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Two", "One"}, []string{ordered[0].Content, ordered[1].Content})
}

func TestEdit_AMessageHandedToTheReplyIsTheReplys(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.running()
	handed, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{Scope: h.scope, Content: "Werner", Steer: true})
	require.NoError(t, err)

	_, err = h.svc.Edit(t.Context(), &EditRequest{Scope: h.scope, ID: handed.Item.ID, Content: "Schneider"})
	assert.True(t, errortypes.IsBusinessError(err))

	h.turns.active = nil
	edited, err := h.svc.Edit(t.Context(), &EditRequest{Scope: h.scope, ID: handed.Item.ID, Content: "Schneider"})
	require.NoError(t, err)
	assert.Equal(t, "Schneider", edited.Content)
	assert.False(t, edited.Steer, "once the reply ended it is an ordinary waiting message")
}

func TestSendNow_RefusesToSteerWithFiles(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.running()
	queued, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{
		Scope:   h.scope,
		Content: "Read the POD",
		Request: conversation.QueuedRequest{AttachmentDocumentIDs: []pulid.ID{pulid.MustNew("doc_")}},
	})
	require.NoError(t, err)

	_, err = h.svc.SendNow(t.Context(), &ItemRequest{Scope: h.scope, ID: queued.Item.ID})
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, h.workflows.signals)
}

// A step the person handed another agent is a task for that agent, never a
// word to the reply under way: asked to steer, it waits in the queue instead.
func TestEnqueue_AStepForAnotherAgentWaitsRatherThanSteers(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.running()
	agentID := pulid.MustNew("agdef_")

	outcome, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{
		Scope:   h.scope,
		Content: "Mark shipment SEED-PAY-001 ready to invoice.",
		Request: conversation.QueuedRequest{DirectedAgentID: agentID},
		Steer:   true,
	})

	require.NoError(t, err)
	assert.False(t, outcome.Steering)
	require.NotNil(t, outcome.Item)
	assert.False(t, outcome.Item.Steer)
	assert.Equal(t, agentID, outcome.Item.Request.DirectedAgentID)
	assert.Empty(t, h.workflows.signals)
	assert.Empty(t, h.turns.started, "the reply under way still holds the conversation")
}

func TestSendNow_RefusesToSteerAStepForAnotherAgent(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.running()
	queued, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{
		Scope:   h.scope,
		Content: "Mark shipment SEED-PAY-001 ready to invoice.",
		Request: conversation.QueuedRequest{DirectedAgentID: pulid.MustNew("agdef_")},
	})
	require.NoError(t, err)

	_, err = h.svc.SendNow(t.Context(), &ItemRequest{Scope: h.scope, ID: queued.Item.ID})
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, h.workflows.signals)
}

// Sent once the conversation is free, the step still goes to the agent the
// person handed it to.
func TestEnqueue_AStepForAnotherAgentStartsItsTurnDirected(t *testing.T) {
	t.Parallel()

	h := newHarness()
	agentID := pulid.MustNew("agdef_")

	outcome, err := h.svc.Enqueue(t.Context(), &EnqueueRequest{
		Scope:   h.scope,
		Content: "Mark shipment SEED-PAY-001 ready to invoice.",
		Request: conversation.QueuedRequest{DirectedAgentID: agentID},
	})

	require.NoError(t, err)
	require.NotNil(t, outcome.Turn)
	require.Len(t, h.workflows.starts, 1)
	payload, ok := h.workflows.starts[0].(*assistantjobs.AssistantTurnPayload)
	require.True(t, ok, "the turn's workflow is started with its payload")
	assert.Equal(t, agentID, payload.Request.DirectedAgentID)
	assert.Equal(t, "Mark shipment SEED-PAY-001 ready to invoice.", payload.Content)
}
