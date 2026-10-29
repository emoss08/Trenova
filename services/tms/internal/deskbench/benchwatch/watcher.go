package benchwatch

import (
	"context"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentjobs/agentcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/assistantjobs/assistantcontract"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
)

type StartedTurn struct {
	At       time.Time
	TurnID   pulid.ID
	ThreadID pulid.ID
	Content  string
	Request  assistantcontract.AssistantTurnRequest
	Run      client.WorkflowRun
}

type TurnQueue struct {
	mu    sync.Mutex
	items []StartedTurn
	ready chan struct{}
}

func newTurnQueue() *TurnQueue {
	return &TurnQueue{ready: make(chan struct{}, 1)}
}

func (q *TurnQueue) push(turn *StartedTurn) {
	q.mu.Lock()
	q.items = append(q.items, *turn)
	q.mu.Unlock()

	select {
	case q.ready <- struct{}{}:
	default:
	}
}

func (q *TurnQueue) pop() (StartedTurn, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return StartedTurn{}, false
	}
	turn := q.items[0]
	q.items = q.items[1:]

	return turn, true
}

func (q *TurnQueue) Next(ctx context.Context, wait <-chan time.Time) (StartedTurn, bool) {
	for {
		if turn, ok := q.pop(); ok {
			return turn, true
		}
		select {
		case <-ctx.Done():
			return StartedTurn{}, false
		case <-wait:
			return q.pop()
		case <-q.ready:
		}
	}
}

type TurnWatcher struct {
	kernelports.WorkflowStarter

	mu     sync.Mutex
	queues map[pulid.ID]*TurnQueue
	runs   map[pulid.ID]client.WorkflowRun
}

func New(inner kernelports.WorkflowStarter) *TurnWatcher {
	return &TurnWatcher{
		WorkflowStarter: inner,
		queues:          make(map[pulid.ID]*TurnQueue),
		runs:            make(map[pulid.ID]client.WorkflowRun),
	}
}

func (w *TurnWatcher) Workflow(runID pulid.ID) (client.WorkflowRun, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	started, ok := w.runs[runID]
	if ok {
		delete(w.runs, runID)
	}

	return started, ok
}

func (w *TurnWatcher) noteRun(run client.WorkflowRun, args []any) {
	if len(args) != 1 {
		return
	}
	payload, ok := args[0].(*agentcontract.AgentRunPayload)
	if !ok || payload == nil {
		return
	}

	w.mu.Lock()
	w.runs[payload.RunID] = run
	w.mu.Unlock()
}

func (w *TurnWatcher) StartWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow any,
	args ...any,
) (client.WorkflowRun, error) {
	startedAt := time.Now()
	run, err := w.WorkflowStarter.StartWorkflow(ctx, options, workflow, args...)
	if err != nil {
		return run, err
	}

	name, _ := workflow.(string)
	if name == agentcontract.AgentRunWorkflowName {
		w.noteRun(run, args)
		return run, nil
	}
	if name != assistantcontract.AssistantTurnWorkflowName {
		return run, nil
	}
	if len(args) != 1 {
		return run, nil
	}
	payload, ok := args[0].(*assistantcontract.AssistantTurnPayload)
	if !ok || payload == nil {
		return run, nil
	}

	w.mu.Lock()
	queue := w.queues[payload.ThreadID]
	w.mu.Unlock()
	if queue != nil {
		queue.push(&StartedTurn{
			At:       startedAt,
			TurnID:   payload.TurnID,
			ThreadID: payload.ThreadID,
			Content:  payload.Content,
			Request:  payload.Request,
			Run:      run,
		})
	}

	return run, nil
}

func (w *TurnWatcher) Watch(threadID pulid.ID) *TurnQueue {
	w.mu.Lock()
	defer w.mu.Unlock()

	queue, ok := w.queues[threadID]
	if !ok {
		queue = newTurnQueue()
		w.queues[threadID] = queue
	}

	return queue
}

func (w *TurnWatcher) Forget(threadID pulid.ID) {
	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.queues, threadID)
}
