package qboconnector

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/shared/quickbooks"
)

var (
	_ accountingsyncports.AccountingJournalWriter = (*Connector)(nil)
	_ accountingsyncports.AccountingLedgerReader  = (*Connector)(nil)
)

var (
	errJournalPosting = errors.New("quickbooks: a journal line is either a debit or a credit")
	errJournalParty   = errors.New("quickbooks: a journal line names a customer or a vendor")
)

func (c *Connector) CreateJournalEntry(
	ctx context.Context,
	doc *accountingsyncports.AccountingJournalDocument,
) (*accountingsyncports.AccountingDocumentResult, error) {
	if !doc.Kind.IsLedger() {
		return nil, errDocumentKind
	}
	txn, err := journalTxnOf(doc)
	if err != nil {
		return nil, err
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	created, err := client.CreateJournalEntry(ctx, doc.RequestID, txn)
	if err != nil {
		return nil, err
	}
	return c.journalResult(created), nil
}

func (c *Connector) UpdateJournalEntry(
	ctx context.Context,
	doc *accountingsyncports.AccountingJournalDocument,
) (*accountingsyncports.AccountingDocumentResult, error) {
	if !doc.Kind.IsLedger() {
		return nil, errDocumentKind
	}
	id := strings.TrimSpace(doc.ExternalID)
	if id == "" {
		return nil, errExternalIDRequired
	}
	txn, err := journalTxnOf(doc)
	if err != nil {
		return nil, err
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	saved, err := client.UpdateJournalEntry(ctx, doc.RequestID, id, txn)
	if err != nil {
		return nil, err
	}
	return c.journalResult(saved), nil
}

func (c *Connector) DeleteJournalEntry(
	ctx context.Context,
	ref *accountingsyncports.AccountingDocumentRef,
) (*accountingsyncports.AccountingDocumentResult, error) {
	if !ref.Kind.IsLedger() {
		return nil, errDocumentKind
	}
	id := strings.TrimSpace(ref.ExternalID)
	if id == "" {
		return nil, errExternalIDRequired
	}
	client, err := c.client(ref.Auth)
	if err != nil {
		return nil, err
	}
	if _, err = client.DeleteJournalEntry(ctx, ref.RequestID, id); err != nil {
		return nil, err
	}
	return &accountingsyncports.AccountingDocumentResult{ExternalID: id}, nil
}

func (c *Connector) journalResult(saved *quickbooks.TxnResult) *accountingsyncports.AccountingDocumentResult {
	return &accountingsyncports.AccountingDocumentResult{
		ExternalID: saved.ID,
		DocNumber:  saved.DocNumber,
		Refs: map[string]string{
			accountingsync.ExternalRefDocument:     saved.ID,
			accountingsync.ExternalRefDocumentType: string(quickbooks.TxnJournalEntry),
			accountingsync.ExternalRefURL: c.appURL(
				quickbooks.TxnJournalEntry.AppPath(),
				saved.ID,
			),
		},
	}
}

func journalTxnOf(doc *accountingsyncports.AccountingJournalDocument) (*quickbooks.JournalTxn, error) {
	txn := &quickbooks.JournalTxn{
		DocNumber:    doc.DocNumber,
		TxnDate:      doc.TxnDate,
		CurrencyCode: doc.CurrencyCode,
		ExchangeRate: doc.ExchangeRate,
		PrivateNote:  doc.PrivateNote,
		Lines:        make([]quickbooks.JournalLine, 0, len(doc.Lines)),
	}
	for idx := range doc.Lines {
		line := &doc.Lines[idx]
		posting, err := postingOf(line.Posting)
		if err != nil {
			return nil, err
		}
		entityType, err := journalEntityOf(line)
		if err != nil {
			return nil, err
		}
		txn.Lines = append(txn.Lines, quickbooks.JournalLine{
			PostingType: posting,
			AccountID:   line.AccountExternalID,
			Amount:      line.Amount,
			Description: line.Description,
			EntityType:  entityType,
			EntityID:    line.PartyExternalID,
		})
	}
	return txn, nil
}

func postingOf(posting accountingsyncports.AccountingJournalPosting) (quickbooks.PostingType, error) {
	switch posting {
	case accountingsyncports.AccountingJournalDebit:
		return quickbooks.PostingDebit, nil
	case accountingsyncports.AccountingJournalCredit:
		return quickbooks.PostingCredit, nil
	default:
		return "", errJournalPosting
	}
}

func journalEntityOf(line *accountingsyncports.AccountingJournalLine) (quickbooks.JournalEntityType, error) {
	switch line.PartyKind {
	case "":
		return "", nil
	case accountingsyncports.AccountingJournalCustomer:
		return quickbooks.JournalEntityCustomer, nil
	case accountingsyncports.AccountingJournalVendor:
		return quickbooks.JournalEntityVendor, nil
	default:
		return "", errJournalParty
	}
}

func (c *Connector) ReadTrialBalance(
	ctx context.Context,
	req *accountingsyncports.ReadTrialBalanceRequest,
) ([]accountingsyncports.AccountingTrialBalanceRow, error) {
	client, err := c.client(req.Auth)
	if err != nil {
		return nil, err
	}
	rows, err := client.TrialBalance(ctx, &quickbooks.TrialBalanceRequest{
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
	})
	if err != nil {
		return nil, err
	}
	out := make([]accountingsyncports.AccountingTrialBalanceRow, 0, len(rows))
	for idx := range rows {
		out = append(out, accountingsyncports.AccountingTrialBalanceRow{
			AccountExternalID: rows[idx].AccountID,
			AccountName:       rows[idx].AccountName,
			Debit:             rows[idx].Debit,
			Credit:            rows[idx].Credit,
		})
	}
	return out, nil
}
