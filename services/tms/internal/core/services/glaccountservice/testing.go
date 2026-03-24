package glaccountservice

import (
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*glaccount.GLAccount]().
			WithModelName("GLAccount").
			Build(),
	}
}

func NewTestValidatorWithDB(conn *postgres.Connection) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*glaccount.GLAccount]().
			WithModelName("GLAccount").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return conn.DB() })).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return conn.DB() })).
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
			WithCustomRule(createParentAccountActiveRule(conn)).
			WithCustomRule(createCircularReferenceRule(conn)).
			WithCustomRule(createBalanceConsistencyRule()).
			WithCustomRule(createDeactivationProtectionRule(conn)).
			Build(),
	}
}
