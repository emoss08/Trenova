package agenttoolpolicy_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func person() *serviceports.RequestActor {
	user := pulid.MustNew("usr_")

	return &serviceports.RequestActor{
		PrincipalType: serviceports.PrincipalTypeUser,
		PrincipalID:   user,
		UserID:        user,
	}
}

func agentPrincipal() *serviceports.RequestActor {
	return &serviceports.RequestActor{
		PrincipalType: serviceports.PrincipalTypeAgent,
		PrincipalID:   pulid.MustNew("agdef_"),
	}
}

func definition(
	ceiling agent.AutonomyTier,
	tiers map[string]agent.AutonomyTier,
) *agentdefinition.Definition {
	return &agentdefinition.Definition{AutonomyCeiling: ceiling, ToolTiers: tiers}
}

func call(actor *serviceports.RequestActor, args map[string]any) serviceports.ToolExecuteParams {
	return serviceports.ToolExecuteParams{Actor: actor, Params: args}
}

func withCondition(
	policy serviceports.ToolPolicy,
	limit agent.AutonomyTier,
) serviceports.ToolPolicy {
	description := policy.Condition.Description
	policy.Condition = &serviceports.TierCondition{
		Description: description,
		Limit: func(context.Context, serviceports.ToolExecuteParams) agent.AutonomyTier {
			return limit
		},
	}

	return policy
}

func clean() *agent.RunTaint { return &agent.RunTaint{} }

func tainted() *agent.RunTaint {
	taint := &agent.RunTaint{}
	taint.Add(agent.TaintMark{
		Source:   agent.TaintSourceInboundMessage,
		ToolName: "get_inbound_message",
		CallID:   "call_1",
		Ref:      &agent.RecordRef{EntityType: "inbound_message", ID: "imsg_1"},
	})

	return taint
}

type decideCase struct {
	name   string
	in     agenttoolpolicy.DecideInput
	tier   agent.AutonomyTier
	egress agent.EgressClass
	heldBy []string
}

func runDecideCases(t *testing.T, cases []decideCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decision := agenttoolpolicy.Decide(t.Context(), tc.in)
			assert.Equal(t, tc.tier, decision.Tier)
			if tc.egress != "" {
				assert.Equal(t, tc.egress, decision.Egress)
			}
			switch {
			case tc.heldBy == nil:
			case len(tc.heldBy) == 0:
				assert.Empty(t, decision.HeldBy)
			default:
				assert.Equal(t, tc.heldBy, decision.HeldBy)
			}
		})
	}
}

