package bcconnector

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/shopspring/decimal"
)

const (
	batchCodePrefix          = "TRN"
	batchCodeHashChars       = 7
	docTypeCustomerPayment   = "CustomerPayment"
	docTypeVendorPayment     = "VendorPayment"
	shortPayNote             = "Short pay on "
	stepShortPayBase         = 10
	unappliedPaymentCode     = "unapplied-payment"
	unappliedPaymentResolved = "Apply the whole payment to invoices in Trenova, then retry"
	receiptsJournalName      = "Trenova receipts "
	paymentsJournalName      = "Trenova payments "
)

func (c *Connector) SavePayment(
	ctx context.Context,
	doc *services.AccountingPaymentDocument,
) (*services.AccountingDocumentResult, error) {
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
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	posting, err := c.postingFor(ctx, client, doc.TxnDate)
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

	number := requestReference(doc.RequestID)
	if err = c.postPayment(ctx, &journalWrite{
		client:         client,
		party:          businesscentral.PartyCustomer,
		accountID:      doc.DepositAccountExternalID,
		documentNumber: number,
		lines:          customerPaymentLines(doc, cash, number),
		refs:           result.Refs,
	}); err != nil {
		return result, err
	}
	finishPayment(result, number, docTypeCustomerPayment)
	return result, nil
}

func cashApplications(
	doc *services.AccountingPaymentDocument,
) ([]*services.AccountingPaymentApplication, error) {
	cash := make([]*services.AccountingPaymentApplication, 0, len(doc.Applications))
	applied := decimal.Zero
	for idx := range doc.Applications {
		app := &doc.Applications[idx]
		if app.AppliedAmount.IsNegative() || app.ShortPayAmount.IsNegative() {
			return nil, businesscentral.ErrAmountInvalid
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
			Message: "The payment applies no cash to an invoice, so " + providerName +
				" has nothing to record",
			Resolution: unappliedPaymentResolved,
		}
	}
	if !sameMoney(applied, doc.TotalAmount) {
		return nil, &accountingsync.SyncError{
			Category: accountingsync.SyncErrorValidation,
			Code:     unappliedPaymentCode,
			Message: providerName + " records a payment line only against one invoice, " +
				"so a payment with an unapplied amount cannot be sent",
			Resolution: unappliedPaymentResolved,
		}
	}
	return cash, nil
}

func customerPaymentLines(
	doc *services.AccountingPaymentDocument,
	cash []*services.AccountingPaymentApplication,
	number string,
) []businesscentral.PaymentInput {
	lines := make([]businesscentral.PaymentInput, 0, len(cash))
	for _, app := range cash {
		lines = append(lines, businesscentral.PaymentInput{
			PartyID:        doc.CustomerExternalID,
			PostingDate:    doc.TxnDate,
			DocumentNumber: number,
			ExternalDocumentNumber: cleanText(
				doc.ReferenceNumber,
				businesscentral.MaxExternalDocumentNumberLength,
			),
			Amount:             businesscentral.CustomerPaymentAmount(app.AppliedAmount),
			AppliesToInvoiceID: strings.TrimSpace(app.InvoiceExternalID),
			Description: cleanText(
				"Payment of "+app.InvoiceNumber,
				businesscentral.MaxDescriptionLength,
			),
		})
	}
	return lines
}

func finishPayment(result *services.AccountingDocumentResult, number, docType string) {
	result.ExternalID = number
	result.DocNumber = number
	result.Refs[accountingsync.ExternalRefDocument] = number
	result.Refs[accountingsync.ExternalRefDocumentType] = docType
}

type journalWrite struct {
	client         *businesscentral.Client
	party          businesscentral.PartyKind
	accountID      string
	documentNumber string
	lines          []businesscentral.PaymentInput
	refs           map[string]string
}

