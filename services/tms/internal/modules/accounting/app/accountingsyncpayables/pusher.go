package accountingsyncpayables

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncpush"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingpayablesports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Kit      *accountingsyncpush.Kit
	Payables accountingpayablesports.AccountingPayablesSource
}

type Pusher struct {
	kit      *accountingsyncpush.Kit
	payables accountingpayablesports.AccountingPayablesSource
}

func New(p Params) *Pusher {
	return &Pusher{kit: p.Kit, payables: p.Payables}
}

func purchaseTotal(lines []accountingconnectorports.AccountingPurchaseLine) decimal.Decimal {
	total := decimal.Zero
	for idx := range lines {
		total = total.Add(lines[idx].Amount)
	}
	return total
}

func payableKind(objectType accountingsync.SyncObjectType) accountingpayablesports.PayableKind {
	if objectType.IsDriverSettlement() {
		return accountingpayablesports.PayableDriver
	}
	return accountingpayablesports.PayableCarrier
}

func (p *Pusher) payableSettlement(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
) (*accountingpayablesports.PayableSettlement, error) {
	settlement, err := p.payables.GetSettlement(
		ctx,
		&accountingpayablesports.GetPayableSettlementRequest{
			TenantInfo: sess.Tenant,
			Kind:       payableKind(record.ObjectType),
			ID:         record.ObjectID,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, accountingsyncpush.Blocked(
				accountingsync.SyncErrorNotFound,
				"The settlement no longer exists in Trenova",
				"Skip this record",
			)
		}
		return nil, err
	}
	if settlement.Kind == accountingpayablesports.PayableDriver && !settlement.OwnerOperator {
		return nil, &accountingsyncpush.NoopError{
			Reason: "Company driver pay belongs to the payroll system, so it is not sent",
		}
	}
	if settlement.PostedAt == nil {
		return nil, accountingsyncpush.Blocked(
			accountingsync.SyncErrorValidation,
			accountingsynceffects.DocumentLabel(
				record.ObjectType,
				settlement.Number,
			)+" was never posted",
			"Skip this record",
		)
	}
	return settlement, nil
}

func payableAccount(
	accountID pulid.ID,
	settlement *accountingpayablesports.PayableSettlement,
	missing string,
) *accountingsyncpush.AccountRefRequest {
	return &accountingsyncpush.AccountRefRequest{
		AccountID: accountID,
		Label:     accountLabel(accountID, settlement.Lines),
		Role:      roleForAccount(accountID, settlement.Defaults),
		Missing:   missing,
	}
}

func roleForAccount(
	accountID pulid.ID,
	defaults accountingpayablesports.PayableDefaultAccounts,
) string {
	switch accountID {
	case defaults.Payable:
		return accountingsync.AccountRoleAP
	case defaults.PurchasedTransportation:
		return accountingsync.AccountRolePurchasedTransportation
	case defaults.Cash:
		return accountingsync.AccountRoleDeposit
	default:
		return ""
	}
}

func accountLabel(accountID pulid.ID, lines []accountingpayablesports.PayableJournalLine) string {
	for idx := range lines {
		if lines[idx].AccountID != accountID {
			continue
		}
		if label := accountingsyncpush.GLAccountLabel(
			lines[idx].AccountCode,
			lines[idx].AccountName,
		); label != "" {
			return label
		}
	}
	return ""
}

