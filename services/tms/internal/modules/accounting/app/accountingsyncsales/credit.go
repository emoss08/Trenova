package accountingsyncsales

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncpush"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (p *Pusher) PushCreditApplication(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsyncpush.Result, error) {
	app, err := p.payments.GetCreditMemoApplicationByID(
		ctx,
		customerpaymentports.GetCreditMemoApplicationRequest{
			ID:         record.ObjectID,
			TenantInfo: sess.Tenant,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, accountingsyncpush.Blocked(accountingsync.SyncErrorNotFound,
				"The credit application no longer exists in Trenova", "Skip this record")
		}
		return nil, err
	}
	invoices, err := p.invoices.GetByIDs(ctx, invoiceports.GetInvoicesByIDsRequest{
		TenantInfo: sess.Tenant,
		InvoiceIDs: []pulid.ID{app.InvoiceID, app.CreditMemoInvoiceID},
	})
	if err != nil {
		return nil, err
	}
	var target, memo *invoice.Invoice
	for _, inv := range invoices {
		switch inv.ID {
		case app.InvoiceID:
			target = inv
		case app.CreditMemoInvoiceID:
			memo = inv
		}
	}
	if target == nil || memo == nil {
		return nil, accountingsyncpush.Blocked(accountingsync.SyncErrorNotFound,
			"A document this credit application links no longer exists", "Skip this record")
	}
	label := accountingsynceffects.DocumentLabel(
		accountingsync.SyncObjectCreditApplication,
		memo.Number,
	)
	if err = sess.CheckBooks(
		memo.CurrencyCode,
		record.SentDate(app.AccountingDate),
		label,
	); err != nil {
		return nil, err
	}

	res := p.kit.NewResolver(sess)
	customerID, err := p.customerRef(ctx, sess, res, target.CustomerID)
	if err != nil {
		return nil, err
	}
	invoiceID, err := p.salesRef(ctx, sess, target)
	if err != nil {
		return nil, err
	}
	memoID, err := p.salesRef(ctx, sess, memo)
	if err != nil {
		return nil, err
	}
	rate, err := p.kit.ExchangeRate(ctx, sess, &accountingsyncpush.DocumentRate{
		Label:          label,
		Currency:       memo.CurrencyCode,
		Stamp:          memo.ExchangeRate,
		StampDate:      memo.ExchangeRateDate,
		DocumentDate:   memo.InvoiceDate,
		AccountingDate: app.AccountingDate,
		Persist:        p.stampInvoice(sess, memo.ID),
	})
	if err != nil {
		return nil, err
	}

	doc := &accountingconnectorports.AccountingCreditApplicationDocument{
		Auth:               sess.Auth,
		RequestID:          record.RequestID,
		CustomerExternalID: customerID,
		TxnDate: timeutils.FormatCalendarDate(
			record.SentDate(app.AccountingDate),
			sess.Loc,
		),
		CurrencyCode:         memo.CurrencyCode,
		ExchangeRate:         rate,
		InvoiceExternalID:    invoiceID,
		CreditMemoExternalID: memoID,
		Amount:               money.DecimalFromMinor(app.AppliedAmountMinor),
		PrivateNote: "Applies " + memo.Number + " to " + target.Number +
			accountingsyncpush.SentDateNote(record, app.AccountingDate, sess.Loc),
	}
	written, err := p.kit.CreateAdopting(
		ctx,
		sess,
		record,
		&accountingconnectorports.AccountingFindDocumentRequest{
			CounterpartyExternalID: customerID,
			TxnDate:                doc.TxnDate,
			Total:                  doc.Amount,
			AppliesToExternalIDs:   []string{invoiceID},
			CreditExternalID:       memoID,
		},
		func() (*accountingconnectorports.AccountingDocumentResult, error) {
			return sess.Writer.CreateCreditApplication(ctx, doc)
		},
	)
	if err != nil {
		return accountingsyncpush.Partial(written), err
	}
	return accountingsyncpush.Finished(sess, record, written, doc, res.MappingIDs())
}

func (p *Pusher) PushCreditApplicationVoid(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsyncpush.Result, error) {
	existing, err := p.kit.ObjectRecords(ctx, sess, record.ObjectType, record.ObjectID)
	if err != nil {
		return nil, err
	}
	created := accountingsyncpush.DocumentRecord(existing)
	if done, voidErr := p.kit.SettleUnsent(ctx, created, record); done {
		return nil, voidErr
	}
	ref := &accountingconnectorports.AccountingDocumentRef{
		Auth:       sess.Auth,
		RequestID:  record.RequestID,
		Kind:       record.ObjectType,
		ExternalID: created.ExternalID,
		Refs:       accountingsyncpush.MergedRefs(existing),
	}
	written, err := sess.Writer.VoidCreditApplication(ctx, ref)
	if err != nil {
		return accountingsyncpush.Partial(written), err
	}
	return accountingsyncpush.Finished(sess, record, written, ref, nil)
}
