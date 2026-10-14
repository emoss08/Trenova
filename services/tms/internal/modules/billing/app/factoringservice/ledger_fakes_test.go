package factoringservice

import (
	"context"
	"strconv"
	"sync"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type fakeTx struct {
	kernelports.DBConnection
}

func (fakeTx) WithTx(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

type fakeReserveEntries struct {
	factoringports.ReserveEntryRepository
	mu      sync.Mutex
	entries []*factoring.ReserveEntry
}

func (f *fakeReserveEntries) Create(
	_ context.Context,
	e *factoring.ReserveEntry,
) (*factoring.ReserveEntry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, existing := range f.entries {
		if e.ReversesEntryID.IsNotNil() && existing.ReversesEntryID == e.ReversesEntryID {
			return nil, errortypes.NewConflictError("reversed")
		}
	}
	e.CreatedAt = testNow + int64(len(f.entries))
	copied := *e
	f.entries = append(f.entries, &copied)
	return e, nil
}

func (f *fakeReserveEntries) ListForItem(
	_ context.Context,
	req factoringports.ListReserveEntriesForItemRequest,
) ([]*factoring.ReserveEntry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*factoring.ReserveEntry, 0, len(f.entries))
	for _, e := range f.entries {
		if e.SubmissionItemID == req.SubmissionItemID {
			copied := *e
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (f *fakeInvoices) LockForUpdate(
	_ context.Context,
	req invoiceports.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	if inv, ok := f.byID[req.ID]; ok {
		copied := *inv
		return &copied, nil
	}
	return nil, errortypes.NewNotFoundError("Invoice not found")
}

func (f *fakeInvoices) Update(_ context.Context, inv *invoice.Invoice) (*invoice.Invoice, error) {
	copied := *inv
	f.byID[inv.ID] = &copied
	return inv, nil
}

type fakeProjection struct {
	customerledgerports.CustomerLedgerProjectionRepository
	entries []*customerledger.CustomerLedgerEntry
}

func (f *fakeProjection) AppendEntries(
	_ context.Context,
	entries []*customerledger.CustomerLedgerEntry,
) error {
	f.entries = append(f.entries, entries...)
	return nil
}

func (f *fakeProjection) balance(invoiceID pulid.ID) int64 {
	var total int64
	for _, e := range f.entries {
		if e.RelatedInvoiceID == invoiceID {
			total += e.AmountMinor
		}
	}
	return total
}

type fakeControls struct {
	tenantports.AccountingControlRepository
	control *tenant.AccountingControl
}

func (f *fakeControls) GetByOrgID(context.Context, pulid.ID) (*tenant.AccountingControl, error) {
	return f.control, nil
}

type fakePeriods struct {
	fiscalperiodports.FiscalPeriodRepository
	periods []*fiscalperiod.FiscalPeriod
}

func (f *fakePeriods) GetPeriodByDate(
	_ context.Context,
	req fiscalperiodports.GetPeriodByDateRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	for _, p := range f.periods {
		if req.Date >= p.StartDate && req.Date <= p.EndDate {
			return p, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Fiscal period not found")
}

func (f *fakePeriods) ListByFiscalYearID(
	context.Context,
	fiscalperiodports.ListByFiscalYearIDRequest,
) ([]*fiscalperiod.FiscalPeriod, error) {
	return f.periods, nil
}

type fakeJournalNumbers struct {
	seqgen.Generator
	next int
}

func (f *fakeJournalNumbers) GenerateJournalBatchNumber(
	context.Context, pulid.ID, pulid.ID, string, string,
) (string, error) {
	f.next++
	return "JB" + strconv.Itoa(f.next), nil
}

func (f *fakeJournalNumbers) GenerateJournalEntryNumber(
	context.Context, pulid.ID, pulid.ID, string, string,
) (string, error) {
	f.next++
	return "JE" + strconv.Itoa(f.next), nil
}

type fakeJournals struct {
	journalentryports.JournalPostingRepository
	postings []journalentryports.CreateJournalPostingParams
}

//nolint:gocritic // the repository contract takes the params by value
func (f *fakeJournals) CreatePosting(
	_ context.Context,
	params journalentryports.CreateJournalPostingParams,
) error {
	for idx := range f.postings {
		if f.postings[idx].SourceIdempotencyKey == params.SourceIdempotencyKey {
			return errortypes.NewConflictError("duplicate journal source")
		}
	}
	f.postings = append(f.postings, params)
	return nil
}

func (f *fakeJournals) balance(accountID pulid.ID) int64 {
	var total int64
	for idx := range f.postings {
		for _, line := range f.postings[idx].Lines {
			if line.GLAccountID == accountID {
				total += line.DebitAmount - line.CreditAmount
			}
		}
	}
	return total
}
