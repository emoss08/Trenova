package aiauditservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type unattendedScenario struct {
	*scenario
	background pulid.ID
	systemUser pulid.ID
	legacy     *agent.AgentProposal
	current    *agent.AgentProposal
}

func newUnattendedScenario() *unattendedScenario {
	s := newScenario()
	u := &unattendedScenario{
		scenario:   s,
		background: pulid.MustNew("ar_"),
		systemUser: pulid.MustNew("usr_"),
	}
	org, bu := s.tenant.OrgID, s.tenant.BuID

	s.source.runs = append(s.source.runs, &agent.AgentRun{
		ID: u.background, OrganizationID: org, BusinessUnitID: bu,
		AgentDefinitionID: s.agentID, Trigger: agent.RunTriggerEvent,
		SubjectType: agent.SubjectType("shipment"), SubjectID: pulid.MustNew("shp_"),
		Status:    agent.RunStatusCompleted,
		StartedAt: testNow - 760, CompletedAt: i64(testNow - 500),
		CreatedAt: testNow - 760, UpdatedAt: testNow - 500,
	})
	u.legacy = &agent.AgentProposal{
		ID: pulid.MustNew("ap_"), OrganizationID: org, BusinessUnitID: bu,
		RunID: u.background, ToolName: "update_worker", Rationale: "keep it current",
		ToolParams:     map[string]any{"firstName": "Ada"},
		AutonomyTier:   agent.TierAutoExecute,
		Status:         agent.ProposalStatusExecuted,
		HeldBy:         []string{},
		TargetResource: "worker", TargetID: pulid.ID("wrk_01J9BBBBBBBBBBBBBBBBBBBBBB"),
		TargetVersion: 2, ExecutedAt: i64(testNow - 700), ExecutedTargetVersion: i64(3),
		CreatedAt: testNow - 700, UpdatedAt: testNow - 700,
	}
	u.current = &agent.AgentProposal{
		ID: pulid.MustNew("ap_"), OrganizationID: org, BusinessUnitID: bu,
		RunID: u.background, ToolName: "update_worker", Rationale: "keep it current",
		ToolParams:     map[string]any{"firstName": "Grace"},
		AutonomyTier:   agent.TierAutoExecute,
		Status:         agent.ProposalStatusExecuted,
		HeldBy:         []string{},
		TargetResource: "worker", TargetID: pulid.ID("wrk_01J9CCCCCCCCCCCCCCCCCCCCCC"),
		TargetVersion: 7, ExecutedAt: i64(testNow - 600), ExecutedByUserID: u.systemUser,
		ExecutedTargetVersion: i64(8),
		CreatedAt:             testNow - 600, UpdatedAt: testNow - 600,
	}
	s.source.proposals = append(s.source.proposals, u.legacy)
	s.source.agents[s.agentID] = "Dispatch Agent"

	return u
}

func TestProjector_AnUnattendedExecutionIsTheSystemUsersWriteForItsAgent(t *testing.T) {
	t.Parallel()

	u := newUnattendedScenario()
	ledger := newFakeLedger()
	keyring := testKeyring(true)
	projector := testProjector(ledger, u.source, keyring)
	for range 10 {
		_, err := projector.RunOnce(t.Context(), nil)
		require.NoError(t, err)
	}
	legacyHash := bySourceKey(ledger.all(u.tenant))[executedKey(u.legacy)].Hash
	require.NotEmpty(t, legacyHash)

	u.source.proposals = append(u.source.proposals, u.current)
	for range 10 {
		_, err := projector.RunOnce(t.Context(), nil)
		require.NoError(t, err)
	}
	events := bySourceKey(ledger.all(u.tenant))

	current := events[executedKey(u.current)]
	require.NotNil(t, current)
	assert.Equal(t, aiaudit.KindProposalExecuted, current.Kind)
	assert.Equal(t, aiaudit.PrincipalUser, current.PrincipalType)
	assert.Equal(t, u.systemUser.String(), current.PrincipalID, "the system account is the user")
	assert.Equal(t, u.agentID, current.AgentDefinitionID, "the agent stays identifiable")
	assert.Equal(t, "Dispatch Agent", current.AgentName)
	assert.True(t, current.OnBehalfOfUserID.IsNil(), "no person is invented")
	assert.True(t, current.DecidedByUserID.IsNil())
	assert.Equal(t, testNow-760, current.WindowStart,
		"an automatic write is looked for from the start of the run that made it")
	assert.False(t, current.Reconstructed)

	legacy := events[executedKey(u.legacy)]
	require.NotNil(t, legacy)
	assert.Equal(t, aiaudit.PrincipalAgent, legacy.PrincipalType,
		"a write recorded before executors were named still derives as the agent's")
	assert.Equal(t, u.agentID.String(), legacy.PrincipalID)
	assert.Equal(t, legacyHash, legacy.Hash, "rows already on the trail are never rewritten")
	assert.Equal(t, testNow-760, legacy.WindowStart)

	result, err := testVerifier(ledger, keyring, nil).VerifyTenant(t.Context(), u.tenant, nil)
	require.NoError(t, err)
	assert.Equal(t, aiaudit.VerificationVerified, result.Status,
		"rows before and after the change verify on one chain")
}

