//nolint:funlen // existing legacy workflow/API shape is intentionally kept stable
package invoiceservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerledger"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// invoiceJournalPlan is the journal entry posting an invoice writes, and the
// customer ledger line beside it, before the batch and entry numbers that
// only a write takes.
type invoiceJournalPlan struct {
	event            tenant.JournalSourceEventType
	period           *fiscalperiod.FiscalPeriod
	postingDate      int64
	amount           int64
	ledgerAmount     int64
	entryStatus      string
	batchStatus      string
	postedAt         *int64
	postedByID       pulid.ID
	requiresApproval bool
	isApproved       bool
	approvedByID     pulid.ID
	approvedAt       *int64
	lines            []repositories.JournalPostingLine
}

// errNoLedgerEntry is an invoice the organization's accounting control books
// no ledger entry for.
var errNoLedgerEntry = errors.New("no ledger entry for this invoice")

// planInvoiceJournal is the ledger output posting the invoice creates, or
// errNoLedgerEntry when the organization's accounting control creates none
// for it.
func (s *Service) planInvoiceJournal(
	ctx context.Context,
	entity *invoice.Invoice,
	actor *servicesports.RequestActor,
) (*invoiceJournalPlan, error) {
	if s.accountingRepo == nil || s.journalRepo == nil || s.sequenceGenerator == nil ||
		entity == nil ||
		actor == nil {
		return nil, errNoLedgerEntry
	}

	accountingControl, err := s.accountingRepo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errNoLedgerEntry
		}
		return nil, err
	}
	event := invoicePostingSourceEvent(entity.BillType)
	if !s.accountingPolicyService().CanCreateInvoiceLedgerEntry(accountingControl, event) {
		return nil, errNoLedgerEntry
	}
	if !invoicePostingHasRequiredAccounts(accountingControl) {
		return nil, errortypes.NewValidationError(
			"accountingControl",
			errortypes.ErrRequired,
			"Invoice posting requires default Accounts Receivable and revenue accounts",
		)
	}

	period, postingDate, err := s.resolveInvoicePostingPeriod(ctx, entity, accountingControl)
	if err != nil {
		return nil, err
	}

	amount := entity.TotalAmountMinor
	if amount < 0 {
		amount = -amount
	}
	if amount == 0 {
		return nil, errNoLedgerEntry
	}

	plan := &invoiceJournalPlan{
		event:        event,
		period:       period,
		postingDate:  postingDate,
		amount:       amount,
		ledgerAmount: entity.TotalAmountMinor,
	}
	plan.entryStatus, plan.batchStatus, plan.postedAt, plan.postedByID, plan.requiresApproval,
		plan.isApproved, plan.approvedByID, plan.approvedAt = invoicePostingWorkflow(
		accountingControl,
		actor.UserID,
		*entity.PostedAt,
	)
	//nolint:exhaustive // only actionable enum states require explicit handling here
	switch entity.BillType {
	case billingqueue.BillTypeCreditMemo:
		plan.ledgerAmount = -amount
	case billingqueue.BillTypeDebitMemo:
		plan.ledgerAmount = amount
	}

	debitAccountID := accountingControl.DefaultARAccountID
	creditAccountID := accountingControl.DefaultRevenueAccountID
	if entity.BillType == billingqueue.BillTypeCreditMemo {
		debitAccountID = accountingControl.DefaultRevenueAccountID
		creditAccountID = accountingControl.DefaultARAccountID
	}
	plan.lines = []repositories.JournalPostingLine{
		{
			ID:           pulid.MustNew("jel_"),
			GLAccountID:  debitAccountID,
			LineNumber:   1,
			Description:  fmt.Sprintf("Invoice posted for %s", entity.Number),
			DebitAmount:  amount,
			CreditAmount: 0,
			NetAmount:    amount,
			CustomerID:   entity.CustomerID,
		},
		{
			ID:           pulid.MustNew("jel_"),
			GLAccountID:  creditAccountID,
			LineNumber:   2,
			Description:  fmt.Sprintf("Invoice posted for %s", entity.Number),
			DebitAmount:  0,
			CreditAmount: amount,
			NetAmount:    -amount,
			CustomerID:   entity.CustomerID,
		},
	}

	return plan, nil
}

