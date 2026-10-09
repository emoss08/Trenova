package deskbench

import (
	"context"
	"sync"
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
)

type StartedTurn struct {
	At       time.Time
	TurnID   pulid.ID
	ThreadID pulid.ID
	Content  string
	Request  assistantjobs.AssistantTurnRequest
	Run      client.WorkflowRun
}

type turnQueue struct {
	mu    sync.Mutex
	items []StartedTurn
	ready chan struct{}
}

func newTurnQueue() *turnQueue {
	return &turnQueue{ready: make(chan struct{}, 1)}
}

func (q *turnQueue) push(turn StartedTurn) {
	q.mu.Lock()
	q.items = append(q.items, turn)
	q.mu.Unlock()

	select {
	case q.ready <- struct{}{}:
	default:
	}
}

func (q *turnQueue) pop() (StartedTurn, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return StartedTurn{}, false
	}
	turn := q.items[0]
	q.items = q.items[1:]

	return turn, true
}

func (q *turnQueue) next(ctx context.Context, wait <-chan time.Time) (StartedTurn, bool) {
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
	serviceports.WorkflowStarter

	mu     sync.Mutex
	queues map[pulid.ID]*turnQueue
}

func NewTurnWatcher(inner serviceports.WorkflowStarter) *TurnWatcher {
	return &TurnWatcher{
		WorkflowStarter: inner,
		queues:          make(map[pulid.ID]*turnQueue),
	}
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

	if name, ok := workflow.(string); !ok || name != assistantjobs.AssistantTurnWorkflowName {
		return run, nil
	}
	if len(args) != 1 {
		return run, nil
	}
	payload, ok := args[0].(*assistantjobs.AssistantTurnPayload)
	if !ok || payload == nil {
		return run, nil
	}

	w.mu.Lock()
	queue := w.queues[payload.ThreadID]
	w.mu.Unlock()
	if queue != nil {
		queue.push(StartedTurn{
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

func (w *TurnWatcher) watch(threadID pulid.ID) *turnQueue {
	w.mu.Lock()
	defer w.mu.Unlock()

	queue, ok := w.queues[threadID]
	if !ok {
		queue = newTurnQueue()
		w.queues[threadID] = queue
	}

	return queue
}

func (w *TurnWatcher) forget(threadID pulid.ID) {
	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.queues, threadID)
}
