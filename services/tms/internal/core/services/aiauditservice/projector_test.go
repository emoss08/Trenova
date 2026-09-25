package aiauditservice

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testNow = int64(1_800_000_000)

type scenario struct {
	tenant   pagination.TenantInfo
	user     pulid.ID
	approver pulid.ID
	agentID  pulid.ID
	thread   pulid.ID
	turn     pulid.ID
	run      pulid.ID
	proposal pulid.ID
	source   *fakeSource
}

func i64(v int64) *int64 { return &v }

func newScenario() *scenario {
	s := &scenario{
		tenant: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		user:     pulid.MustNew("usr_"),
		approver: pulid.MustNew("usr_"),
		agentID:  pulid.MustNew("agdef_"),
		thread:   pulid.MustNew("athr_"),
		turn:     pulid.MustNew("atrn_"),
		run:      pulid.MustNew("ar_"),
		proposal: pulid.MustNew("ap_"),
	}
	org, bu := s.tenant.OrgID, s.tenant.BuID
	shipment := pulid.MustNew("shp_")
	cost := decimal.RequireFromString("0.0125")

	s.source = &fakeSource{
		users:  map[pulid.ID]string{s.user: "Ada Lovelace", s.approver: "Grace Hopper"},
		agents: map[pulid.ID]string{s.agentID: "Dispatcher"},
		threads: []*conversation.Thread{{
			ID: s.thread, OrganizationID: org, BusinessUnitID: bu,
			UserID: s.user, AgentDefinitionID: s.agentID,
		}},
		turns: []*conversation.AssistantTurn{{
			ID: s.turn, OrganizationID: org, BusinessUnitID: bu,
			ThreadID: s.thread, UserID: s.user, RunID: s.run,
			Status:    conversation.AssistantTurnStatusCompleted,
			Origin:    conversation.AssistantTurnOriginPerson,
			StartedAt: testNow - 1000, CompletedAt: i64(testNow - 900),
			Fingerprint: &agent.Fingerprint{DefinitionVersion: 3},
			TraceID:     "4bf92f3577b34da6a3ce929d0e0e4736",
			CreatedAt:   testNow - 1000, UpdatedAt: testNow - 900,
		}},
		runs: []*agent.AgentRun{{
			ID: s.run, OrganizationID: org, BusinessUnitID: bu,
			AgentDefinitionID: s.agentID, Trigger: agent.RunTriggerChat,
			SubjectType: agent.SubjectType("shipment"), SubjectID: shipment,
			Status:    agent.RunStatusAwaitingDecision,
			StartedAt: testNow - 1000, CompletedAt: i64(testNow - 905),
			CreatedAt: testNow - 1000, UpdatedAt: testNow - 905,
		}},
		usage: []*aiusage.AIUsageRecord{
			{
				ID: pulid.MustNew("aiu_"), OrganizationID: org, BusinessUnitID: bu,
				ProviderKind: "anthropic", Model: "model-a", Surface: aiusage.SurfaceChat,
				UserID: s.user, ThreadID: s.thread, Succeeded: true,
				InputTokens: 100, OutputTokens: 20, LatencyMs: 1200, CostUSD: &cost,
				CreatedAt: testNow - 990,
			},
			{
				ID: pulid.MustNew("aiu_"), OrganizationID: org, BusinessUnitID: bu,
				ProviderKind: "openai", Model: "model-b", Surface: aiusage.SurfaceBackground,
				RunID: s.run, Succeeded: false, ErrorClass: "rate_limited",
				ErrorMessage: "slow down, write to bob@example.com", LatencyMs: 10,
				CreatedAt: testNow - 980,
			},
			{
				ID: pulid.MustNew("aiu_"), OrganizationID: org, BusinessUnitID: bu,
				ProviderKind: "openai", Model: "model-b", Surface: aiusage.SurfaceEvaluation,
				AgentDefinitionID: s.agentID, Succeeded: true, LatencyMs: 5,
				CreatedAt: testNow - 970,
			},
		},
		steps: []*agent.AgentRunStep{
			{
				ID: pulid.MustNew("ars_"), OrganizationID: org, BusinessUnitID: bu,
				OwnerKind: string(agent.RunOwnerAssistantTurn), OwnerID: s.turn,
				Kind: string(
					serviceports.RunStepTool,
				), Status: string(serviceports.RunStepCompleted),
				StepKey: "k-read", ToolName: "get_shipment", CallID: "c1",
				Arguments: map[string]any{"shipmentId": shipment.String()},
				Outcome:   map[string]any{"content": "found", "failed": false},
				CreatedAt: testNow - 960, UpdatedAt: testNow - 959,
			},
			{
				ID: pulid.MustNew("ars_"), OrganizationID: org, BusinessUnitID: bu,
				OwnerKind: string(agent.RunOwnerAssistantTurn), OwnerID: s.turn,
				Kind: string(
					serviceports.RunStepTool,
				), Status: string(serviceports.RunStepCompleted),
				StepKey: "k-write", ToolName: "update_worker", CallID: "c2",
				Arguments: map[string]any{"dob": "1990-01-01", "firstName": "Ada"},
				Outcome: map[string]any{"action": map[string]any{
					"toolName": "update_worker", "tier": "AutoExecute", "executed": true,
					"heldBy": []any{}, "proposalId": s.proposal.String(),
					"target": map[string]any{
						"resource": "worker",
						"id":       "wrk_01J9AAAAAAAAAAAAAAAAAAAAAA",
						"version":  4,
					},
					"executedVersion": 5,
				}},
				CreatedAt: testNow - 950, UpdatedAt: testNow - 949,
			},
			{
				ID: pulid.MustNew("ars_"), OrganizationID: org, BusinessUnitID: bu,
				OwnerKind: string(agent.RunOwnerAssistantTurn), OwnerID: s.turn,
				Kind: string(serviceports.RunStepTool), Status: string(serviceports.RunStepStarted),
				StepKey: "k-lost", ToolName: "update_worker", CallID: "c3",
				CreatedAt: testNow - 940, UpdatedAt: testNow - 940,
			},
			{
				ID: pulid.MustNew("ars_"), OrganizationID: org, BusinessUnitID: bu,
				OwnerKind: string(agent.RunOwnerAssistantTurn), OwnerID: s.turn,
				Kind: string(serviceports.RunStepTool), Status: string(serviceports.RunStepFailed),
				StepKey: "k-denied", ToolName: "update_worker", CallID: "c4",
				Outcome:   map[string]any{"failed": true, "reason": "lacks worker:update"},
				CreatedAt: testNow - 935, UpdatedAt: testNow - 934,
			},
		},
		events: []*agent.AgentRunEvent{
			{
				ID: pulid.MustNew("are_"), OrganizationID: org, BusinessUnitID: bu,
				OwnerKind: string(agent.RunOwnerAssistantTurn), OwnerID: s.turn, Sequence: 1,
				Kind: serviceports.AssistantEventToolFinished, CallID: "c9",
				Payload: map[string]any{"callId": "c9", "name": "delete_everything",
					"failed": true, "content": "that tool is not held"},
				OccurredAt: testNow - 930, CreatedAt: testNow - 900,
			},
			{
				ID: pulid.MustNew("are_"), OrganizationID: org, BusinessUnitID: bu,
				OwnerKind: string(agent.RunOwnerAssistantTurn), OwnerID: s.turn, Sequence: 2,
				Kind: serviceports.AssistantEventToolFinished, CallID: "c1",
				Payload:    map[string]any{"callId": "c1", "name": "get_shipment", "failed": true},
				OccurredAt: testNow - 930, CreatedAt: testNow - 900,
			},
			{
				ID: pulid.MustNew("are_"), OrganizationID: org, BusinessUnitID: bu,
				OwnerKind: string(agent.RunOwnerAssistantTurn), OwnerID: s.turn, Sequence: 3,
				Kind: serviceports.AssistantEventDelegateStarted,
				Payload: map[string]any{"delegateCallId": "d1", "agentId": s.agentID.String(),
					"task": "look it up"},
				OccurredAt: testNow - 925, CreatedAt: testNow - 900,
			},
			{
				ID: pulid.MustNew("are_"), OrganizationID: org, BusinessUnitID: bu,
				OwnerKind: string(agent.RunOwnerAssistantTurn), OwnerID: s.turn, Sequence: 4,
				Kind: serviceports.AssistantEventDelegateFinished,
				Payload: map[string]any{"delegateCallId": "d1", "agentId": s.agentID.String(),
					"status": "exhausted", "reason": "ran out of tool calls"},
				OccurredAt: testNow - 920, CreatedAt: testNow - 900,
			},
		},
		proposals: []*agent.AgentProposal{{
			ID: s.proposal, OrganizationID: org, BusinessUnitID: bu,
			RunID: s.run, ToolName: "update_worker", Rationale: "their address changed",
			ToolParams:     map[string]any{"firstName": "Ada", "dob": "1990-01-01"},
			AutonomyTier:   agent.TierActWithApproval,
			Status:         agent.ProposalStatusExecuted,
			HeldBy:         []string{"tool_tier"},
			TargetResource: "worker", TargetID: pulid.ID("wrk_01J9AAAAAAAAAAAAAAAAAAAAAA"),
			TargetVersion: 4, ExecutedAt: i64(testNow - 800), ExecutedByUserID: s.approver,
			ExecutedTargetVersion: i64(5),
			CreatedAt:             testNow - 910, UpdatedAt: testNow - 800,
		}},
		decisions: []*agent.AgentDecision{{
			ID: pulid.MustNew("ad_"), OrganizationID: org, BusinessUnitID: bu,
			ProposalID: &s.proposal, DecidedByUserID: s.approver,
			Decision: agent.DecisionAccepted, ReasonCode: "looks_right",
			CreatedAt: testNow - 805,
		}},
	}

	return s
}

