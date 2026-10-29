package qboconnector

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/emoss08/trenova/shared/quickbookstxn"
)

var (
	_ accountingconnectorports.AccountingJournalWriter = (*Connector)(nil)
	_ accountingconnectorports.AccountingLedgerReader  = (*Connector)(nil)
)

var (
	errJournalPosting = errors.New("quickbooks: a journal line is either a debit or a credit")
	errJournalParty   = errors.New("quickbooks: a journal line names a customer or a vendor")
)

func (c *Connector) CreateJournalEntry(
	ctx context.Context,
	doc *accountingconnectorports.AccountingJournalDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
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
	created, err := quickbookstxn.CreateJournalEntry(ctx, client, doc.RequestID, txn)
	if err != nil {
		return nil, err
	}
	return c.journalResult(created), nil
}

func (c *Connector) UpdateJournalEntry(
	ctx context.Context,
	doc *accountingconnectorports.AccountingJournalDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
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
	saved, err := quickbookstxn.UpdateJournalEntry(ctx, client, doc.RequestID, id, txn)
	if err != nil {
		return nil, err
	}
	return c.journalResult(saved), nil
}

func (c *Connector) DeleteJournalEntry(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
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
	if _, err = quickbookstxn.DeleteJournalEntry(ctx, client, ref.RequestID, id); err != nil {
		return nil, err
	}
	return &accountingconnectorports.AccountingDocumentResult{ExternalID: id}, nil
}

func (c *Connector) journalResult(
	saved *quickbookstxn.TxnResult,
) *accountingconnectorports.AccountingDocumentResult {
	return &accountingconnectorports.AccountingDocumentResult{
		ExternalID: saved.ID,
		DocNumber:  saved.DocNumber,
		Refs: map[string]string{
			accountingsync.ExternalRefDocument:     saved.ID,
			accountingsync.ExternalRefDocumentType: string(quickbookstxn.TxnJournalEntry),
			accountingsync.ExternalRefURL: c.appURL(
				quickbookstxn.TxnJournalEntry.AppPath(),
				saved.ID,
			),
		},
	}
}

func journalTxnOf(
	doc *accountingconnectorports.AccountingJournalDocument,
) (*quickbookstxn.JournalTxn, error) {
	txn := &quickbookstxn.JournalTxn{
		DocNumber:    doc.DocNumber,
		TxnDate:      doc.TxnDate,
		CurrencyCode: doc.CurrencyCode,
		ExchangeRate: doc.ExchangeRate,
		PrivateNote:  doc.PrivateNote,
		Lines:        make([]quickbookstxn.JournalLine, 0, len(doc.Lines)),
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
		txn.Lines = append(txn.Lines, quickbookstxn.JournalLine{
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

func postingOf(
	posting accountingconnectorports.AccountingJournalPosting,
) (quickbookstxn.PostingType, error) {
	switch posting {
	case accountingconnectorports.AccountingJournalDebit:
		return quickbookstxn.PostingDebit, nil
	case accountingconnectorports.AccountingJournalCredit:
		return quickbookstxn.PostingCredit, nil
	default:
		return "", errJournalPosting
	}
}

func journalEntityOf(
	line *accountingconnectorports.AccountingJournalLine,
) (quickbookstxn.JournalEntityType, error) {
	switch line.PartyKind {
	case "":
		return "", nil
	case accountingconnectorports.AccountingJournalCustomer:
		return quickbookstxn.JournalEntityCustomer, nil
	case accountingconnectorports.AccountingJournalVendor:
		return quickbookstxn.JournalEntityVendor, nil
	default:
		return "", errJournalParty
	}
}

func (c *Connector) ReadTrialBalance(
	ctx context.Context,
	req *accountingconnectorports.ReadTrialBalanceRequest,
) ([]accountingconnectorports.AccountingTrialBalanceRow, error) {
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
	out := make([]accountingconnectorports.AccountingTrialBalanceRow, 0, len(rows))
	for idx := range rows {
		out = append(out, accountingconnectorports.AccountingTrialBalanceRow{
			AccountExternalID: rows[idx].AccountID,
			AccountName:       rows[idx].AccountName,
			Debit:             rows[idx].Debit,
			Credit:            rows[idx].Credit,
		})
	}
	return out, nil
}
