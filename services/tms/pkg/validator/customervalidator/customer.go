package customervalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/framework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB                              *postgres.Connection
	CustomerBillingProfileValidator *CustomerBillingProfileValidator
}

type Validator struct {
	customerBillingProfileValidator *CustomerBillingProfileValidator
	getDB                           func(context.Context) (*bun.DB, error)
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		customerBillingProfileValidator: p.CustomerBillingProfileValidator,
		getDB:                           p.DB.DB,
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	entity *customer.Customer,
) *errortypes.MultiError {
	engine := framework.NewValidationEngine(framework.DefaultEngineConfig())

	engine.AddRule(
		framework.NewConcreteRule("customer_validation").
			WithStage(framework.ValidationStageBasic).
			WithPriority(framework.ValidationPriorityHigh).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				entity.Validate(me)
				return nil
			}),
	)

	// engine.AddRule(
	// 	framework.NewUniquenessRule("customer_code_uniqueness", v.getDB).
	// 		ForTable("customers").
	// 		ForModel("Customer").
	// 		WithTenant(func() (organizationID pulid.ID, businessUnitID pulid.ID) {
	// 			return entity.GetOrganizationID(), entity.GetBusinessUnitID()
	// 		}).
	// 		ForOperation(valCtx.IsCreate).
	// 		CheckField("code", func() string {
	// 			return entity.Code
	// 		}, "Customer with code ':value' already exists in the organization."),
	// )

	engine.AddRule(
		framework.NewConcreteRule("customer_billing_profile_validation").
			WithStage(framework.ValidationStageBusinessRules).
			WithPriority(framework.ValidationPriorityHigh).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				v.customerBillingProfileValidator.Validate(
					ctx,
					entity.BillingProfile,
					entity.OrganizationID,
					me,
				)
				return nil
			}),
	)

	return engine.Validate(ctx)
}
