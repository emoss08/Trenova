package validationframework

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type ValidationMode int

const (
	ModeCreate ValidationMode = iota
	ModeUpdate
)

type TenantedValidationContext struct {
	Mode           ValidationMode
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	EntityID       pulid.ID
}

// IsCreate reports whether validation is running for a new entity (create flow).
func (c *TenantedValidationContext) IsCreate() bool {
	return c.Mode == ModeCreate
}

// IsUpdate reports whether validation is running for an existing entity (update flow).
func (c *TenantedValidationContext) IsUpdate() bool {
	return c.Mode == ModeUpdate
}

type TenantedEntity interface {
	GetID() pulid.ID
	GetTableName() string
	GetOrganizationID() pulid.ID
	GetBusinessUnitID() pulid.ID
	Validate(multiErr *errortypes.MultiError)
}

type TenantedRule[T TenantedEntity] interface {
	Name() string
	Stage() ValidationStage
	Priority() ValidationPriority
	ShouldRun(valCtx *TenantedValidationContext) bool
	Validate(
		ctx context.Context,
		entity T,
		valCtx *TenantedValidationContext,
		multiErr *errortypes.MultiError,
	) error
}

type TenantedValidateFn[T TenantedEntity] func(
	ctx context.Context,
	entity T,
	valCtx *TenantedValidationContext,
	multiErr *errortypes.MultiError,
) error

type BaseTenantedRule[T TenantedEntity] struct {
	name       string
	stage      ValidationStage
	priority   ValidationPriority
	onCreate   bool
	onUpdate   bool
	validateFn TenantedValidateFn[T]
}

// NewTenantedRule returns a tenanted rule with default stage (business rules), medium priority,
// and both create and update enabled. Use the fluent setters to narrow when the rule runs and
// to attach validation logic.
func NewTenantedRule[T TenantedEntity](name string) *BaseTenantedRule[T] {
	return &BaseTenantedRule[T]{
		name:     name,
		stage:    ValidationStageBusinessRules,
		priority: ValidationPriorityMedium,
		onCreate: true,
		onUpdate: true,
	}
}

// Name returns the rule identifier used for ordering, logging, and diagnostics.
func (r *BaseTenantedRule[T]) Name() string {
	return r.name
}

// Stage returns which validation pipeline stage this rule belongs to.
func (r *BaseTenantedRule[T]) Stage() ValidationStage {
	return r.stage
}

// Priority returns the relative ordering of this rule within its stage.
func (r *BaseTenantedRule[T]) Priority() ValidationPriority {
	return r.priority
}

// OnCreate configures the rule to run only when valCtx indicates create mode.
func (r *BaseTenantedRule[T]) OnCreate() *BaseTenantedRule[T] {
	r.onCreate = true
	r.onUpdate = false
	return r
}

// OnUpdate configures the rule to run only when valCtx indicates update mode.
func (r *BaseTenantedRule[T]) OnUpdate() *BaseTenantedRule[T] {
	r.onCreate = false
	r.onUpdate = true
	return r
}

// OnBoth configures the rule to run for both create and update flows.
func (r *BaseTenantedRule[T]) OnBoth() *BaseTenantedRule[T] {
	r.onCreate = true
	r.onUpdate = true
	return r
}

// WithStage sets the validation stage for this rule.
func (r *BaseTenantedRule[T]) WithStage(stage ValidationStage) *BaseTenantedRule[T] {
	r.stage = stage
	return r
}

// WithPriority sets the priority used to order this rule relative to others in the same stage.
func (r *BaseTenantedRule[T]) WithPriority(priority ValidationPriority) *BaseTenantedRule[T] {
	r.priority = priority
	return r
}

// WithValidation attaches the function that performs the rule's validation. If unset, Validate is a no-op.
func (r *BaseTenantedRule[T]) WithValidation(
	fn TenantedValidateFn[T],
) *BaseTenantedRule[T] {
	r.validateFn = fn
	return r
}

// ShouldRun reports whether this rule applies to the given validation context (create vs update),
// based on OnCreate, OnUpdate, or OnBoth configuration.
func (r *BaseTenantedRule[T]) ShouldRun(valCtx *TenantedValidationContext) bool {
	if valCtx.IsCreate() && r.onCreate {
		return true
	}
	if valCtx.IsUpdate() && r.onUpdate {
		return true
	}
	return false
}

// Validate runs the configured validation function, passing entity, context, and multiErr for
// field-level errors. Returns nil when no validation function was attached.
func (r *BaseTenantedRule[T]) Validate(
	ctx context.Context,
	entity T,
	valCtx *TenantedValidationContext,
	multiErr *errortypes.MultiError,
) error {
	if r.validateFn == nil {
		return nil
	}
	return r.validateFn(ctx, entity, valCtx, multiErr)
}
