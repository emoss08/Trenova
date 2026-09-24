package agenttoolpolicy_test

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/stretchr/testify/assert"
)

func policyReaching(class agent.EgressClass) serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          "write_" + class.String(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipment,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{class},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
	}
}

/*
Every class against every state of the run's reading. A run that read outside
content, or whose reading is unknown, never runs a write that leaves the
organization on its own; what stays inside is decided as if nothing was read.
Only money notices, because the other classes that leave already stop at
ActWithApproval on their own ceiling.
*/
func TestDecide_TaintAgainstEveryEgressClass(t *testing.T) {
	t.Parallel()

	marked := &agent.RunTaint{}
	marked.Add(agent.TaintMark{
		Source:   agent.TaintSourceDocument,
		ToolName: "get_document_summary",
		CallID:   "call_1",
		Ref:      &agent.RecordRef{EntityType: agent.TaintEntityDocument, ID: "doc_1"},
	})
	taints := map[string]*agent.RunTaint{
		"clean":   {},
		"tainted": marked,
		"unknown": nil,
	}

	for _, class := range agent.EgressClasses() {
		policy := policyReaching(class)
		acting := definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
			policy.Name: agent.TierAutoExecute,
		})

		for state, taint := range taints {
			decision := agenttoolpolicy.Decide(t.Context(), agenttoolpolicy.DecideInput{
				Policy:     policy,
				Params:     call(agentPrincipal(), nil),
				Definition: acting,
				Unattended: true,
				Taint:      taint,
			})

			want := class.Ceiling()
			held := class.Leaves() && state != "clean"
			if held {
				want = want.AtMost(agent.TierActWithApproval)
			}
			assert.Equal(t, want, decision.Tier, "%s, %s", class, state)
			assert.Equal(t, class, decision.Egress, "%s, %s", class, state)
			assert.Equal(t,
				held && class.Ceiling() == agent.TierAutoExecute,
				slices.Contains(decision.HeldBy, agenttoolpolicy.HeldByTainted),
				"%s, %s: tainted names what held it only when nothing else already had",
				class, state)
		}
	}
}

func TestDecide_ADifferentMarkOrderIsTheSameDecision(t *testing.T) {
	t.Parallel()

	first := &agent.RunTaint{}
	first.Add(agent.TaintMark{Source: agent.TaintSourceEDI, CallID: "call_1"})
	first.Add(agent.TaintMark{Source: agent.TaintSourceBankReceipt, CallID: "call_2"})
	second := &agent.RunTaint{}
	second.Add(agent.TaintMark{Source: agent.TaintSourceBankReceipt, CallID: "call_2"})
	second.Add(agent.TaintMark{Source: agent.TaintSourceEDI, CallID: "call_1"})

	policy := policyReaching(agent.EgressMoney)
	acting := definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
		policy.Name: agent.TierAutoExecute,
	})
	decide := func(taint *agent.RunTaint) agenttoolpolicy.Decision {
		return agenttoolpolicy.Decide(t.Context(), agenttoolpolicy.DecideInput{
			Policy:     policy,
			Params:     call(person(), nil),
			Definition: acting,
			Taint:      taint,
		})
	}

	assert.Equal(t, decide(first), decide(second))
}
