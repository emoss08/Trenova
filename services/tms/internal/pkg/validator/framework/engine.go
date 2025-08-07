/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package framework

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/rs/zerolog/log"
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
func NewValidationRule(
	stage ValidationStage,
	priority ValidationPriority,
	validateFunc func(ctx context.Context, multiErr *errors.MultiError) error,
) ValidationRule {
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

// ValidationContext contains contextual information for validation
type ValidationContext struct {
	// Field is the field name for indexed errors
	Field string
	// Index is the index for indexed errors
	Index int
	// Parent is the parent MultiError for indexed errors
	Parent *errors.MultiError
	// IsIndexed indicates if this is an indexed validation
	IsIndexed bool
	// IndexedErr is the indexed error (automatically created when IsIndexed is true)
	IndexedErr *errors.MultiError
}

// ValidationEngine represents a validation engine that can run validation rules
type ValidationEngine struct {
	rules []ValidationRule
	ctx   *ValidationContext
}

// NewValidationEngine creates a new ValidationEngine
func NewValidationEngine() *ValidationEngine {
	return &ValidationEngine{
		rules: make([]ValidationRule, 0),
		ctx:   &ValidationContext{},
	}
}

// ForField specifies the field name for indexed errors
func (v *ValidationEngine) ForField(field string) *ValidationEngine {
	v.ctx.Field = field
	return v
}

// AtIndex specifies the index for indexed errors
func (v *ValidationEngine) AtIndex(idx int) *ValidationEngine {
	v.ctx.Index = idx
	v.ctx.IsIndexed = true
	return v
}

// WithParent specifies the parent MultiError for indexed errors
func (v *ValidationEngine) WithParent(parent *errors.MultiError) *ValidationEngine {
	v.ctx.Parent = parent
	return v
}

// AddRule adds a validation rule to the engine
func (v *ValidationEngine) AddRule(rule ValidationRule) *ValidationEngine {
	v.rules = append(v.rules, rule)
	return v
}

// Validate runs all validation rules and returns any errors as a new MultiError.
// Note: When WithParent is used, this method always returns nil because errors are added directly
// to the parent MultiError. In this case, the return value can be safely discarded with:
//
//	_ = engine.Validate(ctx)
//
// When WithParent is not used, this method returns a MultiError if validation errors occur
// or nil if no errors occur.
func (v *ValidationEngine) Validate(ctx context.Context) *errors.MultiError {
	var multiErr *errors.MultiError

	// *If this is an indexed validation with a parent, use the parent and create indexed error
	if v.ctx.IsIndexed && v.ctx.Parent != nil {
		v.ctx.IndexedErr = v.ctx.Parent.WithIndex(v.ctx.Field, v.ctx.Index)

		// * Execute rules on the indexed error
		v.executeRules(ctx, v.ctx.IndexedErr)

		// * Log trace information about the nil return value when debug is enabled
		log.Trace().
			Str("field", v.ctx.Field).
			Int("index", v.ctx.Index).
			Msg("ValidationEngine.Validate returning nil - errors added to parent MultiError")

		// ! When using WithParent, we intentionally return nil since errors are added to
		// ! the parent MultiError. This is by design and not an error condition.
		// ! executeRules() does not return an error, it adds errors directly to the multiErr.
		//nolint:nilerr // This is by design and not an error condition.
		return nil
	}

	// Otherwise, create a new MultiError
	multiErr = errors.NewMultiError()

	// If this is an indexed validation without a parent, create indexed error
	if v.ctx.IsIndexed {
		indexedErr := multiErr.WithIndex(v.ctx.Field, v.ctx.Index)
		v.executeRules(ctx, indexedErr)
	} else {
		v.executeRules(ctx, multiErr)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// ValidateInto runs all validation rules and adds errors to the provided MultiError.
// This is useful for validators that need to add errors to an existing MultiError.
func (v *ValidationEngine) ValidateInto(ctx context.Context, multiErr *errors.MultiError) {
	// If this is an indexed validation, create indexed error
	if v.ctx.IsIndexed {
		indexedErr := multiErr.WithIndex(v.ctx.Field, v.ctx.Index)
		v.executeRules(ctx, indexedErr)
	} else {
		v.executeRules(ctx, multiErr)
	}
}

// executeRules executes all validation rules on the provided MultiError
func (v *ValidationEngine) executeRules(ctx context.Context, multiErr *errors.MultiError) {
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
}
