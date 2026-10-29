package accountingsyncsales

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncpush"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

func salesTotal(lines []accountingconnectorports.AccountingDocumentLine) decimal.Decimal {
	total := decimal.Zero
	for idx := range lines {
		total = total.Add(lines[idx].Amount)
	}
	return total
}

func (p *Pusher) customerRef(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	res *accountingsyncpush.Resolver,
	customerID pulid.ID,
) (string, error) {
	return p.kit.PartyRef(ctx, sess, res, accountingsync.SyncObjectCustomer, customerID)
}

func (p *Pusher) salesRef(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	inv *invoice.Invoice,
) (string, error) {
	objectType := accountingsync.SyncObjectTypeForBill(inv.BillType)
	label := accountingsynceffects.DocumentLabel(objectType, inv.Number)
	existing, err := p.kit.ObjectRecords(ctx, sess, objectType, inv.ID)
	if err != nil {
		return "", err
	}
	if created := accountingsyncpush.DocumentRecord(existing); created != nil {
		switch {
		case accountingsyncpush.IsSynced(created):
			return created.ExternalID, nil
		case !created.Status.IsFinal():
			return "", accountingsyncpush.WaitingOn(created, label)
		}
	} else if sess.Conn.Covers(inv.InvoiceDate) && inv.PostedAt != nil {
		dependency, enqueueErr := p.kit.EnqueueDependency(
			ctx,
			sess,
			&accountingsyncports.AccountingSyncEnqueueRequest{
				ObjectType:   objectType,
				ObjectID:     inv.ID,
				ObjectNumber: inv.Number,
				Operation:    accountingsync.SyncOperationCreate,
				Revision:     1,
				DocumentDate: inv.InvoiceDate,
			},
		)
		if enqueueErr != nil {
			return "", enqueueErr
		}
		if dependency != nil && !dependency.Status.IsFinal() {
			p.kit.Kick(ctx, sess)
			return "", accountingsyncpush.WaitingOn(dependency, label)
		}
	}

	found, ok, err := sess.Writer.FindDocument(
		ctx,
		&accountingconnectorports.AccountingFindDocumentRequest{
			Auth:      sess.Auth,
			Kind:      objectType,
			DocNumber: inv.Number,
		},
	)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", accountingsyncpush.Blocked(
			accountingsync.SyncErrorNotFound,
			label+" is not in "+sess.ProviderName,
			"Enter "+label+" in "+sess.ProviderName+" with the same number, then retry; or skip this record",
		)
	}
	return found.ExternalID, nil
}

func (p *Pusher) PushSales(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsyncpush.Result, error) {
	inv, err := p.invoices.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         record.ObjectID,
		TenantInfo: sess.Tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, accountingsyncpush.Blocked(accountingsync.SyncErrorNotFound,
				"The document no longer exists in Trenova", "Skip this record")
		}
		return nil, err
	}
	label := accountingsynceffects.DocumentLabel(record.ObjectType, inv.Number)
	if inv.PostedAt == nil ||
		(inv.Status != invoice.StatusPosted && inv.Status != invoice.StatusVoided) {
		return nil, accountingsyncpush.Blocked(accountingsync.SyncErrorValidation,
			label+" is not posted", "Post it in Trenova, or skip this record")
	}
	if err = sess.CheckBooks(
		inv.CurrencyCode,
		record.SentDate(inv.InvoiceDate),
		label,
	); err != nil {
		return nil, err
	}

	var target *accountingsync.AccountingSyncRecord
	if record.Operation == accountingsync.SyncOperationUpdate {
		if target, err = p.kit.UpdateTarget(ctx, sess, record, label); err != nil {
			return nil, err
		}
	}

	res := p.kit.NewResolver(sess)
	customerID, err := p.customerRef(ctx, sess, res, inv.CustomerID)
	if err != nil {
		return nil, err
	}

	doc, err := p.salesDocument(ctx, sess, res, record, inv)
	if err != nil {
		return nil, err
	}
	doc.CustomerExternalID = customerID

	var written *accountingconnectorports.AccountingDocumentResult
	if target != nil {
		doc.ExternalID = target.ExternalID
		doc.Refs = target.ExternalRefs
		written, err = sess.Writer.UpdateSalesDocument(ctx, doc)
	} else {
		written, err = p.kit.CreateAdopting(
			ctx,
			sess,
			record,
			&accountingconnectorports.AccountingFindDocumentRequest{
				DocNumber:              doc.DocNumber,
				CounterpartyExternalID: customerID,
				TxnDate:                doc.TxnDate,
				Total:                  salesTotal(doc.Lines),
			},
			func() (*accountingconnectorports.AccountingDocumentResult, error) {
				return sess.Writer.CreateSalesDocument(ctx, doc)
			},
		)
	}
	if err != nil {
		return accountingsyncpush.Partial(written), err
	}
	return accountingsyncpush.Finished(sess, record, written, doc, res.MappingIDs())
}