/*
A report saved to the person's own list runs as they asked, whatever the
agent's ceiling; one the organization would see waits for approval. These are
the owner's rules from before the spec existed, reproduced exactly.
*/
func TestDecide_CreateReport(t *testing.T) {
	t.Parallel()

	policy := registeredPolicy(t, "create_report")
	private := map[string]any{
		"name":       "Late loads",
		"visibility": string(report.VisibilityPrivate),
	}
	shared := map[string]any{
		"name":       "Late loads",
		"visibility": string(report.VisibilityShared),
	}

	runDecideCases(t, []decideCase{
		{
			name: "a private report runs for the person whatever the ceiling",
			in: agenttoolpolicy.DecideInput{
				Policy:     policy,
				Params:     call(person(), private),
				Definition: definition(agent.TierPropose, nil),
				Taint:      clean(),
			},
			tier:   agent.TierAutoExecute,
			egress: agent.EgressPersonal,
			heldBy: []string{agenttoolpolicy.HeldByPersonalExemption},
		},
		{
			name: "a shared report waits for approval under an agent set to act alone",
			in: agenttoolpolicy.DecideInput{
				Policy:     policy,
				Params:     call(person(), shared),
				Definition: definition(agent.TierAutoExecute, nil),
				Taint:      clean(),
			},
			tier:   agent.TierActWithApproval,
			egress: agent.EgressInternal,
			heldBy: []string{agenttoolpolicy.HeldByEgressClass},
		},
		{
			name: "a private report nobody is watching keeps the agent's tier",
			in: agenttoolpolicy.DecideInput{
				Policy:     policy,
				Params:     call(person(), private),
				Definition: definition(agent.TierPropose, nil),
				Unattended: true,
				Taint:      clean(),
			},
			tier:   agent.TierPropose,
			heldBy: []string{agenttoolpolicy.HeldByAgentCeiling},
		},
		{
			name: "a report an agent principal saves is not anyone's own",
			in: agenttoolpolicy.DecideInput{
				Policy:     policy,
				Params:     call(agentPrincipal(), private),
				Definition: definition(agent.TierAutoExecute, nil),
				Taint:      clean(),
			},
			tier:   agent.TierActWithApproval,
			egress: agent.EgressInternal,
		},
		{
			name: "a tier a person set on the agent is kept",
			in: agenttoolpolicy.DecideInput{
				Policy: policy,
				Params: call(person(), private),
				Definition: definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
					"create_report": agent.TierPropose,
				}),
				TierSetByPerson: true,
				Taint:           clean(),
			},
			tier:   agent.TierPropose,
			heldBy: []string{agenttoolpolicy.HeldByToolTier},
		},
		{
			name: "bug 3: a tier trust wrote does not take the exemption away",
			in: agenttoolpolicy.DecideInput{
				Policy: policy,
				Params: call(person(), private),
				Definition: definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
					"create_report": agent.TierActWithApproval,
				}),
				TierSetByPerson: false,
				Taint:           clean(),
			},
			tier:   agent.TierAutoExecute,
			heldBy: []string{agenttoolpolicy.HeldByPersonalExemption},
		},
	})
}

func TestDecide_SaveTableViewRunsAPrivateViewUnasked(t *testing.T) {
	t.Parallel()

	policy := registeredPolicy(t, "save_table_view")
	runDecideCases(t, []decideCase{
		{
			name: "a private view is the person's own",
			in: agenttoolpolicy.DecideInput{
				Policy:     policy,
				Params:     call(person(), map[string]any{"entity": "shipment"}),
				Definition: definition(agent.TierPropose, nil),
				Taint:      clean(),
			},
			tier:   agent.TierAutoExecute,
			egress: agent.EgressPersonal,
		},
		{
			name: "a shared view keeps the agent's tier",
			in: agenttoolpolicy.DecideInput{
				Policy:     policy,
				Params:     call(person(), map[string]any{"entity": "shipment", "shared": true}),
				Definition: definition(agent.TierPropose, nil),
				Taint:      clean(),
			},
			tier:   agent.TierPropose,
			egress: agent.EgressInternal,
		},
	})
}

// An inbox message runs only as far as the mailbox it arrived on allows,
// whatever the desk has earned; a reply never runs past approval.
func TestDecide_MailboxGate(t *testing.T) {
	t.Parallel()

	earned := definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
		"mark_inbound_message":     agent.TierAutoExecute,
		"reply_to_inbound_message": agent.TierAutoExecute,
	})
	mark := registeredPolicy(t, "mark_inbound_message")
	reply := registeredPolicy(t, "reply_to_inbound_message")

	runDecideCases(t, []decideCase{
		{
			name: "a message held for review is a proposal",
			in: agenttoolpolicy.DecideInput{
				Policy:     withCondition(mark, agent.TierPropose),
				Params:     call(agentPrincipal(), nil),
				Definition: earned,
				Taint:      clean(),
			},
			tier:   agent.TierPropose,
			heldBy: []string{agenttoolpolicy.HeldByCondition},
		},
		{
			name: "a message the mailbox handles runs",
			in: agenttoolpolicy.DecideInput{
				Policy:     withCondition(mark, agent.TierAutoExecute),
				Params:     call(agentPrincipal(), nil),
				Definition: earned,
				Taint:      clean(),
			},
			tier: agent.TierAutoExecute,
		},
		{
			name: "a reply still waits for a person",
			in: agenttoolpolicy.DecideInput{
				Policy:     withCondition(reply, agent.TierAutoExecute),
				Params:     call(agentPrincipal(), nil),
				Definition: earned,
				Taint:      clean(),
			},
			tier:   agent.TierActWithApproval,
			heldBy: []string{agenttoolpolicy.HeldByToolMax},
		},
		{
			name: "a reply on a held message is a proposal",
			in: agenttoolpolicy.DecideInput{
				Policy:     withCondition(reply, agent.TierPropose),
				Params:     call(agentPrincipal(), nil),
				Definition: earned,
				Taint:      clean(),
			},
			tier: agent.TierPropose,
			heldBy: []string{
				agenttoolpolicy.HeldByToolMax,
				agenttoolpolicy.HeldByCondition,
			},
		},
	})
}

