package xerochangefeed

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/xerorefsync"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/xerosession"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountinginbound"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/emoss08/trenova/shared/xerodocs"
	"github.com/emoss08/trenova/shared/xeropay"
	"github.com/emoss08/trenova/shared/xeroref"
	"github.com/shopspring/decimal"
)

type Feed struct {
	*xerosession.Session
	refs *xerorefsync.References
}

func NewFeed(session *xerosession.Session, refs *xerorefsync.References) *Feed {
	return &Feed{Session: session, refs: refs}
}

type changeTarget string

const (
	targetPayments    = changeTarget("payments")
	targetInvoices    = changeTarget("invoices")
	targetCreditNotes = changeTarget("creditnotes")
	targetAccounts    = changeTarget("accounts")
	targetItems       = changeTarget("items")
	targetContacts    = changeTarget("contacts")
	cursorPaging      = "p"
	cursorSeparator   = "|"
	targetSeparator   = ","
	cursorParts       = 7
	cursorPrecision   = time.Second
)

var ErrChangeCursor = errors.New("xero: the change cursor is not one this adapter wrote")

func (t changeTarget) valid() bool {
	switch t {
	case targetPayments, targetInvoices, targetCreditNotes, targetAccounts, targetItems,
		targetContacts:
		return true
	default:
		return false
	}
}

type changeCursor struct {
	since   time.Time
	paging  bool
	started time.Time
	latest  time.Time
	targets []changeTarget
	index   int
	page    int
}

func formatInstant(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Truncate(cursorPrecision).Format(time.RFC3339)
}

func parseInstant(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, ErrChangeCursor
	}
	return parsed.UTC(), nil
}

func (c *changeCursor) String() string {
	if !c.paging {
		return formatInstant(c.since)
	}
	names := make([]string, 0, len(c.targets))
	for _, target := range c.targets {
		names = append(names, string(target))
	}
	return strings.Join([]string{
		cursorPaging,
		formatInstant(c.since),
		formatInstant(c.started),
		formatInstant(c.latest),
		strings.Join(names, targetSeparator),
		strconv.Itoa(c.index),
		strconv.Itoa(c.page),
	}, cursorSeparator)
}

func parseChangeCursor(raw string, now time.Time) (*changeCursor, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return &changeCursor{since: now.Truncate(cursorPrecision)}, nil
	}
	if !strings.HasPrefix(value, cursorPaging+cursorSeparator) {
		since, err := parseInstant(value)
		if err != nil || since.IsZero() {
			return nil, ErrChangeCursor
		}
		return &changeCursor{since: since}, nil
	}

	parts := strings.Split(value, cursorSeparator)
	if len(parts) != cursorParts {
		return nil, ErrChangeCursor
	}
	since, sinceErr := parseInstant(parts[1])
	started, startedErr := parseInstant(parts[2])
	latest, latestErr := parseInstant(parts[3])
	index, indexErr := strconv.Atoi(parts[5])
	page, pageErr := strconv.Atoi(parts[6])
	if errors.Join(sinceErr, startedErr, latestErr, indexErr, pageErr) != nil || since.IsZero() {
		return nil, ErrChangeCursor
	}
	targets := make([]changeTarget, 0, 6)
	for name := range strings.SplitSeq(parts[4], targetSeparator) {
		target := changeTarget(name)
		if !target.valid() {
			return nil, ErrChangeCursor
		}
		targets = append(targets, target)
	}
	if index < 0 || index >= len(targets) || page < 1 {
		return nil, ErrChangeCursor
	}
	return &changeCursor{
		since:   since,
		paging:  true,
		started: started,
		latest:  latest,
		targets: targets,
		index:   index,
		page:    page,
	}, nil
}

func nextSince(since, latest, started time.Time) time.Time {
	if latest.IsZero() {
		return since
	}
	next := latest
	if !started.IsZero() && started.Before(next) {
		next = started
	}
	next = next.Add(-cursorPrecision).Truncate(cursorPrecision)
	if next.Before(since) {
		return since
	}
	return next
}

