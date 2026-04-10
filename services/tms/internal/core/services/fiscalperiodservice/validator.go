package fiscalperiodservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB             *postgres.Connection
	InvoiceRepo    repositories.InvoiceRepository
	AccountingRepo repositories.AccountingControlRepository
}

type Validator struct {
	validator      *validationframework.TenantedValidator[*fiscalperiod.FiscalPeriod]
	invoiceRepo    repositories.InvoiceRepository
	accountingRepo repositories.AccountingControlRepository
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*fiscalperiod.FiscalPeriod]().
			WithModelName("FiscalPeriod").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithReferenceCheck(
				"fiscalYearId",
				"fiscal_years",
				"Fiscal year does not exist or belongs to a different organization",
				func(fp *fiscalperiod.FiscalPeriod) pulid.ID { return fp.FiscalYearID },
			).
			WithCustomRule(createDateValidationRule(p.DB)).
			WithCustomRule(createStatusConsistencyRule()).
			Build(),
		invoiceRepo:    p.InvoiceRepo,
		accountingRepo: p.AccountingRepo,
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}

func (v *Validator) ValidateClose(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) *errortypes.MultiError {
	if v.accountingRepo == nil || v.invoiceRepo == nil {
		return nil
	}

	control, err := v.accountingRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}

		multiErr := errortypes.NewMultiError()
		multiErr.Add("reconciliation", errortypes.ErrSystemError, "Failed to load accounting control")
		return multiErr
	}

	if !control.RequireReconciliationToClose || control.ReconciliationMode == tenant.ReconciliationModeDisabled {
		return nil
	}

	count, err := v.invoiceRepo.CountPostedReconciliationDiscrepancies(
		ctx,
		repositories.CountPostedInvoiceReconciliationDiscrepanciesRequest{
			OrgID:           entity.OrganizationID,
			BuID:            entity.BusinessUnitID,
			PeriodStartDate: entity.StartDate,
			PeriodEndDate:   entity.EndDate,
			ToleranceAmount: control.ReconciliationToleranceAmount,
		},
	)
	if err != nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("reconciliation", errortypes.ErrSystemError, "Failed to validate reconciliation discrepancies")
		return multiErr
	}

	if count == 0 {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"reconciliation",
		errortypes.ErrInvalidOperation,
		fmt.Sprintf("Cannot close fiscal period while %d posted invoice reconciliation discrepancies remain unresolved", count),
	)
	return multiErr
}
