package commodityservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*commodity.Commodity]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*commodity.Commodity]().
			WithModelName("Commodity").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithUniqueField(
				"name",
				"name",
				"Commodity with this name already exists in your organization",
				func(c *commodity.Commodity) any { return c.Name },
			).
			WithOptionalCustomReferenceCheck(
				"hazardousMaterialId",
				"Hazardous material does not exist in your organization",
				func(c *commodity.Commodity) pulid.ID { return c.HazardousMaterialID },
				createHazardousMaterialCheck(p.DB),
			).
			Build(),
	}
}

func createHazardousMaterialCheck(
	db *postgres.Connection,
) validationframework.CustomReferenceCheckFunc {
	return func(ctx context.Context, orgID, buID pulid.ID, refID pulid.ID) (bool, error) {
		if refID.IsNil() {
			return true, nil
		}

		exists, err := db.DB().NewSelect().
			TableExpr("hazardous_materials").
			ColumnExpr("1").
			Where("id = ?", refID).
			Where("organization_id = ?", orgID).
			Where("business_unit_id = ?", buID).
			Exists(ctx)
		if err != nil {
			return false, err
		}
		return exists, nil
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
