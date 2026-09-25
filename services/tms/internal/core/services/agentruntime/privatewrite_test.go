package agentruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type privateActionTool struct {
	*agentruntimetest.StubActionTool

	private bool
}

func (t *privateActionTool) Policy() serviceports.ToolPolicy {
	policy := t.StubActionTool.Policy()
	policy.Egress = []agent.EgressClass{agent.EgressPersonal, agent.EgressInternal}
	policy.PersonalRunsUnasked = true
	policy.Classify = func(serviceports.ToolExecuteParams) serviceports.CallPolicy {
		if t.private {
			return serviceports.CallPolicy{Egress: agent.EgressPersonal}
		}

		return serviceports.CallPolicy{Egress: agent.EgressInternal}
	}

	return policy
}

func runPrivate(
	t *testing.T,
	tool *privateActionTool,
	configure func(*serviceports.RunRequest),
) *serviceports.RunResult {
	t.Helper()

	return runPrivateWithTrust(t, tool, nil, configure)
}

func runPrivateWithTrust(
	t *testing.T,
	tool *privateActionTool,
	trust repositories.AgentToolTrustRepository,
	configure func(*serviceports.RunRequest),
) *serviceports.RunResult {
	t.Helper()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("create_report", map[string]any{"name": "Late loads"}),
		textTurn("Saved."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)
	rt.trust = trust
	req := &serviceports.RunRequest{
		Definition: testDefinition("create_report"),
		Actor:      testActor(),
		Input:      "Save a report of late loads",
	}
	if configure != nil {
		configure(req)
	}

	result, err := rt.Run(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, result.Actions, 1)

	return result
}

// A report saved to the person's own list is theirs to save. An agent held
// to proposing used to put an approval card between them and what they
// asked for.
func TestRun_APrivateWriteRunsWhateverTheAgentsCeiling(t *testing.T) {
	t.Parallel()

	tool := &privateActionTool{
		StubActionTool: actionTool("create_report", agent.TierAutoExecute, nil),
		private:        true,
	}

	result := runPrivate(t, tool, nil)

	assert.Equal(t, agent.TierAutoExecute, result.Actions[0].Tier)
	assert.True(t, result.Actions[0].Executed)
}

func TestRun_AWriteOthersSeeStillWaitsAtTheCeiling(t *testing.T) {
	t.Parallel()

	tool := &privateActionTool{
		StubActionTool: actionTool("create_report", agent.TierAutoExecute, nil),
		private:        false,
	}

	result := runPrivate(t, tool, nil)

	assert.Equal(t, agent.TierPropose, result.Actions[0].Tier)
	assert.False(t, result.Actions[0].Executed)
}

// An administrator who set a tier for the tool on this agent chose it on
// purpose, and an unattended run has nobody the write could be private to.
func TestRun_APrivateWriteKeepsAnAgentsOwnSettingAndNeedsAPerson(t *testing.T) {
	t.Parallel()

	newTool := func() *privateActionTool {
		return &privateActionTool{
			StubActionTool: actionTool("create_report", agent.TierAutoExecute, nil),
			private:        true,
		}
	}

	set := runPrivate(t, newTool(), func(req *serviceports.RunRequest) {
		req.Definition.ToolTiers = map[string]agent.AutonomyTier{"create_report": agent.TierPropose}
	})
	assert.Equal(t, agent.TierPropose, set.Actions[0].Tier)

	unattended := runPrivate(t, newTool(), func(req *serviceports.RunRequest) {
		req.Unattended = true
	})
	assert.Equal(t, agent.TierPropose, unattended.Actions[0].Tier)
}

type ledgerTrust struct {
	repositories.AgentToolTrustRepository

	rows []*agent.ToolTrust
	err  error
	asks int
}

func (l *ledgerTrust) ListByDefinition(
	_ context.Context,
	_ repositories.ListToolTrustRequest,
) ([]*agent.ToolTrust, error) {
	l.asks++

	return l.rows, l.err
}

func onRecordedAgent(tier agent.AutonomyTier) func(*serviceports.RunRequest) {
	return func(req *serviceports.RunRequest) {
		req.Definition.ID = pulid.MustNew("agdef_")
		req.Definition.OrganizationID = pulid.MustNew("org_")
		req.Definition.BusinessUnitID = pulid.MustNew("bu_")
		req.Definition.ToolTiers = map[string]agent.AutonomyTier{"create_report": tier}
	}
}

/*
Trust writes the tiers it earns into the same map a person's choice lives in.
The exemption used to read any entry there as a person's, so the first
promotion took it away and a delegated private report waited for approval.
The ledger names the tier it earned; that one is not a person's choice.
*/
func TestRun_APrivateWriteKeepsItsExemptionThroughATierTrustEarned(t *testing.T) {
	t.Parallel()

	tool := &privateActionTool{
		StubActionTool: actionTool("create_report", agent.TierAutoExecute, nil),
		private:        true,
	}
	ledger := &ledgerTrust{rows: []*agent.ToolTrust{
		{ToolName: "assign_move", EarnedTier: agent.TierPropose},
		{ToolName: "create_report", EarnedTier: agent.TierActWithApproval},
	}}

	result := runPrivateWithTrust(t, tool, ledger, onRecordedAgent(agent.TierActWithApproval))

	assert.Equal(t, agent.TierAutoExecute, result.Actions[0].Tier)
	assert.True(t, result.Actions[0].Executed)
	assert.Equal(t, 1, ledger.asks)
}

func TestRun_APrivateWriteKeepsATierAPersonMovedAfterTrust(t *testing.T) {
	t.Parallel()

	for name, ledger := range map[string]*ledgerTrust{
		"a person moved it since": {rows: []*agent.ToolTrust{
			{ToolName: "create_report", EarnedTier: agent.TierAutoExecute},
		}},
		"trust never earned a tier": {rows: []*agent.ToolTrust{{ToolName: "create_report"}}},
		"the ledger has no row":     {},
		"the ledger cannot be read": {err: errors.New("connection refused")},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tool := &privateActionTool{
				StubActionTool: actionTool("create_report", agent.TierAutoExecute, nil),
				private:        true,
			}

			result := runPrivateWithTrust(
				t, tool, ledger, onRecordedAgent(agent.TierActWithApproval),
			)

			assert.Equal(t, agent.TierPropose, result.Actions[0].Tier)
			assert.False(t, result.Actions[0].Executed)
		})
	}
}

// The ledger is read only for a tool whose tier the agent names, whether or
// not the call is personal, because what set that tier (a person, or trust) is
// recorded with every call; a call on a tool the agent names no tier for
// decides without it.
func TestRun_OnlyATierTheAgentNamesReadsTheLedger(t *testing.T) {
	t.Parallel()

	named := &ledgerTrust{}
	shared := &privateActionTool{
		StubActionTool: actionTool("create_report", agent.TierAutoExecute, nil),
		private:        false,
	}
	result := runPrivateWithTrust(t, shared, named, onRecordedAgent(agent.TierActWithApproval))
	assert.Equal(t, 1, named.asks)
	assert.True(t, result.Actions[0].TierSource.IsValid())

	unnamed := &ledgerTrust{}
	unset := &privateActionTool{
		StubActionTool: actionTool("create_report", agent.TierAutoExecute, nil),
		private:        true,
	}
	result = runPrivateWithTrust(t, unset, unnamed, nil)
	assert.Zero(t, unnamed.asks)
	assert.Equal(t, agent.TierSourcePersonalExemption, result.Actions[0].TierSource)
}
