package fiscalperiodservice

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*fiscalperiod.FiscalPeriod]().
			WithModelName("FiscalPeriod").
			Build(),
	}
}

func NewTestValidatorWithDB(conn *postgres.Connection) *Validator {
	checker := validationframework.NewKeelChecker(conn, fiscalperioddb.FiscalPeriodsSchema())
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*fiscalperiod.FiscalPeriod]().
			WithModelName("FiscalPeriod").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
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
