package agenttoolpolicy

import (
	"context"
	"math"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	HeldByShadowMode     = "shadow_mode"
	HeldBySimulationMode = "simulation_mode"
)

type AssessInput struct {
	Policy     agenttoolports.ToolPolicy
	Definition *agentdefinition.Definition
	Trust      *agent.ToolTrust
	Control    *agentcontrol.AgentControl
	Tainted    bool
}

func Assess(ctx context.Context, in *AssessInput) agenttoolports.ToolAutonomy {
	definition := in.Definition
	if definition == nil {
		definition = &agentdefinition.Definition{AutonomyCeiling: agent.TierAutoExecute}
	}

	taint := representativeTaint(in.Tainted)
	best := bestCase(ctx, in, definition, taint)
	worst := worstCase(ctx, in, definition, taint)

	out := agenttoolports.ToolAutonomy{
		Tier:   best.Tier,
		HeldBy: mergeHeld(best.HeldBy, worst.HeldBy),
	}
	if best.Tier == agent.TierAutoExecute && worst.Tier != agent.TierAutoExecute {
		out.Answer = agent.AutonomyConditional
	} else {
		out.Answer = agent.AnswerForTier(best.Tier)
	}

	shadow := in.Policy.Kind == agent.ToolKindAction &&
		definition.EffectiveShadow(in.Control != nil && in.Control.ShadowMode)
	if in.Policy.Kind == agent.ToolKindAction {
		if shadow {
			out.Answer = agent.AutonomySimulated
			out.HeldBy = mergeHeld(out.HeldBy, []string{HeldByShadowMode})
		}
		if definition.SimulationMode {
			out.Answer = agent.AutonomySimulated
			out.HeldBy = mergeHeld(out.HeldBy, []string{HeldBySimulationMode})
		}
	}

	out.Earned = in.Trust != nil &&
		definition.SetsToolTier(in.Policy.Name) &&
		in.Trust.HoldsEarnedTier(definition.ToolTiers[in.Policy.Name])
	if !shadow {
		out.ApprovalsToNext = approvalsToNext(in, definition)
	}

	return out
}

func bestCase(
	ctx context.Context,
	in *AssessInput,
	definition *agentdefinition.Definition,
	taint *agent.RunTaint,
) Decision {
	attended := !definition.IsBackground()
	policy := in.Policy
	policy.Condition = nil
	if in.Policy.Bounded() {
		policy.Condition = &agenttoolports.TierCondition{
			Description: in.Policy.Condition.Description,
			Bounds:      withinBounds,
			Vouches:     in.Policy.Condition.Vouches,
		}
	}
	policy.TaintHold = nil

	classes := in.Policy.Egress
	if len(classes) == 0 {
		classes = []agent.EgressClass{in.Policy.HighestEgress()}
	}

	var best Decision
	for idx, class := range classes {
		policy.Classify = fixedCall(class)
		decision := Decide(ctx, DecideInput{
			Policy:          policy,
			Params:          representativeParams(attended),
			Definition:      definition,
			Unattended:      !attended,
			TierSetByPerson: TierSetByPerson(definition, policy.Name, in.Trust),
			Taint:           taint,
			MoneyAllowance:  unspentAllowance,
		})
		if idx == 0 || decision.Tier.Above(best.Tier) {
			best = decision
		}
	}

	return best
}

func worstCase(
	ctx context.Context,
	in *AssessInput,
	definition *agentdefinition.Definition,
	taint *agent.RunTaint,
) Decision {
	policy := in.Policy
	policy.Classify = fixedCall(in.Policy.HighestEgress())
	if condition := in.Policy.Condition; condition != nil {
		policy.Condition = &agenttoolports.TierCondition{Description: condition.Description}
		if condition.Limit != nil {
			policy.Condition.Limit = heldToProposal
		}
		if condition.Bounds != nil {
			policy.Condition.Bounds = outsideBounds
		}
	}
	if in.Policy.TaintHold != nil {
		policy.TaintHold = &agenttoolports.TaintHold{
			Description: in.Policy.TaintHold.Description,
			Applies:     alwaysHeld,
		}
	}

	return Decide(ctx, DecideInput{
		Policy:          policy,
		Params:          representativeParams(false),
		Definition:      definition,
		Unattended:      true,
		TierSetByPerson: TierSetByPerson(definition, policy.Name, in.Trust),
		Taint:           taint,
	})
}

func approvalsToNext(in *AssessInput, definition *agentdefinition.Definition) *int {
	control := in.Control
	if in.Policy.Kind != agent.ToolKindAction || control == nil || !control.EarnedAutonomy ||
		control.PromotionThreshold <= 0 {
		return nil
	}

	current := definition.EffectiveTier(in.Policy.Name, in.Policy.DefaultTier)
	next, ok := current.Next()
	if !ok || !definition.WithinCeiling(next) || next.Above(Promotable(in.Policy)) {
		return nil
	}

	streak := 0
	if in.Trust != nil {
		streak = in.Trust.Streak
	}
	remaining := max(control.PromotionThreshold-streak, 1)

	return &remaining
}

type classifier = func(agenttoolports.ToolExecuteParams) agenttoolports.CallPolicy

func fixedCall(class agent.EgressClass) classifier {
	return func(agenttoolports.ToolExecuteParams) agenttoolports.CallPolicy {
		return agenttoolports.CallPolicy{Egress: class}
	}
}

func heldToProposal(context.Context, agenttoolports.ToolExecuteParams) agent.AutonomyTier {
	return agent.TierPropose
}

const representativeCurrency = "USD"

func withinBounds(context.Context, agenttoolports.ToolExecuteParams) agenttoolports.MoneyBound {
	return agenttoolports.WithinMoneyBound(0, representativeCurrency)
}

func outsideBounds(context.Context, agenttoolports.ToolExecuteParams) agenttoolports.MoneyBound {
	return agenttoolports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
}

func unspentAllowance(context.Context) (agent.MoneyAllowance, error) {
	return agent.MoneyAllowance{
		Currency: representativeCurrency,
		Limits: agent.MoneyLimits{
			CallCeilingMinor: math.MaxInt64,
			RunCalls:         math.MaxInt,
			RunAmountMinor:   math.MaxInt64,
			DayCalls:         math.MaxInt,
			DayAmountMinor:   math.MaxInt64,
		},
	}, nil
}

func alwaysHeld(agenttoolports.ToolExecuteParams) bool {
	return true
}

func representativeParams(attended bool) agenttoolports.ToolExecuteParams {
	if !attended {
		return agenttoolports.ToolExecuteParams{}
	}

	return agenttoolports.ToolExecuteParams{
		Actor: &permissionports.RequestActor{PrincipalType: permissionports.PrincipalTypeUser},
	}
}

func representativeTaint(tainted bool) *agent.RunTaint {
	if !tainted {
		return &agent.RunTaint{}
	}

	return &agent.RunTaint{Marks: []agent.TaintMark{{Source: agent.TaintSourceInboundMessage}}}
}

func mergeHeld(first, second []string) []string {
	return sliceutils.Dedupe(slices.Concat(first, second))
}
