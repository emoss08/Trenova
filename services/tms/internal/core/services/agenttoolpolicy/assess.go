package agenttoolpolicy

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	HeldByShadowMode     = "shadow_mode"
	HeldBySimulationMode = "simulation_mode"
)

type AssessInput struct {
	Policy     serviceports.ToolPolicy
	Definition *agentdefinition.Definition
	Trust      *agent.ToolTrust
	Control    *tenant.AgentControl
	Tainted    bool
}

func Assess(ctx context.Context, in *AssessInput) serviceports.ToolAutonomy {
	definition := in.Definition
	if definition == nil {
		definition = &agentdefinition.Definition{AutonomyCeiling: agent.TierAutoExecute}
	}

	taint := representativeTaint(in.Tainted)
	best := bestCase(ctx, in, definition, taint)
	worst := worstCase(ctx, in, definition, taint)

	out := serviceports.ToolAutonomy{
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
	if in.Policy.Condition != nil {
		policy.Condition = &serviceports.TierCondition{
			Description: in.Policy.Condition.Description,
			Limit:       heldToProposal,
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

type classifier = func(serviceports.ToolExecuteParams) serviceports.CallPolicy

func fixedCall(class agent.EgressClass) classifier {
	return func(serviceports.ToolExecuteParams) serviceports.CallPolicy {
		return serviceports.CallPolicy{Egress: class}
	}
}

func heldToProposal(context.Context, serviceports.ToolExecuteParams) agent.AutonomyTier {
	return agent.TierPropose
}

func representativeParams(attended bool) serviceports.ToolExecuteParams {
	if !attended {
		return serviceports.ToolExecuteParams{}
	}

	return serviceports.ToolExecuteParams{
		Actor: &serviceports.RequestActor{PrincipalType: serviceports.PrincipalTypeUser},
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
