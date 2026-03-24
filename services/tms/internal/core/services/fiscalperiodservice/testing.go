package fiscalperiodservice

import (
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*fiscalperiod.FiscalPeriod]().
			WithModelName("FiscalPeriod").
			Build(),
	}
}

func NewTestValidatorWithDB(conn *postgres.Connection) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*fiscalperiod.FiscalPeriod]().
			WithModelName("FiscalPeriod").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return conn.DB() })).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return conn.DB() })).
			WithReferenceCheck(
				"fiscalYearId",
				"fiscal_years",
				"Fiscal year does not exist or belongs to a different organization",
				func(fp *fiscalperiod.FiscalPeriod) pulid.ID { return fp.FiscalYearID },
			).
			WithCustomRule(createDateValidationRule(conn)).
			WithCustomRule(createStatusConsistencyRule()).
			Build(),
	}
}
