package agentports

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingAgentEventPublisher struct {
	events []AgentEvent
}

func (p *recordingAgentEventPublisher) Publish(_ context.Context, event AgentEvent) {
	p.events = append(p.events, event)
}

func TestPublishAgentEventPublishesAtOnceOutsideATransaction(t *testing.T) {
	t.Parallel()

	publisher := &recordingAgentEventPublisher{}
	event := AgentEvent{Kind: agent.EventShipmentCreated, SubjectID: pulid.MustNew("shp_")}

	PublishAgentEvent(t.Context(), publisher, event)

	require.Len(t, publisher.events, 1)
	assert.Equal(t, event, publisher.events[0])
}

func TestPublishAgentEventWaitsForTheTransactionToCommit(t *testing.T) {
	t.Parallel()

	publisher := &recordingAgentEventPublisher{}
	ctx, hooks := kernelports.WithAfterCommitHooks(t.Context())
	event := AgentEvent{Kind: agent.EventShipmentCreated, SubjectID: pulid.MustNew("shp_")}

	PublishAgentEvent(ctx, publisher, event)
	assert.Empty(t, publisher.events)

	hooks.Run(t.Context())

	require.Len(t, publisher.events, 1)
	assert.Equal(t, event, publisher.events[0])
}

func TestPublishAgentEventIsDroppedWhenTheTransactionRollsBack(t *testing.T) {
	t.Parallel()

	publisher := &recordingAgentEventPublisher{}
	ctx, _ := kernelports.WithAfterCommitHooks(t.Context())

	PublishAgentEvent(ctx, publisher, AgentEvent{Kind: agent.EventShipmentCreated})

	assert.Empty(t, publisher.events)
}

func TestPublishAgentEventToleratesANilPublisher(t *testing.T) {
	t.Parallel()

	ctx, hooks := kernelports.WithAfterCommitHooks(t.Context())

	assert.NotPanics(t, func() {
		PublishAgentEvent(ctx, nil, AgentEvent{Kind: agent.EventShipmentCreated})
		hooks.Run(t.Context())
	})
}
