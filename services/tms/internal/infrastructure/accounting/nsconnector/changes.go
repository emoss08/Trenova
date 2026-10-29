package nsconnector

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsdocsync"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsrefsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountinginbound"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
)

var _ accountingconnectorports.AccountingChangeReader = (*Connector)(nil)

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
	spec       *nsbinding.ReferenceSpec
	reference  accountingprovider.ReferenceKind
	payment    accountinginbound.InboundChangeKind
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
		typeCode:   nsdocsync.TypeCustomerPayment,
		recordType: nsdocsync.RecordCustomerPayment,
		spec:       &transactionSpec,
		payment:    accountinginbound.InboundCustomerPayment,
		limit:      paymentsPerRead,
	},
	{
		name:       targetVendPayments,
		typeCode:   nsdocsync.TypeVendorPayment,
		recordType: nsdocsync.RecordVendorPayment,
		spec:       &transactionSpec,
		payment:    accountinginbound.InboundBillPayment,
		limit:      paymentsPerRead,
	},
	{
		name:     targetInvoices,
		typeCode: nsdocsync.TypeInvoice,
		spec:     &transactionSpec,
		objects:  salesObjects,
	},
	{
		name:     targetCreditMemos,
		typeCode: nsdocsync.TypeCreditMemo,
		spec:     &transactionSpec,
		objects:  creditObjects,
	},
	{
		name:     targetBills,
		typeCode: nsdocsync.TypeVendorBill,
		spec:     &transactionSpec,
		objects:  billObjects,
	},
	{
		name:     targetVendCredits,
		typeCode: nsdocsync.TypeVendorCredit,
		spec:     &transactionSpec,
		objects:  billObjects,
	},
	{
		name:      targetCustomers,
		spec:      &nsrefsync.CustomerSpec,
		reference: accountingprovider.ReferenceKindCustomer,
	},
	{
		name:      targetVendors,
		spec:      &nsrefsync.VendorSpec,
		reference: accountingprovider.ReferenceKindVendor,
	},
	{name: targetItems, spec: &nsrefsync.ItemSpec, reference: accountingprovider.ReferenceKindItem},
	{
		name:      targetAccounts,
		spec:      &nsrefsync.AccountSpec,
		reference: accountingprovider.ReferenceKindAccount,
	},
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
	positions map[string]nsbinding.Keyset
}

func (c *changeCursor) String() string {
	names := slices.Sorted(maps.Keys(c.positions))
	parts := make([]string, 0, len(names)+2)
	parts = append(parts, cursorVersion, strconv.FormatInt(c.since.Unix(), 10))
	for _, name := range names {
		pos := c.positions[name]
		parts = append(parts, name+targetAssign+strconv.FormatInt(pos.At.Unix(), 10)+
			positionSeparator+strconv.FormatInt(pos.ID, 10))
	}
	return strings.Join(parts, cursorSeparator)
}

func (c *changeCursor) position(name string) nsbinding.Keyset {
	if pos, ok := c.positions[name]; ok {
		return pos
	}
	return nsbinding.Keyset{At: c.since.Add(-timezoneMargin)}
}