func (c *Feed) ChangeFeedLimits() accountingconnectorports.AccountingChangeFeedLimits {
	return accountingconnectorports.AccountingChangeFeedLimits{
		MaxLookback:    0,
		ReportsDeletes: true,
		MaxPerRead:     xero.MaxPageSize,
	}
}

func (c *Feed) ChangeCursorAt(at time.Time) string {
	return formatInstant(at)
}

func changeTargets(
	req *accountingconnectorports.ReadAccountingChangesRequest,
) ([]changeTarget, error) {
	targets := make([]changeTarget, 0, 6)
	if req.Payments || req.BillPayments {
		targets = append(targets, targetPayments)
	}
	if req.Documents {
		targets = append(targets, targetInvoices, targetCreditNotes)
	}
	for _, kind := range req.ReferenceKinds {
		var target changeTarget
		switch kind {
		case accountingprovider.ReferenceKindAccount:
			target = targetAccounts
		case accountingprovider.ReferenceKindItem:
			target = targetItems
		case accountingprovider.ReferenceKindCustomer, accountingprovider.ReferenceKindVendor:
			target = targetContacts
		case accountingprovider.ReferenceKindTerm, accountingprovider.ReferenceKindPaymentMethod:
			continue
		default:
			return nil, errors.New("xero: unknown reference kind " + string(kind))
		}
		if !slices.Contains(targets, target) {
			targets = append(targets, target)
		}
	}
	return targets, nil
}

func (c *Feed) ReadChanges(
	ctx context.Context,
	req *accountingconnectorports.ReadAccountingChangesRequest,
) (*accountingconnectorports.AccountingChangePage, error) {
	now := req.Now.UTC()
	if req.Now.IsZero() {
		now = time.Now().UTC()
	}
	cursor, err := parseChangeCursor(req.Cursor, now)
	if err != nil {
		return nil, err
	}
	targets, err := changeTargets(req)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return &accountingconnectorports.AccountingChangePage{NextCursor: cursor.String()}, nil
	}
	client, err := c.Client(req.Auth)
	if err != nil {
		return nil, err
	}
	collect := newChangeCollector(c.refs, client, req)

	if cursor.paging {
		return collect.continuePaging(ctx, cursor)
	}

	full := make([]changeTarget, 0, len(targets))
	for _, target := range targets {
		more, readErr := collect.read(ctx, target, 1, cursor.since)
		if readErr != nil {
			return nil, readErr
		}
		if more {
			full = append(full, target)
		}
	}
	if err = collect.finish(ctx); err != nil {
		return nil, err
	}
	if len(full) == 0 {
		done := changeCursor{since: nextSince(cursor.since, collect.latest, time.Time{})}
		collect.page.NextCursor = done.String()
		return collect.page, nil
	}
	paging := changeCursor{
		since:   cursor.since,
		paging:  true,
		started: now,
		latest:  collect.latest,
		targets: full,
		page:    2,
	}
	collect.page.More = true
	collect.page.NextCursor = paging.String()
	return collect.page, nil
}

func laterTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

type changeCollector struct {
	refs     *xerorefsync.References
	client   *xero.Client
	req      *accountingconnectorports.ReadAccountingChangesRequest
	page     *accountingconnectorports.AccountingChangePage
	latest   time.Time
	payments []xeropay.Payment
}

func newChangeCollector(
	refs *xerorefsync.References,
	client *xero.Client,
	req *accountingconnectorports.ReadAccountingChangesRequest,
) *changeCollector {
	return &changeCollector{
		refs:   refs,
		client: client,
		req:    req,
		page: &accountingconnectorports.AccountingChangePage{
			Payments:   []accountingconnectorports.AccountingInboundPayment{},
			Documents:  []accountingconnectorports.AccountingChangedDocument{},
			References: []accountingconnectorports.AccountingChangedReference{},
		},
	}
}

func (c *changeCollector) seen(t time.Time) {
	c.latest = laterTime(c.latest, t)
}

