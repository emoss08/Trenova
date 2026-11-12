package agenteventports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AgentEvent struct {
	Kind       agenttrigger.EventKind
	SubjectID  pulid.ID
	TenantInfo pagination.TenantInfo
	// Related are other records the event is about, such as the stop a move
	// arrived at or the carrier a message came from. A wait watching one of
	// them is told. Agents woken by the event still read only the subject.
	Related []pulid.ID
	// Detail says what happened in a line, for a wait the event ends.
	Detail string
}

type AgentEventPublisher interface {
	Publish(ctx context.Context, event *AgentEvent)
}

// AgentEventObserver is told of every event an organization's plan lets
// automation act on, for work that follows an event without being an agent
// run, such as planning dispatch coverage again.
type AgentEventObserver interface {
	ObserveAgentEvent(ctx context.Context, event *AgentEvent)
}

// EventRunQueue starts an agent's runs on events, queued per agent so it never
// has more runs working at once than it allows (MaxConcurrentRuns). A burst
// of events waits its turn instead of starting every run at once.
type EventRunQueue interface {
	Enqueue(ctx context.Context, req *EventRunRequest) error
}

type EventRunRequest struct {
	TenantInfo   pagination.TenantInfo
	DefinitionID pulid.ID
	Kind         agenttrigger.EventKind
	SubjectID    pulid.ID
}

// EventRoutine takes an event for an agent in code, before any model run:
// the routine part of a job, such as asking for a missing proof of delivery,
// which the agent's model is kept for when the routine cannot do it.
type EventRoutine interface {
	Handles(definition *agentdefinition.Definition, kind agenttrigger.EventKind) bool
	Take(ctx context.Context, definition *agentdefinition.Definition, event *AgentEvent) error
}

// AttachmentExtractionObserver is told when a document's extraction is saved,
// so a file a person attached in a conversation has its card shown there.
type AttachmentExtractionObserver interface {
	AttachmentExtracted(ctx context.Context, event *AgentEvent)
}

func PublishAgentEvent(ctx context.Context, publisher AgentEventPublisher, event AgentEvent) {
	if publisher == nil {
		return
	}

	kernelports.AfterCommit(ctx, func(committedCtx context.Context) {
		publisher.Publish(committedCtx, &event)
	})
}
