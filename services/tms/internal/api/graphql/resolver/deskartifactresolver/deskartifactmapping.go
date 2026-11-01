package deskartifactresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/deskmapping"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

type results struct {
	Results any `json:"results"`
}

func (r *MutationResolver) artifactRequest(
	ctx context.Context,
	threadID, artifactID string,
) (conversationports.GetThreadRequest, pulid.ID, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAssistant, permission.OpRead)
	if err != nil {
		return conversationports.GetThreadRequest{}, pulid.Nil, err
	}
	thread, err := deskmapping.ThreadRequest(authCtx, "threadId", threadID)
	if err != nil {
		return conversationports.GetThreadRequest{}, pulid.Nil, err
	}
	artifact, err := gqlid.ParseIDField("artifactId", artifactID)
	if err != nil {
		return conversationports.GetThreadRequest{}, pulid.Nil, err
	}

	return thread, artifact, nil
}
