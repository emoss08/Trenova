package agenttoolpolicy

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

const (
	HeldByAgentCeiling      = "agent_ceiling"
	HeldByToolMax           = "tool_max"
	HeldByEgressClass       = "egress_class"
	HeldByCondition         = "condition"
	HeldByTainted           = agent.HeldByTaintKey
	HeldByToolTier          = "tool_tier"
	HeldByPersonalExemption = "personal_exemption"
	// HeldByBusinessHours marks a write held for a person because the agent
	// may only change things on its own in business hours, and it was not.
	HeldByBusinessHours = "business_hours"
	// HeldByScriptIntent marks a write a script prepared: it waits for a
	// person to approve the script's batch, whatever else would let it run.
	HeldByScriptIntent = "script_intent"
)

type MoneyAllowanceReader func(context.Context) (agenttooltrust.MoneyAllowance, error)

type DecideInput struct {
	Policy          agenttoolports.ToolPolicy
	Params          agenttoolports.ToolExecuteParams
	Definition      *agentdefinition.Definition
	HolderCeiling   agent.AutonomyTier
	Unattended      bool
	TierSetByPerson bool
	Taint           *agent.RunTaint
	MoneyAllowance  MoneyAllowanceReader
	// FromScript is set for an intent a script prepared, which never runs
	// before a person approves the batch it is filed in.
	FromScript bool
}

type Decision struct {
	Tier          agent.AutonomyTier
	Egress        agent.EgressClass
	HeldBy        []string
	Source        agenttooltrust.TierSource
	MoneyMinor    *int64
	MoneyCurrency string
}

func HeldWhenTainted(class agent.EgressClass) bool {
	return class.Leaves()
}

func Decide(ctx context.Context, in DecideInput) Decision {
	policy := &in.Policy
	if !in.FromScript && Escalates(in.Definition, policy) {
		return Decision{
			Tier:   agent.TierAutoExecute,
			Egress: policy.Classified(in.Params).Egress,
			Source: agenttooltrust.TierSourcePolicyDefault,
		}
	}
	decision := Decision{
		Tier:   toolTier(in.Definition, policy),
		Source: toolTierSource(&in),
	}
	if decision.Tier != agent.TierAutoExecute {
		decision.hold(HeldByToolTier)
	}
	decision.lower(effectiveTier(in.Definition, policy), HeldByAgentCeiling)
	if in.HolderCeiling.IsValid() {
		decision.lower(in.HolderCeiling, HeldByAgentCeiling)
	}
	decision.lower(maxTier(policy), HeldByToolMax)

	call := policy.Classified(in.Params)
	decision.Egress = call.Egress
	if call.MaxTier.IsValid() {
		decision.lower(call.MaxTier, HeldByEgressClass)
	}
	taintHolds := policy.TaintHolds(in.Params, call) && (in.Taint == nil || in.Taint.Tainted())
	if call.Egress == agent.EgressMoney {
		if decision.boundMoney(ctx, &in, taintHolds) {
			taintHolds = false
		}
	} else {
		decision.lower(call.Egress.Ceiling(), HeldByEgressClass)
	}

	if policy.Condition != nil && policy.Condition.Limit != nil {
		decision.lower(conditionLimit(ctx, policy.Condition, &in.Params), HeldByCondition)
	}

	if runsUnasked(&in, call) {
		decision.Tier = agent.TierAutoExecute
		decision.Source = agenttooltrust.TierSourcePersonalExemption
		decision.HeldBy = []string{HeldByPersonalExemption}
		decision.lower(maxTier(policy), HeldByToolMax)
	}

	if taintHolds {
		decision.lower(agent.TierActWithApproval, HeldByTainted)
		decision.hold(HeldByTainted)
	}
	if in.FromScript {
		decision.lower(agent.TierActWithApproval, HeldByScriptIntent)
		decision.hold(HeldByScriptIntent)
	}
	if decision.Tier != agent.TierAutoExecute {
		decision.MoneyMinor = nil
		decision.MoneyCurrency = ""
	}

	return decision
}

func Escalates(definition *agentdefinition.Definition, policy *agenttoolports.ToolPolicy) bool {
	if !policy.Escalates {
		return false
	}

	return definition == nil || !definition.SetsToolTier(policy.Name)
}

func Promotable(policy *agenttoolports.ToolPolicy) agent.AutonomyTier {
	return maxTier(policy).AtMost(policy.EgressCeiling())
}

// AlwaysProposes reports a tool every call of which records a proposal on
// this agent, whatever it reaches and whatever trust it earns: its own
// ceiling or its egress keeps it below running on its own, or the agent's
// ceiling does. The prompt groups only these as changes that wait for a
// person; it used to call a tool whose static tier ran at once a change that
// runs as soon as it is called, and a call that the turn's taint, a
// condition or what it reached held for a person contradicted it.
func AlwaysProposes(
	definition *agentdefinition.Definition,
	policy *agenttoolports.ToolPolicy,
) bool {
	if Escalates(definition, policy) {
		return false
	}
	if agent.TierAutoExecute.Above(Promotable(policy)) {
		return true
	}
	if definition == nil || !definition.AutonomyCeiling.IsValid() {
		return false
	}

	return agent.TierAutoExecute.Above(definition.AutonomyCeiling)
}

