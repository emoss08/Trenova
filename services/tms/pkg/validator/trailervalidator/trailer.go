package trailervalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/trailer"
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
	factory *framework.TenantedValidatorFactory[*trailer.Trailer]
}

func NewValidator(p ValidatorParams) *Validator {
	factory := framework.NewTenantedValidatorFactory[*trailer.Trailer](
		func(ctx context.Context) (*bun.DB, error) {
			return p.DB.DB(ctx)
		},
	).
		WithModelName("Trailer").
		WithUniqueFields(func(entity *trailer.Trailer) []framework.UniqueField {
			return []framework.UniqueField{
				{
					Name:     "code",
					GetValue: func() string { return entity.Code },
					Message:  "Trailer with code ':value' already exists in the organization.",
				},
			}
		}).
		WithCustomRules(func(entity *trailer.Trailer, vc *validator.ValidationContext) []framework.ValidationRule {
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

	return &Validator{
		factory: factory,
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	entity *trailer.Trailer,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, entity, valCtx)
}
