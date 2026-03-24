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

func (c *TenantedValidationContext) IsCreate() bool {
	return c.Mode == ModeCreate
}

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

func NewTenantedRule[T TenantedEntity](name string) *BaseTenantedRule[T] {
	return &BaseTenantedRule[T]{
		name:     name,
		stage:    ValidationStageBusinessRules,
		priority: ValidationPriorityMedium,
		onCreate: true,
		onUpdate: true,
	}
}

func (r *BaseTenantedRule[T]) Name() string {
	return r.name
}

func (r *BaseTenantedRule[T]) Stage() ValidationStage {
	return r.stage
}

func (r *BaseTenantedRule[T]) Priority() ValidationPriority {
	return r.priority
}

func (r *BaseTenantedRule[T]) OnCreate() *BaseTenantedRule[T] {
	r.onCreate = true
	r.onUpdate = false
	return r
}

func (r *BaseTenantedRule[T]) OnUpdate() *BaseTenantedRule[T] {
	r.onCreate = false
	r.onUpdate = true
	return r
}

func (r *BaseTenantedRule[T]) OnBoth() *BaseTenantedRule[T] {
	r.onCreate = true
	r.onUpdate = true
	return r
}

func (r *BaseTenantedRule[T]) WithStage(stage ValidationStage) *BaseTenantedRule[T] {
	r.stage = stage
	return r
}

func (r *BaseTenantedRule[T]) WithPriority(priority ValidationPriority) *BaseTenantedRule[T] {
	r.priority = priority
	return r
}

func (r *BaseTenantedRule[T]) WithValidation(
	fn TenantedValidateFn[T],
) *BaseTenantedRule[T] {
	r.validateFn = fn
	return r
}

func (r *BaseTenantedRule[T]) ShouldRun(valCtx *TenantedValidationContext) bool {
	if valCtx.IsCreate() && r.onCreate {
		return true
	}
	if valCtx.IsUpdate() && r.onUpdate {
		return true
	}
	return false
}

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
