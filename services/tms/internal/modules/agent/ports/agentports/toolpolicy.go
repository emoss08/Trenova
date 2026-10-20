package agentports

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/money"
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
	TaintHold           *TaintHold
	PersonalRunsUnasked bool
	Escalates           bool
	Effect              agent.ToolEffect
	Artifact            string
	Reversible          bool
	Idempotent          bool
	ReadsExternal       agent.ExternalRead
	Source              agent.TaintSource
	Sources             []agent.TaintSource
	CarriesTaint        bool
	CapsLaterWrites     bool
	Script              ScriptUse
	Rationale           string
}

// ScriptUse says what a script may do with a tool: call it through the
// broker (the default for a read), never call it, or run the script itself.
type ScriptUse string

const (
	ScriptCallable = ScriptUse("")
	ScriptExcluded = ScriptUse("excluded")
	ScriptRunner   = ScriptUse("runner")
)

// CallableFromScript says a script's broker may run the tool: a read whose
// policy does not keep it from scripts, that touches only what the caller may
// read and reads nothing from outside the organization a script could relay
// into a later write.
func (p *ToolPolicy) CallableFromScript() bool {
	return p.Kind == agent.ToolKindQuery &&
		p.EffectiveEffect() == agent.ToolEffectLookup &&
		p.Script == ScriptCallable &&
		p.Scope != agent.ToolScopeSelf
}

func (p ToolPolicy) HeldWhenTainted() bool {
	return p.TaintHold != nil || slices.ContainsFunc(p.Egress, agent.EgressClass.Leaves)
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

func (p ToolPolicy) EffectiveEffect() agent.ToolEffect {
	if p.Effect.IsValid() {
		return p.Effect
	}

	switch p.Kind {
	case agent.ToolKindAction:
		return agent.ToolEffectChange
	case agent.ToolKindQuery:
		return agent.ToolEffectLookup
	case agent.ToolKindRuntime:
	}

	return ""
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

func (p ToolPolicy) HasEgress(class agent.EgressClass) bool {
	return slices.Contains(p.Egress, class)
}

func (p *ToolPolicy) MarksFrom(source agent.TaintSource) bool {
	return source == p.Source || slices.Contains(p.Sources, source)
}

func (p ToolPolicy) Conditional() bool {
	return p.Classify != nil || p.Condition != nil || p.PersonalRunsUnasked
}

func (p ToolPolicy) EgressCeiling() agent.AutonomyTier {
	if len(p.Egress) == 0 {
		return agent.TierAutoExecute
	}

	ceiling := agent.TierPropose
	for _, class := range p.Egress {
		if limit := p.ClassCeiling(class); limit.Above(ceiling) {
			ceiling = limit
		}
	}

	return ceiling
}

//nolint:gocritic // ToolPolicy is held and passed by value, as its other methods take it
func (p ToolPolicy) ClassCeiling(class agent.EgressClass) agent.AutonomyTier {
	if class == agent.EgressMoney && p.Bounded() {
		return agent.TierAutoExecute
	}

	return class.Ceiling()
}

//nolint:gocritic // ToolPolicy is held and passed by value, as its other methods take it
func (p ToolPolicy) Bounded() bool {
	return p.Condition != nil && p.Condition.Bounds != nil
}

func (p ToolPolicy) TaintHolds(params ToolExecuteParams, call CallPolicy) bool {
	if call.Egress.Leaves() {
		return true
	}

	return p.TaintHold != nil && (p.TaintHold.Applies == nil || p.TaintHold.Applies(params))
}

type CallPolicy struct {
	Egress  agent.EgressClass
	MaxTier agent.AutonomyTier
}

type TierCondition struct {
	Description string
	Limit       func(context.Context, ToolExecuteParams) agent.AutonomyTier
	Bounds      func(context.Context, ToolExecuteParams) MoneyBound
	Vouches     []agent.TaintSource
}

// MayVouch says whether the condition could vouch for the run's taint at all,
// before its bounds are read: the taint is known and complete, every source in
// it is one the condition vouches for, and every mark names its own record and
// was read by this run rather than across a delegation.
func (c *TierCondition) MayVouch(taint *agent.RunTaint) bool {
	if c == nil || len(c.Vouches) == 0 || taint == nil || !taint.Tainted() || taint.Overflowed {
		return false
	}
	for _, source := range taint.Sources() {
		if !source.Rereadable() || !slices.Contains(c.Vouches, source) {
			return false
		}
	}

	return !slices.ContainsFunc(taint.Marks, func(mark agent.TaintMark) bool {
		return mark.Delegated || mark.Ref == nil
	})
}

// Vouched says the bound re-read, from the organization's own records, each
// record it counts from, and that each is one the run itself read: what the
// call does is taken from those records, never from the text around them.
func (c *TierCondition) Vouched(taint *agent.RunTaint, bound *MoneyBound) bool {
	if !c.MayVouch(taint) || bound == nil || !bound.Within || len(bound.Reread) == 0 {
		return false
	}

	for _, reread := range bound.Reread {
		if !slices.ContainsFunc(taint.Marks, func(mark agent.TaintMark) bool {
			return reread.Source == mark.Source && reread.Ref.EntityType == mark.Ref.EntityType &&
				reread.Ref.ID == mark.Ref.ID
		}) {
			return false
		}
	}

	return true
}

type MoneyBound struct {
	Within      bool
	Held        string
	AmountMinor int64
	Currency    string
	Reread      []agent.SourcedRef
}

func WithinMoneyBound(amountMinor int64, currency string) MoneyBound {
	return MoneyBound{Within: true, AmountMinor: amountMinor, Currency: money.ISOCode(currency)}
}

// Rereading says the bound read this record again from the organization's own
// records and took what it counts from it rather than from the call.
func (b MoneyBound) Rereading(source agent.TaintSource, ref agent.RecordRef) MoneyBound {
	b.Reread = append(slices.Clip(b.Reread), agent.SourcedRef{Source: source, Ref: ref})

	return b
}

func OutsideMoneyBound(held string) MoneyBound {
	return MoneyBound{Held: held}
}

type TaintHold struct {
	Description string
	Applies     func(ToolExecuteParams) bool
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

func IsSelfScoped(tool any) bool {
	policy, ok := PolicyOf(tool)

	return ok && policy.Scope == agent.ToolScopeSelf
}
