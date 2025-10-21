package commodityvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/framework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	factory *framework.TenantedValidatorFactory[*commodity.Commodity]
}

func NewValidator(p ValidatorParams) *Validator {
	factory := framework.NewTenantedValidatorFactory[*commodity.Commodity](
		func(ctx context.Context) (*bun.DB, error) {
			return p.DB.DB(ctx)
		},
	).
		WithModelName("Commodity").
		WithUniqueFields(func(entity *commodity.Commodity) []framework.UniqueField {
			return []framework.UniqueField{
				{
					Name:     "name",
					GetValue: func() string { return entity.Name },
					Message:  "Commodity with name ':value' already exists in the organization.",
				},
			}
		}).WithCustomRules(func(entity *commodity.Commodity, vc *validator.ValidationContext) []framework.ValidationRule {
		var rules []framework.ValidationRule

		if vc.IsCreate {
			rules = append(
				rules,
				framework.NewBusinessRule("id_validation").
					WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
						if entity.ID.IsNotNil() {
							me.Add("id", errortypes.ErrInvalid, "ID cannot be set on create")
						}
						return nil
					}),
			)
		}

		return rules
	})

	return &Validator{
		factory: factory,
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	entity *commodity.Commodity,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, entity, valCtx)
}
