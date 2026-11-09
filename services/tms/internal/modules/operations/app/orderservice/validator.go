package orderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/fieldpath"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const orderNumberField = "orderNumber"

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityReporter[*order.Order]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*order.Order]().
			WithModelName("Order").
			WithIntegrityChecker(p.Integrity).
			WithUniqueField(
				orderNumberField,
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
) error {
	return v.validator.ReportCreate(ctx, entity).Err()
}

func (v *Validator) ValidateCreateUnnumbered(
	ctx context.Context,
	entity *order.Order,
) error {
	probe := *entity
	probe.OrderNumber = pulid.MustNew("ord_").String()
	report := v.validator.ReportCreate(ctx, &probe)
	if report.Internal() != nil {
		return report.Err()
	}

	numberPath := fieldpath.Parse(orderNumberField)
	kept := errortypes.NewMultiError()
	for _, entry := range report.Errors {
		if !entry.Path.HasPrefix(numberPath) {
			kept.AddError(entry)
		}
	}

	return kept.Err()
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *order.Order,
) error {
	return v.validator.ReportUpdate(ctx, entity).Err()
}
