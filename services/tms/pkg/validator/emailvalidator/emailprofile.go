package emailvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/framework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type EmailProfileValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type EmailProfileValidator struct {
	factory *framework.TenantedValidatorFactory[*email.EmailProfile]
}

func NewEmailProfileValidator(p EmailProfileValidatorParams) *EmailProfileValidator {
	factory := framework.NewTenantedValidatorFactory[*email.EmailProfile](
		func(ctx context.Context) (*bun.DB, error) {
			return p.DB.DB(ctx)
		},
	).
		WithModelName("EmailProfile").
		WithUniqueFields(func(entity *email.EmailProfile) []framework.UniqueField {
			return []framework.UniqueField{
				{
					Name:     "name",
					GetValue: func() string { return entity.Name },
					Message:  "Location category with name ':value' already exists in the organization.",
				},
			}
		}).
		WithCustomRules(func(entity *email.EmailProfile, vc *validator.ValidationContext) []framework.ValidationRule {
			var rules []framework.ValidationRule

			if vc.IsCreate {
				rules = append(rules, framework.NewBusinessRule("id_validation").
					WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
						if entity.ID.IsNotNil() {
							multiErr.Add("id", errortypes.ErrInvalid, "ID cannot be set on create")
						}
						return nil
					}),
				)
			}

			return rules
		})

	return &EmailProfileValidator{
		factory: factory,
	}
}

func (v *EmailProfileValidator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	entity *email.EmailProfile,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, entity, valCtx)
}
