package validationframework

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type Stage int

const (
	StageBasic Stage = iota
	StageDataIntegrity
	StageBusinessRules
	StageCompliance
)

type Mode int

const (
	ModeCreate Mode = iota
	ModeUpdate
)

type Entity interface {
	GetID() pulid.ID
	GetTableName() string
	Validate(multiErr *errortypes.MultiError)
}

type TenantOwned interface {
	GetOrganizationID() pulid.ID
	GetBusinessUnitID() pulid.ID
}

type ValidateFn[T Entity] func(
	ctx context.Context,
	entity T,
	valCtx *Context,
	multiErr *errortypes.MultiError,
) error

type Rule[T Entity] interface {
	Name() string
	Stage() Stage
	ShouldRun(mode Mode) bool
	Validate(ctx context.Context, entity T, valCtx *Context, multiErr *errortypes.MultiError) error
}

type BaseRule[T Entity] struct {
	name       string
	stage      Stage
	onCreate   bool
	onUpdate   bool
	validateFn ValidateFn[T]
}

func NewRule[T Entity](name string) *BaseRule[T] {
	return &BaseRule[T]{
		name:     name,
		stage:    StageBusinessRules,
		onCreate: true,
		onUpdate: true,
	}
}

func (r *BaseRule[T]) Name() string {
	return r.name
}

func (r *BaseRule[T]) Stage() Stage {
	return r.stage
}

func (r *BaseRule[T]) OnCreate() *BaseRule[T] {
	r.onCreate = true
	r.onUpdate = false
	return r
}

func (r *BaseRule[T]) OnUpdate() *BaseRule[T] {
	r.onCreate = false
	r.onUpdate = true
	return r
}

func (r *BaseRule[T]) OnBoth() *BaseRule[T] {
	r.onCreate = true
	r.onUpdate = true
	return r
}

func (r *BaseRule[T]) WithStage(stage Stage) *BaseRule[T] {
	r.stage = stage
	return r
}

func (r *BaseRule[T]) WithValidation(fn ValidateFn[T]) *BaseRule[T] {
	r.validateFn = fn
	return r
}

func (r *BaseRule[T]) ShouldRun(mode Mode) bool {
	switch mode {
	case ModeCreate:
		return r.onCreate
	case ModeUpdate:
		return r.onUpdate
	}

	return false
}

func (r *BaseRule[T]) Validate(
	ctx context.Context,
	entity T,
	valCtx *Context,
	multiErr *errortypes.MultiError,
) error {
	if r.validateFn == nil {
		return nil
	}

	return r.validateFn(ctx, entity, valCtx, multiErr)
}
