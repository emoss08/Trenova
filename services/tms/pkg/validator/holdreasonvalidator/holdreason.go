package holdreasonvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
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
	factory *framework.TenantedValidatorFactory[*holdreason.HoldReason]
}

func NewValidator(p ValidatorParams) *Validator {
	factory := framework.NewTenantedValidatorFactory[*holdreason.HoldReason](
		func(ctx context.Context) (*bun.DB, error) {
			return p.DB.DB(ctx)
		},
	).
		WithModelName("HoldReason").
		WithUniqueFields(func(hr *holdreason.HoldReason) []framework.UniqueField {
			return []framework.UniqueField{
				{
					Name:     "code",
					GetValue: func() string { return hr.Code },
					Message:  "Hold reason with code ':value' already exists in the organization.",
				},
				{
					Name:     "label",
					GetValue: func() string { return hr.Label },
					Message:  "Hold reason with label ':value' already exists in the organization.",
				},
			}
		}).
		WithCustomRules(func(hr *holdreason.HoldReason, valCtx *validator.ValidationContext) []framework.ValidationRule {
			var rules []framework.ValidationRule

			rules = append(rules, framework.NewBusinessRule("severity_validation").
				WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
					if hr.DefaultSeverity == holdreason.SeverityBlocking {
						if !hr.DefaultBlocksBilling && !hr.DefaultBlocksDelivery &&
							!hr.DefaultBlocksDispatch {
							multiErr.Add(
								"defaultSeverity",
								errortypes.ErrInvalid,
								"At least one block must be true if the severity is Blocking",
							)
						}
					}
					return nil
				}))

			if valCtx.IsCreate {
				rules = append(rules, framework.NewBusinessRule("id_validation").
					WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
						if hr.ID.IsNotNil() {
							multiErr.Add("id", errortypes.ErrInvalid, "ID cannot be set on create")
						}
						return nil
					}))
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
	hr *holdreason.HoldReason,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, hr, valCtx)
}