func (p *Pusher) PushBill(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsyncpush.Result, error) {
	settlement, err := p.payableSettlement(ctx, sess, record)
	if err != nil {
		return nil, err
	}
	label := accountingsynceffects.DocumentLabel(record.ObjectType, settlement.Number)
	if err = checkBillReady(sess, record, settlement, label); err != nil {
		return nil, err
	}
	var target *accountingsync.AccountingSyncRecord
	if record.Operation == accountingsync.SyncOperationUpdate {
		if target, err = p.kit.UpdateTarget(ctx, sess, record, label); err != nil {
			return nil, err
		}
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
	apAccount, err := res.Account(ctx, payableAccount(
		settlement.PayableAccountID,
		settlement,
		label+" has no payable account",
	))
	if err != nil {
		return nil, err
	}

	lines, err := p.billLines(ctx, res, settlement)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, &accountingsyncpush.NoopError{
			Reason: label + " has no amounts, so there is nothing to send",
		}
	}
	credit := creditLines(lines)

	rate, err := p.kit.ExchangeRate(ctx, sess, &accountingsyncpush.DocumentRate{
		Label:          label,
		Currency:       settlement.CurrencyCode,
		Stamp:          settlement.ExchangeRate,
		StampDate:      settlement.ExchangeRateDate,
		DocumentDate:   *settlement.PostedAt,
		AccountingDate: *settlement.PostedAt,
		Persist:        p.stampPayable(sess, settlement, false),
	})
	if err != nil {
		return nil, err
	}

	doc := &accountingconnectorports.AccountingPurchaseDocument{
		Auth:                sess.Auth,
		RequestID:           record.RequestID,
		Kind:                record.ObjectType,
		VendorCredit:        credit,
		VendorExternalID:    vendorID,
		APAccountExternalID: apAccount,
		DocNumber:           billNumber(settlement),
		TxnDate: timeutils.FormatCalendarDate(
			record.SentDate(*settlement.PostedAt),
			sess.Loc,
		),
		CurrencyCode: settlement.CurrencyCode,
		ExchangeRate: rate,
		PrivateNote: billNote(record.ObjectType, settlement, sess.Loc) +
			accountingsyncpush.SentDateNote(record, *settlement.PostedAt, sess.Loc),
		Lines: lines,
	}
	if !credit {
		doc.DueDate = timeutils.FormatCalendarDate(settlement.PayDate, sess.Loc)
	}
	if limit := sess.Limits.MaxDocNumberLength; limit > 0 && len([]rune(doc.DocNumber)) > limit {
		doc.DocNumber = ""
	}

	written, err := p.sendBill(ctx, sess, record, target, doc, label)
	if err != nil {
		return accountingsyncpush.Partial(written), err
	}
	return accountingsyncpush.WithProviderURL(
		accountingsyncpush.Finished(sess, record, written, doc, res.MappingIDs()),
	)
}

func checkBillReady(
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
	settlement *accountingpayablesports.PayableSettlement,
	label string,
) error {
	if len(settlement.Lines) == 0 {
		if settlement.NetMinor == 0 {
			return &accountingsyncpush.NoopError{
				Reason: label + " has no amounts, so there is nothing to send",
			}
		}
		return accountingsyncpush.Blocked(
			accountingsync.SyncErrorValidation,
			label+" has no journal entry to send",
			"Check the settlement's posting in the general ledger, then retry; or skip this record",
		)
	}
	return sess.CheckBooks(
		settlement.CurrencyCode,
		record.SentDate(*settlement.PostedAt),
		label,
	)
}

func creditLines(lines []accountingconnectorports.AccountingPurchaseLine) bool {
	total := decimal.Zero
	for idx := range lines {
		total = total.Add(lines[idx].Amount)
	}
	credit := total.IsNegative()
	if credit {
		for idx := range lines {
			lines[idx].Amount = lines[idx].Amount.Neg()
		}
	}
	return credit
}

func (p *Pusher) sendBill(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
	target *accountingsync.AccountingSyncRecord,
	doc *accountingconnectorports.AccountingPurchaseDocument,
	label string,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if target != nil {
		return writeBill(ctx, sess, target, doc, label)
	}
	return p.kit.CreateAdopting(
		ctx,
		sess,
		record,
		&accountingconnectorports.AccountingFindDocumentRequest{
			DocNumber:              doc.DocNumber,
			CounterpartyExternalID: doc.VendorExternalID,
			TxnDate:                doc.TxnDate,
			Total:                  purchaseTotal(doc.Lines),
			Credit:                 doc.VendorCredit,
		},
		func() (*accountingconnectorports.AccountingDocumentResult, error) {
			return sess.Writer.CreatePurchaseDocument(ctx, doc)
		},
	)
}

