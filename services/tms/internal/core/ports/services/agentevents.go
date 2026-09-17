package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AgentEvent struct {
	Kind       agent.EventKind
	SubjectID  pulid.ID
	TenantInfo pagination.TenantInfo
}

type AgentEventPublisher interface {
	Publish(ctx context.Context, event AgentEvent)
}

func PublishAgentEvent(ctx context.Context, publisher AgentEventPublisher, event AgentEvent) {
	if publisher == nil {
		return
	}

	publisher.Publish(ctx, event)
}