func (c *Connector) postPayment(ctx context.Context, w *journalWrite) error {
	batch, err := paymentBatch(ctx, w.client, w.party, w.accountID)
	if err != nil {
		return err
	}
	w.refs[refJournal] = batch.ID
	release, err := c.locks.acquire(ctx, w.client.Ref().String()+"|"+batch.ID)
	if err != nil {
		return err
	}
	defer release()

	posted, err := w.client.GeneralLedgerEntries(ctx, w.documentNumber)
	if err != nil {
		return err
	}
	if len(posted) > 0 {
		return nil
	}
	if err = addPaymentLines(ctx, w, batch.ID); err != nil {
		return err
	}
	if err = w.client.PostJournal(ctx, batch.ID); err != nil {
		if businesscentral.IsNotFound(err) ||
			errors.Is(err, businesscentral.ErrUnsupportedAction) {
			return unpostedJournal(batch.Code)
		}
		return err
	}
	return nil
}

func paymentBatch(
	ctx context.Context,
	client *businesscentral.Client,
	party businesscentral.PartyKind,
	accountID string,
) (*businesscentral.PaymentJournal, error) {
	code := batchCode(accountID)
	found, err := client.PaymentJournals(ctx, party, code)
	if err != nil {
		return nil, err
	}
	if len(found) > 0 {
		return &found[0], nil
	}
	name := receiptsJournalName + code
	if party == businesscentral.PartyVendor {
		name = paymentsJournalName + code
	}
	created, err := client.CreatePaymentJournal(ctx, party, &businesscentral.PaymentJournalInput{
		Code:               code,
		DisplayName:        name,
		BalancingAccountID: accountID,
	})
	if err == nil {
		return created, nil
	}
	if !businesscentral.IsDuplicate(err) {
		return nil, err
	}
	found, err = client.PaymentJournals(ctx, party, code)
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, businesscentral.ErrUnexpectedPayload
	}
	return &found[0], nil
}

func batchCode(accountID string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(accountID))))
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:])
	return batchCodePrefix + encoded[:batchCodeHashChars]
}

func addPaymentLines(ctx context.Context, w *journalWrite, batchID string) error {
	existing, err := w.client.Payments(ctx, w.party, batchID, w.documentNumber)
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
		if _, err = w.client.CreatePayment(ctx, w.party, batchID, line); err != nil {
			return err
		}
	}
	return nil
}

type shortPayWrite struct {
	client  *businesscentral.Client
	posting *postingContext
	doc     *services.AccountingPaymentDocument
	idx     int
	result  *services.AccountingDocumentResult
}

func shortPayRefKey(invoiceID string) string {
	return accountingsync.ExternalRefShortPayPrefix + invoiceID
}

func (c *Connector) shortPay(ctx context.Context, w *shortPayWrite) error {
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
			Resolution: "Map the short pay write-off to a " + providerName + " item",
		}
	}
	text := shortPayNote + app.InvoiceNumber
	memo, err := postDocument(ctx, &documentWrite{
		client: w.client,
		kind:   businesscentral.DocumentSalesCreditMemo,
		input: &businesscentral.DocumentInput{
			PartyID: w.doc.CustomerExternalID,
			ExternalDocumentNumber: requestReference(accountingsync.SyncStepRequestID(
				w.doc.RequestID, stepShortPayBase+w.idx,
			)),
			DocumentDate: w.doc.TxnDate,
			PostingDate:  w.doc.TxnDate,
			CurrencyCode: w.posting.currency(w.doc.CurrencyCode),
			InvoiceID:    strings.TrimSpace(app.InvoiceExternalID),
			Lines: []businesscentral.LineInput{{
				ItemID:      strings.TrimSpace(w.doc.ShortPayItemExternalID),
				Description: cleanText(text, businesscentral.MaxDescriptionLength),
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

func (c *Connector) VoidPayment(
	_ context.Context,
	_ *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	return nil, reversalRefused()
}

func (c *Connector) CreateCreditApplication(
	_ context.Context,
	_ *services.AccountingCreditApplicationDocument,
) (*services.AccountingDocumentResult, error) {
	return nil, creditApplicationRefused()
}

func (c *Connector) VoidCreditApplication(
	ctx context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	memoID := appliedMemo(ref.Refs)
	if memoID == "" {
		return nil, creditApplicationRefused()
	}
	client, err := c.client(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID
	if _, err = c.retireDocument(
		ctx,
		client,
		businesscentral.DocumentSalesCreditMemo,
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