func TestDecide_CommentVisibility(t *testing.T) {
	t.Parallel()

	policy := registeredPolicy(t, "add_shipment_comment")
	actsAlone := definition(agent.TierAutoExecute, nil)
	comment := func(visibility string) map[string]any {
		return map[string]any{"shipmentId": "shp_1", "visibility": visibility}
	}

	runDecideCases(t, []decideCase{
		{
			name: "an internal note runs",
			in: agenttoolpolicy.DecideInput{
				Policy:     policy,
				Params:     call(person(), comment("Internal")),
				Definition: actsAlone,
				Taint:      clean(),
			},
			tier:   agent.TierAutoExecute,
			egress: agent.EgressInternal,
		},
		{
			name: "a note the customer reads waits",
			in: agenttoolpolicy.DecideInput{
				Policy:     policy,
				Params:     call(person(), comment("Customer")),
				Definition: actsAlone,
				Taint:      clean(),
			},
			tier:   agent.TierActWithApproval,
			egress: agent.EgressCustomerVisible,
			heldBy: []string{agenttoolpolicy.HeldByEgressClass},
		},
		{
			name: "a note the driver reads waits",
			in: agenttoolpolicy.DecideInput{
				Policy:     policy,
				Params:     call(person(), comment("Driver")),
				Definition: actsAlone,
				Taint:      clean(),
			},
			tier:   agent.TierActWithApproval,
			egress: agent.EgressDriverVisible,
		},
	})
}

// What leaves the organization is a person's decision, however high the agent
// is set; bugs 1 and 2 were two tools the cap never reached.
func TestDecide_OutboundApprovalCap(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"email_customer",
		"send_detention_notice",
		"notify_driver",
		"request_missing_docs",
		"request_credential_renewal",
		"tender_move_to_routing_guide",
		"tender_move_to_carriers",
		"schedule_report",
		"create_shipment",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			policy := registeredPolicy(t, name)
			decision := agenttoolpolicy.Decide(t.Context(), agenttoolpolicy.DecideInput{
				Policy: policy,
				Params: call(person(), nil),
				Definition: definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
					name: agent.TierAutoExecute,
				}),
				Taint: clean(),
			})
			assert.Equal(t, agent.TierActWithApproval, decision.Tier)
			assert.Equal(t, []string{agenttoolpolicy.HeldByToolMax}, decision.HeldBy)
			assert.Equal(t, agent.TierActWithApproval, agenttoolpolicy.Promotable(policy))
		})
	}
}

// A self-scoped tool keeps the agent's tier: it is personal, but not one the
// owner exempted from approval.
func TestDecide_SelfScopedTools(t *testing.T) {
	t.Parallel()

	policy := registeredPolicy(t, "add_home_widget")
	assert.Equal(t, agent.ToolScopeSelf, policy.Scope)

	runDecideCases(t, []decideCase{
		{
			name: "at the tool's default",
			in: agenttoolpolicy.DecideInput{
				Policy:     policy,
				Params:     call(person(), nil),
				Definition: definition(agent.TierAutoExecute, nil),
				Taint:      clean(),
			},
			tier:   agent.TierPropose,
			egress: agent.EgressPersonal,
			heldBy: []string{agenttoolpolicy.HeldByToolTier},
		},
		{
			name: "raised by the agent",
			in: agenttoolpolicy.DecideInput{
				Policy: policy,
				Params: call(person(), nil),
				Definition: definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
					"add_home_widget": agent.TierAutoExecute,
				}),
				Taint: tainted(),
			},
			tier:   agent.TierAutoExecute,
			heldBy: []string{},
		},
	})
}

