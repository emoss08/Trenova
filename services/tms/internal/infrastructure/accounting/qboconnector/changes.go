package qboconnector

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/quickbooks"
)

var _ services.AccountingChangeReader = (*Connector)(nil)

const (
	cursorSteady    = "t"
	cursorPaging    = "q"
	cursorSeparator = ":"
	entitySeparator = "|"
	changeOverlap   = 2 * time.Minute
	lookbackMargin  = time.Hour
)

var errChangeCursor = errors.New("quickbooks: the change cursor is not one this adapter wrote")

type changeCursor struct {
	since   time.Time
	paging  bool
	started time.Time
	targets []quickbooks.ChangeEntity
	index   int
	start   int
}

func (c changeCursor) String() string {
	if !c.paging {
		return cursorSteady + cursorSeparator + strconv.FormatInt(c.since.Unix(), 10)
	}
	names := make([]string, 0, len(c.targets))
	for _, target := range c.targets {
		names = append(names, string(target))
	}
	return strings.Join([]string{
		cursorPaging,
		strconv.FormatInt(c.since.Unix(), 10),
		strconv.FormatInt(c.started.Unix(), 10),
		strings.Join(names, entitySeparator),
		strconv.Itoa(c.index),
		strconv.Itoa(c.start),
	}, cursorSeparator)
}

func parseChangeCursor(raw string, now time.Time) (changeCursor, error) {
	if strings.TrimSpace(raw) == "" {
		return changeCursor{since: now}, nil
	}
	parts := strings.Split(raw, cursorSeparator)
	switch {
	case parts[0] == cursorSteady && len(parts) == 2:
		since, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return changeCursor{}, errChangeCursor
		}
		return changeCursor{since: time.Unix(since, 0).UTC()}, nil
	case parts[0] == cursorPaging && len(parts) == 6:
		since, sinceErr := strconv.ParseInt(parts[1], 10, 64)
		started, startedErr := strconv.ParseInt(parts[2], 10, 64)
		index, indexErr := strconv.Atoi(parts[4])
		start, startErr := strconv.Atoi(parts[5])
		if err := errors.Join(sinceErr, startedErr, indexErr, startErr); err != nil {
			return changeCursor{}, errChangeCursor
		}
		targets := make([]quickbooks.ChangeEntity, 0, 8)
		for name := range strings.SplitSeq(parts[3], entitySeparator) {
			entity := quickbooks.ChangeEntity(name)
			if !entity.IsValid() {
				return changeCursor{}, errChangeCursor
			}
			targets = append(targets, entity)
		}
		if index < 0 || index >= len(targets) || start < 1 {
			return changeCursor{}, errChangeCursor
		}
		return changeCursor{
			since:   time.Unix(since, 0).UTC(),
			paging:  true,
			started: time.Unix(started, 0).UTC(),
			targets: targets,
			index:   index,
			start:   start,
		}, nil
	default:
		return changeCursor{}, errChangeCursor
	}
}

func (c *Connector) ChangeFeedLimits() services.AccountingChangeFeedLimits {
	return services.AccountingChangeFeedLimits{
		MaxLookback:    quickbooks.MaxChangeLookback,
		ReportsDeletes: true,
		MaxPerRead:     quickbooks.MaxChangesPerCall,
	}
}

func (c *Connector) ChangeCursorAt(at time.Time) string {
	return changeCursor{since: at.UTC()}.String()
}

func changeTargets(req *services.ReadAccountingChangesRequest) ([]quickbooks.ChangeEntity, error) {
	targets := make([]quickbooks.ChangeEntity, 0, len(req.ReferenceKinds)+2)
	if req.Payments {
		targets = append(targets, quickbooks.ChangePayment)
	}
	if req.BillPayments {
		targets = append(targets, quickbooks.ChangeBillPayment)
	}
	for _, kind := range req.ReferenceKinds {
		providerRef, err := providerKind(kind)
		if err != nil {
			return nil, err
		}
		targets = append(targets, quickbooks.ReferenceChangeEntity(providerRef))
	}
	return targets, nil
}

func (c *Connector) ReadChanges(
	ctx context.Context,
	req *services.ReadAccountingChangesRequest,
) (*services.AccountingChangePage, error) {
	now := req.Now.UTC()
	cursor, err := parseChangeCursor(req.Cursor, now)
	if err != nil {
		return nil, err
	}
	targets, err := changeTargets(req)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return &services.AccountingChangePage{NextCursor: cursor.String()}, nil
	}
	client, err := c.client(req.Auth)
	if err != nil {
		return nil, err
	}

	if cursor.paging {
		return c.readPage(ctx, client, &cursor, false)
	}

	if now.Sub(cursor.since) > quickbooks.MaxChangeLookback-lookbackMargin {
		return c.readPage(ctx, client, &changeCursor{
			since:   cursor.since,
			paging:  true,
			started: now,
			targets: targets,
			start:   1,
		}, true)
	}

	set, err := client.ChangesSince(ctx, targets, cursor.since, now)
	if err != nil {
		return nil, err
	}
	page := pageOf(set, fullEntities(set.Full))
	if len(set.Full) == 0 {
		page.NextCursor = changeCursor{since: now.Add(-changeOverlap)}.String()
		return page, nil
	}
	page.More = true
	page.NextCursor = changeCursor{
		since:   cursor.since,
		paging:  true,
		started: now,
		targets: set.Full,
		start:   1,
	}.String()
	return page, nil
}

