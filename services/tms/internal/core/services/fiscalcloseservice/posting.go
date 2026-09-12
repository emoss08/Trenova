package fiscalcloseservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	sourceObjectType    = "FiscalYear"
	sourceEventClosed   = "FiscalYearClosed"
	sourceEventOpening  = "FiscalYearOpeningBalance"
	sourceEventReversed = "FiscalYearCloseReversed"

	batchTypeClose = "Close"
	postedStatus   = "Posted"
)

// Post writes the year-end accounting for a fiscal year: the closing entry that
// empties the income statement into retained earnings, and the opening entry
// that carries the balance sheet into the next year. It must run inside the same
// transaction that flips the year to Closed.
func (s *Service) Post(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
	userID pulid.ID,
) (*fiscalclose.PostResult, error) {
	plan, inputs, err := s.prepare(ctx, fy)
	if err != nil {
		return nil, err
	}
	if multiErr := blockerError(plan.Blockers); multiErr != nil {
		return nil, multiErr
	}

	result := &fiscalclose.PostResult{
		FiscalYearID:   fy.ID,
		NetIncomeMinor: plan.NetIncomeMinor,
		Revision:       plan.Revision,
		Entries:        make([]*fiscalclose.PostedEntry, 0, 2),
	}

	if !plan.ClosingEntry.IsEmpty() {
		periodID, perr := s.ensureClosingPeriod(ctx, fy, &inputs.closingPeriod, userID)
		if perr != nil {
			return nil, perr
		}

		posted, perr := s.postPlanEntry(ctx, &postRequest{
			fiscalYear:     fy,
			entry:          plan.ClosingEntry,
			fiscalPeriodID: periodID,
			entryType:      journalentry.EntryTypeClosing,
			sourceEvent:    sourceEventClosed,
			idempotencyKey: closeIdempotencyKey(sourceEventClosed, fy.ID, plan.Revision),
			userID:         userID,
		})
		if perr != nil {
			return nil, perr
		}
		result.Entries = append(result.Entries, posted)
	}

	if !plan.OpeningEntry.IsEmpty() {
		posted, perr := s.postPlanEntry(ctx, &postRequest{
			fiscalYear:     fy,
			entry:          plan.OpeningEntry,
			fiscalPeriodID: plan.OpeningEntry.FiscalPeriodID,
			entryType:      journalentry.EntryTypeOpening,
			sourceEvent:    sourceEventOpening,
			idempotencyKey: closeIdempotencyKey(sourceEventOpening, fy.ID, plan.Revision),
			userID:         userID,
		})
		if perr != nil {
			return nil, perr
		}
		result.Entries = append(result.Entries, posted)
	}

	s.l.Info("posted fiscal year close",
		zap.String("fiscalYearId", fy.ID.String()),
		zap.Int64("netIncomeMinor", plan.NetIncomeMinor),
		zap.Int("entries", len(result.Entries)),
		zap.Int("revision", plan.Revision),
	)

	return result, nil
}

// ensureClosingPeriod returns the period the closing entry posts into and leaves
// it closed behind us. Fiscal years generated after this feature already carry an
// Inactive adjusting period; years created before it do not, so one is made here.
// Either way it ends the close in Closed: it exists to hold the close, not to
// accept further postings.
func (s *Service) ensureClosingPeriod(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
	target *periodTarget,
	userID pulid.ID,
) (pulid.ID, error) {
	now := timeutils.NowUnix()

	if target.MustCreate {
		created, err := s.fiscalPeriodRepo.Create(ctx, &fiscalperiod.FiscalPeriod{
			OrganizationID:        fy.OrganizationID,
			BusinessUnitID:        fy.BusinessUnitID,
			FiscalYearID:          fy.ID,
			PeriodNumber:          target.PeriodNumber,
			PeriodType:            fiscalperiod.PeriodTypeAdjusting,
			Status:                fiscalperiod.StatusClosed,
			Name:                  target.Name,
			StartDate:             target.StartDate,
			EndDate:               target.EndDate,
			IsAdjusting:           true,
			AllowAdjustingEntries: true,
			ClosedAt:              &now,
			ClosedByID:            userID,
		})
		if err != nil {
			return pulid.Nil, err
		}

		return created.ID, nil
	}

	if target.Status == fiscalperiod.StatusClosed ||
		target.Status == fiscalperiod.StatusPermanentlyClosed {
		return target.ID, nil
	}

	if _, err := s.fiscalPeriodRepo.Close(ctx, repositories.CloseFiscalPeriodRequest{
		ID:         target.ID,
		TenantInfo: tenantOf(fy),
		ClosedByID: userID,
		ClosedAt:   now,
	}); err != nil {
		return pulid.Nil, err
	}

	return target.ID, nil
}

