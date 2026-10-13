package nsconnector

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/shared/netsuite"
)

var _ accountingsyncports.AccountingChangeReader = (*Connector)(nil)

const (
	cursorVersion      = "n1"
	cursorSeparator    = ";"
	positionSeparator  = "|"
	targetAssign       = "="
	maxChangesPerRead  = netsuite.MaxQueryLimit
	paymentsPerRead    = 100
	timezoneMargin     = 14 * time.Hour
	targetInvoices     = "invoices"
	targetCreditMemos  = "creditMemos"
	targetCustPayments = "customerPayments"
	targetBills        = "vendorBills"
	targetVendCredits  = "vendorCredits"
	targetVendPayments = "vendorPayments"
	targetCustomers    = "customers"
	targetVendors      = "vendors"
	targetItems        = "items"
	targetAccounts     = "accounts"
	targetDeleted      = "deleted"
)

var errChangeCursor = errors.New("netsuite: the change cursor is not one this adapter wrote")

type targetInfo struct {
	name       string
	typeCode   string
	recordType string
	spec       *referenceSpec
	reference  accountingsync.ReferenceKind
	payment    accountingsync.InboundChangeKind
	objects    []accountingsync.SyncObjectType
	limit      int
}

var (
	salesObjects = []accountingsync.SyncObjectType{
		accountingsync.SyncObjectInvoice,
		accountingsync.SyncObjectDebitMemo,
	}
	creditObjects = []accountingsync.SyncObjectType{accountingsync.SyncObjectCreditMemo}
	billObjects   = []accountingsync.SyncObjectType{
		accountingsync.SyncObjectCarrierBill,
		accountingsync.SyncObjectDriverBill,
	}
)

var targetTable = []targetInfo{
	{
		name:       targetCustPayments,
		typeCode:   typeCustomerPayment,
		recordType: recordCustomerPayment,
		spec:       &transactionSpec,
		payment:    accountingsync.InboundCustomerPayment,
		limit:      paymentsPerRead,
	},
	{
		name:       targetVendPayments,
		typeCode:   typeVendorPayment,
		recordType: recordVendorPayment,
		spec:       &transactionSpec,
		payment:    accountingsync.InboundBillPayment,
		limit:      paymentsPerRead,
	},
	{name: targetInvoices, typeCode: typeInvoice, spec: &transactionSpec, objects: salesObjects},
	{
		name:     targetCreditMemos,
		typeCode: typeCreditMemo,
		spec:     &transactionSpec,
		objects:  creditObjects,
	},
	{name: targetBills, typeCode: typeVendorBill, spec: &transactionSpec, objects: billObjects},
	{
		name:     targetVendCredits,
		typeCode: typeVendorCredit,
		spec:     &transactionSpec,
		objects:  billObjects,
	},
	{name: targetCustomers, spec: &customerSpec, reference: accountingsync.ReferenceKindCustomer},
	{name: targetVendors, spec: &vendorSpec, reference: accountingsync.ReferenceKindVendor},
	{name: targetItems, spec: &itemSpec, reference: accountingsync.ReferenceKindItem},
	{name: targetAccounts, spec: &accountSpec, reference: accountingsync.ReferenceKindAccount},
	{name: targetDeleted},
}

func lookupTarget(name string) *targetInfo {
	for idx := range targetTable {
		if targetTable[idx].name == name {
			return &targetTable[idx]
		}
	}
	return nil
}

func (t *targetInfo) readLimit(budget int) int {
	if t.limit > 0 {
		return min(t.limit, budget)
	}
	return budget
}

type changeCursor struct {
	since     time.Time
	positions map[string]keyset
}

func (c *changeCursor) String() string {
	names := slices.Sorted(maps.Keys(c.positions))
	parts := make([]string, 0, len(names)+2)
	parts = append(parts, cursorVersion, strconv.FormatInt(c.since.Unix(), 10))
	for _, name := range names {
		pos := c.positions[name]
		parts = append(parts, name+targetAssign+strconv.FormatInt(pos.at.Unix(), 10)+
			positionSeparator+strconv.FormatInt(pos.id, 10))
	}
	return strings.Join(parts, cursorSeparator)
}

