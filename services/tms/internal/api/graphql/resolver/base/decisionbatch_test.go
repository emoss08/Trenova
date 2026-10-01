package base

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchDecisionRequest_CarriesTheNoteToTheService(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ap_")
	note := "These customers are on credit hold."
	reason := "rejected_in_conversation"

	req, err := BatchDecisionRequest([]string{id.String()}, &gqlmodel.DecideAgentProposalsInput{
		Decision:   agent.DecisionRejected,
		ReasonCode: &reason,
		Note:       &note,
	}, pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Equal(t, note, req.Note)
	assert.Equal(t, reason, req.ReasonCode)
	assert.Equal(t, []pulid.ID{id}, req.ProposalIDs)
}

func TestBatchDecisionRequest_LeavesTheNoteEmptyWhenAbsent(t *testing.T) {
	t.Parallel()

	req, err := BatchDecisionRequest(
		[]string{pulid.MustNew("ap_").String()},
		&gqlmodel.DecideAgentProposalsInput{Decision: agent.DecisionAccepted},
		pagination.TenantInfo{},
	)
	require.NoError(t, err)

	assert.Empty(t, req.Note)
}
