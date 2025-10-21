package distanceoverridevalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distanceoverride"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
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
	factory *framework.TenantedValidatorFactory[*distanceoverride.Override]
}

func NewValidator(p ValidatorParams) *Validator {
	factory := framework.NewTenantedValidatorFactory[*distanceoverride.Override](
		func(ctx context.Context) (*bun.DB, error) {
			return p.DB.DB(ctx)
		},
	).
		WithModelName("DistanceOverride").
		WithCustomRules(func(do *distanceoverride.Override, valCtx *validator.ValidationContext) []framework.ValidationRule {
			var rules []framework.ValidationRule

			rules = append(rules,
				// Rule #1: Origin and destination location cannot be the same
				framework.NewBusinessRule("origin_destination_validation").
					WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
						if pulid.Equals(do.OriginLocationID, do.DestinationLocationID) {
							multiErr.Add(
								"originLocationId",
								errortypes.ErrInvalid,
								"Origin location and destination location cannot be the same",
							)
						}
						return nil
					}),
			)

			if valCtx.IsCreate {
				rules = append(rules, framework.NewBusinessRule("id_validation").
					WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
						if do.ID.IsNotNil() {
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
	entity *distanceoverride.Override,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, entity, valCtx)
}
