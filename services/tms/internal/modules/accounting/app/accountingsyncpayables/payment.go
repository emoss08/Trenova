package accountingsyncpayables

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncpush"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingpayablesports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/timeutils"
)

func fitDocNumber(number string, limit int) string {
	number = strings.TrimSpace(number)
	if runes := []rune(number); limit > 0 && len(runes) > limit {
		return string(runes[:limit])
	}
	return number
}

func (p *Pusher) PushBillPayment(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsyncpush.Result, error) {
	settlement, err := p.payableSettlement(ctx, sess, record)
	if err != nil {
		return nil, err
	}
	billType := record.ObjectType.BillOf()
	label := accountingsynceffects.DocumentLabel(record.ObjectType, settlement.Number)
	switch {
	case settlement.PaidAt == nil:
		return nil, accountingsyncpush.Blocked(
			accountingsync.SyncErrorValidation,
			accountingsynceffects.DocumentLabel(billType, settlement.Number)+" is not marked paid",
			"Skip this record",
		)
	case settlement.NetMinor < 0:
		return nil, &accountingsyncpush.NoopError{
			Reason: accountingsynceffects.DocumentLabel(billType, settlement.Number) +
				" is owed to you, so it was sent as a vendor credit; record the refund in " +
				sess.ProviderName + " against that credit",
		}
	case settlement.NetMinor == 0:
		return nil, &accountingsyncpush.NoopError{
			Reason: label + " is for nothing, so there is nothing to send",
		}
	}
	if err = sess.CheckBooks(
		settlement.CurrencyCode,
		record.SentDate(*settlement.PaidAt),
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

	billID, err := p.billRef(ctx, sess, billType, settlement)
	if err != nil {
		return nil, err
	}

	res := p.kit.NewResolver(sess)
	vendorID, err := p.kit.PartyRef(
		ctx,
		sess,
		res,
		record.ObjectType.VendorOf(),
		settlement.PartyID,
	)
	if err != nil {
		return nil, err
	}
	bank, err := res.Account(ctx, payableAccount(
		settlement.BankAccountID,
		settlement,
		label+" has no cash account to pay from",
	))
	if err != nil {
		return nil, err
	}

	rate, err := p.kit.ExchangeRate(ctx, sess, &accountingsyncpush.DocumentRate{
		Label:          label,
		Currency:       settlement.CurrencyCode,
		Stamp:          settlement.PaidExchangeRate,
		StampDate:      settlement.PaidExchangeRateDate,
		DocumentDate:   *settlement.PaidAt,
		AccountingDate: *settlement.PaidAt,
		Persist:        p.stampPayable(sess, settlement, true),
	})
	if err != nil {
		return nil, err
	}

	doc := &accountingconnectorports.AccountingBillPaymentDocument{
		Auth:                  sess.Auth,
		RequestID:             record.RequestID,
		Kind:                  record.ObjectType,
		VendorExternalID:      vendorID,
		BankAccountExternalID: bank,
		BillExternalID:        billID,
		DocNumber: fitDocNumber(
			settlement.PaymentReference,
			sess.Limits.MaxDocNumberLength,
		),
		TxnDate:      timeutils.FormatCalendarDate(record.SentDate(*settlement.PaidAt), sess.Loc),
		CurrencyCode: settlement.CurrencyCode,
		ExchangeRate: rate,
		PrivateNote: billPaymentNote(record.ObjectType, settlement) +
			accountingsyncpush.SentDateNote(record, *settlement.PaidAt, sess.Loc),
		Amount: money.DecimalFromMinor(settlement.NetMinor),
	}
	var written *accountingconnectorports.AccountingDocumentResult
	if target != nil {
		doc.ExternalID = target.ExternalID
		written, err = sess.Writer.UpdateBillPayment(ctx, doc)
	} else {
		written, err = p.kit.CreateAdopting(
			ctx,
			sess,
			record,
			&accountingconnectorports.AccountingFindDocumentRequest{
				DocNumber:              doc.DocNumber,
				CounterpartyExternalID: vendorID,
				TxnDate:                doc.TxnDate,
				Total:                  doc.Amount,
				AppliesToExternalIDs:   []string{billID},
			},
			func() (*accountingconnectorports.AccountingDocumentResult, error) {
				return sess.Writer.CreateBillPayment(ctx, doc)
			},
		)
	}
	if err != nil {
		return accountingsyncpush.Partial(written), err
	}
	return accountingsyncpush.Finished(sess, record, written, doc, res.MappingIDs())
}

func (p *Pusher) billRef(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	billType accountingsync.SyncObjectType,
	settlement *accountingpayablesports.PayableSettlement,
) (string, error) {
	label := accountingsynceffects.DocumentLabel(billType, settlement.Number)
	existing, err := p.kit.ObjectRecords(ctx, sess, billType, settlement.ID)
	if err != nil {
		return "", err
	}
	created := accountingsyncpush.DocumentRecord(existing)
	if created == nil {
		if !sess.Conn.Covers(*settlement.PostedAt) {
			return "", &accountingsyncpush.NoopError{
				Reason: label + " was posted before the start date, so its payment is not sent either",
			}
		}
		created, err = p.kit.EnqueueDependency(
			ctx,
			sess,
			&accountingsyncports.AccountingSyncEnqueueRequest{
				ObjectType:   billType,
				ObjectID:     settlement.ID,
				ObjectNumber: settlement.Number,
				Operation:    accountingsync.SyncOperationCreate,
				Revision:     1,
				DocumentDate: *settlement.PostedAt,
			},
		)
		if err != nil {
			return "", err
		}
		if created == nil {
			return "", accountingsyncpush.Blocked(
				accountingsync.SyncErrorNotFound,
				label+" could not be queued",
				"Retry this record",
			)
		}
	}
	switch {
	case accountingsyncpush.IsSynced(created):
		return created.ExternalID, nil
	case created.Status.IsFinal():
		return "", &accountingsyncpush.NoopError{
			Reason: label + " was not sent, so its payment is not sent either",
		}
	default:
		p.kit.Kick(ctx, sess)
		return "", accountingsyncpush.WaitingOn(created, label)
	}
}

func billPaymentNote(
	objectType accountingsync.SyncObjectType,
	settlement *accountingpayablesports.PayableSettlement,
) string {
	kind := "carrier settlement"
	if objectType.IsDriverSettlement() {
		kind = "owner-operator settlement"
	}
	note := accountingsyncpush.NotePrefix + "payment of " + kind + " " + settlement.Number
	if method := strings.TrimSpace(settlement.PaymentMethod); method != "" {
		note += " · Paid by " + method
	}
	if reference := strings.TrimSpace(settlement.PaymentReference); reference != "" {
		note += " · Reference " + reference
	}
	return note
}
