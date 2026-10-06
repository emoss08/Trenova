package bcconnector

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/businesscentral"
)

var _ services.AccountingChangeReader = (*Connector)(nil)

type changeTarget string

const (
	targetSalesInvoices    = changeTarget("salesInvoices")
	targetSalesCreditMemos = changeTarget("salesCreditMemos")
	targetPurchaseInvoices = changeTarget("purchaseInvoices")
	targetPurchaseCredits  = changeTarget("purchaseCreditMemos")
	targetAccounts         = changeTarget("accounts")
	targetItems            = changeTarget("items")
	targetCustomers        = changeTarget("customers")
	targetVendors          = changeTarget("vendors")
	cursorPaging           = "p"
	cursorSeparator        = "|"
	targetSeparator        = ","
	cursorParts            = 6
	cursorPrecision        = time.Second
	maxChangeTargets       = 8
)

var errChangeCursor = errors.New("businesscentral: the change cursor is not one this adapter wrote")

var documentTargets = map[changeTarget]businesscentral.DocumentKind{ //nolint:exhaustive // only document targets have a document kind
	targetSalesInvoices:    businesscentral.DocumentSalesInvoice,
	targetSalesCreditMemos: businesscentral.DocumentSalesCreditMemo,
	targetPurchaseInvoices: businesscentral.DocumentPurchaseInvoice,
	targetPurchaseCredits:  businesscentral.DocumentPurchaseCreditMemo,
}

func (t changeTarget) valid() bool {
	if _, ok := documentTargets[t]; ok {
		return true
	}
	return t == targetAccounts || t == targetItems || t == targetCustomers || t == targetVendors
}

type changeCursor struct {
	since   time.Time
	paging  bool
	started time.Time
	latest  time.Time
	targets []changeTarget
	index   int
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
	}, cursorSeparator)
}

func parseChangeCursor(raw string, now time.Time) (*changeCursor, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return &changeCursor{since: now.UTC().Truncate(cursorPrecision)}, nil
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
	if errors.Join(sinceErr, startedErr, latestErr, indexErr) != nil || since.IsZero() {
		return nil, errChangeCursor
	}
	targets := make([]changeTarget, 0, maxChangeTargets)
	for name := range strings.SplitSeq(parts[4], targetSeparator) {
		target := changeTarget(name)
		if !target.valid() {
			return nil, errChangeCursor
		}
		targets = append(targets, target)
	}
	if index < 0 || index >= len(targets) {
		return nil, errChangeCursor
	}
	return &changeCursor{
		since:   since,
		paging:  true,
		started: started,
		latest:  latest,
		targets: targets,
		index:   index,
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
		ReportsDeletes: false,
		MaxPerRead:     businesscentral.MaxPageSize,
	}
}

func (c *Connector) ChangeCursorAt(at time.Time) string {
	return formatInstant(at)
}

func changeTargets(req *services.ReadAccountingChangesRequest) []changeTarget {
	targets := make([]changeTarget, 0, maxChangeTargets)
	if req.Documents {
		targets = append(
			targets,
			targetSalesInvoices,
			targetSalesCreditMemos,
			targetPurchaseInvoices,
			targetPurchaseCredits,
		)
	}
	for _, kind := range req.ReferenceKinds {
		var target changeTarget
		switch kind {
		case accountingsync.ReferenceKindAccount:
			target = targetAccounts
		case accountingsync.ReferenceKindItem:
			target = targetItems
		case accountingsync.ReferenceKindCustomer:
			target = targetCustomers
		case accountingsync.ReferenceKindVendor:
			target = targetVendors
		case accountingsync.ReferenceKindTerm, accountingsync.ReferenceKindPaymentMethod:
			continue
		default:
			continue
		}
		if !slices.Contains(targets, target) {
			targets = append(targets, target)
		}
	}
	return targets
}

func (c *Connector) ReadChanges(
	ctx context.Context,
	req *services.ReadAccountingChangesRequest,
) (*services.AccountingChangePage, error) {
	now := req.Now.UTC()
	if req.Now.IsZero() {
		now = c.now().UTC()
	}
	cursor, err := parseChangeCursor(req.Cursor, now)
	if err != nil {
		return nil, err
	}
	if !cursor.paging {
		cursor.targets = changeTargets(req)
		cursor.started = now
	}
	if len(cursor.targets) == 0 {
		return &services.AccountingChangePage{
			Payments:   []services.AccountingInboundPayment{},
			Documents:  []services.AccountingChangedDocument{},
			References: []services.AccountingChangedReference{},
			NextCursor: cursor.String(),
		}, nil
	}
	client, err := c.client(req.Auth)
	if err != nil {
		return nil, err
	}
	collect := newChangeCollector(client)
	return collect.run(ctx, cursor)
}