func (p *Pusher) billLines(
	ctx context.Context,
	res *accountingsyncpush.Resolver,
	settlement *accountingpayablesports.PayableSettlement,
) ([]accountingconnectorports.AccountingPurchaseLine, error) {
	lines := make([]accountingconnectorports.AccountingPurchaseLine, 0, len(settlement.Lines))
	for idx := range settlement.Lines {
		line := &settlement.Lines[idx]
		if line.AccountID == settlement.PayableAccountID || line.NetMinor() == 0 {
			continue
		}
		account, err := res.Account(ctx, payableAccount(line.AccountID, settlement, ""))
		if err != nil {
			return nil, err
		}
		lines = append(lines, accountingconnectorports.AccountingPurchaseLine{
			Description:       strings.TrimSpace(line.AccountName),
			AccountExternalID: account,
			Amount:            money.DecimalFromMinor(line.NetMinor()),
		})
	}
	return lines, nil
}

func billNumber(settlement *accountingpayablesports.PayableSettlement) string {
	if len(settlement.InvoiceNumbers) == 1 {
		return settlement.InvoiceNumbers[0]
	}
	return settlement.Number
}

func billNote(
	objectType accountingsync.SyncObjectType,
	settlement *accountingpayablesports.PayableSettlement,
	loc *time.Location,
) string {
	kind := "carrier settlement"
	if objectType.IsDriverSettlement() {
		kind = "owner-operator settlement"
	}
	parts := []string{accountingsyncpush.NotePrefix + kind + " " + settlement.Number}
	if settlement.PeriodStart > 0 && settlement.PeriodEnd > 0 {
		parts = append(parts, "Period "+
			timeutils.FormatCalendarDate(settlement.PeriodStart, loc)+" to "+
			timeutils.FormatCalendarDate(settlement.PeriodEnd, loc))
	}
	switch {
	case settlement.ShipmentCount == 1:
		parts = append(parts, "1 shipment")
	case settlement.ShipmentCount > 1:
		parts = append(parts, strconv.Itoa(settlement.ShipmentCount)+" shipments")
	}
	if len(settlement.InvoiceNumbers) > 0 {
		parts = append(parts, "Carrier invoices "+strings.Join(settlement.InvoiceNumbers, ", "))
	}
	return strings.Join(parts, " · ")
}

func (p *Pusher) PushBillVoid(
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
	written, err := sess.Writer.VoidPurchaseDocument(ctx, ref)
	if err != nil {
		return accountingsyncpush.Partial(written), err
	}
	return accountingsyncpush.Finished(sess, record, written, ref, nil)
}

func sentAsCredit(record *accountingsync.AccountingSyncRecord) bool {
	return accountingsync.IsCreditDocument(record.ExternalRefs)
}

func writeBill(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	target *accountingsync.AccountingSyncRecord,
	doc *accountingconnectorports.AccountingPurchaseDocument,
	label string,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if sentAsCredit(target) != doc.VendorCredit {
		return nil, accountingsyncpush.Blocked(
			accountingsync.SyncErrorValidation,
			label+" now nets to the other side of zero, so "+sess.ProviderName+
				" holds it as the wrong kind of document",
			"Send it again as a new document, then delete the old one in "+sess.ProviderName,
		)
	}
	doc.ExternalID = target.ExternalID
	return sess.Writer.UpdatePurchaseDocument(ctx, doc)
}

func (p *Pusher) stampPayable(
	sess *accountingsyncpush.Session,
	settlement *accountingpayablesports.PayableSettlement,
	paid bool,
) func(context.Context, decimal.Decimal, int64) error {
	return func(ctx context.Context, rate decimal.Decimal, date int64) error {
		return p.payables.StampExchangeRate(
			ctx,
			&accountingpayablesports.StampPayableExchangeRateRequest{
				TenantInfo: sess.Tenant,
				Kind:       settlement.Kind,
				ID:         settlement.ID,
				Paid:       paid,
				Rate:       rate,
				Date:       date,
			},
		)
	}
}
