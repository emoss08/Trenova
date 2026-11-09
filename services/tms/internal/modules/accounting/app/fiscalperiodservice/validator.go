package fiscalperiodservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal/journalreversaldb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource/journalsourcedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal/manualjournaldb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker

	DB             *postgres.Connection
	InvoiceRepo    invoiceports.InvoiceRepository
	AccountingRepo tenantports.AccountingControlRepository
}

type Validator struct {
	validator      tenantedvalidation.EntityValidator[*fiscalperiod.FiscalPeriod]
	db             *postgres.Connection
	invoiceRepo    invoiceports.InvoiceRepository
	accountingRepo tenantports.AccountingControlRepository
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*fiscalperiod.FiscalPeriod]().
			WithModelName("FiscalPeriod").
			WithIntegrityChecker(p.Integrity).
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
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}

//nolint:govet // existing scoped variable reuse is local and behavior-preserving
func (v *Validator) ValidateClose(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) error {
	multiErr := errortypes.NewMultiError()
	v.validateAccountingCloseBlockers(ctx, entity, multiErr)
	if multiErr.HasErrors() {
		return multiErr.Err()
	}

	control, err := v.accountingRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}

		multiErr := errortypes.NewMultiError()
		multiErr.AddInternal(fmt.Errorf("failed to load accounting control: %w", err))
		return multiErr.Err()
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
		multiErr.AddInternal(fmt.Errorf("failed to validate reconciliation discrepancies: %w", err))
		return multiErr.Err()
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
	return multiErr.Err()
}

var (
	errManualJournalBlockers = errors.New("read manual journal close blockers")
	errReversalBlockers      = errors.New("read journal reversal close blockers")
	errJournalSourceBlockers = errors.New("read journal source close blockers")
)

func (v *Validator) validateAccountingCloseBlockers(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
	multiErr *errortypes.MultiError,
) {
	var pendingManualCount, pendingReversalCount, pendingSourceCount int
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

		jr := journalreversaldb.Reversals
		reversals, err := jr.Where(
			jr.ResolvedFiscalPeriodID.Eq(entity.ID),
			jr.Status.In(
				journalreversal.StatusRequested,
				journalreversal.StatusPendingApproval,
				journalreversal.StatusApproved,
			),
		).Count(ctx, db)
		if err != nil {
			return errReversalBlockers
		}
		pendingReversalCount = int(reversals)

		js := journalsourcedb.Sources
		sources, err := js.Where(
			js.JournalBatch().FiscalPeriodID.Eq(entity.ID),
			js.Status.NeCol(keel.ConstText("Posted")),
		).Count(ctx, db)
		if err != nil {
			return errJournalSourceBlockers
		}
		pendingSourceCount = int(sources)

		return nil
	})
	switch {
	case errors.Is(err, errReversalBlockers):
		multiErr.AddInternal(fmt.Errorf("failed to validate journal reversal close blockers: %w", err))
		return
	case errors.Is(err, errJournalSourceBlockers):
		multiErr.AddInternal(fmt.Errorf("failed to validate journal source close blockers: %w", err))
		return
	case err != nil:
		multiErr.AddInternal(fmt.Errorf("failed to validate manual journal close blockers: %w", err))
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
	if pendingReversalCount > 0 {
		multiErr.Add(
			"accounting",
			errortypes.ErrInvalidOperation,
			"Cannot close fiscal period while {0} journal reversals are pending posting or approval",
			pendingReversalCount,
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
