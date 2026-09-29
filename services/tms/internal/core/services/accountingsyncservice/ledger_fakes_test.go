package accountingsyncservice

import (
	"context"
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type fakeLedger struct {
	mu       sync.Mutex
	journals []*repositories.LedgerJournal
	sums     []repositories.SumLedgerRequest
}

func (f *fakeLedger) add(journal *repositories.LedgerJournal) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.journals = append(f.journals, journal)
}

func (f *fakeLedger) sumRequests() []repositories.SumLedgerRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.sums)
}

func (f *fakeLedger) GetJournal(
	_ context.Context,
	req *repositories.GetLedgerJournalRequest,
) (*repositories.LedgerJournal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, journal := range f.journals {
		if journal.ID == req.ID {
			return journal, nil
		}
	}
	return nil, errortypes.NewNotFoundError("JournalEntry not found within your organization")
}

func (f *fakeLedger) GetEntryType(
	ctx context.Context,
	req *repositories.GetLedgerJournalRequest,
) (journalentry.EntryType, error) {
	journal, err := f.GetJournal(ctx, req)
	if err != nil {
		return "", err
	}
	return journalentry.EntryType(journal.EntryType), nil
}

func counted(journal *repositories.LedgerJournal, includeClosing bool) bool {
	entryType := journalentry.EntryType(journal.EntryType)
	reverses := journalentry.EntryType(journal.ReversesEntryType)
	if !includeClosing {
		return accountingsync.JournalSendable(entryType, reverses)
	}
	return entryType != journalentry.EntryTypeOpening && reverses != journalentry.EntryTypeOpening
}

func (f *fakeLedger) ListJournals(
	_ context.Context,
	req *repositories.ListLedgerJournalsRequest,
) ([]*repositories.LedgerJournal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*repositories.LedgerJournal{}
	for _, journal := range f.journals {
		if journal.AccountingDate < req.From || journal.AccountingDate >= req.Before ||
			!counted(journal, false) {
			continue
		}
		out = append(out, journal)
	}
	return out, nil
}

func (f *fakeLedger) SumLines(
	_ context.Context,
	req *repositories.SumLedgerRequest,
) ([]repositories.LedgerAccountBalance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sums = append(f.sums, *req)
	type key struct {
		account pulid.ID
		party   repositories.LedgerParty
	}
	positions := map[key]int{}
	out := []repositories.LedgerAccountBalance{}
	for _, journal := range f.journals {
		if journal.AccountingDate >= req.Before ||
			(req.From != nil && journal.AccountingDate < *req.From) ||
			!counted(journal, req.IncludeClosing) {
			continue
		}
		for _, line := range journal.Lines {
			party := repositories.LedgerParty{}
			if slices.Contains(req.PartyAccountIDs, line.AccountID) {
				party = line.Party
				if party.IsZero() {
					party = journal.Party
				}
			}
			k := key{account: line.AccountID, party: party}
			pos, ok := positions[k]
			if !ok {
				out = append(out, repositories.LedgerAccountBalance{
					AccountID:   line.AccountID,
					AccountCode: line.AccountCode,
					AccountName: line.AccountName,
					Party:       party,
				})
				pos = len(out) - 1
				positions[k] = pos
			}
			out[pos].DebitMinor += line.DebitMinor
			out[pos].CreditMinor += line.CreditMinor
		}
	}
	return out, nil
}

func (f *fakeLedger) ListActiveAccounts(
	context.Context,
	*repositories.ListLedgerAccountsRequest,
) ([]repositories.LedgerAccount, error) {
	return nil, nil
}

type fakeReferences struct {
	repositories.AccountingReferenceObjectRepository

	mu   sync.Mutex
	rows map[string]*accountingsync.AccountingReferenceObject
}

func newFakeReferences() *fakeReferences {
	return &fakeReferences{rows: map[string]*accountingsync.AccountingReferenceObject{}}
}

func (f *fakeReferences) account(externalID string, class accountingsync.AccountClass) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[externalID] = &accountingsync.AccountingReferenceObject{
		Kind:         accountingsync.ReferenceKindAccount,
		ExternalID:   externalID,
		Name:         externalID,
		AccountClass: class,
		Active:       true,
	}
}

func (f *fakeReferences) GetByExternalIDs(
	_ context.Context,
	req *repositories.GetAccountingReferenceObjectsRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*accountingsync.AccountingReferenceObject{}
	for _, id := range req.ExternalIDs {
		if ref, ok := f.rows[id]; ok && ref.Kind == req.Kind {
			out = append(out, ref)
		}
	}
	return out, nil
}

func (f *fakeWriter) CreateJournalEntry(
	_ context.Context,
	doc *services.AccountingJournalDocument,
) (*services.AccountingDocumentResult, error) {
	sent := *doc
	sent.Lines = slices.Clone(doc.Lines)
	return f.record("CreateJournalEntry", &sent)
}

func (f *fakeWriter) UpdateJournalEntry(
	_ context.Context,
	doc *services.AccountingJournalDocument,
) (*services.AccountingDocumentResult, error) {
	sent := *doc
	sent.Lines = slices.Clone(doc.Lines)
	return f.record("UpdateJournalEntry", &sent)
}

func (f *fakeWriter) DeleteJournalEntry(
	_ context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	return f.record("DeleteJournalEntry", ref)
}
