package glaccountservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*glaccount.GLAccount]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*glaccount.GLAccount]().
			WithModelName("GLAccount").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerScoped(p.DB)).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerScoped(p.DB)).
			WithUniqueField(
				"accountCode",
				"account_code",
				"GL account with this account code already exists in your organization",
				func(g *glaccount.GLAccount) any { return g.AccountCode },
			).
			WithReferenceCheck(
				"accountTypeId",
				"account_types",
				"Account type does not exist or belongs to a different organization",
				func(g *glaccount.GLAccount) pulid.ID { return g.AccountTypeID },
			).
			WithOptionalReferenceCheck(
				"parentId",
				"gl_accounts",
				"Parent GL account does not exist or belongs to a different organization",
				func(g *glaccount.GLAccount) pulid.ID { return g.ParentID },
			).
			WithCustomRule(createSystemAccountProtectionRule()).
			WithCustomRule(createParentAccountActiveRule(p.DB)).
			WithCustomRule(createCircularReferenceRule(p.DB)).
			WithCustomRule(createBalanceConsistencyRule()).
			WithCustomRule(createDeactivationProtectionRule(p.DB)).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *glaccount.GLAccount,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *glaccount.GLAccount,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