func (c *changeCursor) position(name string) keyset {
	if pos, ok := c.positions[name]; ok {
		return pos
	}
	return keyset{at: c.since.Add(-timezoneMargin)}
}

func parseChangeCursor(raw string, now time.Time) (*changeCursor, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return &changeCursor{
			since:     now.UTC().Truncate(time.Second),
			positions: map[string]keyset{},
		}, nil
	}
	parts := strings.Split(value, cursorSeparator)
	if len(parts) < 2 || parts[0] != cursorVersion {
		return nil, errChangeCursor
	}
	since, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || since <= 0 {
		return nil, errChangeCursor
	}
	cursor := &changeCursor{
		since:     time.Unix(since, 0).UTC(),
		positions: make(map[string]keyset, len(parts)-2),
	}
	for _, entry := range parts[2:] {
		name, pos, ok := parsePosition(entry)
		if !ok {
			return nil, errChangeCursor
		}
		cursor.positions[name] = pos
	}
	return cursor, nil
}

func parsePosition(entry string) (string, keyset, bool) {
	name, rest, found := strings.Cut(entry, targetAssign)
	if !found || lookupTarget(name) == nil {
		return "", keyset{}, false
	}
	stamp, id, found := strings.Cut(rest, positionSeparator)
	if !found {
		return "", keyset{}, false
	}
	at, atErr := strconv.ParseInt(stamp, 10, 64)
	internalID, idErr := strconv.ParseInt(id, 10, 64)
	if atErr != nil || idErr != nil || at <= 0 || internalID < 0 {
		return "", keyset{}, false
	}
	return name, keyset{at: time.Unix(at, 0).UTC(), id: internalID}, true
}

func (c *Connector) ChangeFeedLimits() accountingsyncports.AccountingChangeFeedLimits {
	return accountingsyncports.AccountingChangeFeedLimits{
		MaxLookback:    0,
		ReportsDeletes: true,
		MaxPerRead:     maxChangesPerRead,
	}
}

func (c *Connector) ChangeCursorAt(at time.Time) string {
	cursor := &changeCursor{since: at.UTC().Truncate(time.Second)}
	return cursor.String()
}

func changeTargets(req *accountingsyncports.ReadAccountingChangesRequest) []*targetInfo {
	wanted := make(map[string]bool, len(targetTable))
	wanted[targetCustPayments] = req.Payments
	wanted[targetVendPayments] = req.BillPayments
	for _, name := range []string{targetInvoices, targetCreditMemos, targetBills, targetVendCredits} {
		wanted[name] = req.Documents
	}
	for _, kind := range req.ReferenceKinds {
		for idx := range targetTable {
			if targetTable[idx].reference == kind && kind != "" {
				wanted[targetTable[idx].name] = true
			}
		}
	}
	out := make([]*targetInfo, 0, len(targetTable))
	for idx := range targetTable {
		if wanted[targetTable[idx].name] {
			out = append(out, &targetTable[idx])
		}
	}
	if len(out) > 0 {
		out = append(out, lookupTarget(targetDeleted))
	}
	return out
}

func (c *Connector) ReadChanges(
	ctx context.Context,
	req *accountingsyncports.ReadAccountingChangesRequest,
) (*accountingsyncports.AccountingChangePage, error) {
	now := req.Now.UTC()
	if req.Now.IsZero() {
		now = c.now().UTC()
	}
	cursor, err := parseChangeCursor(req.Cursor, now)
	if err != nil {
		return nil, err
	}
	collect := &changeCollector{
		targets: changeTargets(req),
		budget:  maxChangesPerRead,
		page: &accountingsyncports.AccountingChangePage{
			Payments:   []accountingsyncports.AccountingInboundPayment{},
			Documents:  []accountingsyncports.AccountingChangedDocument{},
			References: []accountingsyncports.AccountingChangedReference{},
		},
	}
	if len(collect.targets) > 0 {
		if collect.session, err = c.open(req.Auth); err != nil {
			return nil, err
		}
		collect.editors = &editorBook{client: collect.session.client}
		if err = collect.run(ctx, cursor); err != nil {
			return nil, err
		}
	}
	collect.page.NextCursor = cursor.String()
	return collect.page, nil
}

