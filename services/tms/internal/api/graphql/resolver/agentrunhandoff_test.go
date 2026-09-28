package resolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type threadAgentsRepo struct {
	repositories.ConversationRepository

	byID  map[pulid.ID]*conversation.Thread
	reads int
}

func (r *threadAgentsRepo) ListThreadAgentsByIDs(
	_ context.Context,
	req repositories.ListThreadAgentsByIDsRequest,
) ([]*conversation.Thread, error) {
	r.reads++
	out := make([]*conversation.Thread, 0, len(req.IDs))
	for _, id := range req.IDs {
		if thread, ok := r.byID[id]; ok {
			out = append(out, thread)
		}
	}

	return out, nil
}

func handOffLoaders(
	t *testing.T,
	threads *threadAgentsRepo,
	definitions *delegateDefinitionsRepo,
) context.Context {
	t.Helper()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	return loaders.WithLoaders(t.Context(), &loaders.Loaders{
		ThreadAgentByID: loaders.NewThreadAgentByIDLoaderFactory(
			loaders.ThreadAgentByIDLoaderFactoryParams{Conversations: threads},
		).NewForTenant(tenant),
		AgentDefinitionByID: loaders.NewAgentDefinitionByIDLoaderFactory(
			loaders.AgentDefinitionByIDLoaderFactoryParams{Definitions: definitions},
		).NewForTenant(tenant),
	})
}

/*
#628: AI Control lists the run of an agent a conversation's agent handed a
task to, and says who handed it over: the agent of the conversation the run
belongs to, read for a page of runs in one query each.
*/
func TestAgentRunHandedBy_NamesTheAgentOfTheConversationThatAsked(t *testing.T) {
	t.Parallel()

	dispatch := &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Dispatch"}
	desk := &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Shipment Desk"}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_"), AgentDefinitionID: dispatch.ID}
	threads := &threadAgentsRepo{byID: map[pulid.ID]*conversation.Thread{thread.ID: thread}}
	definitions := &delegateDefinitionsRepo{byID: map[pulid.ID]*agentdefinition.Definition{
		dispatch.ID: dispatch,
		desk.ID:     desk,
	}}
	ctx := handOffLoaders(t, threads, definitions)

	handed := &agent.AgentRun{
		AgentDefinitionID: desk.ID,
		SubjectType:       agent.SubjectAssistantThread,
		SubjectID:         thread.ID,
		ParentOwnerKind:   agent.RunOwnerAssistantTurn,
		ParentOwnerID:     pulid.MustNew("atrn_"),
		DelegateCallID:    "call_hand",
	}
	by, err := agentRunHandedBy(ctx, handed)
	require.NoError(t, err)
	require.NotNil(t, by)
	assert.Equal(t, "Dispatch", by.Name)

	own := &agent.AgentRun{
		AgentDefinitionID: dispatch.ID,
		SubjectType:       agent.SubjectAssistantThread,
		SubjectID:         thread.ID,
	}
	by, err = agentRunHandedBy(ctx, own)
	require.NoError(t, err)
	assert.Nil(t, by, "the conversation's own run was handed nothing")
	assert.Equal(t, 1, threads.reads)

	gone := *handed
	gone.SubjectID = pulid.MustNew("athr_")
	by, err = agentRunHandedBy(ctx, &gone)
	require.NoError(t, err)
	assert.Nil(t, by, "a conversation deleted since names nobody")
}

func TestAgentRunParentOwnerKind_IsAbsentForARunNothingHandedATask(t *testing.T) {
	t.Parallel()

	kind, err := agentRunParentOwnerKind(&agent.AgentRun{})
	require.NoError(t, err)
	assert.Nil(t, kind)

	kind, err = agentRunParentOwnerKind(&agent.AgentRun{ParentOwnerKind: agent.RunOwnerAssistantTurn})
	require.NoError(t, err)
	require.NotNil(t, kind)
	assert.Equal(t, gqlmodel.AgentRunEventOwnerKindAssistantTurn, *kind)
}
