package glaccountservice

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*glaccount.GLAccount]().
			WithModelName("GLAccount").
			Build(),
	}
}

func NewTestValidatorWithDB(conn *postgres.Connection) *Validator {
	checker := validationframework.NewKeelChecker(conn, glaccountdb.GLAccountsSchema())
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*glaccount.GLAccount]().
			WithModelName("GLAccount").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
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