type changeCollector struct {
	session *session
	editors *editorBook
	targets []*targetInfo
	page    *accountingsyncports.AccountingChangePage
	budget  int
}

func (c *changeCollector) run(ctx context.Context, cursor *changeCursor) error {
	for _, target := range c.targets {
		limit := target.readLimit(c.budget)
		if limit <= 0 {
			c.page.More = true
			return nil
		}
		pos, more, err := c.read(ctx, target, cursor.position(target.name), limit)
		if err != nil {
			return err
		}
		cursor.positions[target.name] = pos
		if more {
			c.page.More = true
			return nil
		}
	}
	return nil
}

func (c *changeCollector) read(
	ctx context.Context,
	target *targetInfo,
	pos keyset,
	limit int,
) (keyset, bool, error) {
	switch {
	case target.name == targetDeleted:
		return c.readDeleted(ctx, pos, limit)
	case target.typeCode != "":
		return c.readTransactions(ctx, target, pos, limit)
	default:
		return c.readReferences(ctx, target, pos, limit)
	}
}

func advance(pos keyset, at time.Time, id string) keyset {
	internalID, ok := parseInternalID(id)
	if !ok {
		return pos
	}
	if at.IsZero() || at.Before(pos.at) {
		at = pos.at
	}
	return keyset{at: at.Truncate(time.Second), id: internalID}
}

func (c *changeCollector) query(
	ctx context.Context,
	b *sqlBuilder,
	spec *referenceSpec,
	limit int,
	rows any,
) (*netsuite.QueryPage, error) {
	return c.session.client.Query(ctx, netsuite.Query{
		SQL:    b.sql(spec.selectFrom, changeOrder(spec)),
		Params: b.params,
		Limit:  limit,
	}, rows)
}

func (c *changeCollector) readTransactions(
	ctx context.Context,
	target *targetInfo,
	pos keyset,
	limit int,
) (keyset, bool, error) {
	b := changeQuery(&changeFilter{
		spec:         target.spec,
		pos:          pos,
		subsidiaryID: c.session.ref.SubsidiaryID,
		typeCode:     target.typeCode,
	})
	var rows []transactionRow
	page, err := c.query(ctx, b, target.spec, limit, &rows)
	if err != nil {
		return pos, false, err
	}
	editors, err := c.editorsOf(ctx, rows)
	if err != nil {
		return pos, false, err
	}
	for idx := range rows {
		row := &rows[idx]
		pos = advance(pos, row.modified(), row.id())
		c.budget--
		if target.payment != "" {
			if err = c.payment(ctx, target, row, editors[row.id()]); err != nil {
				return pos, false, err
			}
			continue
		}
		operation := accountingsyncports.AccountingChangeUpsert
		if row.Voided.Value() {
			operation = accountingsyncports.AccountingChangeVoid
		}
		c.page.Documents = append(c.page.Documents, accountingsyncports.AccountingChangedDocument{
			ObjectTypes: target.objects,
			ExternalID:  row.id(),
			Operation:   operation,
			ModifiedAt:  unixOrZero(row.modified()),
			ModifiedBy:  editors[row.id()],
		})
	}
	return pos, page.HasMore && len(rows) > 0, nil
}

func (c *changeCollector) editorsOf(
	ctx context.Context,
	rows []transactionRow,
) (map[string]string, error) {
	ids := make([]int64, 0, len(rows))
	for idx := range rows {
		if id, ok := parseInternalID(rows[idx].id()); ok {
			ids = append(ids, id)
		}
	}
	return c.editors.names(ctx, ids)
}

