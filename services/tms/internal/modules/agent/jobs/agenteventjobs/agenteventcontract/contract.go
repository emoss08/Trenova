package agenteventcontract

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// KindPrefix starts every agent's event queue: one queue per agent, so
	// each drains against its own cap.
	KindPrefix        = "agent-event:"
	DrainWorkflowName = "AgentEventDrainWorkflow"
	DrainSignalName   = "agent-event-drain"
	drainSummary      = "Start an agent's queued event runs, as many as it may run at once"
)

// Kind is an agent's event queue.
func Kind(definitionID pulid.ID) string { return KindPrefix + definitionID.String() }

// DefinitionOf is the agent a queue kind belongs to.
func DefinitionOf(kind string) (pulid.ID, bool) {
	raw, ok := strings.CutPrefix(kind, KindPrefix)
	if !ok || raw == "" {
		return pulid.Nil, false
	}

	return pulid.ID(raw), true
}

type QueueParams struct {
	fx.In

	Queue   workqueueports.Queue
	Signals kernelports.WorkflowSignalStarter
	Logger  *zap.Logger
}

// Queue queues an agent's runs on events: the subject is the row, the event
// kind its detail, so a subject queued twice runs once, on the latest event.
type Queue struct {
	queue   workqueueports.Queue
	signals kernelports.WorkflowSignalStarter
	l       *zap.Logger
}

var _ agenteventports.EventRunQueue = (*Queue)(nil)

func NewQueue(p QueueParams) *Queue {
	return &Queue{queue: p.Queue, signals: p.Signals, l: p.Logger.Named("agent-event-queue")}
}

func (q *Queue) drain(definitionID pulid.ID) *temporaljobs.DrainRequester {
	return &temporaljobs.DrainRequester{
		Queue:     q.queue,
		Signals:   q.signals,
		Kind:      Kind(definitionID),
		Workflow:  DrainWorkflowName,
		TaskQueue: temporaltype.TaskQueueAgentBackground.String(),
		Signal:    DrainSignalName,
		Summary:   drainSummary,
		Logger:    q.l,
	}
}

func (q *Queue) Enqueue(ctx context.Context, req *agenteventports.EventRunRequest) error {
	if req.DefinitionID.IsNil() {
		return nil
	}

	return q.drain(req.DefinitionID).
		RequestWithDetail(ctx, req.TenantInfo, string(req.Kind), req.SubjectID)
}

// WakeDue wakes every agent's drain with queued events, across tenants: the
// backstop for a wake that was lost.
func (q *Queue) WakeDue(ctx context.Context, limit int) (int, error) {
	due, err := q.queue.ListDueKinds(ctx, &workqueueports.ListDueKindsRequest{
		Prefix: KindPrefix,
		Now:    timeutils.NowUnix(),
		Limit:  limit,
	})
	if err != nil {
		return 0, err
	}

	woken := 0
	for _, item := range due {
		definitionID, ok := DefinitionOf(item.Kind)
		if !ok {
			continue
		}
		if wErr := q.drain(definitionID).Wake(ctx, item.TenantInfo); wErr != nil {
			q.l.Warn("could not wake an agent's event drain from the sweep",
				zap.String("kind", item.Kind),
				zap.String("organizationId", item.TenantInfo.OrgID.String()),
				zap.Error(wErr))
			continue
		}
		woken++
	}

	return woken, nil
}