func (c *changeCollector) continuePaging(
	ctx context.Context,
	cursor *changeCursor,
) (*accountingconnectorports.AccountingChangePage, error) {
	more, err := c.read(ctx, cursor.targets[cursor.index], cursor.page, cursor.since)
	if err != nil {
		return nil, err
	}
	if err = c.finish(ctx); err != nil {
		return nil, err
	}
	cursor.latest = laterTime(cursor.latest, c.latest)
	switch {
	case more:
		cursor.page++
		c.page.More = true
		c.page.NextCursor = cursor.String()
	case cursor.index+1 < len(cursor.targets):
		cursor.index++
		cursor.page = 2
		c.page.More = true
		c.page.NextCursor = cursor.String()
	default:
		done := changeCursor{since: nextSince(cursor.since, cursor.latest, cursor.started)}
		c.page.NextCursor = done.String()
	}
	return c.page, nil
}

func (c *changeCollector) read(
	ctx context.Context,
	target changeTarget,
	page int,
	since time.Time,
) (bool, error) {
	switch target {
	case targetPayments:
		return c.readPayments(ctx, page, &since)
	case targetInvoices:
		return c.readInvoices(ctx, page, &since)
	case targetCreditNotes:
		return c.readCreditNotes(ctx, page, &since)
	case targetContacts:
		return c.readContacts(ctx, page, &since)
	case targetAccounts:
		return false, c.readAccounts(ctx, &since)
	case targetItems:
		return false, c.readItems(ctx, &since)
	default:
		return false, ErrChangeCursor
	}
}

func (c *changeCollector) readPayments(
	ctx context.Context,
	page int,
	since *time.Time,
) (bool, error) {
	listed, err := xeropay.Payments(ctx, c.client, page, since)
	if err != nil {
		return false, err
	}
	for idx := range listed.Payments {
		c.seen(listed.Payments[idx].UpdatedAt)
	}
	c.payments = append(c.payments, listed.Payments...)
	return listed.More, nil
}

func (c *changeCollector) readInvoices(
	ctx context.Context,
	page int,
	since *time.Time,
) (bool, error) {
	listed, err := xerodocs.Invoices(ctx, c.client, page, since)
	if err != nil {
		return false, err
	}
	for idx := range listed.Invoices {
		invoice := &listed.Invoices[idx]
		c.seen(invoice.UpdatedAt)
		c.document(&changedDocument{
			types:      invoiceObjectTypes(invoice.Type),
			externalID: invoice.InvoiceID,
			status:     invoice.Status,
			updated:    invoice.UpdatedAt,
		})
	}
	return listed.More, nil
}

func (c *changeCollector) readCreditNotes(
	ctx context.Context,
	page int,
	since *time.Time,
) (bool, error) {
	listed, err := xerodocs.CreditNotes(ctx, c.client, page, since)
	if err != nil {
		return false, err
	}
	for idx := range listed.CreditNotes {
		note := &listed.CreditNotes[idx]
		c.seen(note.UpdatedAt)
		c.document(&changedDocument{
			types:      creditNoteObjectTypes(note.Type),
			externalID: note.CreditNoteID,
			status:     note.Status,
			updated:    note.UpdatedAt,
		})
	}
	return listed.More, nil
}

func (c *changeCollector) readContacts(
	ctx context.Context,
	page int,
	since *time.Time,
) (bool, error) {
	listed, err := xeroref.Contacts(ctx, c.client, page, since)
	if err != nil {
		return false, err
	}
	for idx := range listed.Contacts {
		c.seen(listed.Contacts[idx].UpdatedAt)
		c.contact(&listed.Contacts[idx])
	}
	return listed.More, nil
}

func (c *changeCollector) readAccounts(ctx context.Context, since *time.Time) error {
	accounts, err := xeroref.Accounts(ctx, c.client, since)
	if err != nil {
		return err
	}
	for idx := range accounts {
		account := &accounts[idx]
		c.seen(account.UpdatedAt)
		c.page.References = append(
			c.page.References,
			accountingconnectorports.AccountingChangedReference{
				Object:  xerorefsync.AccountObjectOf(account),
				Deleted: strings.EqualFold(account.Status, accountStatusDeleted),
			},
		)
	}
	return nil
}

