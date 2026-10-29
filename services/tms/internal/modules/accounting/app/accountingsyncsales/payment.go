package accountingsyncsales

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncpush"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func appliedInvoices(
	applications []accountingconnectorports.AccountingPaymentApplication,
) []string {
	ids := make([]string, 0, len(applications))
	for idx := range applications {
		ids = append(ids, applications[idx].InvoiceExternalID)
	}
	return ids
}

func paymentMethodLabel(method customerpayment.Method) string {
	return "Payment method " + string(method)
}

func (p *Pusher) PushPayment(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsyncpush.Result, error) {
	payment, err := p.payments.GetByID(ctx, customerpaymentports.GetCustomerPaymentByIDRequest{
		ID:         record.ObjectID,
		TenantInfo: sess.Tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, accountingsyncpush.Blocked(accountingsync.SyncErrorNotFound,
				"The payment no longer exists in Trenova", "Skip this record")
		}
		return nil, err
	}
	label := accountingsynceffects.DocumentLabel(
		accountingsync.SyncObjectCustomerPayment,
		payment.ReferenceNumber,
	)
	if err = sess.CheckBooks(
		payment.CurrencyCode,
		record.SentDate(payment.AccountingDate),
		label,
	); err != nil {
		return nil, err
	}

	existing, err := p.kit.ObjectRecords(ctx, sess, record.ObjectType, record.ObjectID)
	if err != nil {
		return nil, err
	}
	externalID := ""
	if record.Operation == accountingsync.SyncOperationUpdate {
		externalID, err = p.paymentUpdateTarget(ctx, sess, &paymentUpdate{
			record:   record,
			payment:  payment,
			existing: existing,
			label:    label,
		})
		if err != nil {
			return nil, err
		}
	}

	res := p.kit.NewResolver(sess)
	refs, err := p.paymentRefs(ctx, sess, res, payment)
	if err != nil {
		return nil, err
	}
	rate, err := p.kit.ExchangeRate(ctx, sess, &accountingsyncpush.DocumentRate{
		Label:          label,
		Currency:       payment.CurrencyCode,
		Stamp:          payment.ExchangeRate,
		StampDate:      payment.ExchangeRateDate,
		DocumentDate:   payment.PaymentDate,
		AccountingDate: payment.AccountingDate,
		Persist:        p.stampPayment(sess, payment.ID),
	})
	if err != nil {
		return nil, err
	}

	doc := &accountingconnectorports.AccountingPaymentDocument{
		Auth:               sess.Auth,
		RequestID:          record.RequestID,
		ExternalID:         externalID,
		CustomerExternalID: refs.customer,
		TxnDate: timeutils.FormatCalendarDate(
			record.SentDate(payment.AccountingDate),
			sess.Loc,
		),
		CurrencyCode:             payment.CurrencyCode,
		ExchangeRate:             rate,
		PaymentMethodExternalID:  refs.method,
		DepositAccountExternalID: refs.deposit,
		ReferenceNumber:          payment.ReferenceNumber,
		PrivateNote: paymentNote(payment) +
			accountingsyncpush.SentDateNote(record, payment.AccountingDate, sess.Loc),
		TotalAmount: money.DecimalFromMinor(payment.AmountMinor),
		Applications: make(
			[]accountingconnectorports.AccountingPaymentApplication,
			0,
			len(payment.Applications),
		),
		Refs: record.ExternalRefs,
	}
	if err = p.paymentApplications(ctx, sess, res, payment, existing, doc); err != nil {
		return nil, err
	}

	written, err := p.savePayment(ctx, sess, record, doc)
	if err != nil {
		return accountingsyncpush.Partial(written), err
	}
	return accountingsyncpush.Finished(sess, record, written, doc, res.MappingIDs())
}

type paymentUpdate struct {
	record   *accountingsync.AccountingSyncRecord
	payment  *customerpayment.Payment
	existing []*accountingsync.AccountingSyncRecord
	label    string
}

func (p *Pusher) paymentUpdateTarget(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	update *paymentUpdate,
) (string, error) {
	record := update.record
	created := accountingsyncpush.DocumentRecord(update.existing)
	if created == nil {
		dependency, err := p.kit.EnqueueDependency(
			ctx,
			sess,
			&accountingsyncports.AccountingSyncEnqueueRequest{
				ObjectType:   record.ObjectType,
				ObjectID:     record.ObjectID,
				ObjectNumber: record.ObjectNumber,
				Operation:    accountingsync.SyncOperationCreate,
				Revision:     1,
				DocumentDate: update.payment.AccountingDate,
			},
		)
		if err != nil {
			return "", err
		}
		created = dependency
	}
	switch {
	case created == nil:
		return "", &accountingsyncpush.NoopError{
			Reason: "The payment was never sent, so there is nothing to update",
		}
	case accountingsyncpush.IsSynced(created) && created.SharesProviderDocument():
		return "", accountingsyncpush.SharedDocumentError(sess, update.label)
	case accountingsyncpush.IsSynced(created):
		return created.ExternalID, nil
	case created.Status.IsFinal():
		return "", &accountingsyncpush.NoopError{
			Reason: "The payment was skipped, so its changes are not sent",
		}
	default:
		p.kit.Kick(ctx, sess)
		return "", accountingsyncpush.WaitingOn(created, update.label)
	}
}

func (p *Pusher) savePayment(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
	doc *accountingconnectorports.AccountingPaymentDocument,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	save := func() (*accountingconnectorports.AccountingDocumentResult, error) {
		return sess.Writer.SavePayment(ctx, doc)
	}
	if doc.ExternalID != "" {
		return save()
	}
	return p.kit.CreateAdopting(
		ctx,
		sess,
		record,
		&accountingconnectorports.AccountingFindDocumentRequest{
			DocNumber:              doc.ReferenceNumber,
			CounterpartyExternalID: doc.CustomerExternalID,
			TxnDate:                doc.TxnDate,
			Total:                  doc.TotalAmount,
			AppliesToExternalIDs:   appliedInvoices(doc.Applications),
		},
		save,
	)
}