func (c *changeCollector) payment(
	ctx context.Context,
	target *targetInfo,
	row *transactionRow,
	modifiedBy string,
) error {
	var record transactionRecord
	err := c.session.client.Get(ctx, netsuite.RecordRead{
		Type:               target.recordType,
		ID:                 row.id(),
		ExpandSubResources: true,
	}, &record)
	if err != nil {
		if netsuite.IsNotFound(err) {
			return nil
		}
		return err
	}
	payment := inboundPaymentOf(target.payment, row, &record)
	payment.ModifiedBy = modifiedBy
	c.page.Payments = append(c.page.Payments, payment)
	return nil
}

func inboundPaymentOf(
	kind accountingsync.InboundChangeKind,
	row *transactionRow,
	record *transactionRecord,
) accountingsyncports.AccountingInboundPayment {
	party := record.party()
	operation := accountingsyncports.AccountingChangeUpsert
	if row.Voided.Value() {
		operation = accountingsyncports.AccountingChangeVoid
	}
	total := record.Payment
	if kind == accountingsync.InboundBillPayment || total.IsZero() {
		total = record.Total
	}
	if total.IsZero() {
		total = row.total()
	}
	number := strings.TrimSpace(record.TranID)
	if number == "" {
		number = row.TranID.Value()
	}
	reference := strings.TrimSpace(record.CheckNum)
	if reference == "" {
		reference = number
	}
	out := accountingsyncports.AccountingInboundPayment{
		Kind:              kind,
		Operation:         operation,
		ExternalID:        row.id(),
		Number:            number,
		ModifiedAt:        unixOrZero(row.modified()),
		PartyExternalID:   party.ID.Value(),
		PartyName:         party.RefName,
		TxnDate:           record.TranDate,
		CurrencyCode:      strings.ToUpper(row.Currency.Value()),
		Amount:            total.Abs(),
		Unapplied:         record.Unapplied.Abs(),
		MethodExternalID:  record.PaymentMethod.ID.Value(),
		MethodName:        record.PaymentMethod.RefName,
		AccountExternalID: record.Account.ID.Value(),
		ReferenceNumber:   reference,
		Lines: make(
			[]accountingsyncports.AccountingInboundLine,
			0,
			len(record.Apply.Items),
		),
	}
	for idx := range record.Apply.Items {
		line := &record.Apply.Items[idx]
		if !line.Apply.Value() || string(line.Doc) == "" {
			continue
		}
		out.Lines = append(out.Lines, accountingsyncports.AccountingInboundLine{
			DocumentKind:       inboundDocumentKind(kind, line.Type),
			DocumentExternalID: string(line.Doc),
			Amount:             line.Amount.Abs(),
		})
	}
	return out
}

func compactName(raw string) string {
	return strings.ToLower(strings.NewReplacer(" ", "", "_", "", "-", "").Replace(raw))
}

func inboundDocumentKind(
	kind accountingsync.InboundChangeKind,
	docType string,
) accountingsync.InboundDocumentKind {
	switch compactName(docType) {
	case "invoice", "custinvc":
		return accountingsync.InboundDocInvoice
	case "creditmemo", "custcred":
		return accountingsync.InboundDocCreditMemo
	case "bill", "vendorbill", "vendbill":
		return accountingsync.InboundDocBill
	case "vendorcredit", "billcredit", "vendcred":
		return accountingsync.InboundDocVendorCredit
	case "":
		if kind == accountingsync.InboundBillPayment {
			return accountingsync.InboundDocBill
		}
		return accountingsync.InboundDocInvoice
	default:
		return accountingsync.InboundDocOther
	}
}

func (c *changeCollector) readReferences(
	ctx context.Context,
	target *targetInfo,
	pos keyset,
	limit int,
) (keyset, bool, error) {
	b := changeQuery(&changeFilter{
		spec:         target.spec,
		pos:          pos,
		subsidiaryID: c.session.ref.SubsidiaryID,
	})
	var rows []referenceRow
	page, err := c.query(ctx, b, target.spec, limit, &rows)
	if err != nil {
		return pos, false, err
	}
	for idx := range rows {
		row := &rows[idx]
		pos = advance(pos, row.modified(), row.ID.Value())
		c.budget--
		c.page.References = append(
			c.page.References,
			accountingsyncports.AccountingChangedReference{
				Object: referenceObjectOf(target.reference, row),
			},
		)
	}
	return pos, page.HasMore && len(rows) > 0, nil
}

