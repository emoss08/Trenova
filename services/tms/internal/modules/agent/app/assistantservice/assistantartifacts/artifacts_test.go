package assistantartifacts

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// A draft never holds its own decision state. Listing reads the status off
// the proposal, so a draft sent from AI Control reads as sent in the Desk.
func TestListThreadArtifacts_DraftStatusFollowsTheProposal(t *testing.T) {
	t.Parallel()

	thread := &conversation.Thread{ID: pulid.MustNew("thr_")}
	sent := pulid.MustNew("ap_")
	rejected := pulid.MustNew("ap_")
	repo := &assistanttest.StubArtifactRepo{Listed: []*assistantartifact.Artifact{
		{
			ID:         pulid.MustNew("art_"),
			Kind:       assistantartifact.KindEmailDraft,
			Status:     assistantartifact.StatusPending,
			ProposalID: sent,
		},
		{
			ID:         pulid.MustNew("art_"),
			Kind:       assistantartifact.KindEmailDraft,
			Status:     assistantartifact.StatusPending,
			ProposalID: rejected,
		},
		{
			ID:     pulid.MustNew("art_"),
			Kind:   assistantartifact.KindEntityCard,
			Status: assistantartifact.StatusReady,
		},
	}}
	svc := &Service{
		logger:        zap.NewNop(),
		artifacts:     repo,
		conversations: &assistanttest.StubConversationRepo{Thread: thread},
		proposals: &assistanttest.StubProposalRepo{ByThread: []*agent.AgentProposal{
			{ID: sent, Status: agent.ProposalStatusExecuted},
			{ID: rejected, Status: agent.ProposalStatusRejected},
		}},
	}

	page, err := svc.ListThreadArtifacts(
		t.Context(),
		&conversationports.GetThreadRequest{ID: thread.ID},
		assistantartifactports.ListArtifactsOptions{},
	)
	require.NoError(t, err)
	listed := page.Results
	require.Len(t, listed, 3)
	assert.Equal(t, assistantartifact.StatusSent, listed[0].Status)
	assert.Equal(t, assistantartifact.StatusFailed, listed[1].Status)
	assert.Equal(t, assistantartifact.StatusReady, listed[2].Status)
}