func (p *Pusher) salesDocument(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	res *accountingsyncpush.Resolver,
	record *accountingsync.AccountingSyncRecord,
	inv *invoice.Invoice,
) (*accountingconnectorports.AccountingSalesDocument, error) {
	sign := decimal.NewFromInt(1)
	if record.ObjectType == accountingsync.SyncObjectCreditMemo && inv.TotalAmount.IsNegative() {
		sign = decimal.NewFromInt(-1)
	}

	postedAt := inv.InvoiceDate
	if inv.PostedAt != nil {
		postedAt = *inv.PostedAt
	}
	rate, err := p.kit.ExchangeRate(ctx, sess, &accountingsyncpush.DocumentRate{
		Label:          accountingsynceffects.DocumentLabel(record.ObjectType, inv.Number),
		Currency:       inv.CurrencyCode,
		Stamp:          inv.ExchangeRate,
		StampDate:      inv.ExchangeRateDate,
		DocumentDate:   inv.InvoiceDate,
		AccountingDate: postedAt,
		Persist:        p.stampInvoice(sess, inv.ID),
	})
	if err != nil {
		return nil, err
	}

	doc := &accountingconnectorports.AccountingSalesDocument{
		Auth:         sess.Auth,
		RequestID:    record.RequestID,
		Kind:         record.ObjectType,
		DocNumber:    inv.Number,
		TxnDate:      timeutils.FormatCalendarDate(record.SentDate(inv.InvoiceDate), sess.Loc),
		CurrencyCode: inv.CurrencyCode,
		ExchangeRate: rate,
		PrivateNote: salesNote(
			inv,
		) + accountingsyncpush.SentDateNote(
			record,
			inv.InvoiceDate,
			sess.Loc,
		),
		CustomerMemo: customerMemo(inv),
		Refs:         record.ExternalRefs,
	}
	if limit := sess.Limits.MaxDocNumberLength; limit > 0 && len([]rune(inv.Number)) > limit {
		doc.DocNumber = ""
	}
	if inv.DueDate != nil && record.ObjectType != accountingsync.SyncObjectCreditMemo {
		doc.DueDate = timeutils.FormatCalendarDate(*inv.DueDate, sess.Loc)
	}
	if doc.TermExternalID, err = salesTerm(ctx, sess, res, record, inv); err != nil {
		return nil, err
	}
	if doc.Lines, err = salesLines(ctx, sess, res, inv, sign); err != nil {
		return nil, err
	}
	if len(doc.Lines) == 0 {
		return nil, &accountingsyncpush.NoopError{
			Reason: accountingsynceffects.DocumentLabel(record.ObjectType, inv.Number) +
				" has no amounts to send",
		}
	}

	if record.ObjectType == accountingsync.SyncObjectCreditMemo {
		if err = p.applyReversal(ctx, sess, inv, doc); err != nil {
			return nil, err
		}
	}
	return doc, nil
}

func salesTerm(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	res *accountingsyncpush.Resolver,
	record *accountingsync.AccountingSyncRecord,
	inv *invoice.Invoice,
) (string, error) {
	if inv.PaymentTerm == "" || record.ObjectType == accountingsync.SyncObjectCreditMemo ||
		!sess.Profile.Offers(accountingsync.TargetPaymentTerm) {
		return "", nil
	}
	return res.Optional(ctx, accountingsyncpush.MappingTarget{
		TargetType: accountingsync.TargetPaymentTerm,
		Key:        string(inv.PaymentTerm),
	})
}

func salesLines(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	res *accountingsyncpush.Resolver,
	inv *invoice.Invoice,
	sign decimal.Decimal,
) ([]accountingconnectorports.AccountingDocumentLine, error) {
	lines := make([]accountingconnectorports.AccountingDocumentLine, 0, len(inv.Lines))
	for _, line := range inv.Lines {
		if line == nil || line.Amount.IsZero() {
			continue
		}
		target := lineTarget(line)
		externalID, err := res.Require(ctx, target)
		if err != nil {
			return nil, err
		}
		entry := accountingconnectorports.AccountingDocumentLine{
			Description: line.Description,
			Quantity:    line.Quantity.Abs(),
			UnitPrice:   line.UnitPrice.Mul(sign),
			Amount:      line.Amount.Mul(sign),
		}
		setLineRef(&entry, sess.Profile.KindFor(target.TargetType), externalID)
		if inv.ServiceDate != nil {
			entry.ServiceDate = timeutils.FormatCalendarDate(*inv.ServiceDate, sess.Loc)
		}
		lines = append(lines, entry)
	}
	return lines, nil
}

