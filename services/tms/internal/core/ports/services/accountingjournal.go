package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/shopspring/decimal"
)

type AccountingJournalPosting string

const (
	AccountingJournalDebit  = AccountingJournalPosting("Debit")
	AccountingJournalCredit = AccountingJournalPosting("Credit")
)

type AccountingJournalPartyKind string

const (
	AccountingJournalCustomer = AccountingJournalPartyKind("Customer")
	AccountingJournalVendor   = AccountingJournalPartyKind("Vendor")
)

type AccountingJournalLine struct {
	Posting           AccountingJournalPosting
	AccountExternalID string
	Amount            decimal.Decimal
	Description       string
	PartyKind         AccountingJournalPartyKind
	PartyExternalID   string
}

type AccountingJournalDocument struct {
	Auth         AccountingDocumentAuth
	RequestID    string
	ExternalID   string
	Kind         accountingsync.SyncObjectType
	DocNumber    string
	TxnDate      string
	CurrencyCode string
	ExchangeRate decimal.Decimal
	PrivateNote  string
	Lines        []AccountingJournalLine
}

type AccountingJournalWriter interface {
	CreateJournalEntry(
		ctx context.Context,
		doc *AccountingJournalDocument,
	) (*AccountingDocumentResult, error)
	UpdateJournalEntry(
		ctx context.Context,
		doc *AccountingJournalDocument,
	) (*AccountingDocumentResult, error)
	DeleteJournalEntry(
		ctx context.Context,
		ref *AccountingDocumentRef,
	) (*AccountingDocumentResult, error)
}

type ReadTrialBalanceRequest struct {
	Auth      AccountingDocumentAuth
	StartDate string
	EndDate   string
}

type AccountingTrialBalanceRow struct {
	AccountExternalID string
	AccountName       string
	Debit             decimal.Decimal
	Credit            decimal.Decimal
}

func (r *AccountingTrialBalanceRow) Net() decimal.Decimal {
	return r.Debit.Sub(r.Credit)
}

type AccountingLedgerReader interface {
	ReadTrialBalance(
		ctx context.Context,
		req *ReadTrialBalanceRequest,
	) ([]AccountingTrialBalanceRow, error)
}
