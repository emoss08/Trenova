package framework

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
)

type Validator[T any] interface {
	Validate(ctx context.Context, entity T) *errortypes.MultiError
}

type ValidatorFunc[T any] func(ctx context.Context, entity T) *errortypes.MultiError

func (f ValidatorFunc[T]) Validate(ctx context.Context, entity T) *errortypes.MultiError {
	return f(ctx, entity)
}

type FieldValidator[T any, F any] struct {
	fieldName  string
	getField   func(T) F
	validators []func(F) *errortypes.Error
}

func NewFieldValidator[T any, F any](fieldName string, getField func(T) F) *FieldValidator[T, F] {
	return &FieldValidator[T, F]{
		fieldName:  fieldName,
		getField:   getField,
		validators: make([]func(F) *errortypes.Error, 0),
	}
}

func (fv *FieldValidator[T, F]) Required(message string) *FieldValidator[T, F] {
	fv.validators = append(fv.validators, func(value F) *errortypes.Error {
		var zero F
		if fmt.Sprintf("%v", value) == fmt.Sprintf("%v", zero) {
			return errortypes.NewValidationError(fv.fieldName, errortypes.ErrRequired, message)
		}
		return nil
	})
	return fv
}

func (fv *FieldValidator[T, F]) Custom(validator func(F) *errortypes.Error) *FieldValidator[T, F] {
	fv.validators = append(fv.validators, validator)
	return fv
}

func (fv *FieldValidator[T, F]) Validate(entity T) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	value := fv.getField(entity)

	for _, validator := range fv.validators {
		if err := validator(value); err != nil {
			multiErr.AddError(err)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type ConcreteRule struct {
	name        string
	stage       ValidationStage
	priority    ValidationPriority
	validateFn  func(context.Context, *errortypes.MultiError) error
	condition   func() bool
	description string
}

func NewConcreteRule(name string) *ConcreteRule {
	return &ConcreteRule{
		name:      name,
		stage:     ValidationStageBasic,
		priority:  ValidationPriorityMedium,
		condition: func() bool { return true }, // Default: always execute
	}
}

func (r *ConcreteRule) WithStage(stage ValidationStage) *ConcreteRule {
	r.stage = stage
	return r
}

func (r *ConcreteRule) WithPriority(priority ValidationPriority) *ConcreteRule {
	r.priority = priority
	return r
}

func (r *ConcreteRule) WithValidation(
	fn func(context.Context, *errortypes.MultiError) error,
) *ConcreteRule {
	r.validateFn = fn
	return r
}

func (r *ConcreteRule) WithCondition(condition func() bool) *ConcreteRule {
	r.condition = condition
	return r
}

func (r *ConcreteRule) WithDescription(desc string) *ConcreteRule {
	r.description = desc
	return r
}

func (r *ConcreteRule) Stage() ValidationStage {
	return r.stage
}

func (r *ConcreteRule) Priority() ValidationPriority {
	return r.priority
}

func (r *ConcreteRule) Validate(ctx context.Context, multiErr *errortypes.MultiError) error {
	// Check condition first
	if !r.condition() {
		return nil
	}

	if r.validateFn == nil {
		return fmt.Errorf("validation function not set for rule: %s", r.name)
	}

	return r.validateFn(ctx, multiErr)
}

func (r *ConcreteRule) Name() string {
	return r.name
}

func (r *ConcreteRule) Description() string {
	return r.description
}

type FieldRule struct {
	ConcreteRule
	fieldName string
}

func NewFieldRule(fieldName string) *FieldRule {
	return &FieldRule{
		ConcreteRule: *NewConcreteRule(fmt.Sprintf("field_%s", fieldName)),
		fieldName:    fieldName,
	}
}

type BusinessRule struct {
	*ConcreteRule
	dependencies []string
}

func NewBusinessRule(name string) *BusinessRule {
	rule := &BusinessRule{
		ConcreteRule: NewConcreteRule(name),
		dependencies: make([]string, 0),
	}
	rule.stage = ValidationStageBusinessRules
	return rule
}

func (r *BusinessRule) WithValidation(
	fn func(context.Context, *errortypes.MultiError) error,
) *BusinessRule {
	r.ConcreteRule.WithValidation(fn)
	return r
}

func (r *BusinessRule) WithDependencies(deps ...string) *BusinessRule {
	r.dependencies = deps
	return r
}

func (r *BusinessRule) GetDependencies() []string {
	return r.dependencies
}

type ComplianceRule struct {
	*ConcreteRule
	regulation string
	section    string
}

func NewComplianceRule(regulation, section string) *ComplianceRule {
	rule := &ComplianceRule{
		ConcreteRule: NewConcreteRule(fmt.Sprintf("compliance_%s_%s", regulation, section)),
		regulation:   regulation,
		section:      section,
	}
	rule.stage = ValidationStageCompliance
	rule.priority = ValidationPriorityHigh
	return rule
}

func (r *ComplianceRule) WithValidation(
	fn func(context.Context, *errortypes.MultiError) error,
) *ComplianceRule {
	r.ConcreteRule.WithValidation(fn)
	return r
}

func (r *ComplianceRule) GetRegulation() string {
	return r.regulation
}

func (r *ComplianceRule) GetSection() string {
	return r.section
}
