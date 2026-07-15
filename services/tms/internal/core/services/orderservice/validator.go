package orderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*order.Order]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*order.Order]().
			WithModelName("Order").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithUniqueField(
				"orderNumber",
				"order_number",
				"Order with this number already exists in your organization",
				func(o *order.Order) any { return o.OrderNumber },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *order.Order,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *order.Order,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
