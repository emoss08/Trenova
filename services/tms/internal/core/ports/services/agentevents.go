package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AgentEvent struct {
	Kind       agent.EventKind
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

func PublishAgentEvent(ctx context.Context, publisher AgentEventPublisher, event AgentEvent) {
	if publisher == nil {
		return
	}

	ports.AfterCommit(ctx, func(committedCtx context.Context) {
		publisher.Publish(committedCtx, event)
	})
}
