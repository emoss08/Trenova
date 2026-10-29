package fiscalperiodservice

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/shared/pulid"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*fiscalperiod.FiscalPeriod]().
			WithModelName("FiscalPeriod").
			Build(),
	}
}

func NewTestValidatorWithDB(conn *postgres.Connection) *Validator {
	checker := keelcheck.NewChecker(conn, fiscalperioddb.FiscalPeriodsSchema())
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*fiscalperiod.FiscalPeriod]().
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