type paymentExternalRefs struct {
	customer string
	deposit  string
	method   string
}

func (p *Pusher) paymentRefs(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	res *accountingsyncpush.Resolver,
	payment *customerpayment.Payment,
) (paymentExternalRefs, error) {
	customerID, err := p.customerRef(ctx, sess, res, payment.CustomerID)
	if err != nil {
		return paymentExternalRefs{}, err
	}
	deposit, err := res.Require(ctx, accountingsyncpush.MappingTarget{
		TargetType: accountingsync.TargetAccountRole,
		Key:        accountingsync.AccountRoleDeposit,
		Label:      "The deposit account",
	})
	if err != nil {
		return paymentExternalRefs{}, err
	}
	refs := paymentExternalRefs{customer: customerID, deposit: deposit}
	if !sess.Profile.Offers(accountingsync.TargetPaymentMethod) {
		return refs, nil
	}
	refs.method, err = res.Require(ctx, accountingsyncpush.MappingTarget{
		TargetType: accountingsync.TargetPaymentMethod,
		Key:        string(payment.PaymentMethod),
		Label:      paymentMethodLabel(payment.PaymentMethod),
	})
	if err != nil {
		return paymentExternalRefs{}, err
	}
	return refs, nil
}

func paymentNote(payment *customerpayment.Payment) string {
	note := accountingsyncpush.NotePrefix + "payment"
	if payment.ReferenceNumber != "" {
		note += " " + payment.ReferenceNumber
	}
	if payment.Memo != "" {
		note += " · " + payment.Memo
	}
	return note
}

func (p *Pusher) paymentApplications(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	res *accountingsyncpush.Resolver,
	payment *customerpayment.Payment,
	existing []*accountingsync.AccountingSyncRecord,
	doc *accountingconnectorports.AccountingPaymentDocument,
) error {
	if len(payment.Applications) == 0 {
		return nil
	}
	ids := make([]pulid.ID, 0, len(payment.Applications))
	for _, app := range payment.Applications {
		ids = append(ids, app.InvoiceID)
	}
	invoices, err := p.invoices.GetByIDs(ctx, invoiceports.GetInvoicesByIDsRequest{
		TenantInfo: sess.Tenant,
		InvoiceIDs: ids,
	})
	if err != nil {
		return err
	}
	byID := make(map[pulid.ID]*invoice.Invoice, len(invoices))
	for _, inv := range invoices {
		byID[inv.ID] = inv
	}

	priorRefs := accountingsyncpush.MergedRefs(existing)
	shortPaid := false
	for _, app := range payment.Applications {
		inv, ok := byID[app.InvoiceID]
		if !ok {
			return accountingsyncpush.Blocked(accountingsync.SyncErrorNotFound,
				"An invoice this payment applies to no longer exists", "Skip this record")
		}
		externalID, refErr := p.salesRef(ctx, sess, inv)
		if refErr != nil {
			return refErr
		}
		entry := accountingconnectorports.AccountingPaymentApplication{
			InvoiceExternalID: externalID,
			InvoiceNumber:     inv.Number,
			AppliedAmount:     money.DecimalFromMinor(app.AppliedAmountMinor),
			ShortPayAmount:    money.DecimalFromMinor(app.ShortPayAmountMinor),
			ShortPayCreditExternalID: priorRefs[accountingsync.ExternalRefShortPayPrefix+
				externalID],
		}
		shortPaid = shortPaid || app.ShortPayAmountMinor > 0
		doc.Applications = append(doc.Applications, entry)
	}

	if !shortPaid {
		return nil
	}
	kind := sess.Profile.KindFor(accountingsync.TargetItemRole)
	label := "The short-pay write-off item"
	if kind == accountingprovider.ReferenceKindAccount {
		label = "The short-pay write-off account"
	}
	writeOff, err := res.Require(ctx, accountingsyncpush.MappingTarget{
		TargetType: accountingsync.TargetItemRole,
		Key:        accountingsync.ItemRoleShortPayWriteOff,
		Label:      label,
	})
	if err != nil {
		return err
	}
	if kind == accountingprovider.ReferenceKindAccount {
		doc.ShortPayAccountExternalID = writeOff
		return nil
	}
	doc.ShortPayItemExternalID = writeOff
	return nil
}

func (p *Pusher) PushPaymentVoid(
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
	if created.SharesProviderDocument() {
		return nil, accountingsyncpush.SharedDocumentError(
			sess,
			accountingsynceffects.DocumentLabel(record.ObjectType, record.ObjectNumber),
		)
	}
	for _, other := range existing {
		if other.Operation == accountingsync.SyncOperationUpdate &&
			other.Status == accountingsync.SyncStatusInFlight {
			return nil, accountingsyncpush.WaitFor(
				other.ID,
				"Waiting for a change to the payment to finish sending",
			)
		}
	}
	if err = p.kit.WithdrawUpdates(ctx, record); err != nil {
		return nil, err
	}

	ref := &accountingconnectorports.AccountingDocumentRef{
		Auth:       sess.Auth,
		RequestID:  record.RequestID,
		Kind:       record.ObjectType,
		ExternalID: created.ExternalID,
		Refs:       accountingsyncpush.MergedRefs(existing),
	}
	written, err := sess.Writer.VoidPayment(ctx, ref)
	if err != nil {
		return accountingsyncpush.Partial(written), err
	}
	return accountingsyncpush.Finished(sess, record, written, ref, nil)
}
