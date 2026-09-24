package agenttoolpolicy

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

const (
	HeldByAgentCeiling      = "agent_ceiling"
	HeldByToolMax           = "tool_max"
	HeldByEgressClass       = "egress_class"
	HeldByCondition         = "condition"
	HeldByTainted           = "tainted"
	HeldByToolTier          = "tool_tier"
	HeldByPersonalExemption = "personal_exemption"
)

type DecideInput struct {
	Policy          serviceports.ToolPolicy
	Params          serviceports.ToolExecuteParams
	Definition      *agentdefinition.Definition
	Unattended      bool
	TierSetByPerson bool
	Taint           *agent.RunTaint
}

type Decision struct {
	Tier   agent.AutonomyTier
	Egress agent.EgressClass
	HeldBy []string
}

func HeldWhenTainted(class agent.EgressClass) bool {
	return class.Leaves()
}

func Decide(ctx context.Context, in DecideInput) Decision {
	policy := in.Policy
	decision := Decision{Tier: toolTier(in.Definition, policy)}
	if decision.Tier != agent.TierAutoExecute {
		decision.hold(HeldByToolTier)
	}
	decision.lower(effectiveTier(in.Definition, policy), HeldByAgentCeiling)
	decision.lower(maxTier(policy), HeldByToolMax)

	call := policy.Classified(in.Params)
	decision.Egress = call.Egress
	if call.MaxTier.IsValid() {
		decision.lower(call.MaxTier, HeldByEgressClass)
	}
	decision.lower(call.Egress.Ceiling(), HeldByEgressClass)

	if policy.Condition != nil && policy.Condition.Limit != nil {
		decision.lower(conditionLimit(ctx, policy.Condition, in.Params), HeldByCondition)
	}

	if runsUnasked(in, call) {
		decision.Tier = agent.TierAutoExecute
		decision.HeldBy = []string{HeldByPersonalExemption}
		decision.lower(maxTier(policy), HeldByToolMax)
	}

	if HeldWhenTainted(call.Egress) && (in.Taint == nil || in.Taint.Tainted()) {
		decision.lower(agent.TierActWithApproval, HeldByTainted)
	}

	return decision
}

func Promotable(policy serviceports.ToolPolicy) agent.AutonomyTier {
	return maxTier(policy).AtMost(policy.EgressCeiling())
}

func StaticTier(
	definition *agentdefinition.Definition,
	policy serviceports.ToolPolicy,
) agent.AutonomyTier {
	return effectiveTier(definition, policy).AtMost(maxTier(policy))
}

func ExplainTier(policy serviceports.ToolPolicy) string {
	parts := make([]string, 0, 5)
	if limit := Promotable(policy); limit != agent.TierAutoExecute {
		parts = append(parts, fmt.Sprintf(
			"Runs at the tier the agent sets for it, never past %s.", limit))
	} else {
		parts = append(parts, "Runs at the tier the agent sets for it.")
	}
	if policy.Classify != nil {
		parts = append(parts, fmt.Sprintf(
			"What each call reaches decides how far it may go: %s.", egressList(policy.Egress)))
	}
	if policy.Condition != nil {
		parts = append(parts, policy.Condition.Description)
	}
	if policy.PersonalRunsUnasked {
		parts = append(parts, "A call that changes only the caller's own records runs "+
			"without a decision while they are in the conversation, unless a person set "+
			"the tool's tier on the agent.")
	}
	if slices.ContainsFunc(policy.Egress, HeldWhenTainted) {
		parts = append(parts, "A call that leaves the organization waits for approval "+
			"once the run has read text from outside it.")
	}

	return strings.Join(parts, " ")
}

func TierSetByPerson(
	definition *agentdefinition.Definition,
	toolName string,
	trust *agent.ToolTrust,
) bool {
	if definition == nil || !definition.SetsToolTier(toolName) {
		return false
	}

	return trust == nil || !trust.HoldsEarnedTier(definition.ToolTiers[toolName])
}

func (d *Decision) lower(limit agent.AutonomyTier, key string) {
	if !d.Tier.Above(limit) {
		return
	}
	d.Tier = limit
	d.hold(key)
}

func (d *Decision) hold(key string) {
	if !slices.Contains(d.HeldBy, key) {
		d.HeldBy = append(d.HeldBy, key)
	}
}

func SeeksPersonalExemption(in DecideInput) bool {
	if !in.Policy.PersonalRunsUnasked {
		return false
	}

	return personalCall(in, in.Policy.Classified(in.Params))
}

func runsUnasked(in DecideInput, call serviceports.CallPolicy) bool {
	return personalCall(in, call) && !in.TierSetByPerson
}

func personalCall(in DecideInput, call serviceports.CallPolicy) bool {
	actor := in.Params.Actor

	return in.Policy.PersonalRunsUnasked &&
		call.Egress == agent.EgressPersonal &&
		!in.Unattended &&
		actor != nil &&
		actor.PrincipalType == serviceports.PrincipalTypeUser
}

func toolTier(
	definition *agentdefinition.Definition,
	policy serviceports.ToolPolicy,
) agent.AutonomyTier {
	requested := policy.DefaultTier
	if definition != nil {
		if override, ok := definition.ToolTiers[policy.Name]; ok && override.IsValid() {
			requested = override
		}
	}
	if !requested.IsValid() {
		return agent.TierPropose
	}

	return requested
}

func effectiveTier(
	definition *agentdefinition.Definition,
	policy serviceports.ToolPolicy,
) agent.AutonomyTier {
	if definition == nil {
		return toolTier(nil, policy)
	}

	return definition.EffectiveTier(policy.Name, policy.DefaultTier)
}

func maxTier(policy serviceports.ToolPolicy) agent.AutonomyTier {
	if policy.MaxTier.IsValid() {
		return policy.MaxTier
	}

	return agent.TierAutoExecute
}

func conditionLimit(
	ctx context.Context,
	condition *serviceports.TierCondition,
	params serviceports.ToolExecuteParams,
) agent.AutonomyTier {
	limit := condition.Limit(ctx, params)
	if !limit.IsValid() {
		return agent.TierPropose
	}

	return limit
}

func egressList(classes []agent.EgressClass) string {
	names := make([]string, 0, len(classes))
	for _, class := range classes {
		names = append(names, strings.ReplaceAll(class.String(), "_", " "))
	}

	return strings.Join(names, ", ")
}