func (c *changeCollector) readItems(ctx context.Context, since *time.Time) error {
	items, err := xeroref.Items(ctx, c.client, since)
	if err != nil {
		return err
	}
	for idx := range items {
		c.seen(items[idx].UpdatedAt)
	}
	objects, err := c.refs.ItemObjects(ctx, c.client, items)
	if err != nil {
		return err
	}
	for _, obj := range objects {
		c.page.References = append(
			c.page.References,
			accountingconnectorports.AccountingChangedReference{Object: obj},
		)
	}
	return nil
}

type changedDocument struct {
	types      []accountingsync.SyncObjectType
	externalID string
	status     string
	updated    time.Time
}

func (c *changeCollector) document(doc *changedDocument) {
	if len(doc.types) == 0 || doc.externalID == "" {
		return
	}
	c.page.Documents = append(c.page.Documents, accountingconnectorports.AccountingChangedDocument{
		ObjectTypes: doc.types,
		ExternalID:  doc.externalID,
		Operation:   documentOperation(doc.status),
		ModifiedAt:  unixOrZero(doc.updated),
	})
}

func (c *changeCollector) contact(contact *xeroref.Contact) {
	for _, kind := range []accountingprovider.ReferenceKind{
		accountingprovider.ReferenceKindCustomer,
		accountingprovider.ReferenceKindVendor,
	} {
		if !slices.Contains(c.req.ReferenceKinds, kind) ||
			!xerorefsync.ContactHolds(contact, kind) {
			continue
		}
		c.page.References = append(
			c.page.References,
			accountingconnectorports.AccountingChangedReference{
				Object: xerorefsync.ContactObjectOf(kind, contact),
			},
		)
	}
}

func documentOperation(status string) accountingconnectorports.AccountingChangeOperation {
	switch {
	case strings.EqualFold(status, xerodocs.StatusDeleted):
		return accountingconnectorports.AccountingChangeDelete
	case strings.EqualFold(status, xerodocs.StatusVoided):
		return accountingconnectorports.AccountingChangeVoid
	default:
		return accountingconnectorports.AccountingChangeUpsert
	}
}

func payableTypes() []accountingsync.SyncObjectType {
	return []accountingsync.SyncObjectType{
		accountingsync.SyncObjectCarrierBill,
		accountingsync.SyncObjectDriverBill,
	}
}

func invoiceObjectTypes(kind string) []accountingsync.SyncObjectType {
	switch strings.ToUpper(kind) {
	case xerodocs.InvoiceTypeReceivable:
		return []accountingsync.SyncObjectType{
			accountingsync.SyncObjectInvoice,
			accountingsync.SyncObjectDebitMemo,
		}
	case xerodocs.InvoiceTypePayable:
		return payableTypes()
	default:
		return nil
	}
}

func creditNoteObjectTypes(kind string) []accountingsync.SyncObjectType {
	switch strings.ToUpper(kind) {
	case xerodocs.CreditNoteTypeReceivable:
		return []accountingsync.SyncObjectType{accountingsync.SyncObjectCreditMemo}
	case xerodocs.CreditNoteTypePayable:
		return payableTypes()
	default:
		return nil
	}
}

type inboundShape struct {
	kind     accountinginbound.InboundChangeKind
	document accountinginbound.InboundDocumentKind
}

func inboundShapeOf(paymentType string) (inboundShape, bool) {
	switch strings.ToUpper(paymentType) {
	case xeropay.PaymentTypeReceivable:
		return inboundShape{
			kind:     accountinginbound.InboundCustomerPayment,
			document: accountinginbound.InboundDocInvoice,
		}, true
	case xeropay.PaymentTypePayable:
		return inboundShape{
			kind:     accountinginbound.InboundBillPayment,
			document: accountinginbound.InboundDocBill,
		}, true
	default:
		return inboundShape{}, false
	}
}

