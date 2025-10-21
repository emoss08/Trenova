package hazardousmaterialvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
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
	factory *framework.TenantedValidatorFactory[*hazardousmaterial.HazardousMaterial]
}

func NewValidator(p ValidatorParams) *Validator {
	factory := framework.NewTenantedValidatorFactory[*hazardousmaterial.HazardousMaterial](
		func(ctx context.Context) (*bun.DB, error) {
			return p.DB.DB(ctx)
		},
	).
		WithModelName("HazardousMaterial").
		WithUniqueFields(func(hm *hazardousmaterial.HazardousMaterial) []framework.UniqueField {
			return []framework.UniqueField{
				{
					Name:     "code",
					GetValue: func() string { return hm.Code },
					Message:  "Hazardous material with code ':value' already exists in the organization.",
				},
			}
		}).WithCustomRules(func(hm *hazardousmaterial.HazardousMaterial, vc *validator.ValidationContext) []framework.ValidationRule {
		var rules []framework.ValidationRule

		if vc.IsCreate {
			rules = append(
				rules,
				framework.NewBusinessRule("id_validation").
					WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
						if hm.ID.IsNotNil() {
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
	hm *hazardousmaterial.HazardousMaterial,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, hm, valCtx)
}
