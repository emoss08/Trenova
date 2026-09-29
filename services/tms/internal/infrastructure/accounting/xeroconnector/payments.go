package xeroconnector

import (
	"context"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/shopspring/decimal"
)

const (
	docTypePayment           = "Payment"
	docTypeBatchPayment      = "BatchPayment"
	refPaymentPrefix         = "payment:"
	refReplaced              = "replaced"
	refShortPayAllocPrefix   = "shortPayAllocation:"
	shortPayNote             = "Short pay on "
	stepReplaceDelete        = 1
	stepReplaceCreate        = 2
	stepShortPayBase         = 10
	stepsPerShortPay         = 2
	unappliedPaymentCode     = "unapplied-payment"
	unappliedPaymentResolved = "Apply the whole payment to invoices in Trenova, then retry"
)

type paymentWrite struct {
	requestID  string
	externalID string
	refs       map[string]string
	result     *services.AccountingDocumentResult
}

func (w *paymentWrite) createKey() string {
	if strings.TrimSpace(w.externalID) == "" {
		return w.requestID
	}
	return accountingsync.SyncStepRequestID(w.requestID, stepReplaceCreate)
}

func (c *Connector) SavePayment(
	ctx context.Context,
	doc *services.AccountingPaymentDocument,
) (*services.AccountingDocumentResult, error) {
	cash, err := cashApplications(doc)
	if err != nil {
		return nil, err
	}
	date, err := parseDate(doc.TxnDate)
	if err != nil {
		return nil, err
	}
	client, err := c.client(doc.Auth)
	if err != nil {
		return nil, err
	}
	write := &paymentWrite{
		requestID:  doc.RequestID,
		externalID: doc.ExternalID,
		refs:       doc.Refs,
		result:     newResult(doc.Refs),
	}

	if err = replacePrevious(ctx, client, write); err != nil {
		return write.result, err
	}
	for idx := range doc.Applications {
		if !doc.Applications[idx].ShortPayAmount.IsPositive() {
			continue
		}
		if err = c.shortPay(ctx, client, doc, idx, date, write.result); err != nil {
			return write.result, err
		}
	}

	paymentRef := paymentReference(doc.ReferenceNumber, doc.RequestID)
	clearPaymentRefs(write.result.Refs)
	if len(cash) == 1 {
		payment, createErr := client.CreatePayment(ctx, write.createKey(), xero.PaymentInput{
			InvoiceID:    cash[0].InvoiceExternalID,
			AccountID:    doc.DepositAccountExternalID,
			Date:         date,
			Amount:       cash[0].AppliedAmount,
			CurrencyRate: doc.ExchangeRate,
			Reference:    paymentRef,
		})
		if createErr != nil {
			return write.result, createErr
		}
		finishPayment(write.result, payment)
		return write.result, nil
	}

	lines := make([]xero.BatchPaymentLine, 0, len(cash))
	for idx := range cash {
		lines = append(lines, xero.BatchPaymentLine{
			InvoiceID: cash[idx].InvoiceExternalID,
			Amount:    cash[idx].AppliedAmount,
		})
	}
	batch, err := client.CreateBatchPayment(ctx, write.createKey(), xero.BatchPaymentInput{
		AccountID: doc.DepositAccountExternalID,
		Date:      date,
		Reference: paymentRef,
		Payments:  lines,
	})
	if err != nil {
		return write.result, err
	}
	finishBatch(write.result, batch)
	return write.result, nil
}

func cashApplications(
	doc *services.AccountingPaymentDocument,
) ([]*services.AccountingPaymentApplication, error) {
	cash := make([]*services.AccountingPaymentApplication, 0, len(doc.Applications))
	applied := decimal.Zero
	for idx := range doc.Applications {
		app := &doc.Applications[idx]
		if app.AppliedAmount.IsNegative() || app.ShortPayAmount.IsNegative() {
			return nil, xero.ErrAmountNotPositive
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
			Message: providerName + " records a payment only against invoices, " +
				"so a payment with an unapplied amount cannot be sent",
			Resolution: unappliedPaymentResolved,
		}
	}
	return cash, nil
}

func replacePrevious(ctx context.Context, client *xero.Client, write *paymentWrite) error {
	previous := strings.TrimSpace(write.externalID)
	if previous == "" || write.result.Refs[refReplaced] == previous {
		return nil
	}
	if err := deletePaymentDocument(
		ctx,
		client,
		accountingsync.SyncStepRequestID(write.requestID, stepReplaceDelete),
		previous,
		write.refs,
	); err != nil {
		return err
	}
	write.result.Refs[refReplaced] = previous
	return nil
}

func deletePaymentDocument(
	ctx context.Context,
	client *xero.Client,
	key, externalID string,
	refs map[string]string,
) error {
	var err error
	if refs[accountingsync.ExternalRefDocumentType] == docTypeBatchPayment {
		err = client.DeleteBatchPayment(ctx, key, externalID)
	} else {
		err = client.DeletePayment(ctx, key, externalID)
	}
	if err != nil && !xero.IsNotFound(err) {
		return err
	}
	return nil
}

func clearPaymentRefs(refs map[string]string) {
	for key := range refs {
		if strings.HasPrefix(key, refPaymentPrefix) {
			refs[key] = ""
		}
	}
}

