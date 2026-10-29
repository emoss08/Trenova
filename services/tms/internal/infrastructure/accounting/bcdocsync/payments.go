package bcdocsync

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcrefsync"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/bcsession"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentraldocs"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/businesscentralpay"
	"github.com/emoss08/trenova/shared/businesscentralref"
	"github.com/shopspring/decimal"
)

const (
	batchCodePrefix          = "TRN"
	batchCodeHashChars       = 7
	shortPayNote             = "Short pay on "
	stepShortPayBase         = 10
	unappliedPaymentCode     = "unapplied-payment"
	unappliedPaymentResolved = "Apply the whole payment to invoices in Trenova, then retry"
	receiptsJournalName      = "Trenova receipts "
	paymentsJournalName      = "Trenova payments "
)

func (c *Documents) SavePayment(
	ctx context.Context,
	doc *accountingconnectorports.AccountingPaymentDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if strings.TrimSpace(doc.ExternalID) != "" {
		return nil, reversalRefused()
	}
	cash, err := cashApplications(doc)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(doc.DepositAccountExternalID) == "" {
		return nil, unmappedAccount("The deposit account")
	}
	client, err := c.Client(doc.Auth)
	if err != nil {
		return nil, err
	}
	posting, err := c.PostingFor(ctx, client, doc.TxnDate)
	if err != nil {
		return nil, err
	}
	result := newResult(doc.Refs)
	for idx := range doc.Applications {
		if !doc.Applications[idx].ShortPayAmount.IsPositive() {
			continue
		}
		if err = c.shortPay(ctx, &shortPayWrite{
			client: client, posting: posting, doc: doc, idx: idx, result: result,
		}); err != nil {
			return result, err
		}
	}

	number := RequestReference(doc.RequestID)
	if err = c.postPayment(ctx, &journalWrite{
		client:         client,
		party:          businesscentralref.PartyCustomer,
		accountID:      doc.DepositAccountExternalID,
		documentNumber: number,
		lines:          customerPaymentLines(doc, cash, number),
		refs:           result.Refs,
	}); err != nil {
		return result, err
	}
	finishPayment(result, number, DocTypeCustomerPayment)
	return result, nil
}

func cashApplications(
	doc *accountingconnectorports.AccountingPaymentDocument,
) ([]*accountingconnectorports.AccountingPaymentApplication, error) {
	cash := make([]*accountingconnectorports.AccountingPaymentApplication, 0, len(doc.Applications))
	applied := decimal.Zero
	for idx := range doc.Applications {
		app := &doc.Applications[idx]
		if app.AppliedAmount.IsNegative() || app.ShortPayAmount.IsNegative() {
			return nil, businesscentralerrors.ErrAmountInvalid
		}
		if app.AppliedAmount.IsPositive() {
			cash = append(cash, app)
			applied = applied.Add(app.AppliedAmount)
		}
	}
	if len(cash) == 0 {
		return nil, &accountingsync.SyncError{
			Category: accountingsync.SyncErrorValidation,
			Code:     unappliedPaymentCode,
			Message: "The payment applies no cash to an invoice, so " + bcsession.ProviderName +
				" has nothing to record",
			Resolution: unappliedPaymentResolved,
		}
	}
	if !sameMoney(applied, doc.TotalAmount) {
		return nil, &accountingsync.SyncError{
			Category: accountingsync.SyncErrorValidation,
			Code:     unappliedPaymentCode,
			Message: bcsession.ProviderName + " records a payment line only against one invoice, " +
				"so a payment with an unapplied amount cannot be sent",
			Resolution: unappliedPaymentResolved,
		}
	}
	return cash, nil
}

func customerPaymentLines(
	doc *accountingconnectorports.AccountingPaymentDocument,
	cash []*accountingconnectorports.AccountingPaymentApplication,
	number string,
) []businesscentralpay.PaymentInput {
	lines := make([]businesscentralpay.PaymentInput, 0, len(cash))
	for _, app := range cash {
		lines = append(lines, businesscentralpay.PaymentInput{
			PartyID:        doc.CustomerExternalID,
			PostingDate:    doc.TxnDate,
			DocumentNumber: number,
			ExternalDocumentNumber: bcrefsync.CleanText(
				doc.ReferenceNumber,
				businesscentral.MaxExternalDocumentNumberLength,
			),
			Amount:             businesscentralpay.CustomerPaymentAmount(app.AppliedAmount),
			AppliesToInvoiceID: strings.TrimSpace(app.InvoiceExternalID),
			Description: bcrefsync.CleanText(
				"Payment of "+app.InvoiceNumber,
				businesscentral.MaxDescriptionLength,
			),
		})
	}
	return lines
}

type journalWrite struct {
	client         *businesscentral.Client
	party          businesscentralref.PartyKind
	accountID      string
	documentNumber string
	lines          []businesscentralpay.PaymentInput
	refs           map[string]string
}

func (c *Documents) postPayment(ctx context.Context, w *journalWrite) error {
	batch, err := paymentBatch(ctx, w.client, w.party, w.accountID)
	if err != nil {
		return err
	}
	w.refs[RefJournal] = batch.ID
	release, err := c.LockBatch(ctx, w.client.Ref().String()+"|"+batch.ID)
	if err != nil {
		return err
	}
	defer release()

	posted, err := businesscentralpay.GeneralLedgerEntries(ctx, w.client, w.documentNumber)
	if err != nil {
		return err
	}
	if len(posted) > 0 {
		return nil
	}
	if err = addPaymentLines(ctx, w, batch.ID); err != nil {
		return err
	}
	if err = businesscentralpay.PostJournal(ctx, w.client, batch.ID); err != nil {
		if businesscentralerrors.IsNotFound(err) ||
			errors.Is(err, businesscentralerrors.ErrUnsupportedAction) {
			return unpostedJournal(batch.Code)
		}
		return err
	}
	return nil
}