func (s *Service) createInvoiceJournalPosting(
	ctx context.Context,
	entity *invoice.Invoice,
	actor *servicesports.RequestActor,
) error {
	plan, err := s.planInvoiceJournal(ctx, entity, actor)
	if errors.Is(err, errNoLedgerEntry) {
		return nil
	}
	if err != nil {
		return err
	}

	batchNumber, err := s.sequenceGenerator.GenerateJournalBatchNumber(
		ctx,
		entity.OrganizationID,
		entity.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return err
	}
	entryNumber, err := s.sequenceGenerator.GenerateJournalEntryNumber(
		ctx,
		entity.OrganizationID,
		entity.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return err
	}

	event := plan.event
	err = s.journalRepo.CreatePosting(ctx, repositories.CreateJournalPostingParams{
		BatchID:              pulid.MustNew("jb_"),
		OrganizationID:       entity.OrganizationID,
		BusinessUnitID:       entity.BusinessUnitID,
		BatchNumber:          batchNumber,
		BatchType:            "System",
		BatchStatus:          plan.batchStatus,
		BatchDescription:     fmt.Sprintf("Invoice posted for %s", entity.Number),
		FiscalYearID:         plan.period.FiscalYearID,
		FiscalPeriodID:       plan.period.ID,
		AccountingDate:       plan.postingDate,
		PostedAt:             plan.postedAt,
		PostedByID:           plan.postedByID,
		CreatedByID:          actor.UserID,
		UpdatedByID:          actor.UserID,
		EntryID:              pulid.MustNew("je_"),
		EntryNumber:          entryNumber,
		EntryType:            "Standard",
		EntryStatus:          plan.entryStatus,
		ReferenceNumber:      entity.Number,
		ReferenceType:        event.String(),
		ReferenceID:          entity.ID.String(),
		EntryDescription:     fmt.Sprintf("Invoice posted for %s", entity.Number),
		TotalDebit:           plan.amount,
		TotalCredit:          plan.amount,
		IsPosted:             plan.postedAt != nil,
		IsAutoGenerated:      true,
		RequiresApproval:     plan.requiresApproval,
		IsApproved:           plan.isApproved,
		ApprovedByID:         plan.approvedByID,
		ApprovedAt:           plan.approvedAt,
		SourceID:             pulid.MustNew("jsrc_"),
		SourceObjectType:     "Invoice",
		SourceObjectID:       entity.ID.String(),
		SourceEventType:      event.String(),
		SourceStatus:         plan.entryStatus,
		SourceDocumentNumber: entity.Number,
		SourceIdempotencyKey: "invoice-posted:" + entity.ID.String(),
		Lines:                plan.lines,
	})
	if err != nil {
		return err
	}
	if s.customerLedgerRepo == nil {
		return nil
	}

	return s.customerLedgerRepo.AppendEntries(ctx, []*customerledger.CustomerLedgerEntry{{
		ID:               pulid.MustNew("cledg_"),
		OrganizationID:   entity.OrganizationID,
		BusinessUnitID:   entity.BusinessUnitID,
		CustomerID:       entity.CustomerID,
		SourceObjectType: "Invoice",
		SourceObjectID:   entity.ID.String(),
		SourceEventType:  event.String(),
		DocumentNumber:   entity.Number,
		TransactionDate:  plan.postingDate,
		LineNumber:       1,
		AmountMinor:      plan.ledgerAmount,
		CreatedByID:      actor.UserID,
	}})
}

