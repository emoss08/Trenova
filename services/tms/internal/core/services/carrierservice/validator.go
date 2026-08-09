package carrierservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
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
	validator *validationframework.TenantedValidator[*carrier.Carrier]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*carrier.Carrier]().
			WithModelName("Carrier").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithUniqueField(
				"code",
				"code",
				"Carrier with this code already exists in your organization",
				func(c *carrier.Carrier) any { return c.Code },
			).
			WithUniqueField(
				"dot_number",
				"dotNumber",
				"Carrier with this DOT number already exists in your organization",
				func(c *carrier.Carrier) any { return c.DOTNumber },
			).
			WithUniqueField(
				"scac",
				"scac",
				"Carrier with this SCAC already exists in your organization",
				func(c *carrier.Carrier) any { return c.SCAC },
			).
			WithCustomReferenceCheck(
				"stateId",
				"State does not exist",
				func(c *carrier.Carrier) pulid.ID {
					if c.StateID == nil {
						return ""
					}
					return *c.StateID
				},
				createStateCheck(p.DB),
			).
			WithCustomReferenceCheck(
				"remitStateId",
				"Remit state does not exist",
				func(c *carrier.Carrier) pulid.ID {
					if c.RemitStateID == nil {
						return ""
					}
					return *c.RemitStateID
				},
				createStateCheck(p.DB),
			).
			Build(),
	}
}

func createStateCheck(
	db *postgres.Connection,
) validationframework.CustomReferenceCheckFunc {
	return func(ctx context.Context, _, _ pulid.ID, refID pulid.ID) (bool, error) {
		if refID.IsNil() {
			return true, nil
		}

		exists, err := db.DB().NewSelect().
			TableExpr("us_states").
			ColumnExpr("1").
			Where("id = ?", refID).
			Exists(ctx)
		if err != nil {
			return false, err
		}
		return exists, nil
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *carrier.Carrier,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *carrier.Carrier,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