func (c *changeCollector) wants(shape inboundShape) bool {
	if shape.kind == accountinginbound.InboundCustomerPayment {
		return c.req.Payments
	}
	return c.req.BillPayments
}

type inboundGroup struct {
	shape    inboundShape
	payments []*xeropay.Payment
}

func (c *changeCollector) finish(ctx context.Context) error {
	if len(c.payments) == 0 {
		return nil
	}
	groups := make([]*inboundGroup, 0, len(c.payments))
	byKey := make(map[string]*inboundGroup, len(c.payments))
	for idx := range c.payments {
		payment := &c.payments[idx]
		shape, ok := inboundShapeOf(payment.PaymentType)
		if !ok || !c.wants(shape) {
			continue
		}
		key := payment.BatchPaymentID
		if key == "" {
			key = payment.PaymentID
		}
		group, found := byKey[key]
		if !found {
			group = &inboundGroup{shape: shape}
			byKey[key] = group
			groups = append(groups, group)
		}
		group.payments = append(group.payments, payment)
	}

	currencies, err := c.foreignCurrencies(ctx, groups)
	if err != nil {
		return err
	}
	for _, group := range groups {
		c.page.Payments = append(c.page.Payments, group.inbound(currencies))
	}
	return nil
}

func isForeign(payment *xeropay.Payment) bool {
	return !payment.CurrencyRate.IsZero() && !payment.CurrencyRate.Equal(decimal.NewFromInt(1))
}

func (c *changeCollector) foreignCurrencies(
	ctx context.Context,
	groups []*inboundGroup,
) (map[string]string, error) {
	ids := make([]string, 0, len(groups))
	seen := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		for _, payment := range group.payments {
			id := strings.ToLower(payment.InvoiceID)
			if !isForeign(payment) || id == "" {
				continue
			}
			if _, dup := seen[id]; !dup {
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
		}
	}
	currencies := make(map[string]string, len(ids))
	for _, chunk := range chunks(ids) {
		invoices, err := xerodocs.InvoicesByID(ctx, c.client, chunk)
		if err != nil {
			return nil, err
		}
		for idx := range invoices {
			currencies[strings.ToLower(invoices[idx].InvoiceID)] = invoices[idx].CurrencyCode
		}
	}
	return currencies, nil
}

func (g *inboundGroup) inbound(
	currencies map[string]string,
) accountingconnectorports.AccountingInboundPayment {
	first := g.payments[0]
	externalID := first.PaymentID
	if first.BatchPaymentID != "" {
		externalID = first.BatchPaymentID
	}
	out := accountingconnectorports.AccountingInboundPayment{
		Kind:              g.shape.kind,
		Operation:         accountingconnectorports.AccountingChangeDelete,
		ExternalID:        externalID,
		Number:            first.Reference,
		PartyExternalID:   first.ContactID,
		TxnDate:           dateOf(first.Date),
		Amount:            decimal.Zero,
		Unapplied:         decimal.Zero,
		AccountExternalID: first.AccountID,
		ReferenceNumber:   first.Reference,
		Lines: make(
			[]accountingconnectorports.AccountingInboundLine,
			0,
			len(g.payments),
		),
	}
	if isForeign(first) {
		out.CurrencyCode = currencies[strings.ToLower(first.InvoiceID)]
	}
	for _, payment := range g.payments {
		out.ModifiedAt = max(out.ModifiedAt, unixOrZero(payment.UpdatedAt))
		if strings.EqualFold(payment.Status, xerodocs.StatusDeleted) {
			continue
		}
		out.Operation = accountingconnectorports.AccountingChangeUpsert
		out.Amount = out.Amount.Add(payment.Amount)
		out.Lines = append(out.Lines, accountingconnectorports.AccountingInboundLine{
			DocumentKind:       g.shape.document,
			DocumentExternalID: payment.InvoiceID,
			Amount:             payment.Amount,
		})
	}
	return out
}

func dateOf(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.DateOnly)
}