func (s *Service) resolveInvoicePostingPeriod(
	ctx context.Context,
	entity *invoice.Invoice,
	accountingControl *tenant.AccountingControl,
) (*fiscalperiod.FiscalPeriod, int64, error) {
	period, err := s.validator.fiscalPeriodRepo.GetPeriodByDate(
		ctx,
		repositories.GetPeriodByDateRequest{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
			Date:  *entity.PostedAt,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, 0, errortypes.NewValidationError(
				"postedAt",
				errortypes.ErrRequired,
				"No fiscal period covers the invoice posting date",
			)
		}
		return nil, 0, err
	}

	//nolint:exhaustive // only actionable enum states require explicit handling here
	switch period.Status {
	case fiscalperiod.StatusOpen, fiscalperiod.StatusLocked:
		return period, *entity.PostedAt, nil
	case fiscalperiod.StatusClosed, fiscalperiod.StatusPermanentlyClosed:
		if accountingControl.ClosedPeriodPostingPolicy != tenant.ClosedPeriodPostingPolicyPostToNextOpen {
			return nil, 0, errortypes.NewBusinessError(
				"Invoice posting cannot create ledger output in a closed period; reopen the period first",
			)
		}

		periods, listErr := s.validator.fiscalPeriodRepo.ListByFiscalYearID(
			ctx,
			repositories.ListByFiscalYearIDRequest{
				FiscalYearID: period.FiscalYearID,
				OrgID:        entity.OrganizationID,
				BuID:         entity.BusinessUnitID,
			},
		)
		if listErr != nil {
			return nil, 0, listErr
		}
		for _, candidate := range periods {
			if candidate == nil || candidate.PeriodNumber <= period.PeriodNumber {
				continue
			}
			if candidate.Status == fiscalperiod.StatusOpen ||
				candidate.Status == fiscalperiod.StatusLocked {
				return candidate, candidate.StartDate, nil
			}
		}
		return nil, 0, errortypes.NewBusinessError(
			"No next open fiscal period is available for invoice ledger posting",
		)
	default:
		return nil, 0, errortypes.NewBusinessError(
			"Invoice posting cannot create ledger output in an inactive fiscal period",
		)
	}
}

func invoicePostingHasRequiredAccounts(accountingControl *tenant.AccountingControl) bool {
	return !accountingControl.DefaultARAccountID.IsNil() &&
		!accountingControl.DefaultRevenueAccountID.IsNil()
}

func invoicePostingSourceEvent(billType billingqueue.BillType) tenant.JournalSourceEventType {
	//nolint:exhaustive // only actionable enum states require explicit handling here
	switch billType {
	case billingqueue.BillTypeCreditMemo:
		return tenant.JournalSourceEventCreditMemoPosted
	case billingqueue.BillTypeDebitMemo:
		return tenant.JournalSourceEventDebitMemoPosted
	default:
		return tenant.JournalSourceEventInvoicePosted
	}
}

func invoicePostingWorkflow( //nolint:gocritic // stable workflow tuple
	accountingControl *tenant.AccountingControl,
	userID pulid.ID,
	now int64,
) (entryStatus, batchStatus string, postedAt *int64, postedByID pulid.ID, requiresApproval, isApproved bool, approvedByID pulid.ID, approvedAt *int64) {
	entryStatus = "Posted"
	batchStatus = "Posted"
	postedAt = &now
	postedByID = userID
	requiresApproval = false
	isApproved = true
	approvedByID = userID
	approvedAt = &now

	if accountingControl.JournalPostingMode != tenant.JournalPostingModeManual {
		return entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved, approvedByID, approvedAt
	}

	entryStatus = "Pending"
	batchStatus = "Pending"
	postedAt = nil
	postedByID = pulid.Nil
	requiresApproval = accountingControl.RequireManualJEApproval
	isApproved = !accountingControl.RequireManualJEApproval
	if !requiresApproval {
		entryStatus = "Approved"
		batchStatus = "Approved"
		return entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved, approvedByID, approvedAt
	}
	approvedByID = pulid.Nil
	approvedAt = nil
	return entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved, approvedByID, approvedAt
}
