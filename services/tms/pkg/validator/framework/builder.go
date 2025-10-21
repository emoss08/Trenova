package framework

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
)

type ValidationBuilder struct {
	engine *ValidationEngine
	rules  []ValidationRule
}

func NewValidationBuilder() *ValidationBuilder {
	return &ValidationBuilder{
		engine: NewValidationEngine(DefaultEngineConfig()),
		rules:  make([]ValidationRule, 0),
	}
}

func (vb *ValidationBuilder) WithConfig(config *EngineConfig) *ValidationBuilder {
	vb.engine = NewValidationEngine(config)
	return vb
}

func (vb *ValidationBuilder) ForField(field string) *ValidationBuilder {
	vb.engine.ForField(field)
	return vb
}

func (vb *ValidationBuilder) AtIndex(idx int) *ValidationBuilder {
	vb.engine.AtIndex(idx)
	return vb
}

func (vb *ValidationBuilder) WithParent(parent *errortypes.MultiError) *ValidationBuilder {
	vb.engine.WithParent(parent)
	return vb
}

func (vb *ValidationBuilder) AddRule(rule ValidationRule) *ValidationBuilder {
	vb.rules = append(vb.rules, rule)
	vb.engine.AddRule(rule)
	return vb
}

func (vb *ValidationBuilder) Basic(
	name string,
	fn func(context.Context, *errortypes.MultiError) error,
) *ValidationBuilder {
	rule := NewConcreteRule(name).
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(fn)
	return vb.AddRule(rule)
}

func (vb *ValidationBuilder) DataIntegrity(
	name string,
	fn func(context.Context, *errortypes.MultiError) error,
) *ValidationBuilder {
	rule := NewConcreteRule(name).
		WithStage(ValidationStageDataIntegrity).
		WithPriority(ValidationPriorityHigh).
		WithValidation(fn)
	return vb.AddRule(rule)
}

func (vb *ValidationBuilder) BusinessRule(
	name string,
	fn func(context.Context, *errortypes.MultiError) error,
) *ValidationBuilder {
	rule := NewBusinessRule(name)
	rule.WithValidation(fn)
	return vb.AddRule(rule)
}

func (vb *ValidationBuilder) Compliance(
	regulation, section string,
	fn func(context.Context, *errortypes.MultiError) error,
) *ValidationBuilder {
	rule := NewComplianceRule(regulation, section)
	rule.WithValidation(fn)
	return vb.AddRule(rule)
}

func (vb *ValidationBuilder) When(
	condition func() bool,
	ruleFn func(*ValidationBuilder),
) *ValidationBuilder {
	if condition() {
		ruleFn(vb)
	}
	return vb
}

func (vb *ValidationBuilder) Build() *ValidationEngine {
	return vb.engine
}

func (vb *ValidationBuilder) Validate(ctx context.Context) *errortypes.MultiError {
	return vb.engine.Validate(ctx)
}

type RuleBuilder struct {
	rule *ConcreteRule
}

func NewRuleBuilder(name string) *RuleBuilder {
	return &RuleBuilder{
		rule: NewConcreteRule(name),
	}
}

func (rb *RuleBuilder) Stage(stage ValidationStage) *RuleBuilder {
	rb.rule.WithStage(stage)
	return rb
}

func (rb *RuleBuilder) Priority(priority ValidationPriority) *RuleBuilder {
	rb.rule.WithPriority(priority)
	return rb
}

func (rb *RuleBuilder) Validate(
	fn func(context.Context, *errortypes.MultiError) error,
) *RuleBuilder {
	rb.rule.WithValidation(fn)
	return rb
}

func (rb *RuleBuilder) When(condition func() bool) *RuleBuilder {
	rb.rule.WithCondition(condition)
	return rb
}

func (rb *RuleBuilder) Description(desc string) *RuleBuilder {
	rb.rule.WithDescription(desc)
	return rb
}

func (rb *RuleBuilder) Build() ValidationRule {
	return rb.rule
}
