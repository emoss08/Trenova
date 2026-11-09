package tenantedvalidation

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
)

const stageCount = int(validationframework.StageCompliance) + 1

type EntityValidator[T validationframework.Entity] interface {
	ValidateCreate(ctx context.Context, entity T) error
	ValidateUpdate(ctx context.Context, entity T) error
	ValidateUpdateWithOriginal(ctx context.Context, entity, original T) error
}

type EntityReporter[T validationframework.Entity] interface {
	ReportCreate(ctx context.Context, entity T) *errortypes.MultiError
	ReportUpdate(ctx context.Context, entity T) *errortypes.MultiError
	ReportUpdateWithOriginal(ctx context.Context, entity, original T) *errortypes.MultiError
}

type Validator[T validationframework.Entity] struct {
	modelName             string
	checker               validationframework.IntegrityChecker
	integrityScope        func(context.Context) context.Context
	uniqueFields          []UniqueFieldConfig[T]
	compositeUniqueFields []CompositeUniqueFieldConfig[T]
	references            []ReferenceConfig[T]
	referenceLists        []ReferenceListConfig[T]
	immutableFields       []ImmutableFieldConfig[T]
	dateComparisons       []DateComparisonConfig[T]
	numericRanges         []NumericRangeConfig[T]
	rules                 [stageCount][]validationframework.Rule[T]
}

func (v *Validator[T]) ValidateCreate(ctx context.Context, entity T) error {
	return v.ReportCreate(ctx, entity).Err()
}

func (v *Validator[T]) ValidateUpdate(ctx context.Context, entity T) error {
	return v.ReportUpdate(ctx, entity).Err()
}

func (v *Validator[T]) ValidateUpdateWithOriginal(ctx context.Context, entity, original T) error {
	return v.ReportUpdateWithOriginal(ctx, entity, original).Err()
}

func (v *Validator[T]) ReportCreate(ctx context.Context, entity T) *errortypes.MultiError {
	return v.run(
		ctx,
		entity,
		validationframework.NewContext(ctx, entity, validationframework.ModeCreate),
	)
}

func (v *Validator[T]) ReportUpdate(ctx context.Context, entity T) *errortypes.MultiError {
	return v.run(
		ctx,
		entity,
		validationframework.NewContext(ctx, entity, validationframework.ModeUpdate),
	)
}

func (v *Validator[T]) ReportUpdateWithOriginal(
	ctx context.Context,
	entity, original T,
) *errortypes.MultiError {
	valCtx := validationframework.NewContext(ctx, entity, validationframework.ModeUpdate)
	valCtx.Original = original

	return v.run(ctx, entity, valCtx)
}

func (v *Validator[T]) run(
	ctx context.Context,
	entity T,
	valCtx *validationframework.Context,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	v.validateBasics(entity, valCtx, multiErr)
	if multiErr.Internal() != nil {
		return multiErr
	}

	for stage := range stageCount {
		if validationframework.Stage(stage) == validationframework.StageDataIntegrity {
			v.checkIntegrity(ctx, entity, valCtx, multiErr)
			if multiErr.Internal() != nil {
				return multiErr
			}
		}

		for _, rule := range v.rules[stage] {
			if !rule.ShouldRun(valCtx.Mode) {
				continue
			}
			if err := rule.Validate(ctx, entity, valCtx, multiErr); err != nil {
				multiErr.AddInternal(fmt.Errorf("%s rule %s: %w", v.modelName, rule.Name(), err))
				return multiErr
			}
			if multiErr.Internal() != nil {
				return multiErr
			}
		}
	}

	return multiErr
}