func StaticTier(
	definition *agentdefinition.Definition,
	policy *agenttoolports.ToolPolicy,
) agent.AutonomyTier {
	if Escalates(definition, policy) {
		return agent.TierAutoExecute
	}

	return effectiveTier(definition, policy).AtMost(maxTier(policy))
}

func ExplainTier(policy *agenttoolports.ToolPolicy) string {
	parts := make([]string, 0, 6)
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
	if policy.Escalates {
		parts = append(parts, "Runs on its own whatever the agent's ceiling, because it only "+
			"puts the run in front of a person in the organization, unless a person set the "+
			"tool's tier on the agent.")
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
	if policy.TaintHold != nil {
		parts = append(parts, policy.TaintHold.Description)
	}

	return strings.Join(parts, " ")
}

func TierSetByPerson(
	definition *agentdefinition.Definition,
	toolName string,
	trust *agenttooltrust.ToolTrust,
) bool {
	if definition == nil || !definition.SetsToolTier(toolName) {
		return false
	}

	return trust == nil || !trust.HoldsEarnedTier(definition.ToolTiers[toolName])
}

func (d *Decision) boundMoney(ctx context.Context, in *DecideInput, taintHolds bool) bool {
	if d.Tier != agent.TierAutoExecute {
		return false
	}
	if !in.Policy.Bounded() {
		d.lower(agent.TierActWithApproval, agenttooltrust.HeldByMoneyUnbounded)
		return false
	}
	if taintHolds && !in.Policy.Condition.MayVouch(in.Taint) {
		return false
	}

	held, check := moneyHeld(ctx, in)
	if held != "" {
		d.lower(agent.TierActWithApproval, held)
		return false
	}
	if taintHolds && !in.Policy.Condition.Vouched(in.Taint, &check) {
		return false
	}
	amount := check.AmountMinor
	d.MoneyMinor = &amount
	d.MoneyCurrency = check.Currency

	return taintHolds
}

func moneyHeld(ctx context.Context, in *DecideInput) (string, agenttoolports.MoneyBound) {
	check, proven := provenBound(ctx, in)
	if !proven {
		return agenttooltrust.HeldByMoneyUnproven, check
	}
	if !check.Within {
		if slices.Contains(agenttooltrust.MoneyConditionHeldByKeys(), check.Held) {
			return check.Held, check
		}
		return agenttooltrust.HeldByMoneyUnproven, check
	}
	if in.MoneyAllowance == nil {
		return agenttooltrust.HeldByMoneyLimitsUnread, check
	}

	allowance, err := in.MoneyAllowance(ctx)
	if err != nil {
		return agenttooltrust.HeldByMoneyLimitsUnread, check
	}

	return allowance.Exceeded(check.AmountMinor, check.Currency), check
}

func provenBound(
	ctx context.Context,
	in *DecideInput,
) (bound agenttoolports.MoneyBound, proven bool) {
	defer func() {
		if recover() != nil {
			bound, proven = agenttoolports.MoneyBound{}, false
		}
	}()

	return in.Policy.Condition.Bounds(ctx, in.Params), true
}

func (d *Decision) lower(limit agent.AutonomyTier, key string) {
	if !d.Tier.Above(limit) {
		return
	}
	d.Tier = limit
	d.Source = agenttooltrust.TierSourcePolicyDefault
	d.hold(key)
}

func toolTierSource(in *DecideInput) agenttooltrust.TierSource {
	if in.Definition == nil || !in.Definition.SetsToolTier(in.Policy.Name) {
		return agenttooltrust.TierSourcePolicyDefault
	}
	if in.TierSetByPerson {
		return agenttooltrust.TierSourcePersonSetting
	}

	return agenttooltrust.TierSourceTrustEarned
}

func (d *Decision) hold(key string) {
	if !slices.Contains(d.HeldBy, key) {
		d.HeldBy = append(d.HeldBy, key)
	}
}

func SeeksPersonalExemption(in *DecideInput) bool {
	if !in.Policy.PersonalRunsUnasked {
		return false
	}

	return personalCall(in, in.Policy.Classified(in.Params))
}

func runsUnasked(in *DecideInput, call agenttoolports.CallPolicy) bool {
	return personalCall(in, call) && !in.TierSetByPerson
}

func personalCall(in *DecideInput, call agenttoolports.CallPolicy) bool {
	actor := in.Params.Actor

	return in.Policy.PersonalRunsUnasked &&
		call.Egress == agent.EgressPersonal &&
		!in.Unattended &&
		actor != nil &&
		actor.PrincipalType == permissionports.PrincipalTypeUser
}

func toolTier(
	definition *agentdefinition.Definition,
	policy *agenttoolports.ToolPolicy,
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
	policy *agenttoolports.ToolPolicy,
) agent.AutonomyTier {
	if definition == nil {
		return toolTier(nil, policy)
	}

	return definition.EffectiveTier(policy.Name, policy.DefaultTier)
}

func maxTier(policy *agenttoolports.ToolPolicy) agent.AutonomyTier {
	if policy.MaxTier.IsValid() {
		return policy.MaxTier
	}

	return agent.TierAutoExecute
}

func conditionLimit(
	ctx context.Context,
	condition *agenttoolports.TierCondition,
	params *agenttoolports.ToolExecuteParams,
) agent.AutonomyTier {
	limit := condition.Limit(ctx, *params)
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