func parseChangeCursor(raw string, now time.Time) (*changeCursor, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return &changeCursor{
			since:     now.UTC().Truncate(time.Second),
			positions: map[string]nsbinding.Keyset{},
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
		positions: make(map[string]nsbinding.Keyset, len(parts)-2),
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

func parsePosition(entry string) (string, nsbinding.Keyset, bool) {
	name, rest, found := strings.Cut(entry, targetAssign)
	if !found || lookupTarget(name) == nil {
		return "", nsbinding.Keyset{}, false
	}
	stamp, id, found := strings.Cut(rest, positionSeparator)
	if !found {
		return "", nsbinding.Keyset{}, false
	}
	at, atErr := strconv.ParseInt(stamp, 10, 64)
	internalID, idErr := strconv.ParseInt(id, 10, 64)
	if atErr != nil || idErr != nil || at <= 0 || internalID < 0 {
		return "", nsbinding.Keyset{}, false
	}
	return name, nsbinding.Keyset{At: time.Unix(at, 0).UTC(), ID: internalID}, true
}

func (c *Connector) ChangeFeedLimits() accountingconnectorports.AccountingChangeFeedLimits {
	return accountingconnectorports.AccountingChangeFeedLimits{
		MaxLookback:    0,
		ReportsDeletes: true,
		MaxPerRead:     maxChangesPerRead,
	}
}

func (c *Connector) ChangeCursorAt(at time.Time) string {
	cursor := &changeCursor{since: at.UTC().Truncate(time.Second)}
	return cursor.String()
}

func changeTargets(req *accountingconnectorports.ReadAccountingChangesRequest) []*targetInfo {
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
	req *accountingconnectorports.ReadAccountingChangesRequest,
) (*accountingconnectorports.AccountingChangePage, error) {
	now := req.Now.UTC()
	if req.Now.IsZero() {
		now = c.Clock().UTC()
	}
	cursor, err := parseChangeCursor(req.Cursor, now)
	if err != nil {
		return nil, err
	}
	collect := &changeCollector{
		targets: changeTargets(req),
		budget:  maxChangesPerRead,
		page: &accountingconnectorports.AccountingChangePage{
			Payments:   []accountingconnectorports.AccountingInboundPayment{},
			Documents:  []accountingconnectorports.AccountingChangedDocument{},
			References: []accountingconnectorports.AccountingChangedReference{},
		},
	}
	if len(collect.targets) > 0 {
		if collect.session, err = c.Open(req.Auth); err != nil {
			return nil, err
		}
		collect.editors = &nsbinding.EditorBook{Client: collect.session.Client}
		if err = collect.run(ctx, cursor); err != nil {
			return nil, err
		}
	}
	collect.page.NextCursor = cursor.String()
	return collect.page, nil
}

type changeCollector struct {
	session *nsbinding.Session
	editors *nsbinding.EditorBook
	targets []*targetInfo
	page    *accountingconnectorports.AccountingChangePage
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
	pos nsbinding.Keyset,
	limit int,
) (nsbinding.Keyset, bool, error) {
	switch {
	case target.name == targetDeleted:
		return c.readDeleted(ctx, pos, limit)
	case target.typeCode != "":
		return c.readTransactions(ctx, target, pos, limit)
	default:
		return c.readReferences(ctx, target, pos, limit)
	}
}

func advance(pos nsbinding.Keyset, at time.Time, id string) nsbinding.Keyset {
	internalID, ok := nsbinding.ParseInternalID(id)
	if !ok {
		return pos
	}
	if at.IsZero() || at.Before(pos.At) {
		at = pos.At
	}
	return nsbinding.Keyset{At: at.Truncate(time.Second), ID: internalID}
}

func (c *changeCollector) query(
	ctx context.Context,
	b *nsbinding.SQLBuilder,
	spec *nsbinding.ReferenceSpec,
	limit int,
	rows any,
) (*netsuite.QueryPage, error) {
	return c.session.Client.Query(ctx, netsuite.Query{
		SQL:    b.SQL(spec.SelectFrom, changeOrder(spec)),
		Params: b.Params,
		Limit:  limit,
	}, rows)
}

func (c *changeCollector) readTransactions(
	ctx context.Context,
	target *targetInfo,
	pos nsbinding.Keyset,
	limit int,
) (nsbinding.Keyset, bool, error) {
	b := changeQuery(&changeFilter{
		spec:         target.spec,
		pos:          pos,
		subsidiaryID: c.session.Ref.SubsidiaryID,
		typeCode:     target.typeCode,
	})
	var rows []nsbinding.TransactionRow
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
		pos = advance(pos, row.ModifiedAt(), row.IDValue())
		c.budget--
		if target.payment != "" {
			if err = c.payment(ctx, target, row, editors[row.IDValue()]); err != nil {
				return pos, false, err
			}
			continue
		}
		operation := accountingconnectorports.AccountingChangeUpsert
		if row.Voided.Value() {
			operation = accountingconnectorports.AccountingChangeVoid
		}
		c.page.Documents = append(
			c.page.Documents,
			accountingconnectorports.AccountingChangedDocument{
				ObjectTypes: target.objects,
				ExternalID:  row.IDValue(),
				Operation:   operation,
				ModifiedAt:  unixOrZero(row.ModifiedAt()),
				ModifiedBy:  editors[row.IDValue()],
			},
		)
	}
	return pos, page.HasMore && len(rows) > 0, nil
}

func (c *changeCollector) editorsOf(
	ctx context.Context,
	rows []nsbinding.TransactionRow,
) (map[string]string, error) {
	ids := make([]int64, 0, len(rows))
	for idx := range rows {
		if id, ok := nsbinding.ParseInternalID(rows[idx].IDValue()); ok {
			ids = append(ids, id)
		}
	}
	return c.editors.Names(ctx, ids)
}

func (c *changeCollector) payment(
	ctx context.Context,
	target *targetInfo,
	row *nsbinding.TransactionRow,
	modifiedBy string,
) error {
	var record nsbinding.TransactionRecord
	err := c.session.Client.Get(ctx, netsuite.RecordRead{
		Type:               target.recordType,
		ID:                 row.IDValue(),
		ExpandSubResources: true,
	}, &record)
	if err != nil {
		if netsuiteerrors.IsNotFound(err) {
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
	kind accountinginbound.InboundChangeKind,
	row *nsbinding.TransactionRow,
	record *nsbinding.TransactionRecord,
) accountingconnectorports.AccountingInboundPayment {
	party := record.Party()
	operation := accountingconnectorports.AccountingChangeUpsert
	if row.Voided.Value() {
		operation = accountingconnectorports.AccountingChangeVoid
	}
	total := record.Payment
	if kind == accountinginbound.InboundBillPayment || total.IsZero() {
		total = record.Total
	}
	if total.IsZero() {
		total = row.AbsTotal()
	}
	number := strings.TrimSpace(record.TranID)
	if number == "" {
		number = row.TranID.Value()
	}
	reference := strings.TrimSpace(record.CheckNum)
	if reference == "" {
		reference = number
	}
	out := accountingconnectorports.AccountingInboundPayment{
		Kind:              kind,
		Operation:         operation,
		ExternalID:        row.IDValue(),
		Number:            number,
		ModifiedAt:        unixOrZero(row.ModifiedAt()),
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
			[]accountingconnectorports.AccountingInboundLine,
			0,
			len(record.Apply.Items),
		),
	}
	for idx := range record.Apply.Items {
		line := &record.Apply.Items[idx]
		if !line.Apply.Value() || string(line.Doc) == "" {
			continue
		}
		out.Lines = append(out.Lines, accountingconnectorports.AccountingInboundLine{
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
	kind accountinginbound.InboundChangeKind,
	docType string,
) accountinginbound.InboundDocumentKind {
	switch compactName(docType) {
	case "invoice", "custinvc":
		return accountinginbound.InboundDocInvoice
	case "creditmemo", "custcred":
		return accountinginbound.InboundDocCreditMemo
	case "bill", "vendorbill", "vendbill":
		return accountinginbound.InboundDocBill
	case "vendorcredit", "billcredit", "vendcred":
		return accountinginbound.InboundDocVendorCredit
	case "":
		if kind == accountinginbound.InboundBillPayment {
			return accountinginbound.InboundDocBill
		}
		return accountinginbound.InboundDocInvoice
	default:
		return accountinginbound.InboundDocOther
	}
}

func (c *changeCollector) readReferences(
	ctx context.Context,
	target *targetInfo,
	pos nsbinding.Keyset,
	limit int,
) (nsbinding.Keyset, bool, error) {
	b := changeQuery(&changeFilter{
		spec:         target.spec,
		pos:          pos,
		subsidiaryID: c.session.Ref.SubsidiaryID,
	})
	var rows []nsbinding.ReferenceRow
	page, err := c.query(ctx, b, target.spec, limit, &rows)
	if err != nil {
		return pos, false, err
	}
	for idx := range rows {
		row := &rows[idx]
		pos = advance(pos, row.ModifiedAt(), row.ID.Value())
		c.budget--
		c.page.References = append(
			c.page.References,
			accountingconnectorports.AccountingChangedReference{
				Object: nsrefsync.ReferenceObjectOf(target.reference, row),
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
	pos nsbinding.Keyset,
	limit int,
) (nsbinding.Keyset, bool, error) {
	b := (&nsbinding.SQLBuilder{}).Where(
		keysetCondition(modifiedKey("dr.deleteddate"), "dr.recordid"),
		pos.Params()...,
	)
	var rows []deletedRow
	page, err := c.session.Client.Query(ctx, netsuite.Query{
		SQL:    b.SQL(deletedSQL, modifiedKey("dr.deleteddate")+", dr.recordid"),
		Params: b.Params,
		Limit:  limit,
	}, &rows)
	if err != nil {
		if netsuiteerrors.IsUnsupportedTable(err) || nsbinding.IsPermissionFault(err) {
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
	c.page.Documents = append(c.page.Documents, accountingconnectorports.AccountingChangedDocument{
		ObjectTypes: lookupTarget(name).objects,
		ExternalID:  id,
		Operation:   accountingconnectorports.AccountingChangeDelete,
		ModifiedAt:  unixOrZero(at),
	})
}

func (c *changeCollector) deletedPayment(name, id string, at time.Time) {
	if !c.wants(name) {
		return
	}
	c.page.Payments = append(c.page.Payments, accountingconnectorports.AccountingInboundPayment{
		Kind:       lookupTarget(name).payment,
		Operation:  accountingconnectorports.AccountingChangeDelete,
		ExternalID: id,
		ModifiedAt: unixOrZero(at),
		Lines:      []accountingconnectorports.AccountingInboundLine{},
	})
}

func (c *changeCollector) deletedReference(name, id string) {
	if !c.wants(name) {
		return
	}
	c.page.References = append(
		c.page.References,
		accountingconnectorports.AccountingChangedReference{
			Object: &accountingmapping.AccountingReferenceObject{
				Kind:       lookupTarget(name).reference,
				ExternalID: id,
			},
			Deleted: true,
		},
	)
}
