package assistantthreads

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStartThread_ASecondQuestionAboutARecordContinuesItsConversation(t *testing.T) {
	t.Parallel()

	actor := assistanttest.Actor()
	desk := assistanttest.RestrictedChatDefinition("Credential desk")
	worker := pulid.MustNew("wrk_")
	earlier := &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		UserID:            actor.UserID,
		AgentDefinitionID: desk.ID,
		Origin:            conversation.ThreadOriginDesk,
		SubjectType:       agent.SubjectWorker,
		SubjectID:         worker,
		LastMessageAt:     100,
	}
	created := &assistanttest.CreatingConversations{}
	svc := &Service{
		logger:        zap.NewNop(),
		conversations: created,
		definitions: &assistanttest.DelegateDefinitions{
			ByID: map[pulid.ID]*agentdefinition.Definition{desk.ID: desk},
		},
		permissions: &assistanttest.SubjectPermissions{
			Allowed: map[string]bool{"assistant:create": true, "worker:read": true},
			Granted: []pulid.ID{desk.ID},
		},
		pageThreads: &assistanttest.PageThreads{
			Live: map[conversationports.PageThreadKey]*conversation.Thread{
				{SubjectID: worker}: earlier,
			},
		},
		cases: assistanttest.NoCases{},
	}
	ask := func(req *assistantports.StartThreadRequest) *conversation.Thread {
		req.AgentDefinitionID = desk.ID
		thread, err := startAs(t, svc, req, actor)
		require.NoError(t, err)

		return thread
	}

	again := ask(&assistantports.StartThreadRequest{
		Origin:      conversation.ThreadOriginDesk,
		SubjectType: agent.SubjectWorker,
		SubjectID:   worker,
	})
	assert.Equal(t, earlier.ID, again.ID, "the record's conversation is continued, not opened twice")

	elsewhere := ask(&assistantports.StartThreadRequest{
		Origin:      conversation.ThreadOriginDesk,
		SubjectType: agent.SubjectWorker,
		SubjectID:   pulid.MustNew("wrk_"),
	})
	assert.NotEqual(t, earlier.ID, elsewhere.ID, "another record has its own conversation")

	handed := ask(&assistantports.StartThreadRequest{
		Origin:             conversation.ThreadOriginDesk,
		SubjectType:        agent.SubjectWorker,
		SubjectID:          worker,
		HandedFromThreadID: earlier.ID,
	})
	assert.NotEqual(t, earlier.ID, handed.ID, "a hand-off always opens its own conversation")
}