func (c *changeCollector) wants(name string) bool {
	return slices.ContainsFunc(c.targets, func(t *targetInfo) bool { return t.name == name })
}

func (c *changeCollector) readDeleted(
	ctx context.Context,
	pos keyset,
	limit int,
) (keyset, bool, error) {
	b := (&sqlBuilder{}).where(
		keysetCondition(modifiedKey("dr.deleteddate"), "dr.recordid"),
		pos.params()...,
	)
	var rows []deletedRow
	page, err := c.session.client.Query(ctx, netsuite.Query{
		SQL:    b.sql(deletedSQL, modifiedKey("dr.deleteddate")+", dr.recordid"),
		Params: b.params,
		Limit:  limit,
	}, &rows)
	if err != nil {
		if netsuite.IsUnsupportedTable(err) || isPermissionFault(err) {
			c.page.DeletesUnread = true
			return pos, false, nil
		}
		return pos, false, err
	}
	for idx := range rows {
		row := &rows[idx]
		at, _ := netsuite.ParseQueryTime(row.Modified.Value())
		pos = advance(pos, at, row.ID.Value())
		c.budget--
		c.deleted(row.ID.Value(), row.RecordType.Value(), at)
	}
	return pos, page.HasMore && len(rows) > 0, nil
}

func (c *changeCollector) deleted(id, recordType string, at time.Time) {
	switch compactName(recordType) {
	case "invoice", "custinvc":
		c.deletedDocument(targetInvoices, id, at)
	case "creditmemo", "custcred":
		c.deletedDocument(targetCreditMemos, id, at)
	case "vendorbill", "vendbill":
		c.deletedDocument(targetBills, id, at)
	case "vendorcredit", "vendcred":
		c.deletedDocument(targetVendCredits, id, at)
	case "customerpayment", "custpymt", "payment":
		c.deletedPayment(targetCustPayments, id, at)
	case "vendorpayment", "vendpymt", "billpayment":
		c.deletedPayment(targetVendPayments, id, at)
	case "customer":
		c.deletedReference(targetCustomers, id)
	case "vendor":
		c.deletedReference(targetVendors, id)
	case "account":
		c.deletedReference(targetAccounts, id)
	case "item", "serviceitem", "servicesaleitem", "noninventoryitem", "noninventorysaleitem",
		"otherchargeitem", "otherchargesaleitem":
		c.deletedReference(targetItems, id)
	}
}

func (c *changeCollector) deletedDocument(name, id string, at time.Time) {
	if !c.wants(name) {
		return
	}
	c.page.Documents = append(c.page.Documents, accountingsyncports.AccountingChangedDocument{
		ObjectTypes: lookupTarget(name).objects,
		ExternalID:  id,
		Operation:   accountingsyncports.AccountingChangeDelete,
		ModifiedAt:  unixOrZero(at),
	})
}

func (c *changeCollector) deletedPayment(name, id string, at time.Time) {
	if !c.wants(name) {
		return
	}
	c.page.Payments = append(c.page.Payments, accountingsyncports.AccountingInboundPayment{
		Kind:       lookupTarget(name).payment,
		Operation:  accountingsyncports.AccountingChangeDelete,
		ExternalID: id,
		ModifiedAt: unixOrZero(at),
		Lines:      []accountingsyncports.AccountingInboundLine{},
	})
}

func (c *changeCollector) deletedReference(name, id string) {
	if !c.wants(name) {
		return
	}
	c.page.References = append(c.page.References, accountingsyncports.AccountingChangedReference{
		Object: &accountingsync.AccountingReferenceObject{
			Kind:       lookupTarget(name).reference,
			ExternalID: id,
		},
		Deleted: true,
	})
}
