package commodityservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*commodity.Commodity]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*commodity.Commodity]().
			WithModelName("Commodity").
			WithIntegrityChecker(p.Integrity).
			WithUniqueField(
				"name",
				"name",
				"Commodity with this name already exists in your organization",
				func(c *commodity.Commodity) any { return c.Name },
			).
			WithOptionalReferenceCheck(
				"hazardousMaterialId",
				"hazardous_materials",
				"Hazardous material does not exist in your organization",
				func(c *commodity.Commodity) pulid.ID { return c.HazardousMaterialID },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *commodity.Commodity,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *commodity.Commodity,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