type changeCollector struct {
	client *businesscentral.Client
	page   *services.AccountingChangePage
	latest time.Time
	read   int
}

func newChangeCollector(client *businesscentral.Client) *changeCollector {
	return &changeCollector{
		client: client,
		page: &services.AccountingChangePage{
			Payments:   []services.AccountingInboundPayment{},
			Documents:  []services.AccountingChangedDocument{},
			References: []services.AccountingChangedReference{},
		},
	}
}

func (c *changeCollector) run(
	ctx context.Context,
	cursor *changeCursor,
) (*services.AccountingChangePage, error) {
	since := cursor.since
	for idx := cursor.index; idx < len(cursor.targets); idx++ {
		if err := c.readTarget(ctx, cursor.targets[idx], &since); err != nil {
			return nil, err
		}
		if c.read >= businesscentral.MaxPageSize && idx+1 < len(cursor.targets) {
			paging := changeCursor{
				since:   cursor.since,
				paging:  true,
				started: cursor.started,
				latest:  laterTime(cursor.latest, c.latest),
				targets: cursor.targets,
				index:   idx + 1,
			}
			c.page.More = true
			c.page.NextCursor = paging.String()
			return c.page, nil
		}
	}
	latest := laterTime(cursor.latest, c.latest)
	done := changeCursor{since: nextSince(cursor.since, latest, cursor.started)}
	c.page.NextCursor = done.String()
	return c.page, nil
}

func laterTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func (c *changeCollector) seen(t time.Time) {
	c.latest = laterTime(c.latest, t)
	c.read++
}

func (c *changeCollector) readTarget(
	ctx context.Context,
	target changeTarget,
	since *time.Time,
) error {
	switch target {
	case targetSalesInvoices, targetSalesCreditMemos, targetPurchaseInvoices, targetPurchaseCredits:
		return c.readDocuments(ctx, documentTargets[target], since)
	case targetAccounts:
		accounts, err := c.client.Accounts(ctx, since)
		for idx := range accounts {
			c.seen(accounts[idx].LastModified)
			c.reference(accountObjectOf(&accounts[idx]))
		}
		return err
	case targetItems:
		items, err := c.client.Items(ctx, since)
		for idx := range items {
			c.seen(items[idx].LastModified)
			c.reference(itemObjectOf(&items[idx]))
		}
		return err
	case targetCustomers:
		return c.readParties(ctx, businesscentral.PartyCustomer, since)
	case targetVendors:
		return c.readParties(ctx, businesscentral.PartyVendor, since)
	default:
		return errChangeCursor
	}
}

func (c *changeCollector) readParties(
	ctx context.Context,
	kind businesscentral.PartyKind,
	since *time.Time,
) error {
	parties, err := c.client.Parties(ctx, kind, since)
	if err != nil {
		return err
	}
	referenceKind := referenceKindOf(kind)
	for idx := range parties {
		c.seen(parties[idx].LastModified)
		c.reference(partyObjectOf(referenceKind, &parties[idx]))
	}
	return nil
}

func (c *changeCollector) reference(obj *accountingsync.AccountingReferenceObject) {
	c.page.References = append(c.page.References, services.AccountingChangedReference{
		Object: obj,
	})
}

func (c *changeCollector) readDocuments(
	ctx context.Context,
	kind businesscentral.DocumentKind,
	since *time.Time,
) error {
	docs, err := c.client.Documents(ctx, kind, since)
	if err != nil {
		return err
	}
	types := documentObjectTypes(kind)
	for idx := range docs {
		doc := &docs[idx]
		c.seen(doc.LastModified)
		if doc.ID == "" {
			continue
		}
		operation := services.AccountingChangeUpsert
		if retired(doc.Status) {
			operation = services.AccountingChangeVoid
		}
		c.page.Documents = append(c.page.Documents, services.AccountingChangedDocument{
			ObjectTypes: types,
			ExternalID:  doc.ID,
			Operation:   operation,
			ModifiedAt:  unixOrZero(doc.LastModified),
		})
	}
	return nil
}

func documentObjectTypes(kind businesscentral.DocumentKind) []accountingsync.SyncObjectType {
	switch kind {
	case businesscentral.DocumentSalesInvoice:
		return []accountingsync.SyncObjectType{
			accountingsync.SyncObjectInvoice,
			accountingsync.SyncObjectDebitMemo,
		}
	case businesscentral.DocumentSalesCreditMemo:
		return []accountingsync.SyncObjectType{accountingsync.SyncObjectCreditMemo}
	case businesscentral.DocumentPurchaseInvoice, businesscentral.DocumentPurchaseCreditMemo:
		return []accountingsync.SyncObjectType{
			accountingsync.SyncObjectCarrierBill,
			accountingsync.SyncObjectDriverBill,
		}
	default:
		return nil
	}
}