func testKeyring(signed bool) *Keyring {
	if !signed {
		return NewKeyring(&config.AIAuditChainConfig{}, zap.NewNop())
	}

	return NewKeyring(&config.AIAuditChainConfig{
		Keys:        []config.AIAuditChainKey{{ID: "k1", Secret: strings.Repeat("s", 32)}},
		ActiveKeyID: "k1",
	}, zap.NewNop())
}

func testProjector(ledger *fakeLedger, source *fakeSource, keyring *Keyring) *Projector {
	return NewProjector(&ProjectorParams{
		Ledger:    ledger,
		Source:    source,
		Keyring:   keyring,
		Redactor:  testRedactor(),
		BatchSize: 2,
		Now:       func() time.Time { return time.Unix(testNow, 0) },
		Logger:    zap.NewNop(),
	})
}

func bySourceKey(events []*aiaudit.AIAuditEvent) map[string]*aiaudit.AIAuditEvent {
	out := make(map[string]*aiaudit.AIAuditEvent, len(events))
	for _, event := range events {
		out[event.SourceKey] = event
	}

	return out
}

func projected(
	t *testing.T,
	signed bool,
) (*scenario, *fakeLedger, map[string]*aiaudit.AIAuditEvent) {
	t.Helper()

	s := newScenario()
	ledger := newFakeLedger()
	projector := testProjector(ledger, s.source, testKeyring(signed))

	for range 10 {
		_, err := projector.RunOnce(t.Context(), nil)
		require.NoError(t, err)
	}

	return s, ledger, bySourceKey(ledger.all(s.tenant))
}

