package framework

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
)

type CompositeRule struct {
	name        string
	stage       ValidationStage
	priority    ValidationPriority
	rules       []ValidationRule
	operator    LogicalOperator
	stopOnFirst bool // For AND operator, stop on first failure
}

type LogicalOperator int

const (
	AND LogicalOperator = iota
	OR
	XOR
)

func NewCompositeRule(name string, operator LogicalOperator) *CompositeRule {
	return &CompositeRule{
		name:     name,
		stage:    ValidationStageBasic,
		priority: ValidationPriorityMedium,
		rules:    make([]ValidationRule, 0),
		operator: operator,
	}
}

func (cr *CompositeRule) WithStage(stage ValidationStage) *CompositeRule {
	cr.stage = stage
	return cr
}

func (cr *CompositeRule) WithPriority(priority ValidationPriority) *CompositeRule {
	cr.priority = priority
	return cr
}

func (cr *CompositeRule) AddRule(rule ValidationRule) *CompositeRule {
	cr.rules = append(cr.rules, rule)
	return cr
}

func (cr *CompositeRule) AddRules(rules ...ValidationRule) *CompositeRule {
	cr.rules = append(cr.rules, rules...)
	return cr
}

func (cr *CompositeRule) WithStopOnFirst(stop bool) *CompositeRule {
	cr.stopOnFirst = stop
	return cr
}

func (cr *CompositeRule) Stage() ValidationStage {
	return cr.stage
}

func (cr *CompositeRule) Priority() ValidationPriority {
	return cr.priority
}

func (cr *CompositeRule) Validate(ctx context.Context, multiErr *errortypes.MultiError) error {
	switch cr.operator {
	case AND:
		return cr.validateAND(ctx, multiErr)
	case OR:
		return cr.validateOR(ctx, multiErr)
	case XOR:
		return cr.validateXOR(ctx, multiErr)
	default:
		return fmt.Errorf("unknown logical operator: %d", cr.operator)
	}
}

func (cr *CompositeRule) validateAND(ctx context.Context, multiErr *errortypes.MultiError) error {
	for _, rule := range cr.rules {
		tempErr := errortypes.NewMultiError()
		if err := rule.Validate(ctx, tempErr); err != nil {
			return err
		}

		if tempErr.HasErrors() {
			for _, e := range tempErr.Errors {
				multiErr.AddError(e)
			}

			if cr.stopOnFirst {
				break
			}
		}
	}
	return nil
}

func (cr *CompositeRule) validateOR(ctx context.Context, multiErr *errortypes.MultiError) error {
	allErrors := make([]*errortypes.Error, 0)
	passed := false

	for _, rule := range cr.rules {
		tempErr := errortypes.NewMultiError()
		if err := rule.Validate(ctx, tempErr); err != nil {
			return err
		}

		if !tempErr.HasErrors() {
			passed = true
			break
		}

		allErrors = append(allErrors, tempErr.Errors...)
	}

	if !passed {
		for _, e := range allErrors {
			multiErr.AddError(e)
		}
	}

	return nil
}

func (cr *CompositeRule) validateXOR(ctx context.Context, multiErr *errortypes.MultiError) error {
	passCount := 0

	for _, rule := range cr.rules {
		tempErr := errortypes.NewMultiError()
		if err := rule.Validate(ctx, tempErr); err != nil {
			return err
		}

		if !tempErr.HasErrors() {
			passCount++
		}
	}

	if passCount != 1 {
		multiErr.Add(
			"",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"XOR validation failed: expected exactly 1 rule to pass, but %d passed",
				passCount,
			),
		)
	}

	return nil
}

type ConditionalRule struct {
	rule      ValidationRule
	condition func(context.Context) bool
}

func NewConditionalRule(
	rule ValidationRule,
	condition func(context.Context) bool,
) *ConditionalRule {
	return &ConditionalRule{
		rule:      rule,
		condition: condition,
	}
}

func (cr *ConditionalRule) Stage() ValidationStage {
	return cr.rule.Stage()
}

func (cr *ConditionalRule) Priority() ValidationPriority {
	return cr.rule.Priority()
}

func (cr *ConditionalRule) Validate(ctx context.Context, multiErr *errortypes.MultiError) error {
	if cr.condition(ctx) {
		return cr.rule.Validate(ctx, multiErr)
	}
	return nil
}

type RuleSet struct {
	name  string
	rules []ValidationRule
}

func NewRuleSet(name string) *RuleSet {
	return &RuleSet{
		name:  name,
		rules: make([]ValidationRule, 0),
	}
}

func (rs *RuleSet) Add(rule ValidationRule) *RuleSet {
	rs.rules = append(rs.rules, rule)
	return rs
}

func (rs *RuleSet) AddAll(rules ...ValidationRule) *RuleSet {
	rs.rules = append(rs.rules, rules...)
	return rs
}

func (rs *RuleSet) GetRules() []ValidationRule {
	return rs.rules
}

func (rs *RuleSet) ApplyTo(engine *ValidationEngine) {
	for _, rule := range rs.rules {
		engine.AddRule(rule)
	}
}

func (rs *RuleSet) Name() string {
	return rs.name
}

func (rs *RuleSet) Count() int {
	return len(rs.rules)
}
