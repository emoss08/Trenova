package deskqueueresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/deskmapping"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantqueueservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationqueue"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

type queueList struct {
	Items []*conversationqueue.QueuedMessage `json:"items"`
}

func outcomeOf(outcome *assistantqueueservice.Outcome) *gqlmodel.AssistantQueueOutcome {
	model := &gqlmodel.AssistantQueueOutcome{
		Steering: outcome.Steering,
		Held:     outcome.Held,
	}
	if outcome.Item != nil {
		model.Item = outcome.Item
	}
	if outcome.Turn != nil {
		model.Turn = deskmapping.StartedTurn(outcome.Turn)
	}

	return model
}

func scopeOf(authCtx *authctx.AuthContext, threadID string) (*assistantqueueservice.Scope, error) {
	thread, err := gqlid.ParseIDField("threadId", threadID)
	if err != nil {
		return nil, err
	}

	return &assistantqueueservice.Scope{
		ThreadID: thread,
		Actor:    *actorutil.FromAuthContext(authCtx),
	}, nil
}

func (r *MutationResolver) queueScope(
	ctx context.Context,
	operation permission.Operation,
	threadID string,
) (*assistantqueueservice.Scope, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAssistant, operation)
	if err != nil {
		return nil, err
	}

	return scopeOf(authCtx, threadID)
}

func (r *MutationResolver) queueItem(
	ctx context.Context,
	operation permission.Operation,
	threadID, itemID string,
) (*assistantqueueservice.Scope, pulid.ID, error) {
	scope, err := r.queueScope(ctx, operation, threadID)
	if err != nil {
		return nil, pulid.Nil, err
	}
	item, err := gqlid.ParseIDField("itemId", itemID)
	if err != nil {
		return nil, pulid.Nil, err
	}

	return scope, item, nil
}