func (c *Connector) readPage(
	ctx context.Context,
	client *quickbooks.Client,
	cursor *changeCursor,
	expired bool,
) (*services.AccountingChangePage, error) {
	set, next, err := client.QueryChangedSince(
		ctx,
		cursor.targets[cursor.index],
		cursor.since,
		cursor.start,
		quickbooks.MaxQueryResults,
	)
	if err != nil {
		return nil, err
	}
	page := pageOf(set, nil)
	page.CursorExpired = expired

	switch {
	case next > 0:
		cursor.start = next
		page.More = true
		page.NextCursor = cursor.String()
	case cursor.index+1 < len(cursor.targets):
		cursor.index++
		cursor.start = 1
		page.More = true
		page.NextCursor = cursor.String()
	default:
		page.NextCursor = changeCursor{since: cursor.started.Add(-changeOverlap)}.String()
	}
	return page, nil
}

func fullEntities(full []quickbooks.ChangeEntity) map[quickbooks.ChangeEntity]bool {
	out := make(map[quickbooks.ChangeEntity]bool, len(full))
	for _, entity := range full {
		out[entity] = true
	}
	return out
}

func pageOf(
	set *quickbooks.ChangeSet,
	skip map[quickbooks.ChangeEntity]bool,
) *services.AccountingChangePage {
	page := &services.AccountingChangePage{
		Payments: make(
			[]services.AccountingInboundPayment,
			0,
			len(set.Payments)+len(set.BillPayments),
		),
		References: make([]services.AccountingChangedReference, 0, len(set.References)),
	}
	if !skip[quickbooks.ChangePayment] {
		for idx := range set.Payments {
			page.Payments = append(page.Payments, inboundPaymentOf(&set.Payments[idx]))
		}
	}
	if !skip[quickbooks.ChangeBillPayment] {
		for idx := range set.BillPayments {
			page.Payments = append(page.Payments, inboundBillPaymentOf(&set.BillPayments[idx]))
		}
	}
	for idx := range set.References {
		ref := &set.References[idx]
		if skip[quickbooks.ReferenceChangeEntity(ref.Kind)] {
			continue
		}
		kind, ok := referenceKindOf(ref.Kind)
		if !ok {
			continue
		}
		page.References = append(page.References, services.AccountingChangedReference{
			Object:  referenceObjectOf(kind, &ref.ReferenceObject),
			Deleted: ref.Deleted,
		})
	}
	return page
}

func referenceKindOf(kind quickbooks.ReferenceKind) (accountingsync.ReferenceKind, bool) {
	for _, candidate := range accountingsync.AllReferenceKinds() {
		providerRef, err := providerKind(candidate)
		if err == nil && providerRef == kind {
			return candidate, true
		}
	}
	return "", false
}

func changeOperation(meta *quickbooks.ChangeMeta) services.AccountingChangeOperation {
	switch {
	case meta.Deleted:
		return services.AccountingChangeDelete
	case meta.Voided:
		return services.AccountingChangeVoid
	default:
		return services.AccountingChangeUpsert
	}
}

func inboundDocumentKind(txnType string) accountingsync.InboundDocumentKind {
	switch quickbooks.TxnKind(txnType) {
	case quickbooks.TxnInvoice:
		return accountingsync.InboundDocInvoice
	case quickbooks.TxnCreditMemo:
		return accountingsync.InboundDocCreditMemo
	case quickbooks.TxnBill:
		return accountingsync.InboundDocBill
	case quickbooks.TxnVendorCredit:
		return accountingsync.InboundDocVendorCredit
	case quickbooks.TxnPayment, quickbooks.TxnBillPayment:
		return accountingsync.InboundDocOther
	default:
		return accountingsync.InboundDocOther
	}
}

func inboundLines(links []quickbooks.ChangedLink) []services.AccountingInboundLine {
	lines := make([]services.AccountingInboundLine, 0, len(links))
	for _, link := range links {
		lines = append(lines, services.AccountingInboundLine{
			DocumentKind:       inboundDocumentKind(link.TxnType),
			DocumentExternalID: link.TxnID,
			Amount:             link.Amount,
		})
	}
	return lines
}

func inboundPaymentOf(payment *quickbooks.ChangedPayment) services.AccountingInboundPayment {
	return services.AccountingInboundPayment{
		Kind:              accountingsync.InboundCustomerPayment,
		Operation:         changeOperation(&payment.ChangeMeta),
		ExternalID:        payment.ID,
		Number:            payment.ReferenceNumber,
		ModifiedAt:        payment.LastUpdatedAt,
		ModifiedBy:        payment.LastModifiedBy,
		PartyExternalID:   payment.CustomerID,
		PartyName:         payment.CustomerName,
		TxnDate:           payment.TxnDate,
		CurrencyCode:      payment.CurrencyCode,
		Amount:            payment.TotalAmount,
		Unapplied:         payment.UnappliedAmount,
		MethodExternalID:  payment.MethodID,
		MethodName:        payment.MethodName,
		AccountExternalID: payment.DepositAccountID,
		ReferenceNumber:   payment.ReferenceNumber,
		Lines:             inboundLines(payment.Lines),
	}
}

func inboundBillPaymentOf(
	payment *quickbooks.ChangedBillPayment,
) services.AccountingInboundPayment {
	return services.AccountingInboundPayment{
		Kind:              accountingsync.InboundBillPayment,
		Operation:         changeOperation(&payment.ChangeMeta),
		ExternalID:        payment.ID,
		Number:            payment.DocNumber,
		ModifiedAt:        payment.LastUpdatedAt,
		ModifiedBy:        payment.LastModifiedBy,
		PartyExternalID:   payment.VendorID,
		PartyName:         payment.VendorName,
		TxnDate:           payment.TxnDate,
		CurrencyCode:      payment.CurrencyCode,
		Amount:            payment.TotalAmount,
		MethodName:        payment.PayType,
		AccountExternalID: payment.AccountID,
		ReferenceNumber:   payment.DocNumber,
		Lines:             inboundLines(payment.Lines),
	}
}
