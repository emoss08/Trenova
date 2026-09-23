package agenttrustservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeTrustRepo struct {
	repositories.AgentToolTrustRepository
	row      *agent.ToolTrust
	recorded []agent.TrustOutcome
	marks    []repositories.MarkToolTierChangeRequest
}

func (f *fakeTrustRepo) Record(
	_ context.Context,
	req repositories.RecordToolTrustRequest,
) (*agent.ToolTrust, error) {
	f.recorded = append(f.recorded, req.Outcome)
	f.row.ToolName = req.ToolName
	f.row.AgentDefinitionID = req.AgentDefinitionID
	switch req.Outcome {
	case agent.TrustOutcomeApproved:
		f.row.Streak++
		f.row.Approvals++
	case agent.TrustOutcomeModified:
		f.row.Streak = 0
		f.row.Modifications++
	case agent.TrustOutcomeRejected:
		f.row.Streak = 0
		f.row.Rejections++
	case agent.TrustOutcomeExecutionFailed:
		f.row.Streak = 0
		f.row.ExecutionFailures++
	}

	return f.row, nil
}

func (f *fakeTrustRepo) MarkTierChange(
	_ context.Context,
	req repositories.MarkToolTierChangeRequest,
) (*agent.ToolTrust, error) {
	f.marks = append(f.marks, req)

	return f.row, nil
}

type fakeDefinitions struct {
	repositories.AgentDefinitionRepository
	definition *agentdefinition.Definition
	tiers      []repositories.SetAgentDefinitionToolTierRequest
}

func (f *fakeDefinitions) GetByID(
	context.Context,
	repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return f.definition, nil
}

func (f *fakeDefinitions) SetToolTier(
	_ context.Context,
	req repositories.SetAgentDefinitionToolTierRequest,
) error {
	f.tiers = append(f.tiers, req)

	return nil
}

type fakeControls struct {
	repositories.AgentControlRepository
	control *tenant.AgentControl
}

func (f fakeControls) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	return f.control, nil
}

type fakeRuns struct {
	repositories.AgentRunRepository
	run *agent.AgentRun
}

func (f fakeRuns) GetByID(
	context.Context,
	repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return f.run, nil
}

type fakeTool struct {
	services.AgentTool
	tier agent.AutonomyTier
}

func (f fakeTool) DefaultAutonomyTier() agent.AutonomyTier { return f.tier }

type fakeRegistry struct {
	services.AgentToolRegistry
	tools map[string]agent.AutonomyTier
}

func (f fakeRegistry) Get(name string) (services.AgentTool, bool) {
	tier, ok := f.tools[name]
	if !ok {
		return nil, false
	}

	return fakeTool{tier: tier}, true
}

type fakeAudit struct{ entries []*services.LogActionParams }

func (f *fakeAudit) LogAction(params *services.LogActionParams, _ ...services.LogOption) error {
	f.entries = append(f.entries, params)

	return nil
}

type fakeNotifier struct{ created []*notification.Notification }

func (f *fakeNotifier) Create(
	_ context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	f.created = append(f.created, entity)

	return entity, nil
}

type harness struct {
	svc         *Service
	trust       *fakeTrustRepo
	definitions *fakeDefinitions
	audit       *fakeAudit
	notifier    *fakeNotifier
	proposal    *agent.AgentProposal
}

type harnessOptions struct {
	earned    bool
	threshold int
	ceiling   agent.AutonomyTier
	toolTiers map[string]agent.AutonomyTier
	row       *agent.ToolTrust
}

