package agenttrustservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type readyTrustRepo struct {
	*fakeTrustRepo
}

func (r readyTrustRepo) ListReady(
	_ context.Context,
	req repositories.ListReadyToolTrustRequest,
) ([]*agent.ToolTrust, error) {
	if r.row.Streak < req.MinStreak {
		return nil, nil
	}
	read := *r.row
	return []*agent.ToolTrust{&read}, nil
}

type listingDefinitions struct {
	*fakeDefinitions
}

func (d listingDefinitions) ListByIDs(
	context.Context,
	repositories.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	return []*agentdefinition.Definition{d.definition}, nil
}

func readyHarness(t *testing.T, streak int, ceiling agent.AutonomyTier) *harness {
	t.Helper()

	h := newHarness(t, harnessOptions{
		ceiling: ceiling,
		row:     &agent.ToolTrust{Streak: streak, Version: 4},
	})
	h.trust.row.ToolName = "assign_move"
	h.trust.row.AgentDefinitionID = h.definitions.definition.ID
	h.svc.trust = readyTrustRepo{h.trust}
	h.svc.definitions = listingDefinitions{h.definitions}
	return h
}

func readyRequest(h *harness, threshold int) *services.PromoteReadyRequest {
	return &services.PromoteReadyRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: h.definitions.definition.OrganizationID,
			BuID:  h.definitions.definition.BusinessUnitID,
		},
		Threshold: threshold,
	}
}

func TestPromotionCandidatesNameTheToolAndItsNextTier(t *testing.T) {
	t.Parallel()

	h := readyHarness(t, 12, agent.TierAutoExecute)

	candidates, err := h.svc.PromotionCandidates(t.Context(), readyRequest(h, 10))

	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, "Dispatch coverage", candidates[0].AgentName)
	assert.Equal(t, agent.TierActWithApproval, candidates[0].From)
	assert.Equal(t, agent.TierAutoExecute, candidates[0].To)
	assert.Empty(t, h.trust.marks, "a preview changes nothing")
}

func TestPromotionCandidatesStopAtTheCeiling(t *testing.T) {
	t.Parallel()

	h := readyHarness(t, 12, agent.TierActWithApproval)

	candidates, err := h.svc.PromotionCandidates(t.Context(), readyRequest(h, 10))

	require.NoError(t, err)
	assert.Empty(t, candidates)
}

func TestPromoteReadyMovesTheToolAndStartsAFreshStreak(t *testing.T) {
	t.Parallel()

	h := readyHarness(t, 12, agent.TierAutoExecute)

	promoted, err := h.svc.PromoteReady(t.Context(), readyRequest(h, 10))

	require.NoError(t, err)
	require.Len(t, promoted, 1)
	require.Len(t, h.trust.marks, 1)
	assert.True(t, h.trust.marks[0].Promoted)
	require.Len(t, h.definitions.tiers, 1)
	assert.Equal(t, agent.TierAutoExecute, h.definitions.tiers[0].Tier)
}

func TestPromoteReadyBelowTheStreakDoesNothing(t *testing.T) {
	t.Parallel()

	h := readyHarness(t, 4, agent.TierAutoExecute)

	promoted, err := h.svc.PromoteReady(t.Context(), readyRequest(h, 10))

	require.NoError(t, err)
	assert.Empty(t, promoted)
	assert.Empty(t, h.trust.marks)
}

func TestPromoteToolMovesOnlyTheToolNamed(t *testing.T) {
	t.Parallel()

	h := readyHarness(t, 12, agent.TierAutoExecute)

	promoted, err := h.svc.PromoteTool(t.Context(), &services.PromoteToolRequest{
		PromoteReadyRequest: *readyRequest(h, 10),
		AgentDefinitionID:   h.definitions.definition.ID,
		ToolName:            "assign_move",
	})

	require.NoError(t, err)
	require.NotNil(t, promoted)
	assert.Equal(t, agent.TierAutoExecute, promoted.To)
	require.Len(t, h.trust.marks, 1)
	assert.True(t, h.trust.marks[0].Promoted)
}

func TestPromoteToolRefusesAToolThatIsNotReady(t *testing.T) {
	t.Parallel()

	h := readyHarness(t, 12, agent.TierAutoExecute)

	_, err := h.svc.PromoteTool(t.Context(), &services.PromoteToolRequest{
		PromoteReadyRequest: *readyRequest(h, 10),
		AgentDefinitionID:   h.definitions.definition.ID,
		ToolName:            "release_hold",
	})
	require.Error(t, err)

	short := readyHarness(t, 4, agent.TierAutoExecute)
	_, err = short.svc.PromoteTool(t.Context(), &services.PromoteToolRequest{
		PromoteReadyRequest: *readyRequest(short, 10),
		AgentDefinitionID:   short.definitions.definition.ID,
		ToolName:            "assign_move",
	})
	require.Error(t, err)
	assert.Empty(t, h.trust.marks)
	assert.Empty(t, short.trust.marks)
}