func finishPayment(result *services.AccountingDocumentResult, payment *xero.Payment) {
	result.ExternalID = payment.PaymentID
	result.DocNumber = payment.Reference
	result.Refs[accountingsync.ExternalRefDocument] = payment.PaymentID
	result.Refs[accountingsync.ExternalRefDocumentType] = docTypePayment
	result.Refs[refReplaced] = ""
	if payment.InvoiceID != "" {
		result.Refs[refPaymentPrefix+strings.ToLower(payment.InvoiceID)] = payment.PaymentID
	}
}

func finishBatch(result *services.AccountingDocumentResult, batch *xero.BatchPayment) {
	result.ExternalID = batch.BatchPaymentID
	result.DocNumber = batch.Reference
	result.Refs[accountingsync.ExternalRefDocument] = batch.BatchPaymentID
	result.Refs[accountingsync.ExternalRefDocumentType] = docTypeBatchPayment
	result.Refs[refReplaced] = ""
	for idx := range batch.Payments {
		item := &batch.Payments[idx]
		if item.InvoiceID != "" && item.PaymentID != "" {
			result.Refs[refPaymentPrefix+strings.ToLower(item.InvoiceID)] = item.PaymentID
		}
	}
}

func paymentReference(number, requestID string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r == '"' || r == '\\' || r < ' ' || r == 0x7f {
			return ' '
		}
		return r
	}, number)
	cleaned = stringutils.CollapseWhitespace(cleaned)
	id := strings.TrimSpace(requestID)
	if id == "" {
		return reference(cleaned)
	}
	budget := xero.MaxReferenceLength - len([]rune(id)) - 1
	if budget <= 0 {
		return stringutils.TruncateRunes(id, xero.MaxReferenceLength)
	}
	return strings.TrimSpace(
		strings.TrimSpace(stringutils.TruncateRunes(cleaned, budget)) + " " + id,
	)
}

func shortPayRefKey(invoiceID string) string {
	return accountingsync.ExternalRefShortPayPrefix + invoiceID
}

func (c *Connector) shortPay(
	ctx context.Context,
	client *xero.Client,
	doc *services.AccountingPaymentDocument,
	idx int,
	date time.Time,
	result *services.AccountingDocumentResult,
) error {
	app := &doc.Applications[idx]
	if strings.TrimSpace(app.ShortPayCreditExternalID) != "" {
		return nil
	}
	key := shortPayRefKey(app.InvoiceExternalID)
	allocationKey := refShortPayAllocPrefix + app.InvoiceExternalID
	noteID := result.Refs[key]

	if noteID == "" {
		if strings.TrimSpace(doc.ShortPayAccountExternalID) == "" {
			return &accountingsync.SyncError{
				Category:   accountingsync.SyncErrorMapping,
				Code:       accountingsync.ItemRoleShortPayWriteOff,
				Message:    "The short pay write-off account is not mapped",
				Resolution: "Map the short pay write-off to a " + providerName + " account",
			}
		}
		code, err := c.accountCode(
			ctx,
			client,
			doc.ShortPayAccountExternalID,
			"The short pay write-off",
		)
		if err != nil {
			return err
		}
		text := shortPayNote + app.InvoiceNumber
		note, err := client.CreateCreditNote(
			ctx,
			accountingsync.SyncStepRequestID(doc.RequestID, stepShortPayBase+idx*stepsPerShortPay),
			xero.CreditNoteInput{
				Type:            xero.CreditNoteTypeReceivable,
				ContactID:       doc.CustomerExternalID,
				Reference:       reference(text),
				Date:            date,
				CurrencyCode:    doc.CurrencyCode,
				CurrencyRate:    doc.ExchangeRate,
				Status:          xero.StatusAuthorised,
				LineAmountTypes: xero.LineAmountsNoTax,
				Lines: []xero.LineItem{lineItem(&lineSpec{
					description: text,
					amount:      app.ShortPayAmount,
					accountCode: code,
				})},
			},
		)
		if err != nil {
			return err
		}
		noteID = note.CreditNoteID
		result.Refs[key] = noteID
	}
	if result.Refs[allocationKey] != "" {
		return nil
	}

	allocation, err := client.Allocate(
		ctx,
		accountingsync.SyncStepRequestID(doc.RequestID, stepShortPayBase+idx*stepsPerShortPay+1),
		noteID,
		xero.AllocationInput{
			InvoiceID: app.InvoiceExternalID,
			Amount:    app.ShortPayAmount,
			Date:      date,
		},
	)
	if err != nil {
		return err
	}
	result.Refs[allocationKey] = allocation.AllocationID
	return nil
}

func (c *Connector) VoidPayment(
	ctx context.Context,
	ref *services.AccountingDocumentRef,
) (*services.AccountingDocumentResult, error) {
	client, err := c.client(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := newResult(ref.Refs)
	result.ExternalID = ref.ExternalID

	if err = deletePaymentDocument(ctx, client, ref.RequestID, ref.ExternalID, ref.Refs); err != nil {
		return result, err
	}

	step := 0
	for _, key := range slices.Sorted(maps.Keys(result.Refs)) {
		if !strings.HasPrefix(key, accountingsync.ExternalRefShortPayPrefix) || result.Refs[key] == "" {
			continue
		}
		step++
		if err = voidCreditNote(
			ctx,
			client,
			accountingsync.SyncStepRequestID(ref.RequestID, step),
			result.Refs[key],
		); err != nil {
			return result, err
		}
	}
	return result, nil
}