type postRequest struct {
	fiscalYear     *fiscalyear.FiscalYear
	entry          *fiscalclose.PlanEntry
	fiscalPeriodID pulid.ID
	entryType      journalentry.EntryType
	sourceEvent    string
	idempotencyKey string
	userID         pulid.ID
}

func (s *Service) postPlanEntry(
	ctx context.Context,
	req *postRequest,
) (*fiscalclose.PostedEntry, error) {
	batchNumber, entryNumber, err := s.nextNumbers(ctx, req.fiscalYear)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	batchID := pulid.MustNew("jb_")
	entryID := pulid.MustNew("je_")

	lines := make([]repositories.JournalPostingLine, 0, len(req.entry.Lines))
	for idx, line := range req.entry.Lines {
		lines = append(lines, repositories.JournalPostingLine{
			ID:           pulid.MustNew("jel_"),
			GLAccountID:  line.GLAccountID,
			LineNumber:   int16(idx + 1), //nolint:gosec // line counts never approach int16 overflow
			Description:  planLineDescription(req.entry, line, req.fiscalYear),
			DebitAmount:  line.DebitMinor,
			CreditAmount: line.CreditMinor,
			NetAmount:    line.DebitMinor - line.CreditMinor,
		})
	}

	if err = s.journalPostRepo.CreatePosting(ctx, repositories.CreateJournalPostingParams{
		BatchID:              batchID,
		OrganizationID:       req.fiscalYear.OrganizationID,
		BusinessUnitID:       req.fiscalYear.BusinessUnitID,
		BatchNumber:          batchNumber,
		BatchType:            batchTypeClose,
		BatchStatus:          postedStatus,
		BatchDescription:     req.entry.Description,
		FiscalYearID:         req.entry.FiscalYearID,
		FiscalPeriodID:       req.fiscalPeriodID,
		AccountingDate:       req.entry.AccountingDate,
		PostedAt:             &now,
		PostedByID:           req.userID,
		CreatedByID:          req.userID,
		UpdatedByID:          req.userID,
		EntryID:              entryID,
		EntryNumber:          entryNumber,
		EntryType:            req.entryType.String(),
		EntryStatus:          postedStatus,
		ReferenceNumber:      req.fiscalYear.Name,
		ReferenceType:        sourceObjectType,
		ReferenceID:          req.fiscalYear.ID.String(),
		EntryDescription:     req.entry.Description,
		TotalDebit:           req.entry.TotalDebitMinor,
		TotalCredit:          req.entry.TotalCreditMinor,
		IsPosted:             true,
		IsAutoGenerated:      true,
		IsApproved:           true,
		ApprovedByID:         req.userID,
		ApprovedAt:           &now,
		SourceID:             pulid.MustNew("jsrc_"),
		SourceObjectType:     sourceObjectType,
		SourceObjectID:       req.fiscalYear.ID.String(),
		SourceEventType:      req.sourceEvent,
		SourceStatus:         postedStatus,
		SourceDocumentNumber: req.fiscalYear.Name,
		SourceIdempotencyKey: req.idempotencyKey,
		Lines:                lines,
	}); err != nil {
		return nil, err
	}

	return &fiscalclose.PostedEntry{
		Kind:             req.entry.Kind,
		JournalEntryID:   entryID,
		JournalBatchID:   batchID,
		EntryNumber:      entryNumber,
		FiscalYearID:     req.entry.FiscalYearID,
		FiscalPeriodID:   req.fiscalPeriodID,
		TotalDebitMinor:  req.entry.TotalDebitMinor,
		TotalCreditMinor: req.entry.TotalCreditMinor,
		LineCount:        len(lines),
	}, nil
}

// Reverse unwinds a posted close so the year can be reopened. The original
// entries are left in place and offset by reversing entries, because an audited
// ledger records corrections rather than erasing them.
func (s *Service) Reverse(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
	userID pulid.ID,
	reason string,
) (*fiscalclose.ReverseResult, error) {
	entries, err := s.postedCloseEntries(ctx, fy)
	if err != nil {
		return nil, err
	}

	result := &fiscalclose.ReverseResult{
		FiscalYearID: fy.ID,
		Entries:      make([]*fiscalclose.ReversedEntry, 0, len(entries)),
	}

	for _, entry := range entries {
		if err = s.assertYearAcceptsReversal(ctx, fy, entry); err != nil {
			return nil, err
		}
		if err = s.assertPeriodAcceptsReversal(ctx, fy, entry); err != nil {
			return nil, err
		}

		reversed, rerr := s.reverseEntry(ctx, fy, entry, userID, reason)
		if rerr != nil {
			return nil, rerr
		}
		result.Entries = append(result.Entries, reversed)
	}

	s.l.Info("reversed fiscal year close",
		zap.String("fiscalYearId", fy.ID.String()),
		zap.Int("entries", len(result.Entries)),
	)

	return result, nil
}

