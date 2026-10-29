package orderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*order.Order]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := keelcheck.NewChecker(p.DB, orderdb.OrdersSchema())

	return &Validator{
		validator: tenantedvalidation.NewBuilder[*order.Order]().
			WithModelName("Order").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
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