// Simulation and shadow mode decide whether an automatic write is made, not
// which tier it has; the tier is the same either way.
func TestDecide_SimulationAndShadowDoNotMoveTheTier(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"assign_move", "create_report", "email_customer"} {
		policy := registeredPolicy(t, name)
		plain := definition(agent.TierAutoExecute, nil)
		simulated := definition(agent.TierAutoExecute, nil)
		simulated.SimulationMode = true
		simulated.ShadowMode = true

		params := call(person(), map[string]any{"name": "Late loads"})
		want := agenttoolpolicy.Decide(t.Context(), agenttoolpolicy.DecideInput{
			Policy: policy, Params: params, Definition: plain, Taint: clean(),
		})
		got := agenttoolpolicy.Decide(t.Context(), agenttoolpolicy.DecideInput{
			Policy: policy, Params: params, Definition: simulated, Taint: clean(),
		})
		assert.Equal(t, want, got, name)
	}
}

func TestDecide_AgentCeiling(t *testing.T) {
	t.Parallel()

	runDecideCases(t, []decideCase{
		{
			name: "the agent's ceiling holds a tool set higher",
			in: agenttoolpolicy.DecideInput{
				Policy: registeredPolicy(t, "assign_move"),
				Params: call(person(), nil),
				Definition: definition(agent.TierActWithApproval, map[string]agent.AutonomyTier{
					"assign_move": agent.TierAutoExecute,
				}),
				TierSetByPerson: true,
				Taint:           clean(),
			},
			tier:   agent.TierActWithApproval,
			heldBy: []string{agenttoolpolicy.HeldByAgentCeiling},
		},
	})
}

/*
A run that read outside text, or whose reading is unknown, cannot send or move
money on its own. Internal and personal work is untouched. The dispatch path
passes the taint the turn had when it made the call.
*/
func TestDecide_TaintHoldsWhatLeaves(t *testing.T) {
	t.Parallel()

	money := registeredPolicy(t, "post_customer_payment")
	internal := registeredPolicy(t, "assign_move")
	actsAlone := func(name string) *agentdefinition.Definition {
		return definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
			name: agent.TierAutoExecute,
		})
	}

	runDecideCases(t, []decideCase{
		{
			name: "a clean run posts a payment it earned",
			in: agenttoolpolicy.DecideInput{
				Policy:     money,
				Params:     call(agentPrincipal(), nil),
				Definition: actsAlone("post_customer_payment"),
				Taint:      clean(),
			},
			tier: agent.TierAutoExecute,
		},
		{
			name: "a tainted run waits",
			in: agenttoolpolicy.DecideInput{
				Policy:     money,
				Params:     call(agentPrincipal(), nil),
				Definition: actsAlone("post_customer_payment"),
				Taint:      tainted(),
			},
			tier:   agent.TierActWithApproval,
			heldBy: []string{agenttoolpolicy.HeldByTainted},
		},
		{
			name: "an unknown taint is treated as tainted",
			in: agenttoolpolicy.DecideInput{
				Policy:     money,
				Params:     call(agentPrincipal(), nil),
				Definition: actsAlone("post_customer_payment"),
			},
			tier:   agent.TierActWithApproval,
			heldBy: []string{agenttoolpolicy.HeldByTainted},
		},
		{
			name: "internal work is untouched",
			in: agenttoolpolicy.DecideInput{
				Policy:     internal,
				Params:     call(agentPrincipal(), nil),
				Definition: actsAlone("assign_move"),
				Taint:      tainted(),
			},
			tier: agent.TierAutoExecute,
		},
	})

	for _, class := range agent.EgressClasses() {
		assert.Equal(t, class.Leaves(), agenttoolpolicy.HeldWhenTainted(class), class)
	}
}

