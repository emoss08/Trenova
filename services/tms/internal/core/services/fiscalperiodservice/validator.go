package fiscalperiodservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
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
	db             *postgres.Connection
	invoiceRepo    repositories.InvoiceRepository
	accountingRepo repositories.AccountingControlRepository
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*fiscalperiod.FiscalPeriod]().
			WithModelName("FiscalPeriod").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerScoped(p.DB)).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerScoped(p.DB)).
			WithReferenceCheck(
				"fiscalYearId",
				"fiscal_years",
				"Fiscal year does not exist or belongs to a different organization",
				func(fp *fiscalperiod.FiscalPeriod) pulid.ID { return fp.FiscalYearID },
			).
			WithCustomRule(createDateValidationRule(p.DB)).
			WithCustomRule(createStatusConsistencyRule()).
			Build(),
		db:             p.DB,
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

//nolint:govet // existing scoped variable reuse is local and behavior-preserving
func (v *Validator) ValidateClose(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	v.validateAccountingCloseBlockers(ctx, entity, multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	if v.accountingRepo == nil || v.invoiceRepo == nil {
		return nil
	}

	control, err := v.accountingRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}

		multiErr := errortypes.NewMultiError()
		multiErr.Add(
			"reconciliation",
			errortypes.ErrSystemError,
			"Failed to load accounting control",
		)
		return multiErr
	}

	if !control.RequireReconciliationToClose ||
		control.ReconciliationMode == tenant.ReconciliationModeDisabled {
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
		multiErr.Add(
			"reconciliation",
			errortypes.ErrSystemError,
			"Failed to validate reconciliation discrepancies",
		)
		return multiErr
	}

	if count == 0 {
		return nil
	}

	multiErr = errortypes.NewMultiError()
	multiErr.Add(
		"reconciliation",
		errortypes.ErrInvalidOperation,
		"Cannot close fiscal period while {0} posted invoice reconciliation discrepancies remain unresolved",
		count,
	)
	return multiErr
}

var (
	errManualJournalBlockers = errors.New("read manual journal close blockers")
	errJournalSourceBlockers = errors.New("read journal source close blockers")
)

func (v *Validator) validateAccountingCloseBlockers(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
	multiErr *errortypes.MultiError,
) {
	if v.db == nil {
		return
	}

	var pendingManualCount, pendingSourceCount int
	err := v.db.RunScoped(ctx, true, func(ctx context.Context) error {
		if err := v.db.DBForContext(ctx).NewRaw(`
		SELECT COUNT(*)
		FROM manual_journal_requests
		WHERE organization_id = ?
		  AND business_unit_id = ?
		  AND requested_fiscal_period_id = ?
		  AND status IN ('PendingApproval', 'Approved')
	`, entity.OrganizationID, entity.BusinessUnitID, entity.ID).Scan(ctx, &pendingManualCount); err != nil {
			return errManualJournalBlockers
		}

		if err := v.db.DBForContext(ctx).NewRaw(`
		SELECT COUNT(*)
		FROM journal_sources js
		JOIN journal_batches jb
		  ON jb.id = js.journal_batch_id
		 AND jb.organization_id = js.organization_id
		 AND jb.business_unit_id = js.business_unit_id
		WHERE js.organization_id = ?
		  AND js.business_unit_id = ?
		  AND jb.fiscal_period_id = ?
		  AND js.status <> 'Posted'
	`, entity.OrganizationID, entity.BusinessUnitID, entity.ID).Scan(ctx, &pendingSourceCount); err != nil {
			return errJournalSourceBlockers
		}

		return nil
	})
	switch {
	case errors.Is(err, errJournalSourceBlockers):
		multiErr.Add(
			"accounting",
			errortypes.ErrSystemError,
			"Failed to validate journal source close blockers",
		)
		return
	case err != nil:
		multiErr.Add(
			"accounting",
			errortypes.ErrSystemError,
			"Failed to validate manual journal close blockers",
		)
		return
	}

	if pendingManualCount > 0 {
		multiErr.Add(
			"accounting",
			errortypes.ErrInvalidOperation,
			"Cannot close fiscal period while {0} manual journal requests are pending posting or approval",
			pendingManualCount,
		)
	}
	if pendingSourceCount > 0 {
		multiErr.Add(
			"accounting",
			errortypes.ErrInvalidOperation,
			"Cannot close fiscal period while {0} accounting sources remain unposted",
			pendingSourceCount,
		)
	}
}
