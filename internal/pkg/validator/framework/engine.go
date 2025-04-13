package framework

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/errors"
)

// ValidationRule represents a single validation rule that can be applied
type ValidationRule interface {
	// Stage returns the validation stage for this rule
	Stage() ValidationStage
	// Priority returns the validation priority for this rule
	Priority() ValidationPriority
	// Validate performs the validation and adds errors to the provided MultiError
	Validate(ctx context.Context, multiErr *errors.MultiError) error
}

// ValidationRuleFunc is a function that implements ValidationRule
type ValidationRuleFunc struct {
	StageFunc    func() ValidationStage
	PriorityFunc func() ValidationPriority
	ValidateFunc func(ctx context.Context, multiErr *errors.MultiError) error
}

// Stage returns the validation stage for this rule
func (v ValidationRuleFunc) Stage() ValidationStage {
	return v.StageFunc()
}

// Priority returns the validation priority for this rule
func (v ValidationRuleFunc) Priority() ValidationPriority {
	return v.PriorityFunc()
}

// Validate performs the validation and adds errors to the provided MultiError
func (v ValidationRuleFunc) Validate(ctx context.Context, multiErr *errors.MultiError) error {
	return v.ValidateFunc(ctx, multiErr)
}

// NewValidationRule creates a new ValidationRule with the provided stage, priority, and validate function
func NewValidationRule(stage ValidationStage, priority ValidationPriority, validateFunc func(ctx context.Context, multiErr *errors.MultiError) error) ValidationRule {
	return ValidationRuleFunc{
		StageFunc: func() ValidationStage {
			return stage
		},
		PriorityFunc: func() ValidationPriority {
			return priority
		},
		ValidateFunc: validateFunc,
	}
}

// ValidationEngine represents a validation engine that can run validation rules
type ValidationEngine struct {
	rules []ValidationRule
}

// NewValidationEngine creates a new ValidationEngine
func NewValidationEngine() *ValidationEngine {
	return &ValidationEngine{
		rules: make([]ValidationRule, 0),
	}
}

// AddRule adds a validation rule to the engine
func (v *ValidationEngine) AddRule(rule ValidationRule) *ValidationEngine {
	v.rules = append(v.rules, rule)
	return v
}

// Validate runs all validation rules and returns any errors
func (v *ValidationEngine) Validate(ctx context.Context) *errors.MultiError {
	multiErr := errors.NewMultiError()

	// Execute validation rules by stage and priority
	for stage := ValidationStageBasic; stage <= ValidationStageCompliance; stage++ {
		for priority := ValidationPriorityHigh; priority <= ValidationPriorityLow; priority++ {
			for _, rule := range v.rules {
				if rule.Stage() == stage && rule.Priority() == priority {
					if err := rule.Validate(ctx, multiErr); err != nil {
						// System errors are added to the multi error and we continue
						multiErr.Add("system", errors.ErrSystemError, err.Error())
					}
				}
			}
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