func TestProjector_DerivesEveryKind(t *testing.T) {
	t.Parallel()

	s, _, events := projected(t, true)

	turnStarted := events["turn:"+s.turn.String()+":started"]
	require.NotNil(t, turnStarted)
	assert.Equal(t, aiaudit.KindRunStarted, turnStarted.Kind)
	assert.Equal(t, aiaudit.PrincipalUser, turnStarted.PrincipalType)
	assert.Equal(t, s.user, turnStarted.OnBehalfOfUserID)
	assert.Equal(t, "Ada Lovelace", turnStarted.OnBehalfOfUserName)
	assert.Equal(t, s.agentID, turnStarted.AgentDefinitionID)
	assert.Equal(t, "Dispatcher", turnStarted.AgentName)
	assert.Equal(t, int64(3), *turnStarted.AgentDefinitionVersion)
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", turnStarted.TraceID)

	assert.Equal(t, aiaudit.OutcomeCompleted, events["turn:"+s.turn.String()+":ended"].Outcome)

	runStarted := events["run:"+s.run.String()+":started"]
	require.NotNil(t, runStarted)
	assert.Equal(t, aiaudit.PrincipalAgent, runStarted.PrincipalType)
	assert.Equal(
		t,
		s.user,
		runStarted.OnBehalfOfUserID,
		"a run opened for a turn acts for its person",
	)
	assert.Equal(t, "Chat", runStarted.ResultSummary)
	assert.Equal(t, aiaudit.KindRunEnded, events["run:"+s.run.String()+":ended"].Kind)

	chat := events["usage:"+s.source.usage[0].ID.String()]
	require.NotNil(t, chat)
	assert.Equal(t, aiaudit.KindModelCall, chat.Kind)
	assert.Equal(t, s.turn, chat.OwnerID, "a chat call with no turn is matched to its running turn")
	assert.True(t, chat.Reconstructed)
	assert.Equal(t, "0.0125", chat.CostUSD.String())

	failed := events["usage:"+s.source.usage[1].ID.String()]
	assert.Equal(t, aiaudit.OutcomeFailed, failed.Outcome)
	assert.NotContains(t, failed.Reason, "bob@example.com")
	assert.Equal(t, s.run, failed.OwnerID)

	evaluation := events["usage:"+s.source.usage[2].ID.String()]
	assert.Equal(t, aiaudit.PurposeEvaluation, evaluation.Purpose)

	read := events["step:"+s.turn.String()+":k-read"]
	require.NotNil(t, read)
	assert.Equal(t, aiaudit.KindToolCall, read.Kind)
	assert.Equal(t, aiaudit.OutcomeRan, read.Outcome)

	write := events["step:"+s.turn.String()+":k-write"]
	require.NotNil(t, write)
	assert.Equal(t, aiaudit.OutcomeRan, write.Outcome)
	assert.Equal(t, "worker", write.EntityType)
	assert.Equal(t, int64(4), *write.VersionBefore)
	assert.Equal(t, int64(5), *write.VersionAfter)
	assert.Equal(t, s.proposal, write.ProposalID)
	assert.Equal(t, aiaudit.ConfidentialPlaceholder, write.Arguments["dob"])
	assert.Equal(t, permission.ResourceWorker, write.ArgumentSensitivity.Resource)

	assert.Equal(t, aiaudit.OutcomeUnknown, events["step:"+s.turn.String()+":k-lost"].Outcome)
	denied := events["step:"+s.turn.String()+":k-denied"]
	assert.Equal(t, aiaudit.OutcomeDenied, denied.Outcome)
	assert.Equal(t, "lacks worker:update", denied.Reason)

	refused := events["event:"+s.source.events[0].ID.String()]
	require.NotNil(t, refused)
	assert.Equal(t, aiaudit.KindToolRefused, refused.Kind)
	assert.Equal(t, "delete_everything", refused.ToolName)
	assert.Nil(t, events["event:"+s.source.events[1].ID.String()],
		"a failed call with a step behind it is the step's row")

	assert.Equal(
		t,
		aiaudit.KindDelegationStarted,
		events["event:"+s.source.events[2].ID.String()].Kind,
	)
	ended := events["event:"+s.source.events[3].ID.String()]
	assert.Equal(t, aiaudit.KindDelegationEnded, ended.Kind)
	assert.Equal(t, aiaudit.OutcomeExhausted, ended.Outcome)
	assert.Equal(t, "d1", ended.DelegateCallID)

	filed := events["proposal:"+s.proposal.String()+":filed"]
	require.NotNil(t, filed)
	assert.Equal(t, aiaudit.KindProposalFiled, filed.Kind)
	assert.Equal(t, []string{"tool_tier"}, filed.HeldBy)
	assert.Equal(t, aiaudit.ConfidentialPlaceholder, filed.Arguments["dob"])

	executed := events["proposal:"+s.proposal.String()+":executed"]
	require.NotNil(t, executed)
	assert.Equal(t, aiaudit.KindProposalExecuted, executed.Kind)
	assert.Equal(t, s.approver.String(), executed.PrincipalID)
	assert.Equal(t, int64(5), *executed.VersionAfter)

	decided := events["decision:"+s.source.decisions[0].ID.String()]
	require.NotNil(t, decided)
	assert.Equal(t, aiaudit.KindProposalDecided, decided.Kind)
	assert.Equal(t, aiaudit.OutcomeAccepted, decided.Outcome)
	assert.Equal(t, "Grace Hopper", decided.DecidedByUserName)
	assert.Equal(t, "update_worker", decided.ToolName)
	assert.Equal(t, s.user, decided.OnBehalfOfUserID)
}