func TestPromotable(t *testing.T) {
	t.Parallel()

	cases := map[string]agent.AutonomyTier{
		"assign_move":              agent.TierAutoExecute,
		"create_report":            agent.TierAutoExecute,
		"add_shipment_comment":     agent.TierAutoExecute,
		"post_customer_payment":    agent.TierAutoExecute,
		"email_customer":           agent.TierActWithApproval,
		"reply_to_inbound_message": agent.TierActWithApproval,
		"create_shipment":          agent.TierActWithApproval,
		"schedule_report":          agent.TierActWithApproval,
		"cancel_shipment":          agent.TierActWithApproval,
	}
	for name, want := range cases {
		assert.Equal(t, want, agenttoolpolicy.Promotable(registeredPolicy(t, name)), name)
	}

	uncapped := serviceports.ToolPolicy{
		MaxTier: agent.TierAutoExecute,
		Egress:  []agent.EgressClass{agent.EgressDriverVisible},
	}
	assert.Equal(t, agent.TierActWithApproval, agenttoolpolicy.Promotable(uncapped),
		"a class that leaves caps a max tier set above it")
}

func TestStaticTierAndExplainTier(t *testing.T) {
	t.Parallel()

	actsAlone := definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
		"create_shipment": agent.TierAutoExecute,
	})
	assert.Equal(t, agent.TierActWithApproval,
		agenttoolpolicy.StaticTier(actsAlone, registeredPolicy(t, "create_shipment")))

	explained := agenttoolpolicy.ExplainTier(registeredPolicy(t, "add_shipment_comment"))
	assert.Contains(t, explained, "customer visible")
	assert.Contains(t, explained, "waits for approval")
}

func TestTierSetByPerson(t *testing.T) {
	t.Parallel()

	d := definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
		"create_report": agent.TierAutoExecute,
	})
	earned := &agent.ToolTrust{EarnedTier: agent.TierAutoExecute}
	chosen := &agent.ToolTrust{EarnedTier: agent.TierActWithApproval}

	assert.False(t, agenttoolpolicy.TierSetByPerson(d, "create_report", earned))
	assert.True(t, agenttoolpolicy.TierSetByPerson(d, "create_report", chosen))
	assert.True(t, agenttoolpolicy.TierSetByPerson(d, "create_report", nil))
	assert.False(t, agenttoolpolicy.TierSetByPerson(d, "assign_move", nil))
}

// Whether a person set the tier matters only to a call that would otherwise
// run unasked, so the runtime reads the trust ledger for that call alone.
func TestSeeksPersonalExemption(t *testing.T) {
	t.Parallel()

	policy := registeredPolicy(t, "create_report")
	private := map[string]any{"name": "Late loads", "visibility": string(report.VisibilityPrivate)}
	shared := map[string]any{"name": "Late loads", "visibility": string(report.VisibilityShared)}

	cases := []struct {
		name string
		in   agenttoolpolicy.DecideInput
		want bool
	}{
		{
			name: "a private report for a person in the conversation",
			in:   agenttoolpolicy.DecideInput{Policy: policy, Params: call(person(), private)},
			want: true,
		},
		{
			name: "whatever tier is set on the agent",
			in: agenttoolpolicy.DecideInput{
				Policy:          policy,
				Params:          call(person(), private),
				TierSetByPerson: true,
			},
			want: true,
		},
		{
			name: "a shared report",
			in:   agenttoolpolicy.DecideInput{Policy: policy, Params: call(person(), shared)},
		},
		{
			name: "a run nobody is watching",
			in: agenttoolpolicy.DecideInput{
				Policy:     policy,
				Params:     call(person(), private),
				Unattended: true,
			},
		},
		{
			name: "an agent principal",
			in: agenttoolpolicy.DecideInput{
				Policy: policy,
				Params: call(agentPrincipal(), private),
			},
		},
		{
			name: "a personal tool the owner did not exempt",
			in: agenttoolpolicy.DecideInput{
				Policy: registeredPolicy(t, "add_home_widget"),
				Params: call(person(), nil),
			},
		},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, agenttoolpolicy.SeeksPersonalExemption(tc.in), tc.name)
	}
}
