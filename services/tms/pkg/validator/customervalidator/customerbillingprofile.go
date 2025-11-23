package customervalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/validator/framework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type CustomerBillingProfileValidatorParams struct {
	fx.In

	DB                          *postgres.Connection
	AccountingControlRepository repositories.AccountingControlRepository
	ValidationEngineFactory     framework.ValidationEngineFactory
}

type CustomerBillingProfileValidator struct {
	engine                framework.ValidationEngineFactory
	accountingControlRepo repositories.AccountingControlRepository
	getDB                 func(context.Context) (*bun.DB, error)
}

func NewCustomerBillingProfileValidator(
	p CustomerBillingProfileValidatorParams,
) *CustomerBillingProfileValidator {
	return &CustomerBillingProfileValidator{
		engine:                p.ValidationEngineFactory,
		accountingControlRepo: p.AccountingControlRepository,
		getDB:                 p.DB.DB,
	}
}

func (v *CustomerBillingProfileValidator) Validate(
	ctx context.Context,
	entity *customer.CustomerBillingProfile,
	orgID pulid.ID,
	me *errortypes.MultiError,
) {
	engine := v.engine.CreateEngine().
		ForField("billingProfile").
		WithParent(me)

	engine.AddRule(
		framework.NewConcreteRule("accounting_control_overrides_validation").
			WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
				v.validateAccountingControlOverrides(ctx, entity, orgID, multiErr)
				return nil
			}),
	)

	engine.ValidateInto(ctx, me)
}

func (v *CustomerBillingProfileValidator) validateAccountingControlOverrides(
	ctx context.Context,
	entity *customer.CustomerBillingProfile,
	orgID pulid.ID,
	me *errortypes.MultiError,
) {
	ac, err := v.accountingControlRepo.GetByOrgID(ctx, orgID)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	if entity.RevenueAccountID.IsNotNil() &&
		pulid.Equals(
			pulid.ConvertFromPtr(entity.RevenueAccountID),
			pulid.ConvertFromPtr(ac.DefaultRevenueAccountID),
		) {
		me.Add(
			"revenueAccountId",
			errortypes.ErrInvalid,
			"Revenue account cannot be the same as the default revenue account",
		)
	}

	if entity.ARAccountID.IsNotNil() &&
		pulid.Equals(
			pulid.ConvertFromPtr(entity.ARAccountID),
			pulid.ConvertFromPtr(ac.DefaultARAccountID),
		) {
		me.Add(
			"arAccountId",
			errortypes.ErrInvalid,
			"AR account cannot be the same as the default AR account",
		)
	}
}
