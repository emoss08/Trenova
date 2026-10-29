package fiscalperiodservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource/journalsourcedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal/manualjournaldb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB             *postgres.Connection
	InvoiceRepo    invoiceports.InvoiceRepository
	AccountingRepo tenantports.AccountingControlRepository
}

type Validator struct {
	validator      validationframework.TenantedEntityValidator[*fiscalperiod.FiscalPeriod]
	db             *postgres.Connection
	invoiceRepo    invoiceports.InvoiceRepository
	accountingRepo tenantports.AccountingControlRepository
}

func NewValidator(p ValidatorParams) *Validator {
	checker := validationframework.NewKeelChecker(p.DB, fiscalperioddb.FiscalPeriodsSchema())
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
		control.ReconciliationMode == accountingcontrol.ReconciliationModeDisabled {
		return nil
	}

	count, err := v.invoiceRepo.CountPostedReconciliationDiscrepancies(
		ctx,
		invoiceports.CountPostedInvoiceReconciliationDiscrepanciesRequest{
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
	err := dbtx.ReadErr(ctx, v.db, func(ctx context.Context) error {
		db := v.db.Keel(ctx)
		mjr := manualjournaldb.Requests
		manual, err := mjr.Where(
			mjr.RequestedFiscalPeriodID.Eq(entity.ID),
			mjr.Status.In(manualjournal.StatusPendingApproval, manualjournal.StatusApproved),
		).Count(ctx, db)
		if err != nil {
			return errManualJournalBlockers
		}
		pendingManualCount = int(manual)

		js := journalsourcedb.Sources
		sources, err := js.Where(
			js.JournalBatch().FiscalPeriodID.Eq(entity.ID),
			js.Status.Ne("Posted"),
		).Count(ctx, db)
		if err != nil {
			return errJournalSourceBlockers
		}
		pendingSourceCount = int(sources)

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
