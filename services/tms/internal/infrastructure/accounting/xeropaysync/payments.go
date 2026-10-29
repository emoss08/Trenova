package xeropaysync

import (
	"context"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/xerodocsync"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/xerosession"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/emoss08/trenova/shared/xerodocs"
	"github.com/emoss08/trenova/shared/xeroerrors"
	"github.com/emoss08/trenova/shared/xeropay"
	"github.com/shopspring/decimal"
)

const (
	RefShortPayAllocPrefix   = "shortPayAllocation:"
	shortPayNote             = "Short pay on "
	stepShortPayBase         = 10
	stepsPerShortPay         = 2
	UnappliedPaymentCode     = "unapplied-payment"
	unappliedPaymentResolved = "Apply the whole payment to invoices in Trenova, then retry"
)

type Payments struct {
	*xerosession.Session
}

type paymentWrite struct {
	requestID  string
	externalID string
	refs       map[string]string
	result     *accountingconnectorports.AccountingDocumentResult
}

func (w *paymentWrite) createKey() string {
	if strings.TrimSpace(w.externalID) == "" {
		return w.requestID
	}
	return accountingsync.SyncStepRequestID(w.requestID, xerodocsync.StepReplaceCreate)
}

func (c *Payments) SavePayment(
	ctx context.Context,
	doc *accountingconnectorports.AccountingPaymentDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	cash, err := cashApplications(doc)
	if err != nil {
		return nil, err
	}
	date, err := xerodocsync.ParseDate(doc.TxnDate)
	if err != nil {
		return nil, err
	}
	client, err := c.Client(doc.Auth)
	if err != nil {
		return nil, err
	}
	write := &paymentWrite{
		requestID:  doc.RequestID,
		externalID: doc.ExternalID,
		refs:       doc.Refs,
		result:     xerodocsync.NewResult(doc.Refs),
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

	paymentRef := xerodocsync.PaymentReference(doc.ReferenceNumber, doc.RequestID)
	clearPaymentRefs(write.result.Refs)
	if len(cash) == 1 {
		payment, createErr := xeropay.CreatePayment(
			ctx,
			client,
			write.createKey(),
			xeropay.PaymentInput{
				InvoiceID:    cash[0].InvoiceExternalID,
				AccountID:    doc.DepositAccountExternalID,
				Date:         date,
				Amount:       cash[0].AppliedAmount,
				CurrencyRate: doc.ExchangeRate,
				Reference:    paymentRef,
			},
		)
		if createErr != nil {
			return write.result, createErr
		}
		xerodocsync.FinishPayment(write.result, payment)
		return write.result, nil
	}

	lines := make([]xeropay.BatchPaymentLine, 0, len(cash))
	for idx := range cash {
		lines = append(lines, xeropay.BatchPaymentLine{
			InvoiceID: cash[idx].InvoiceExternalID,
			Amount:    cash[idx].AppliedAmount,
		})
	}
	batch, err := xeropay.CreateBatchPayment(
		ctx,
		client,
		write.createKey(),
		xeropay.BatchPaymentInput{
			AccountID: doc.DepositAccountExternalID,
			Date:      date,
			Reference: paymentRef,
			Payments:  lines,
		},
	)
	if err != nil {
		return write.result, err
	}
	finishBatch(write.result, batch)
	return write.result, nil
}

func cashApplications(
	doc *accountingconnectorports.AccountingPaymentDocument,
) ([]*accountingconnectorports.AccountingPaymentApplication, error) {
	cash := make([]*accountingconnectorports.AccountingPaymentApplication, 0, len(doc.Applications))
	applied := decimal.Zero
	for idx := range doc.Applications {
		app := &doc.Applications[idx]
		if app.AppliedAmount.IsNegative() || app.ShortPayAmount.IsNegative() {
			return nil, xeroerrors.ErrAmountNotPositive
		}
		if app.AppliedAmount.IsPositive() {
			cash = append(cash, app)
			applied = applied.Add(app.AppliedAmount)
		}
	}
	if len(cash) == 0 {
		return nil, &accountingsync.SyncError{
			Category: accountingsync.SyncErrorValidation,
			Code:     UnappliedPaymentCode,
			Message: "The payment applies no cash to an invoice, so " + xerosession.ProviderName +
				" has nothing to record",
			Resolution: unappliedPaymentResolved,
		}
	}
	if !xerodocsync.SameMoney(applied, doc.TotalAmount) {
		return nil, &accountingsync.SyncError{
			Category: accountingsync.SyncErrorValidation,
			Code:     UnappliedPaymentCode,
			Message: xerosession.ProviderName + " records a payment only against invoices, " +
				"so a payment with an unapplied amount cannot be sent",
			Resolution: unappliedPaymentResolved,
		}
	}
	return cash, nil
}

func replacePrevious(ctx context.Context, client *xero.Client, write *paymentWrite) error {
	previous := strings.TrimSpace(write.externalID)
	if previous == "" || write.result.Refs[xerodocsync.RefReplaced] == previous {
		return nil
	}
	if err := deletePaymentDocument(
		ctx,
		client,
		accountingsync.SyncStepRequestID(write.requestID, xerodocsync.StepReplaceDelete),
		previous,
		write.refs,
	); err != nil {
		return err
	}
	write.result.Refs[xerodocsync.RefReplaced] = previous
	return nil
}

func deletePaymentDocument(
	ctx context.Context,
	client *xero.Client,
	key, externalID string,
	refs map[string]string,
) error {
	var err error
	if refs[accountingsync.ExternalRefDocumentType] == xerodocsync.DocTypeBatchPayment {
		err = xeropay.DeleteBatchPayment(ctx, client, key, externalID)
	} else {
		err = xeropay.DeletePayment(ctx, client, key, externalID)
	}
	if err != nil && !xeroerrors.IsNotFound(err) {
		return err
	}
	return nil
}

func clearPaymentRefs(refs map[string]string) {
	for key := range refs {
		if strings.HasPrefix(key, xerodocsync.RefPaymentPrefix) {
			refs[key] = ""
		}
	}
}

func finishBatch(
	result *accountingconnectorports.AccountingDocumentResult,
	batch *xeropay.BatchPayment,
) {
	result.ExternalID = batch.BatchPaymentID
	result.DocNumber = batch.Reference
	result.Refs[accountingsync.ExternalRefDocument] = batch.BatchPaymentID
	result.Refs[accountingsync.ExternalRefDocumentType] = xerodocsync.DocTypeBatchPayment
	result.Refs[xerodocsync.RefReplaced] = ""
	for idx := range batch.Payments {
		item := &batch.Payments[idx]
		if item.InvoiceID != "" && item.PaymentID != "" {
			result.Refs[xerodocsync.RefPaymentPrefix+strings.ToLower(item.InvoiceID)] = item.PaymentID
		}
	}
}

func shortPayRefKey(invoiceID string) string {
	return accountingsync.ExternalRefShortPayPrefix + invoiceID
}

func (c *Payments) shortPay(
	ctx context.Context,
	client *xero.Client,
	doc *accountingconnectorports.AccountingPaymentDocument,
	idx int,
	date time.Time,
	result *accountingconnectorports.AccountingDocumentResult,
) error {
	app := &doc.Applications[idx]
	if strings.TrimSpace(app.ShortPayCreditExternalID) != "" {
		return nil
	}
	key := shortPayRefKey(app.InvoiceExternalID)
	allocationKey := RefShortPayAllocPrefix + app.InvoiceExternalID
	noteID := result.Refs[key]

	if noteID == "" {
		if strings.TrimSpace(doc.ShortPayAccountExternalID) == "" {
			return &accountingsync.SyncError{
				Category:   accountingsync.SyncErrorMapping,
				Code:       accountingsync.ItemRoleShortPayWriteOff,
				Message:    "The short pay write-off account is not mapped",
				Resolution: "Map the short pay write-off to a " + xerosession.ProviderName + " account",
			}
		}
		code, err := c.AccountCode(
			ctx,
			client,
			doc.ShortPayAccountExternalID,
			"The short pay write-off",
		)
		if err != nil {
			return err
		}
		text := shortPayNote + app.InvoiceNumber
		note, err := xerodocs.CreateCreditNote(
			ctx,
			client,
			accountingsync.SyncStepRequestID(doc.RequestID, stepShortPayBase+idx*stepsPerShortPay),
			xerodocs.CreditNoteInput{
				Type:            xerodocs.CreditNoteTypeReceivable,
				ContactID:       doc.CustomerExternalID,
				Reference:       xerodocsync.Reference(text),
				Date:            date,
				CurrencyCode:    doc.CurrencyCode,
				CurrencyRate:    doc.ExchangeRate,
				Status:          xerodocs.StatusAuthorised,
				LineAmountTypes: xerodocs.LineAmountsNoTax,
				Lines: []xerodocs.LineItem{xerodocsync.LineItem(&xerodocsync.LineSpec{
					Description: text,
					Amount:      app.ShortPayAmount,
					AccountCode: code,
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

	allocation, err := xerodocs.Allocate(
		ctx,
		client,
		accountingsync.SyncStepRequestID(doc.RequestID, stepShortPayBase+idx*stepsPerShortPay+1),
		noteID,
		xerodocs.AllocationInput{
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

func (c *Payments) VoidPayment(
	ctx context.Context,
	ref *accountingconnectorports.AccountingDocumentRef,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	client, err := c.Client(ref.Auth)
	if err != nil {
		return nil, err
	}
	result := xerodocsync.NewResult(ref.Refs)
	result.ExternalID = ref.ExternalID

	if err = deletePaymentDocument(
		ctx,
		client,
		ref.RequestID,
		ref.ExternalID,
		ref.Refs,
	); err != nil {
		return result, err
	}

	step := 0
	for _, key := range slices.Sorted(maps.Keys(result.Refs)) {
		if !strings.HasPrefix(key, accountingsync.ExternalRefShortPayPrefix) ||
			result.Refs[key] == "" {
			continue
		}
		step++
		if err = xerodocsync.VoidCreditNote(
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