func setLineRef(
	line *accountingconnectorports.AccountingDocumentLine,
	kind accountingprovider.ReferenceKind,
	externalID string,
) {
	if kind == accountingprovider.ReferenceKindAccount {
		line.AccountExternalID = externalID
		return
	}
	line.ItemExternalID = externalID
}

func lineTarget(line *invoice.InvoiceLine) accountingsyncpush.MappingTarget {
	if !line.AccessorialChargeID.IsNil() {
		label := "Charge code " + line.ChargeCode
		if line.ChargeCode == "" {
			label = "Accessorial charge " + line.Description
		}
		return accountingsyncpush.MappingTarget{
			TargetType: accountingsync.TargetAccessorialCharge,
			ObjectID:   line.AccessorialChargeID,
			Label:      label,
		}
	}
	key := string(line.Type)
	if line.Type != invoice.InvoiceLineTypeMemo {
		key = string(invoice.InvoiceLineTypeFreight)
	}
	return accountingsyncpush.MappingTarget{
		TargetType: accountingsync.TargetLineType,
		Key:        key,
		Label:      strings.ToLower(key) + " lines",
	}
}

func salesNote(inv *invoice.Invoice) string {
	parts := []string{accountingsyncpush.NotePrefix + inv.Number}
	if inv.ShipmentProNumber != "" {
		parts = append(parts, "PRO "+inv.ShipmentProNumber)
	}
	if inv.MemoReason != "" {
		parts = append(parts, inv.MemoReason)
	}
	return strings.Join(parts, " · ")
}

func customerMemo(inv *invoice.Invoice) string {
	parts := make([]string, 0, 3)
	if inv.ShipmentProNumber != "" {
		parts = append(parts, "PRO "+inv.ShipmentProNumber)
	}
	if inv.ShipmentBOL != "" {
		parts = append(parts, "BOL "+inv.ShipmentBOL)
	}
	if inv.OrderNumber != "" {
		parts = append(parts, "Order "+inv.OrderNumber)
	}
	return strings.Join(parts, " · ")
}

func (p *Pusher) applyReversal(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	memo *invoice.Invoice,
	doc *accountingconnectorports.AccountingSalesDocument,
) error {
	if memo.SourceInvoiceAdjustmentID.IsNil() {
		return nil
	}
	adjustment, err := p.adjustments.GetByID(
		ctx,
		invoiceadjustmentports.GetInvoiceAdjustmentRequest{
			ID:         memo.SourceInvoiceAdjustmentID,
			TenantInfo: sess.Tenant,
		},
	)
	if err != nil {
		return err
	}
	if adjustment.Kind != invoiceadjustment.KindFullReversal {
		return nil
	}
	original, err := p.invoices.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         adjustment.OriginalInvoiceID,
		TenantInfo: sess.Tenant,
	})
	if err != nil {
		return err
	}

	externalID, err := p.salesRef(ctx, sess, original)
	if err != nil {
		var syncErr *accountingsync.SyncError
		if !errors.As(err, &syncErr) || syncErr.Category != accountingsync.SyncErrorNotFound {
			return err
		}
		return &accountingsyncpush.NoopError{
			Reason: accountingsynceffects.DocumentLabel(
				accountingsync.SyncObjectInvoice,
				original.Number,
			) +
				" was never in " + sess.ProviderName + ", so its reversal is not sent either",
		}
	}
	doc.ApplyToExternalID = externalID
	doc.ApplyAmount = memo.TotalAmount.Abs()
	return nil
}

func (p *Pusher) PushSalesVoid(
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
	ref := &accountingconnectorports.AccountingDocumentRef{
		Auth:       sess.Auth,
		RequestID:  record.RequestID,
		Kind:       record.ObjectType,
		ExternalID: created.ExternalID,
		Refs:       accountingsyncpush.MergedRefs(existing),
	}
	written, err := sess.Writer.VoidSalesDocument(ctx, ref)
	if err != nil {
		return accountingsyncpush.Partial(written), err
	}
	return accountingsyncpush.Finished(sess, record, written, ref, nil)
}
