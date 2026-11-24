package fiscalperiodservice

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalcloseblockers"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalclose"
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

func (v *Validator) ValidateClose(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) error {
	multiErr := errortypes.NewMultiError()
	counts, err := v.closeCounts(ctx, entity)
	if err != nil {
		multiErr.AddInternal(err)
		return multiErr.Err()
	}
	for _, count := range counts {
		if count.count > 0 {
			multiErr.Add(count.field, errortypes.ErrInvalidOperation, count.message, count.count)
		}
	}

	return multiErr.Err()
}

func (v *Validator) CloseChecks(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) (blockers []*fiscalclose.Blocker, cleared []fiscalclose.Check, err error) {
	counts, err := v.closeCounts(ctx, entity)
	if err != nil {
		return nil, nil, err
	}
	for _, count := range counts {
		if count.count == 0 {
			cleared = append(cleared, count.check)
			continue
		}
		blockers = append(blockers, fiscalcloseblockers.Counted(
			ctx,
			fiscalcloseblockers.Refusal{
				Field:    count.field,
				Category: closeCategory,
				Check:    count.check,
			},
			count.count,
			count.message,
		))
	}

	return blockers, cleared, nil
}

type closeCount struct {
	check   fiscalclose.Check
	field   string
	message string
	count   int
}

const (
	closeCategory          = "period"
	fieldAccounting        = "accounting"
	fieldReconciliation    = "reconciliation"
	messageManualJournals  = "Cannot close fiscal period while {0} manual journal requests are pending posting or approval"
	messageReversals       = "Cannot close fiscal period while {0} journal reversals are pending posting or approval"
	messageUnposted        = "Cannot close fiscal period while {0} accounting sources remain unposted"
	messageReconciliations = "Cannot close fiscal period while {0} posted invoice reconciliation discrepancies remain unresolved"
)

func (v *Validator) closeCounts(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) ([]closeCount, error) {
	counts, err := v.countAccountingBlockers(ctx, entity)
	if err != nil {
		return nil, err
	}
	discrepancies, required, err := v.countReconciliationDiscrepancies(ctx, entity)
	if err != nil {
		return nil, err
	}
	if required {
		counts = append(counts, closeCount{
			check:   fiscalclose.CheckReconciliation,
			field:   fieldReconciliation,
			message: messageReconciliations,
			count:   discrepancies,
		})
	}

	return counts, nil
}

func (v *Validator) countReconciliationDiscrepancies(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) (count int, required bool, err error) {
	control, err := v.accountingRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to load accounting control: %w", err)
	}
	if !control.RequireReconciliationToClose ||
		control.ReconciliationMode == accountingcontrol.ReconciliationModeDisabled {
		return 0, false, nil
	}

	discrepancies, err := v.invoiceRepo.CountPostedReconciliationDiscrepancies(
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
		return 0, false, fmt.Errorf("failed to validate reconciliation discrepancies: %w", err)
	}

	return discrepancies, true, nil
}

func (v *Validator) countAccountingBlockers(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) ([]closeCount, error) {
	return dbtx.Read(ctx, v.db, func(ctx context.Context) ([]closeCount, error) {
		db := v.db.Keel(ctx)
		mjr := manualjournaldb.Requests
		manual, err := mjr.Where(
			mjr.RequestedFiscalPeriodID.Eq(entity.ID),
			mjr.Status.In(manualjournal.StatusPendingApproval, manualjournal.StatusApproved),
		).Count(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("failed to validate manual journal close blockers: %w", err)
		}

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
			return nil, fmt.Errorf("failed to validate journal reversal close blockers: %w", err)
		}

		js := journalsourcedb.Sources
		posted := keel.ConstText(postedStatus)
		sources, err := js.Where(
			js.JournalBatch().FiscalPeriodID.Eq(entity.ID),
			js.JournalBatch().Status.NeCol(posted),
			js.Status.NeCol(posted),
		).Count(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("failed to validate journal source close blockers: %w", err)
		}

		return []closeCount{
			{
				check:   fiscalclose.CheckManualJournals,
				field:   fieldAccounting,
				message: messageManualJournals,
				count:   int(manual),
			},
			{
				check:   fiscalclose.CheckJournalReversals,
				field:   fieldAccounting,
				message: messageReversals,
				count:   int(reversals),
			},
			{
				check:   fiscalclose.CheckUnpostedSources,
				field:   fieldAccounting,
				message: messageUnposted,
				count:   int(sources),
			},
		}, nil
	})
}

const postedStatus = "Posted"