func TestProjector_ChainsEveryTenantsRowsInOrder(t *testing.T) {
	t.Parallel()

	s, ledger, _ := projected(t, true)
	keyring := testKeyring(true)

	rows := ledger.all(s.tenant)
	require.NotEmpty(t, rows)
	prev := aiaudit.GenesisHash
	for i, row := range rows {
		assert.Equal(t, int64(i+1), row.Seq)
		assert.Equal(t, prev, row.PrevHash)
		assert.Equal(t, aiaudit.HashVersionSigned, row.HashVersion)
		assert.Equal(t, "k1", row.HashKeyID)
		key, missing := keyring.keyFor(row)
		require.False(t, missing)
		ok, err := aiaudit.VerifyHash(row, key)
		require.NoError(t, err)
		assert.True(t, ok, "row %d", row.Seq)
		prev = row.Hash
	}
}

func TestProjector_RunningAgainInsertsNothing(t *testing.T) {
	t.Parallel()

	s := newScenario()
	ledger := newFakeLedger()
	projector := testProjector(ledger, s.source, testKeyring(true))

	for range 10 {
		_, err := projector.RunOnce(t.Context(), nil)
		require.NoError(t, err)
	}
	before := len(ledger.all(s.tenant))

	ledger.watermarks = map[aiaudit.Source]*aiaudit.AIAuditProjectorState{}
	for range 10 {
		result, err := projector.RunOnce(t.Context(), nil)
		require.NoError(t, err)
		assert.Zero(t, result.Inserted)
	}

	assert.Len(t, ledger.all(s.tenant), before)
}

