package commodityservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator validationframework.TenantedEntityValidator[*commodity.Commodity]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := validationframework.NewKeelChecker(p.DB, commoditydb.CommoditiesSchema())

	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*commodity.Commodity]().
			WithModelName("Commodity").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
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
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *commodity.Commodity,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