// postedCloseEntries returns the close entries that are still standing: an entry
// already offset by a reversal is skipped so a second reopen cannot double-count.
func (s *Service) postedCloseEntries(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
) ([]*journalentry.JournalEntry, error) {
	sources, err := s.sourceRepo.ListByObject(ctx, repositories.GetJournalSourceByObjectRequest{
		TenantInfo:       tenantOf(fy),
		SourceObjectType: sourceObjectType,
		SourceObjectID:   fy.ID.String(),
	})
	if err != nil {
		return nil, err
	}

	entries := make([]*journalentry.JournalEntry, 0, len(sources))
	seen := make(map[pulid.ID]struct{}, len(sources))
	for _, source := range sources {
		if source == nil || source.JournalEntryID.IsNil() {
			continue
		}
		if source.SourceEventType != sourceEventClosed &&
			source.SourceEventType != sourceEventOpening {
			continue
		}
		if _, ok := seen[source.JournalEntryID]; ok {
			continue
		}
		seen[source.JournalEntryID] = struct{}{}

		entry, entryErr := s.journalEntryRepo.GetByID(ctx, repositories.GetJournalEntryByIDRequest{
			ID:         source.JournalEntryID,
			TenantInfo: tenantOf(fy),
		})
		if entryErr != nil {
			return nil, entryErr
		}
		if entry.ReversedByID.IsNotNil() || entry.Status == journalentry.StatusReversed {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// assertYearAcceptsReversal refuses to unwind opening balances out of a later
// fiscal year that has itself been closed. Years come apart newest first, the
// same order they were closed in.
func (s *Service) assertYearAcceptsReversal(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
	entry *journalentry.JournalEntry,
) error {
	if entry.FiscalYearID == fy.ID {
		return nil
	}

	target, err := s.fiscalYearRepo.GetByID(ctx, repositories.GetFiscalYearByIDRequest{
		ID:         entry.FiscalYearID,
		TenantInfo: tenantOf(fy),
	})
	if err != nil {
		return err
	}

	if target.Status == fiscalyear.StatusClosed ||
		target.Status == fiscalyear.StatusPermanentlyClosed {
		return errortypes.NewBusinessError("{0} is {1} and holds the opening balances carried out of {2}. Reopen {3} first.", target.Name, target.Status, fy.Name, target.Name)
	}

	return nil
}

// assertPeriodAcceptsReversal refuses to unwind a close into a period that has
// been permanently closed, which is the one state no authorization can reopen.
func (s *Service) assertPeriodAcceptsReversal(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
	entry *journalentry.JournalEntry,
) error {
	period, err := s.fiscalPeriodRepo.GetByID(ctx, repositories.GetFiscalPeriodByIDRequest{
		ID:         entry.FiscalPeriodID,
		TenantInfo: tenantOf(fy),
	})
	if err != nil {
		return err
	}

	if period.Status == fiscalperiod.StatusPermanentlyClosed {
		return errortypes.NewBusinessError("{0} is permanently closed, so entry {1} cannot be reversed and the year cannot be reopened.", period.Name, entry.EntryNumber)
	}

	return nil
}

func (s *Service) reverseEntry(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
	entry *journalentry.JournalEntry,
	userID pulid.ID,
	reason string,
) (*fiscalclose.ReversedEntry, error) {
	batchNumber, entryNumber, err := s.nextNumbers(ctx, fy)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	batchID := pulid.MustNew("jb_")
	reversalEntryID := pulid.MustNew("je_")

	lines := make([]repositories.JournalPostingLine, 0, len(entry.Lines))
	for _, line := range entry.Lines {
		if line == nil {
			continue
		}
		lines = append(lines, repositories.JournalPostingLine{
			ID:           pulid.MustNew("jel_"),
			GLAccountID:  line.GLAccountID,
			LineNumber:   line.LineNumber,
			Description:  line.Description,
			DebitAmount:  line.CreditAmount,
			CreditAmount: line.DebitAmount,
			NetAmount:    -line.NetAmount,
			CustomerID:   line.CustomerID,
			LocationID:   line.LocationID,
		})
	}

	description := fmt.Sprintf("Reversal of %s for reopening %s", entry.EntryNumber, fy.Name)

	if err = s.journalPostRepo.CreatePosting(ctx, repositories.CreateJournalPostingParams{
		BatchID:              batchID,
		OrganizationID:       fy.OrganizationID,
		BusinessUnitID:       fy.BusinessUnitID,
		BatchNumber:          batchNumber,
		BatchType:            batchTypeClose,
		BatchStatus:          postedStatus,
		BatchDescription:     description,
		FiscalYearID:         entry.FiscalYearID,
		FiscalPeriodID:       entry.FiscalPeriodID,
		AccountingDate:       entry.AccountingDate,
		PostedAt:             &now,
		PostedByID:           userID,
		CreatedByID:          userID,
		UpdatedByID:          userID,
		EntryID:              reversalEntryID,
		EntryNumber:          entryNumber,
		EntryType:            journalentry.EntryTypeReversal.String(),
		EntryStatus:          postedStatus,
		ReferenceNumber:      entry.EntryNumber,
		ReferenceType:        sourceObjectType,
		ReferenceID:          fy.ID.String(),
		EntryDescription:     description,
		TotalDebit:           entry.TotalCredit,
		TotalCredit:          entry.TotalDebit,
		IsPosted:             true,
		IsAutoGenerated:      true,
		IsReversal:           true,
		ReversalOfID:         entry.ID,
		ReversalDate:         &now,
		ReversalReason:       reason,
		IsApproved:           true,
		ApprovedByID:         userID,
		ApprovedAt:           &now,
		SourceID:             pulid.MustNew("jsrc_"),
		SourceObjectType:     sourceObjectType,
		SourceObjectID:       fy.ID.String(),
		SourceEventType:      sourceEventReversed,
		SourceStatus:         postedStatus,
		SourceDocumentNumber: fy.Name,
		SourceIdempotencyKey: "fiscal-year-close-reversal:" + entry.ID.String(),
		Lines:                lines,
	}); err != nil {
		return nil, err
	}

	if err = s.journalEntryRepo.MarkReversed(
		ctx,
		repositories.MarkJournalEntryReversedRequest{
			OriginalEntryID: entry.ID,
			ReversalEntryID: reversalEntryID,
			OrganizationID:  fy.OrganizationID,
			BusinessUnitID:  fy.BusinessUnitID,
			ReversalDate:    entry.AccountingDate,
			ReversalReason:  reason,
			UpdatedByID:     userID,
		},
	); err != nil {
		return nil, err
	}

	return &fiscalclose.ReversedEntry{
		Kind:              reversalKind(entry.EntryType),
		OriginalEntryID:   entry.ID,
		ReversalEntryID:   reversalEntryID,
		ReversalEntryNum:  entryNumber,
		FiscalPeriodID:    entry.FiscalPeriodID,
		TotalDebitMinor:   entry.TotalCredit,
		TotalCreditMinor:  entry.TotalDebit,
		ReversedLineCount: len(lines),
	}, nil
}

func (s *Service) nextNumbers(
	ctx context.Context,
	fy *fiscalyear.FiscalYear,
) (batchNumber, entryNumber string, err error) {
	batchNumber, err = s.generator.GenerateJournalBatchNumber(
		ctx,
		fy.OrganizationID,
		fy.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return "", "", err
	}

	entryNumber, err = s.generator.GenerateJournalEntryNumber(
		ctx,
		fy.OrganizationID,
		fy.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return "", "", err
	}

	return batchNumber, entryNumber, nil
}

func planLineDescription(
	entry *fiscalclose.PlanEntry,
	line *fiscalclose.PlanLine,
	fy *fiscalyear.FiscalYear,
) string {
	if entry.Kind == fiscalclose.EntryKindOpening {
		return fmt.Sprintf("Opening balance carried forward from %s", fy.Name)
	}
	if line.IsRetainedEarn {
		if line.CreditMinor > 0 {
			return fmt.Sprintf("Net income for %s", fy.Name)
		}
		return fmt.Sprintf("Net loss for %s", fy.Name)
	}

	return fmt.Sprintf("Close %s to retained earnings", line.AccountName)
}

func reversalKind(entryType journalentry.EntryType) fiscalclose.EntryKind {
	if entryType == journalentry.EntryTypeOpening {
		return fiscalclose.EntryKindOpening
	}

	return fiscalclose.EntryKindClosing
}

func closeIdempotencyKey(event string, fiscalYearID pulid.ID, revision int) string {
	return fmt.Sprintf("%s:%s:%d", event, fiscalYearID.String(), revision)
}