func TestProjector_PicksUpARunsEndOnALaterPass(t *testing.T) {
	t.Parallel()

	s := newScenario()
	s.source.runs[0].CompletedAt = nil
	s.source.runs[0].Status = agent.RunStatusDiagnosing
	ledger := newFakeLedger()
	projector := testProjector(ledger, s.source, testKeyring(true))
	for range 10 {
		_, err := projector.RunOnce(t.Context(), nil)
		require.NoError(t, err)
	}
	assert.Nil(t, bySourceKey(ledger.all(s.tenant))["run:"+s.run.String()+":ended"])

	s.source.runs[0].CompletedAt = i64(testNow - 100)
	s.source.runs[0].Status = agent.RunStatusCompleted
	s.source.runs[0].UpdatedAt = testNow - 100
	_, err := projector.RunOnce(t.Context(), nil)
	require.NoError(t, err)

	ended := bySourceKey(ledger.all(s.tenant))["run:"+s.run.String()+":ended"]
	require.NotNil(t, ended)
	assert.Equal(t, aiaudit.OutcomeCompleted, ended.Outcome)
}

func TestProjector_WithoutAKeyWritesAnUnsignedChain(t *testing.T) {
	t.Parallel()

	s, ledger, _ := projected(t, false)

	for _, row := range ledger.all(s.tenant) {
		assert.Equal(t, aiaudit.HashVersionUnsigned, row.HashVersion)
		assert.Empty(t, row.HashKeyID)
	}

	verifier := testVerifier(ledger, testKeyring(false), nil)
	result, err := verifier.VerifyTenant(t.Context(), s.tenant, nil)
	require.NoError(t, err)
	assert.Equal(t, aiaudit.VerificationVerified, result.Status)

	status, err := verifier.ChainStatus(t.Context(), s.tenant)
	require.NoError(t, err)
	assert.False(t, status.Signed)
}

func TestProjector_SourcePruneHorizonTrailsTheWatermark(t *testing.T) {
	t.Parallel()

	s := newScenario()
	ledger := newFakeLedger()
	projector := testProjector(ledger, s.source, testKeyring(true))

	horizon, err := projector.SourcePruneHorizon(t.Context(), aiaudit.SourceRunEvents)
	require.NoError(t, err)
	assert.Zero(t, horizon, "a source never read may not be pruned at all")

	for range 10 {
		_, err = projector.RunOnce(t.Context(), nil)
		require.NoError(t, err)
	}
	horizon, err = projector.SourcePruneHorizon(t.Context(), aiaudit.SourceRunEvents)
	require.NoError(t, err)
	assert.Equal(t, testNow-900-int64(ProjectorOverlap.Seconds()), horizon)
}
