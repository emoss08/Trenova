package services

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
)

type ToolPolicy struct {
	Name                string
	Kind                agent.ToolKind
	Resource            permission.Resource
	Operation           permission.Operation
	Scope               agent.ToolScope
	DefaultTier         agent.AutonomyTier
	MaxTier             agent.AutonomyTier
	Egress              []agent.EgressClass
	Classify            func(ToolExecuteParams) CallPolicy
	Condition           *TierCondition
	PersonalRunsUnasked bool
	Effect              agent.ToolEffect
	Artifact            string
	Reversible          bool
	Idempotent          bool
	ReadsExternal       agent.ExternalRead
	Source              agent.TaintSource
	CarriesTaint        bool
	Rationale           string
}

type CallPolicy struct {
	Egress  agent.EgressClass
	MaxTier agent.AutonomyTier
}

type TierCondition struct {
	Description string
	Limit       func(context.Context, ToolExecuteParams) agent.AutonomyTier
}

type ToolPolicyDeclarer interface {
	Policy() ToolPolicy
}

type RuntimeToolPolicies []ToolPolicy

func PolicyOf(tool any) (ToolPolicy, bool) {
	declarer, ok := tool.(ToolPolicyDeclarer)
	if !ok {
		return ToolPolicy{}, false
	}

	return declarer.Policy(), true
}

func (p ToolPolicy) Classified(params ToolExecuteParams) CallPolicy {
	if p.Classify != nil {
		return p.Classify(params)
	}
	if len(p.Egress) == 1 {
		return CallPolicy{Egress: p.Egress[0]}
	}

	return CallPolicy{Egress: p.HighestEgress()}
}

func (p ToolPolicy) HighestEgress() agent.EgressClass {
	highest := agent.EgressNone
	for _, class := range p.Egress {
		if egressRank(class) > egressRank(highest) {
			highest = class
		}
	}

	return highest
}

func (p ToolPolicy) EgressCeiling() agent.AutonomyTier {
	if len(p.Egress) == 0 {
		return agent.TierAutoExecute
	}

	ceiling := agent.TierPropose
	for _, class := range p.Egress {
		if class.Ceiling().Above(ceiling) {
			ceiling = class.Ceiling()
		}
	}

	return ceiling
}

func (p ToolPolicy) HasEgress(class agent.EgressClass) bool {
	return slices.Contains(p.Egress, class)
}

func (p ToolPolicy) Conditional() bool {
	return p.Classify != nil || p.Condition != nil || p.PersonalRunsUnasked
}

func egressRank(class agent.EgressClass) int {
	return slices.Index(agent.EgressClasses(), class)
}

func EffectOf(tool any) agent.ToolEffect {
	policy, ok := PolicyOf(tool)
	if !ok {
		return ""
	}

	return policy.EffectiveEffect()
}

func (p ToolPolicy) EffectiveEffect() agent.ToolEffect {
	if p.Effect.IsValid() {
		return p.Effect
	}

	switch p.Kind {
	case agent.ToolKindAction:
		return agent.ToolEffectChange
	case agent.ToolKindQuery:
		return agent.ToolEffectLookup
	default:
		return ""
	}
}

func IsSelfScoped(tool any) bool {
	policy, ok := PolicyOf(tool)

	return ok && policy.Scope == agent.ToolScopeSelf
}