func newHarness(t *testing.T, opts harnessOptions) *harness {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	definitionID := pulid.MustNew("agdef_")
	runID := pulid.MustNew("arun_")

	row := opts.row
	if row == nil {
		row = &agent.ToolTrust{}
	}
	row.ID = pulid.MustNew("att_")
	if opts.ceiling == "" {
		opts.ceiling = agent.TierAutoExecute
	}
	if opts.threshold == 0 {
		opts.threshold = 3
	}

	h := &harness{
		trust: &fakeTrustRepo{row: row},
		definitions: &fakeDefinitions{definition: &agentdefinition.Definition{
			ID:              definitionID,
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			Name:            "Dispatch coverage",
			ToolNames:       []string{"assign_move"},
			ToolTiers:       opts.toolTiers,
			AutonomyCeiling: opts.ceiling,
		}},
		audit:    &fakeAudit{},
		notifier: &fakeNotifier{},
		proposal: &agent.AgentProposal{
			ID:             pulid.MustNew("ap_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			RunID:          runID,
			ToolName:       "assign_move",
		},
	}
	h.svc = &Service{
		l:           zap.NewNop(),
		trust:       h.trust,
		definitions: h.definitions,
		controls: fakeControls{control: &tenant.AgentControl{
			EarnedAutonomy:     opts.earned,
			PromotionThreshold: opts.threshold,
		}},
		runs: fakeRuns{run: &agent.AgentRun{ID: runID, AgentDefinitionID: definitionID}},
		tools: fakeRegistry{
			tools: map[string]agent.AutonomyTier{"assign_move": agent.TierActWithApproval},
		},
		audit:         h.audit,
		notifications: h.notifier,
	}

	return h
}

func accepted(mods map[string]any) *agent.AgentDecision {
	return &agent.AgentDecision{
		Decision:        agent.DecisionAccepted,
		Modifications:   mods,
		DecidedByUserID: pulid.MustNew("usr_"),
	}
}

func TestRecordDecision_PromotesAtThreshold(t *testing.T) {
	t.Parallel()

	h := newHarness(t, harnessOptions{earned: true, threshold: 3})

	for range 2 {
		require.NoError(t, h.svc.RecordDecision(t.Context(), h.proposal, accepted(nil)))
	}
	assert.Empty(t, h.definitions.tiers, "two approvals are short of the threshold")

	require.NoError(t, h.svc.RecordDecision(t.Context(), h.proposal, accepted(nil)))

	require.Len(t, h.definitions.tiers, 1)
	assert.Equal(t, "assign_move", h.definitions.tiers[0].ToolName)
	assert.Equal(t, agent.TierAutoExecute, h.definitions.tiers[0].Tier,
		"the tool's default was ask-first, so the next tier up is automatic")

	require.Len(t, h.trust.marks, 1)
	assert.True(t, h.trust.marks[0].Promoted)
	assert.Equal(t, agent.TierAutoExecute, h.trust.marks[0].EarnedTier)

	require.Len(t, h.audit.entries, 1)
	assert.Equal(t, permission.ResourceAgentDefinition, h.audit.entries[0].Resource)
	assert.Equal(t, services.PrincipalTypeUser, h.audit.entries[0].PrincipalType)

	require.Len(t, h.notifier.created, 1)
	assert.Equal(t, EventToolPromoted, h.notifier.created[0].EventType)
	assert.Equal(t, agentsLink, h.notifier.created[0].Data["link"])
	assert.Equal(t, "AutoExecute", h.notifier.created[0].Data["toTier"])
}

func TestRecordDecision_NeverPromotesWhenTheOrganizationHasNotOptedIn(t *testing.T) {
	t.Parallel()

	h := newHarness(t, harnessOptions{earned: false, threshold: 1})

	require.NoError(t, h.svc.RecordDecision(t.Context(), h.proposal, accepted(nil)))

	assert.Equal(t, []agent.TrustOutcome{agent.TrustOutcomeApproved}, h.trust.recorded,
		"the ledger still counts so the switch can be turned on later with history")
	assert.Empty(t, h.definitions.tiers)
	assert.Empty(t, h.notifier.created)
}

func TestRecordDecision_StopsAtTheCeiling(t *testing.T) {
	t.Parallel()

	h := newHarness(
		t,
		harnessOptions{earned: true, threshold: 1, ceiling: agent.TierActWithApproval},
	)

	require.NoError(t, h.svc.RecordDecision(t.Context(), h.proposal, accepted(nil)))

	assert.Empty(t, h.definitions.tiers, "automatic is above the agent's ceiling")
	assert.Empty(t, h.trust.marks)
}

func TestRecordDecision_ModificationResetsWithoutDemoting(t *testing.T) {
	t.Parallel()

	h := newHarness(t, harnessOptions{
		earned:    true,
		threshold: 5,
		toolTiers: map[string]agent.AutonomyTier{"assign_move": agent.TierAutoExecute},
		row:       &agent.ToolTrust{Streak: 4, EarnedTier: agent.TierAutoExecute},
	})

	require.NoError(
		t,
		h.svc.RecordDecision(
			t.Context(),
			h.proposal,
			accepted(map[string]any{"workerId": "other"}),
		),
	)

	assert.Equal(t, 0, h.trust.row.Streak)
	assert.Empty(t, h.definitions.tiers, "a modification is not a setback")
	assert.Empty(t, h.notifier.created)
}

func TestRecordDecision_RejectionTakesBackAnEarnedTier(t *testing.T) {
	t.Parallel()

	h := newHarness(t, harnessOptions{
		// The organization has since switched earned autonomy off; a tier
		// the ledger granted is still the ledger's to take back.
		earned:    false,
		toolTiers: map[string]agent.AutonomyTier{"assign_move": agent.TierAutoExecute},
		row:       &agent.ToolTrust{EarnedTier: agent.TierAutoExecute},
	})

	require.NoError(t, h.svc.RecordDecision(t.Context(), h.proposal, &agent.AgentDecision{
		Decision:        agent.DecisionRejected,
		DecidedByUserID: pulid.MustNew("usr_"),
	}))

	require.Len(t, h.definitions.tiers, 1)
	assert.Equal(t, agent.TierActWithApproval, h.definitions.tiers[0].Tier)
	require.Len(t, h.trust.marks, 1)
	assert.False(t, h.trust.marks[0].Promoted)
	require.Len(t, h.notifier.created, 1)
	assert.Equal(t, EventToolDemoted, h.notifier.created[0].EventType)
	assert.Equal(t, notification.PriorityHigh, h.notifier.created[0].Priority)
}

func TestRecordDecision_RejectionLeavesAChosenTierAlone(t *testing.T) {
	t.Parallel()

	h := newHarness(t, harnessOptions{
		earned:    true,
		toolTiers: map[string]agent.AutonomyTier{"assign_move": agent.TierAutoExecute},
		// Nothing was earned: a person set automatic by hand.
		row: &agent.ToolTrust{},
	})

	require.NoError(t, h.svc.RecordDecision(t.Context(), h.proposal, &agent.AgentDecision{
		Decision:        agent.DecisionRejected,
		DecidedByUserID: pulid.MustNew("usr_"),
	}))

	assert.Empty(t, h.definitions.tiers, "the ledger never overrides a person's choice")
	assert.Empty(t, h.notifier.created)
}

func TestRecordExecutionFailure_DemotesWithoutAPerson(t *testing.T) {
	t.Parallel()

	h := newHarness(t, harnessOptions{
		earned:    true,
		toolTiers: map[string]agent.AutonomyTier{"assign_move": agent.TierAutoExecute},
		row:       &agent.ToolTrust{EarnedTier: agent.TierAutoExecute},
	})

	require.NoError(t, h.svc.RecordExecutionFailure(t.Context(), h.proposal))

	assert.Equal(t, []agent.TrustOutcome{agent.TrustOutcomeExecutionFailed}, h.trust.recorded)
	require.Len(t, h.definitions.tiers, 1)
	assert.Equal(t, agent.TierActWithApproval, h.definitions.tiers[0].Tier)
	require.Len(t, h.audit.entries, 1)
	assert.Equal(t, services.PrincipalTypeSystem, h.audit.entries[0].PrincipalType,
		"no person decided anything, so the change is the system's")
}

func TestRecord_SkipsRunsWithoutAnAgent(t *testing.T) {
	t.Parallel()

	h := newHarness(t, harnessOptions{earned: true, threshold: 1})
	h.svc.runs = fakeRuns{run: &agent.AgentRun{ID: h.proposal.RunID}}

	require.NoError(t, h.svc.RecordDecision(t.Context(), h.proposal, accepted(nil)))

	assert.Empty(t, h.trust.recorded)
}

func TestRecordDecision_NilInputsAreNoOps(t *testing.T) {
	t.Parallel()

	h := newHarness(t, harnessOptions{earned: true, threshold: 1})

	require.NoError(t, h.svc.RecordDecision(t.Context(), nil, accepted(nil)))
	require.NoError(t, h.svc.RecordDecision(t.Context(), h.proposal, nil))
	require.NoError(t, h.svc.RecordExecutionFailure(t.Context(), nil))

	assert.Empty(t, h.trust.recorded)
}
