package deskqueueresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantqueueservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationqueue"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

/*
Writing a message that will be sent is writing to the conversation, gated like
sending one; taking one back is arranging one's own queue.
*/
func TestQueueOperationsCheckTheRESTGrants(t *testing.T) {
	t.Parallel()

	threadID, itemID := pulid.MustNew("athr_").String(), pulid.MustNew("aqm_").String()
	cases := []struct {
		name      string
		operation permission.Operation
		call      func(r *MutationResolver) error
	}{
		{"enqueue", permission.OpCreate, func(r *MutationResolver) error {
			_, err := r.EnqueueAssistantMessage(
				resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
				threadID,
				gqlmodel.EnqueueAssistantMessageInput{Content: "next"},
			)
			return err
		}},
		{"send", permission.OpCreate, func(r *MutationResolver) error {
			_, err := r.SendAssistantQueuedMessage(
				resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
				threadID,
				itemID,
			)
			return err
		}},
		{"remove", permission.OpRead, func(r *MutationResolver) error {
			_, err := r.RemoveAssistantQueuedMessage(
				resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
				threadID,
				itemID,
			)
			return err
		}},
		{"reorder", permission.OpRead, func(r *MutationResolver) error {
			_, err := r.ReorderAssistantQueue(
				resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
				threadID,
				[]string{itemID},
			)
			return err
		}},
	}

	for _, tc := range cases {
		engine := resolvertest.RefusingEngine()
		err := tc.call(&MutationResolver{&Deps{Core: &resolvercore.Core{
			L:                zap.NewNop(),
			PermissionEngine: engine,
		}}})

		resolvertest.RequireRefused(t, err, engine, permission.ResourceAssistant, tc.operation)
	}
}

func TestQueuedRequestKeepsAnExplicitChoiceOfTheOrganizationsOrder(t *testing.T) {
	t.Parallel()

	empty := ""
	request, err := queuedRequest(&gqlmodel.EnqueueAssistantMessageInput{
		Content:    "next",
		ProviderID: &empty,
	})

	require.NoError(t, err)
	assert.True(t, request.ProviderChosen, "an empty choice is the organization's order, chosen")
	assert.True(t, request.ProviderID.IsNil())

	request, err = queuedRequest(&gqlmodel.EnqueueAssistantMessageInput{Content: "next"})
	require.NoError(t, err)
	assert.False(t, request.ProviderChosen, "no choice leaves the conversation's model as it is")
}

func TestOutcomeOfCarriesTheTurnToFollow(t *testing.T) {
	t.Parallel()

	turn := &conversationturn.AssistantTurn{
		ID:       pulid.MustNew("atrn_"),
		ThreadID: pulid.MustNew("athr_"),
		Status:   conversation.AssistantTurnStatusPending,
	}

	sent := outcomeOf(&assistantqueueservice.Outcome{Turn: turn})
	require.NotNil(t, sent.Turn)
	assert.Equal(t, turn.ID.String(), sent.Turn.TurnID)
	assert.Equal(t, "/api/v1/assistant/turns/"+turn.ID.String()+"/stream/", sent.Turn.StreamURL)
	assert.Nil(t, sent.Item)

	item := &conversationqueue.QueuedMessage{ID: pulid.MustNew("aqm_")}
	held := outcomeOf(&assistantqueueservice.Outcome{Item: item, Held: "busy"})
	assert.Same(t, item, held.Item)
	assert.Equal(t, "busy", held.Held)
	assert.Nil(t, held.Turn)
}
