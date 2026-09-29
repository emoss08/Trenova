package xeroconnector

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/shopspring/decimal"
)

var _ services.AccountingChangeReader = (*Connector)(nil)

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

var errChangeCursor = errors.New("xero: the change cursor is not one this adapter wrote")

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
		return time.Time{}, errChangeCursor
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
			return nil, errChangeCursor
		}
		return &changeCursor{since: since}, nil
	}

	parts := strings.Split(value, cursorSeparator)
	if len(parts) != cursorParts {
		return nil, errChangeCursor
	}
	since, sinceErr := parseInstant(parts[1])
	started, startedErr := parseInstant(parts[2])
	latest, latestErr := parseInstant(parts[3])
	index, indexErr := strconv.Atoi(parts[5])
	page, pageErr := strconv.Atoi(parts[6])
	if errors.Join(sinceErr, startedErr, latestErr, indexErr, pageErr) != nil || since.IsZero() {
		return nil, errChangeCursor
	}
	targets := make([]changeTarget, 0, 6)
	for name := range strings.SplitSeq(parts[4], targetSeparator) {
		target := changeTarget(name)
		if !target.valid() {
			return nil, errChangeCursor
		}
		targets = append(targets, target)
	}
	if index < 0 || index >= len(targets) || page < 1 {
		return nil, errChangeCursor
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

func (c *Connector) ChangeFeedLimits() services.AccountingChangeFeedLimits {
	return services.AccountingChangeFeedLimits{
		MaxLookback:    0,
		ReportsDeletes: true,
		MaxPerRead:     xero.MaxPageSize,
	}
}

func (c *Connector) ChangeCursorAt(at time.Time) string {
	return formatInstant(at)
}

func changeTargets(req *services.ReadAccountingChangesRequest) ([]changeTarget, error) {
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
		case accountingsync.ReferenceKindAccount:
			target = targetAccounts
		case accountingsync.ReferenceKindItem:
			target = targetItems
		case accountingsync.ReferenceKindCustomer, accountingsync.ReferenceKindVendor:
			target = targetContacts
		case accountingsync.ReferenceKindTerm, accountingsync.ReferenceKindPaymentMethod:
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

func (c *Connector) ReadChanges(
	ctx context.Context,
	req *services.ReadAccountingChangesRequest,
) (*services.AccountingChangePage, error) {
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
		return &services.AccountingChangePage{NextCursor: cursor.String()}, nil
	}
	client, err := c.client(req.Auth)
	if err != nil {
		return nil, err
	}
	collect := newChangeCollector(c, client, req)

	if cursor.paging {
		more, readErr := collect.read(ctx, cursor.targets[cursor.index], cursor.page, cursor.since)
		if readErr != nil {
			return nil, readErr
		}
		if err = collect.finish(ctx); err != nil {
			return nil, err
		}
		cursor.latest = laterTime(cursor.latest, collect.latest)
		switch {
		case more:
			cursor.page++
			collect.page.More = true
			collect.page.NextCursor = cursor.String()
		case cursor.index+1 < len(cursor.targets):
			cursor.index++
			cursor.page = 2
			collect.page.More = true
			collect.page.NextCursor = cursor.String()
		default:
			done := changeCursor{since: nextSince(cursor.since, cursor.latest, cursor.started)}
			collect.page.NextCursor = done.String()
		}
		return collect.page, nil
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
	conn     *Connector
	client   *xero.Client
	req      *services.ReadAccountingChangesRequest
	page     *services.AccountingChangePage
	latest   time.Time
	payments []xero.Payment
}

func newChangeCollector(
	conn *Connector,
	client *xero.Client,
	req *services.ReadAccountingChangesRequest,
) *changeCollector {
	return &changeCollector{
		conn:   conn,
		client: client,
		req:    req,
		page: &services.AccountingChangePage{
			Payments:   []services.AccountingInboundPayment{},
			Documents:  []services.AccountingChangedDocument{},
			References: []services.AccountingChangedReference{},
		},
	}
}

func (c *changeCollector) seen(t time.Time) {
	c.latest = laterTime(c.latest, t)
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
		return false, errChangeCursor
	}
}

func (c *changeCollector) readPayments(
	ctx context.Context,
	page int,
	since *time.Time,
) (bool, error) {
	listed, err := c.client.Payments(ctx, page, since)
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
	listed, err := c.client.Invoices(ctx, page, since)
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
	listed, err := c.client.CreditNotes(ctx, page, since)
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
	listed, err := c.client.Contacts(ctx, page, since)
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
	accounts, err := c.client.Accounts(ctx, since)
	if err != nil {
		return err
	}
	for idx := range accounts {
		account := &accounts[idx]
		c.seen(account.UpdatedAt)
		c.page.References = append(c.page.References, services.AccountingChangedReference{
			Object:  accountObjectOf(account),
			Deleted: strings.EqualFold(account.Status, accountStatusDeleted),
		})
	}
	return nil
}

func (c *changeCollector) readItems(ctx context.Context, since *time.Time) error {
	items, err := c.client.Items(ctx, since)
	if err != nil {
		return err
	}
	for idx := range items {
		c.seen(items[idx].UpdatedAt)
	}
	objects, err := c.conn.itemObjects(ctx, c.client, items)
	if err != nil {
		return err
	}
	for _, obj := range objects {
		c.page.References = append(
			c.page.References,
			services.AccountingChangedReference{Object: obj},
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
	c.page.Documents = append(c.page.Documents, services.AccountingChangedDocument{
		ObjectTypes: doc.types,
		ExternalID:  doc.externalID,
		Operation:   documentOperation(doc.status),
		ModifiedAt:  unixOrZero(doc.updated),
	})
}

func (c *changeCollector) contact(contact *xero.Contact) {
	for _, kind := range []accountingsync.ReferenceKind{
		accountingsync.ReferenceKindCustomer,
		accountingsync.ReferenceKindVendor,
	} {
		if !slices.Contains(c.req.ReferenceKinds, kind) || !contactHolds(contact, kind) {
			continue
		}
		c.page.References = append(c.page.References, services.AccountingChangedReference{
			Object: contactObjectOf(kind, contact),
		})
	}
}

func documentOperation(status string) services.AccountingChangeOperation {
	switch {
	case strings.EqualFold(status, xero.StatusDeleted):
		return services.AccountingChangeDelete
	case strings.EqualFold(status, xero.StatusVoided):
		return services.AccountingChangeVoid
	default:
		return services.AccountingChangeUpsert
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
	case xero.InvoiceTypeReceivable:
		return []accountingsync.SyncObjectType{
			accountingsync.SyncObjectInvoice,
			accountingsync.SyncObjectDebitMemo,
		}
	case xero.InvoiceTypePayable:
		return payableTypes()
	default:
		return nil
	}
}

func creditNoteObjectTypes(kind string) []accountingsync.SyncObjectType {
	switch strings.ToUpper(kind) {
	case xero.CreditNoteTypeReceivable:
		return []accountingsync.SyncObjectType{accountingsync.SyncObjectCreditMemo}
	case xero.CreditNoteTypePayable:
		return payableTypes()
	default:
		return nil
	}
}

type inboundShape struct {
	kind     accountingsync.InboundChangeKind
	document accountingsync.InboundDocumentKind
}

func inboundShapeOf(paymentType string) (inboundShape, bool) {
	switch strings.ToUpper(paymentType) {
	case xero.PaymentTypeReceivable:
		return inboundShape{
			kind:     accountingsync.InboundCustomerPayment,
			document: accountingsync.InboundDocInvoice,
		}, true
	case xero.PaymentTypePayable:
		return inboundShape{
			kind:     accountingsync.InboundBillPayment,
			document: accountingsync.InboundDocBill,
		}, true
	default:
		return inboundShape{}, false
	}
}

func (c *changeCollector) wants(shape inboundShape) bool {
	if shape.kind == accountingsync.InboundCustomerPayment {
		return c.req.Payments
	}
	return c.req.BillPayments
}

type inboundGroup struct {
	shape    inboundShape
	payments []*xero.Payment
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

func isForeign(payment *xero.Payment) bool {
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
		invoices, err := c.client.InvoicesByID(ctx, chunk)
		if err != nil {
			return nil, err
		}
		for idx := range invoices {
			currencies[strings.ToLower(invoices[idx].InvoiceID)] = invoices[idx].CurrencyCode
		}
	}
	return currencies, nil
}

func (g *inboundGroup) inbound(currencies map[string]string) services.AccountingInboundPayment {
	first := g.payments[0]
	externalID := first.PaymentID
	if first.BatchPaymentID != "" {
		externalID = first.BatchPaymentID
	}
	out := services.AccountingInboundPayment{
		Kind:              g.shape.kind,
		Operation:         services.AccountingChangeDelete,
		ExternalID:        externalID,
		Number:            first.Reference,
		PartyExternalID:   first.ContactID,
		TxnDate:           dateOf(first.Date),
		Amount:            decimal.Zero,
		Unapplied:         decimal.Zero,
		AccountExternalID: first.AccountID,
		ReferenceNumber:   first.Reference,
		Lines:             make([]services.AccountingInboundLine, 0, len(g.payments)),
	}
	if isForeign(first) {
		out.CurrencyCode = currencies[strings.ToLower(first.InvoiceID)]
	}
	for _, payment := range g.payments {
		out.ModifiedAt = max(out.ModifiedAt, unixOrZero(payment.UpdatedAt))
		if strings.EqualFold(payment.Status, xero.StatusDeleted) {
			continue
		}
		out.Operation = services.AccountingChangeUpsert
		out.Amount = out.Amount.Add(payment.Amount)
		out.Lines = append(out.Lines, services.AccountingInboundLine{
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
