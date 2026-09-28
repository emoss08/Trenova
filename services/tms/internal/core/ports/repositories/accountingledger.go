package repositories

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type LedgerPartyKind string

const (
	LedgerPartyCustomer = LedgerPartyKind("Customer")
	LedgerPartyCarrier  = LedgerPartyKind("Carrier")
	LedgerPartyDriver   = LedgerPartyKind("Driver")
)

type LedgerParty struct {
	Kind LedgerPartyKind
	ID   pulid.ID
	Name string
}

func (p LedgerParty) IsZero() bool {
	return p.Kind == "" || p.ID.IsNil()
}

type LedgerJournalLine struct {
	AccountID   pulid.ID
	AccountCode string
	AccountName string
	DebitMinor  int64
	CreditMinor int64
	Description string
	Party       LedgerParty
}

func (l *LedgerJournalLine) NetMinor() int64 {
	return l.DebitMinor - l.CreditMinor
}

type LedgerJournal struct {
	ID                   pulid.ID
	EntryNumber          string
	EntryType            string
	Description          string
	AccountingDate       int64
	PostedAt             int64
	IsReversal           bool
	ReversalOfNumber     string
	SourceObjectType     string
	SourceObjectID       string
	SourceDocumentNumber string
	Party                LedgerParty
	Lines                []LedgerJournalLine
}

type GetLedgerJournalRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type ListLedgerJournalsRequest struct {
	TenantInfo pagination.TenantInfo
	From       int64
	Before     int64
}

type LedgerAccountBalance struct {
	AccountID   pulid.ID
	AccountCode string
	AccountName string
	Category    string
	Party       LedgerParty
	DebitMinor  int64
	CreditMinor int64
}

func (b *LedgerAccountBalance) NetMinor() int64 {
	return b.DebitMinor - b.CreditMinor
}

type SumLedgerRequest struct {
	TenantInfo      pagination.TenantInfo
	From            *int64
	Before          int64
	PartyAccountIDs []pulid.ID
	IncludeClosing  bool
}

type ListLedgerAccountsRequest struct {
	TenantInfo pagination.TenantInfo
	Since      *int64
}

type LedgerAccount struct {
	ID       pulid.ID
	Code     string
	Name     string
	Category string
}

type AccountingLedgerSource interface {
	GetJournal(ctx context.Context, req *GetLedgerJournalRequest) (*LedgerJournal, error)
	ListJournals(ctx context.Context, req *ListLedgerJournalsRequest) ([]*LedgerJournal, error)
	SumLines(ctx context.Context, req *SumLedgerRequest) ([]LedgerAccountBalance, error)
	ListActiveAccounts(
		ctx context.Context,
		req *ListLedgerAccountsRequest,
	) ([]LedgerAccount, error)
}