func TestProjector_AnAutomaticWriteInAConversationKeepsItsShortWindow(t *testing.T) {
	t.Parallel()

	s := newScenario()
	s.source.proposals[0].AutonomyTier = agent.TierAutoExecute
	s.source.proposals[0].ExecutedByUserID = s.user
	s.source.decisions = nil

	ledger := newFakeLedger()
	projector := testProjector(ledger, s.source, testKeyring(true))
	for range 10 {
		_, err := projector.RunOnce(t.Context(), nil)
		require.NoError(t, err)
	}

	executed := bySourceKey(ledger.all(s.tenant))["proposal:"+s.proposal.String()+":executed"]
	require.NotNil(t, executed)
	assert.Equal(t, aiaudit.PrincipalUser, executed.PrincipalType)
	assert.Equal(t, s.user.String(), executed.PrincipalID)
	assert.Equal(t, testNow-800-executionWindowSeconds, executed.WindowStart,
		"a person's write in a conversation is unchanged")
}

func TestCorrelate_AnUnattendedWriteMatchesTheSystemUsersAuditRows(t *testing.T) {
	t.Parallel()

	u := newUnattendedScenario()
	u.source.proposals = append(u.source.proposals, u.current)
	ledger := newFakeLedger()
	projector := testProjector(ledger, u.source, testKeyring(true))
	for range 10 {
		_, err := projector.RunOnce(t.Context(), nil)
		require.NoError(t, err)
	}
	executed := bySourceKey(ledger.all(u.tenant))[executedKey(u.current)]
	require.NotNil(t, executed)

	ledger.entries = []*audit.Entry{
		{
			ID: "ae_unattended", ResourceID: executed.EntityID,
			PrincipalType: string(serviceports.PrincipalTypeAgent), PrincipalID: u.agentID,
			UserID: u.systemUser, Timestamp: executed.WindowEnd,
		},
		{
			ID: "ae_person", ResourceID: executed.EntityID, PrincipalID: u.user,
			UserID: u.user, Timestamp: executed.WindowEnd,
		},
	}

	linked, err := Correlate(t.Context(), ledger, u.tenant, []*aiaudit.AIAuditEvent{executed})
	require.NoError(t, err)
	require.Len(t, linked[executed.ID], 1)
	assert.Equal(t, "ae_unattended", linked[executed.ID][0].ID.String())
}

func TestCorrelationMatches_AnAgentsCallMatchesItsDefinitionAndTheGenericAgent(t *testing.T) {
	t.Parallel()

	definition := pulid.MustNew("agdef_")
	event := &aiaudit.AIAuditEvent{
		ID:                pulid.MustNew(aiaudit.EventIDPrefix),
		Kind:              aiaudit.KindToolCall,
		Outcome:           aiaudit.OutcomeRan,
		PrincipalType:     aiaudit.PrincipalAgent,
		PrincipalID:       definition.String(),
		AgentDefinitionID: definition,
		EntityID:          "shp_1",
		WindowStart:       testNow - 10,
		WindowEnd:         testNow,
	}

	matches := CorrelationMatches(event)
	require.Len(t, matches, 2)
	assert.Equal(t, definition, matches[0].PrincipalID, "rows an unattended run wrote")
	assert.Equal(t, serviceports.AgentPrincipalID, matches[1].PrincipalID, "rows written before")

	ledger := newFakeLedger()
	ledger.entries = []*audit.Entry{
		{ID: "ae_named", ResourceID: "shp_1", PrincipalID: definition, Timestamp: testNow},
		{
			ID: "ae_generic", ResourceID: "shp_1", PrincipalID: serviceports.AgentPrincipalID,
			Timestamp: testNow - 5,
		},
		{
			ID: "ae_other_agent", ResourceID: "shp_1", PrincipalID: pulid.MustNew("agdef_"),
			Timestamp: testNow,
		},
	}
	linked, err := Correlate(t.Context(), ledger, newScenario().tenant, []*aiaudit.AIAuditEvent{event})
	require.NoError(t, err)
	ids := make([]string, 0, len(linked[event.ID]))
	for _, entry := range linked[event.ID] {
		ids = append(ids, entry.ID.String())
	}
	assert.ElementsMatch(t, []string{"ae_named", "ae_generic"}, ids)

	generic := *event
	generic.PrincipalID = ""
	generic.AgentDefinitionID = pulid.Nil
	matches = CorrelationMatches(&generic)
	require.Len(t, matches, 1)
	assert.Equal(t, serviceports.AgentPrincipalID, matches[0].PrincipalID)
}

func executedKey(proposal *agent.AgentProposal) string {
	return "proposal:" + proposal.ID.String() + ":executed"
}