func paymentBatch(
	ctx context.Context,
	client *businesscentral.Client,
	party businesscentralref.PartyKind,
	accountID string,
) (*businesscentralpay.PaymentJournal, error) {
	code := BatchCode(accountID)
	found, err := businesscentralpay.PaymentJournals(ctx, client, party, code)
	if err != nil {
		return nil, err
	}
	if len(found) > 0 {
		return &found[0], nil
	}
	name := receiptsJournalName + code
	if party == businesscentralref.PartyVendor {
		name = paymentsJournalName + code
	}
	created, err := businesscentralpay.CreatePaymentJournal(
		ctx,
		client,
		party,
		&businesscentralpay.PaymentJournalInput{
			Code:               code,
			DisplayName:        name,
			BalancingAccountID: accountID,
		},
	)
	if err == nil {
		return created, nil
	}
	if !businesscentralerrors.IsDuplicate(err) {
		return nil, err
	}
	found, err = businesscentralpay.PaymentJournals(ctx, client, party, code)
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, businesscentralerrors.ErrUnexpectedPayload
	}
	return &found[0], nil
}

func BatchCode(accountID string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(accountID))))
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:])
	return batchCodePrefix + encoded[:batchCodeHashChars]
}

func addPaymentLines(ctx context.Context, w *journalWrite, batchID string) error {
	existing, err := businesscentralpay.Payments(ctx, w.client, w.party, batchID, w.documentNumber)
	if err != nil {
		return err
	}
	held := make(map[string]struct{}, len(existing))
	for idx := range existing {
		held[strings.ToLower(existing[idx].AppliesToInvoiceID)] = struct{}{}
	}
	for idx := range w.lines {
		line := &w.lines[idx]
		if _, ok := held[strings.ToLower(line.AppliesToInvoiceID)]; ok {
			continue
		}
		if _, err = businesscentralpay.CreatePayment(ctx, w.client, w.party, batchID, line); err != nil {
			return err
		}
	}
	return nil
}

type shortPayWrite struct {
	client  *businesscentral.Client
	posting *bcsession.PostingContext
	doc     *accountingconnectorports.AccountingPaymentDocument
	idx     int
	result  *accountingconnectorports.AccountingDocumentResult
}

func shortPayRefKey(invoiceID string) string {
	return accountingsync.ExternalRefShortPayPrefix + invoiceID
}

func (c *Documents) shortPay(ctx context.Context, w *shortPayWrite) error {
	app := &w.doc.Applications[w.idx]
	if strings.TrimSpace(app.ShortPayCreditExternalID) != "" {
		return nil
	}
	key := shortPayRefKey(app.InvoiceExternalID)
	if strings.TrimSpace(w.doc.ShortPayItemExternalID) == "" {
		return &accountingsync.SyncError{
			Category:   accountingsync.SyncErrorMapping,
			Code:       accountingsync.ItemRoleShortPayWriteOff,
			Message:    "The short pay write-off item is not mapped",
			Resolution: "Map the short pay write-off to a " + bcsession.ProviderName + " item",
		}
	}
	text := shortPayNote + app.InvoiceNumber
	memo, err := postDocument(ctx, &documentWrite{
		client: w.client,
		kind:   businesscentraldocs.DocumentSalesCreditMemo,
		input: &businesscentraldocs.DocumentInput{
			PartyID: w.doc.CustomerExternalID,
			ExternalDocumentNumber: RequestReference(accountingsync.SyncStepRequestID(
				w.doc.RequestID, stepShortPayBase+w.idx,
			)),
			DocumentDate: w.doc.TxnDate,
			PostingDate:  w.doc.TxnDate,
			CurrencyCode: w.posting.Currency(w.doc.CurrencyCode),
			InvoiceID:    strings.TrimSpace(app.InvoiceExternalID),
			Lines: []businesscentraldocs.LineInput{{
				ItemID:      strings.TrimSpace(w.doc.ShortPayItemExternalID),
				Description: bcrefsync.CleanText(text, businesscentral.MaxDescriptionLength),
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   app.ShortPayAmount,
			}},
		},
		refs:   w.result.Refs,
		refKey: key,
	})
	if err != nil {
		return err
	}
	w.result.Refs[key] = memo.ID
	return nil
}

func (c *Documents) VoidPayment(
	_ context.Context,
	_ *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return nil, reversalRefused()
}

func (c *Documents) CreateCreditApplication(
	_ context.Context,
	_ *accountingconnectorports.AccountingCreditApplicationDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	return nil, creditApplicationRefused()
}

func (c *Documents) VoidCreditApplication(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	memoID := appliedMemo(ref.Refs)
	if memoID == "" {
		return nil, creditApplicationRefused()
	}
	client, err := c.Client(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID
	if _, err = retireDocument(
		ctx,
		client,
		businesscentraldocs.DocumentSalesCreditMemo,
		memoID,
	); err != nil {
		return result, err
	}
	return result, nil
}

func appliedMemo(refs map[string]string) string {
	if strings.TrimSpace(refs[accountingsync.ExternalRefApplication]) == "" {
		return ""
	}
	return strings.TrimSpace(refs[accountingsync.ExternalRefDocument])
}
