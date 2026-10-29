package agentports

import (
	"context"

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
	Publish(ctx context.Context, event AgentEvent)
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
		publisher.Publish(committedCtx, event)
	})
}
