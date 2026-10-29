package factoringservicetest

import (
	"context"
	"strconv"
	"sync"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringwatchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
)

type FakeTx struct{}

func (FakeTx) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	return fn(ctx)
}

type FakeReserveEntries struct {
	factoringledgerports.ReserveEntryRepository
	Mu      sync.Mutex
	Entries []*factoringledger.ReserveEntry
}

func (f *FakeReserveEntries) Create(
	_ context.Context,
	e *factoringledger.ReserveEntry,
) (*factoringledger.ReserveEntry, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	for _, existing := range f.Entries {
		if e.ReversesEntryID.IsNotNil() && existing.ReversesEntryID == e.ReversesEntryID {
			return nil, errortypes.NewConflictError("reversed")
		}
	}
	e.CreatedAt = Now + int64(len(f.Entries))
	copied := *e
	f.Entries = append(f.Entries, &copied)
	return e, nil
}

func (f *FakeReserveEntries) ListForItem(
	_ context.Context,
	req factoringledgerports.ListReserveEntriesForItemRequest,
) ([]*factoringledger.ReserveEntry, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	out := make([]*factoringledger.ReserveEntry, 0, len(f.Entries))
	for _, e := range f.Entries {
		if e.SubmissionItemID == req.SubmissionItemID {
			copied := *e
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (f *FakeInvoices) LockForUpdate(
	_ context.Context,
	req invoiceports.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	if inv, ok := f.ByID[req.ID]; ok {
		copied := *inv
		return &copied, nil
	}
	return nil, errortypes.NewNotFoundError("Invoice not found")
}

func (f *FakeInvoices) Update(_ context.Context, inv *invoice.Invoice) (*invoice.Invoice, error) {
	copied := *inv
	f.ByID[inv.ID] = &copied
	return inv, nil
}

type FakeProjection struct {
	customerledgerports.CustomerLedgerProjectionRepository
	Entries []*customerledger.CustomerLedgerEntry
}

func (f *FakeProjection) AppendEntries(
	_ context.Context,
	entries []*customerledger.CustomerLedgerEntry,
) error {
	f.Entries = append(f.Entries, entries...)
	return nil
}

func (f *FakeProjection) Balance(invoiceID pulid.ID) int64 {
	var total int64
	for _, e := range f.Entries {
		if e.RelatedInvoiceID == invoiceID {
			total += e.AmountMinor
		}
	}
	return total
}

type FakeControls struct {
	tenantports.AccountingControlRepository
	Control *accountingcontrol.AccountingControl
}

func (f *FakeControls) GetByOrgID(
	context.Context,
	pulid.ID,
) (*accountingcontrol.AccountingControl, error) {
	return f.Control, nil
}

type FakePeriods struct {
	fiscalperiodports.FiscalPeriodRepository
	Periods []*fiscalperiod.FiscalPeriod
}

func (f *FakePeriods) GetPeriodByDate(
	_ context.Context,
	req fiscalperiodports.GetPeriodByDateRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	for _, p := range f.Periods {
		if req.Date >= p.StartDate && req.Date <= p.EndDate {
			return p, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Fiscal period not found")
}

func (f *FakePeriods) ListByFiscalYearID(
	context.Context,
	fiscalperiodports.ListByFiscalYearIDRequest,
) ([]*fiscalperiod.FiscalPeriod, error) {
	return f.Periods, nil
}

type FakeJournalNumbers struct {
	seqgen.Generator
	next int
}

func (f *FakeJournalNumbers) GenerateJournalBatchNumber(
	context.Context, pulid.ID, pulid.ID, string, string,
) (string, error) {
	f.next++
	return "JB" + strconv.Itoa(f.next), nil
}

func (f *FakeJournalNumbers) GenerateJournalEntryNumber(
	context.Context, pulid.ID, pulid.ID, string, string,
) (string, error) {
	f.next++
	return "JE" + strconv.Itoa(f.next), nil
}

type FakeJournals struct {
	journalentryports.JournalPostingRepository
	Postings []journalentryports.CreateJournalPostingParams
}

//nolint:gocritic // the repository contract takes the params by value
func (f *FakeJournals) CreatePosting(
	_ context.Context,
	params journalentryports.CreateJournalPostingParams,
) error {
	for idx := range f.Postings {
		if f.Postings[idx].SourceIdempotencyKey == params.SourceIdempotencyKey {
			return errortypes.NewConflictError("duplicate journal source")
		}
	}
	f.Postings = append(f.Postings, params)
	return nil
}

func (f *FakeJournals) Balance(accountID pulid.ID) int64 {
	var total int64
	for idx := range f.Postings {
		for _, line := range f.Postings[idx].Lines {
			if line.GLAccountID == accountID {
				total += line.DebitAmount - line.CreditAmount
			}
		}
	}
	return total
}

func (f *FakeSubmissions) ClaimChargebackWarning(
	_ context.Context,
	req factoringsubmissionports.ClaimChargebackWarningRequest,
) (bool, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	for _, sub := range f.ByID {
		for _, item := range sub.Items {
			if item.ID != req.ItemID {
				continue
			}
			if item.ChargebackWarnedAt != nil ||
				item.Status != factoringsubmission.SubmissionItemStatusFunded {
				return false, nil
			}
			at := req.At
			item.ChargebackWarnedAt = &at
			return true, nil
		}
	}
	return false, nil
}

type FakeAttention struct {
	factoringwatchtowerports.WatchtowerReader
	Due []*factoringwatchtowerports.ChargebackDue
	Req *factoringwatchtowerports.ListChargebacksDueRequest
}

func (f *FakeAttention) ListChargebacksDue(
	_ context.Context,
	req *factoringwatchtowerports.ListChargebacksDueRequest,
) ([]*factoringwatchtowerports.ChargebackDue, error) {
	f.Req = req
	return f.Due, nil
}
