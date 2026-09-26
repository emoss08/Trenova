package quickbooks

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
)

type ChangeEntity string

const (
	ChangePayment      = ChangeEntity("Payment")
	ChangeBillPayment  = ChangeEntity("BillPayment")
	ChangeInvoice      = ChangeEntity("Invoice")
	ChangeCreditMemo   = ChangeEntity("CreditMemo")
	ChangeBill         = ChangeEntity("Bill")
	ChangeVendorCredit = ChangeEntity("VendorCredit")
)

const (
	MaxChangeLookback  = 30 * 24 * time.Hour
	MaxChangesPerCall  = 1000
	statusDeleted      = "Deleted"
	statusVoided       = "Voided"
	voidedNotePrefix   = "Voided"
	changedSinceLayout = time.RFC3339
	queryResource      = "query"
)

var (
	ErrNoChangeEntities    = errors.New("quickbooks: at least one entity is required")
	ErrUnknownChangeEntity = errors.New("quickbooks: unknown change entity")
	ErrChangeWindow        = errors.New(
		"quickbooks: change data capture only looks back 30 days",
	)
)

func ReferenceChangeEntity(kind ReferenceKind) ChangeEntity {
	return ChangeEntity(kind)
}

func DocumentChangeEntity(kind TxnKind) ChangeEntity {
	return ChangeEntity(kind)
}

func (e ChangeEntity) IsValid() bool {
	switch e {
	case ChangePayment,
		ChangeBillPayment,
		ChangeInvoice,
		ChangeCreditMemo,
		ChangeBill,
		ChangeVendorCredit:
		return true
	default:
		return ReferenceKind(e).IsValid()
	}
}

func (e ChangeEntity) Document() (TxnKind, bool) {
	switch e {
	case ChangeInvoice, ChangeCreditMemo, ChangeBill, ChangeVendorCredit:
		return TxnKind(e), true
	case ChangePayment, ChangeBillPayment:
		return "", false
	default:
		return "", false
	}
}

func (e ChangeEntity) Reference() (ReferenceKind, bool) {
	kind := ReferenceKind(e)
	return kind, kind.IsValid()
}

type ChangedLink struct {
	TxnID   string
	TxnType string
	Amount  decimal.Decimal
}

type ChangeMeta struct {
	ID             string
	LastUpdatedAt  int64
	LastModifiedBy string
	Deleted        bool
	Voided         bool
}

type ChangedPayment struct {
	ChangeMeta
	CustomerID       string
	CustomerName     string
	TotalAmount      decimal.Decimal
	UnappliedAmount  decimal.Decimal
	TxnDate          string
	CurrencyCode     string
	MethodID         string
	MethodName       string
	DepositAccountID string
	ReferenceNumber  string
	PrivateNote      string
	Lines            []ChangedLink
}

type ChangedBillPayment struct {
	ChangeMeta
	DocNumber    string
	VendorID     string
	VendorName   string
	TotalAmount  decimal.Decimal
	TxnDate      string
	CurrencyCode string
	PayType      string
	AccountID    string
	PrivateNote  string
	Lines        []ChangedLink
}

type ChangedReference struct {
	ReferenceObject
	Deleted bool
}

type ChangedDocument struct {
	ChangeMeta
	Kind        TxnKind
	DocNumber   string
	TotalAmount decimal.Decimal
}

type ChangeSet struct {
	Payments     []ChangedPayment
	BillPayments []ChangedBillPayment
	Documents    []ChangedDocument
	References   []ChangedReference
	Full         []ChangeEntity
}

func (s *ChangeSet) merge(other *ChangeSet) {
	s.Documents = append(s.Documents, other.Documents...)
	s.Payments = append(s.Payments, other.Payments...)
	s.BillPayments = append(s.BillPayments, other.BillPayments...)
	s.References = append(s.References, other.References...)
	s.Full = append(s.Full, other.Full...)
}

type namedRef struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

func (r *namedRef) value() string {
	if r == nil {
		return ""
	}
	return r.Value
}

func (r *namedRef) name() string {
	if r == nil {
		return ""
	}
	return r.Name
}

type changeMetaData struct {
	LastUpdatedTime   string    `json:"LastUpdatedTime"`
	LastModifiedByRef *namedRef `json:"LastModifiedByRef"`
}

type wireChangeLine struct {
	Amount    decimal.Decimal `json:"Amount"`
	LinkedTxn []linkedTxn     `json:"LinkedTxn"`
}

type wireChangeTxn struct {
	ID                  string           `json:"Id"`
	Status              string           `json:"status"`
	DocNumber           string           `json:"DocNumber"`
	TxnDate             string           `json:"TxnDate"`
	TotalAmt            decimal.Decimal  `json:"TotalAmt"`
	UnappliedAmt        decimal.Decimal  `json:"UnappliedAmt"`
	CustomerRef         *namedRef        `json:"CustomerRef"`
	VendorRef           *namedRef        `json:"VendorRef"`
	CurrencyRef         *namedRef        `json:"CurrencyRef"`
	PaymentMethodRef    *namedRef        `json:"PaymentMethodRef"`
	DepositToAccountRef *namedRef        `json:"DepositToAccountRef"`
	PaymentRefNum       string           `json:"PaymentRefNum"`
	PrivateNote         string           `json:"PrivateNote"`
	PayType             string           `json:"PayType"`
	CheckPayment        *wireBankPayment `json:"CheckPayment"`
	CreditCardPayment   *wireCardPayment `json:"CreditCardPayment"`
	Line                []wireChangeLine `json:"Line"`
	MetaData            *changeMetaData  `json:"MetaData"`
}

type wireBankPayment struct {
	BankAccountRef *namedRef `json:"BankAccountRef"`
}

type wireCardPayment struct {
	CCAccountRef *namedRef `json:"CCAccountRef"`
}

type wireChangeEntity struct {
	wireEntity
	Status string `json:"status"`
}

type changeQueryResponse struct {
	Payment       []wireChangeTxn    `json:"Payment"`
	BillPayment   []wireChangeTxn    `json:"BillPayment"`
	Invoice       []wireChangeTxn    `json:"Invoice"`
	CreditMemo    []wireChangeTxn    `json:"CreditMemo"`
	Bill          []wireChangeTxn    `json:"Bill"`
	VendorCredit  []wireChangeTxn    `json:"VendorCredit"`
	Account       []wireChangeEntity `json:"Account"`
	Item          []wireChangeEntity `json:"Item"`
	Customer      []wireChangeEntity `json:"Customer"`
	Vendor        []wireChangeEntity `json:"Vendor"`
	Term          []wireChangeEntity `json:"Term"`
	PaymentMethod []wireChangeEntity `json:"PaymentMethod"`
}

type cdcEnvelope struct {
	CDCResponse []struct {
		QueryResponse []changeQueryResponse `json:"QueryResponse"`
	} `json:"CDCResponse"`
}

type changeQueryEnvelope struct {
	QueryResponse changeQueryResponse `json:"QueryResponse"`
}

func (c *Client) ChangesSince(
	ctx context.Context,
	entities []ChangeEntity,
	since time.Time,
	now time.Time,
) (*ChangeSet, error) {
	if len(entities) == 0 {
		return nil, ErrNoChangeEntities
	}
	names := make([]string, 0, len(entities))
	for _, entity := range entities {
		if !entity.IsValid() {
			return nil, ErrUnknownChangeEntity
		}
		names = append(names, string(entity))
	}
	if now.Sub(since) > MaxChangeLookback {
		return nil, ErrChangeWindow
	}

	query := c.query()
	query.Set("entities", strings.Join(names, ","))
	query.Set("changedSince", since.UTC().Format(changedSinceLayout))

	var out cdcEnvelope
	if _, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: "cdc",
		Method:   http.MethodGet,
		Path:     c.companyPath("cdc"),
		Query:    query,
		Out:      &out,
	}); err != nil {
		return nil, err
	}

	set := &ChangeSet{}
	for idx := range out.CDCResponse {
		for respIdx := range out.CDCResponse[idx].QueryResponse {
			set.merge(decodeChanges(&out.CDCResponse[idx].QueryResponse[respIdx]))
		}
	}
	return set, nil
}

func (c *Client) QueryChangedSince(
	ctx context.Context,
	entity ChangeEntity,
	since time.Time,
	startPosition, pageSize int,
) (*ChangeSet, int, error) {
	if !entity.IsValid() {
		return nil, 0, ErrUnknownChangeEntity
	}
	if startPosition < 1 || pageSize < 1 || pageSize > MaxQueryResults {
		return nil, 0, ErrInvalidPaging
	}

	statement := "select * from " + string(entity) +
		" where MetaData.LastUpdatedTime >= '" + since.UTC().Format(changedSinceLayout) + "'"
	if _, reference := entity.Reference(); reference {
		statement += " and Active in (true, false)"
	}
	statement += " orderby MetaData.LastUpdatedTime startposition " +
		strconv.Itoa(startPosition) + " maxresults " + strconv.Itoa(pageSize)

	query := c.query()
	query.Set(queryResource, statement)

	var out changeQueryEnvelope
	if _, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: queryResource,
		Method:   http.MethodGet,
		Path:     c.companyPath(queryResource),
		Query:    query,
		Out:      &out,
	}); err != nil {
		return nil, 0, err
	}

	set := decodeChanges(&out.QueryResponse)
	set.Full = nil
	next := 0
	if set.count() == pageSize {
		next = startPosition + pageSize
	}
	return set, next, nil
}

func (s *ChangeSet) count() int {
	return len(s.Payments) + len(s.BillPayments) + len(s.Documents) + len(s.References)
}

func decodeChanges(resp *changeQueryResponse) *ChangeSet {
	set := &ChangeSet{
		Payments:     make([]ChangedPayment, 0, len(resp.Payment)),
		BillPayments: make([]ChangedBillPayment, 0, len(resp.BillPayment)),
	}
	for idx := range resp.Payment {
		set.Payments = append(set.Payments, resp.Payment[idx].payment())
	}
	for idx := range resp.BillPayment {
		set.BillPayments = append(set.BillPayments, resp.BillPayment[idx].billPayment())
	}
	for _, kind := range documentChangeKinds() {
		wire := resp.documents(kind)
		for idx := range wire {
			set.Documents = append(set.Documents, wire[idx].document(kind))
		}
		if len(wire) >= MaxChangesPerCall {
			set.Full = append(set.Full, DocumentChangeEntity(kind))
		}
	}
	for _, kind := range AllReferenceKinds() {
		wire := resp.references(kind)
		for idx := range wire {
			set.References = append(set.References, wire[idx].reference(kind))
		}
	}
	if len(resp.Payment) >= MaxChangesPerCall {
		set.Full = append(set.Full, ChangePayment)
	}
	if len(resp.BillPayment) >= MaxChangesPerCall {
		set.Full = append(set.Full, ChangeBillPayment)
	}
	for _, kind := range AllReferenceKinds() {
		if len(resp.references(kind)) >= MaxChangesPerCall {
			set.Full = append(set.Full, ReferenceChangeEntity(kind))
		}
	}
	return set
}

func documentChangeKinds() []TxnKind {
	return []TxnKind{TxnInvoice, TxnCreditMemo, TxnBill, TxnVendorCredit}
}

func (r *changeQueryResponse) documents(kind TxnKind) []wireChangeTxn {
	switch kind {
	case TxnInvoice:
		return r.Invoice
	case TxnCreditMemo:
		return r.CreditMemo
	case TxnBill:
		return r.Bill
	case TxnVendorCredit:
		return r.VendorCredit
	case TxnPayment, TxnBillPayment:
		return nil
	default:
		return nil
	}
}

func (w *wireChangeTxn) document(kind TxnKind) ChangedDocument {
	return ChangedDocument{
		ChangeMeta:  w.MetaData.meta(w.ID, w.Status, w.PrivateNote, w.TotalAmt),
		Kind:        kind,
		DocNumber:   w.DocNumber,
		TotalAmount: w.TotalAmt,
	}
}

func (r *changeQueryResponse) references(kind ReferenceKind) []wireChangeEntity {
	switch kind {
	case KindAccount:
		return r.Account
	case KindItem:
		return r.Item
	case KindCustomer:
		return r.Customer
	case KindVendor:
		return r.Vendor
	case KindTerm:
		return r.Term
	case KindPaymentMethod:
		return r.PaymentMethod
	default:
		return nil
	}
}

func (m *changeMetaData) meta(id, status, privateNote string, total decimal.Decimal) ChangeMeta {
	out := ChangeMeta{
		ID:      id,
		Deleted: strings.EqualFold(status, statusDeleted),
		Voided: strings.EqualFold(status, statusVoided) ||
			(total.IsZero() && strings.HasPrefix(strings.TrimSpace(privateNote), voidedNotePrefix)),
	}
	if m == nil {
		return out
	}
	if updated, err := time.Parse(time.RFC3339, m.LastUpdatedTime); err == nil {
		out.LastUpdatedAt = updated.Unix()
	}
	out.LastModifiedBy = m.LastModifiedByRef.name()
	if out.LastModifiedBy == "" {
		out.LastModifiedBy = m.LastModifiedByRef.value()
	}
	return out
}

func changedLinks(lines []wireChangeLine) []ChangedLink {
	links := make([]ChangedLink, 0, len(lines))
	for idx := range lines {
		for _, linked := range lines[idx].LinkedTxn {
			links = append(links, ChangedLink{
				TxnID:   linked.TxnID,
				TxnType: linked.TxnType,
				Amount:  lines[idx].Amount,
			})
		}
	}
	return links
}

func (w *wireChangeTxn) payment() ChangedPayment {
	return ChangedPayment{
		ChangeMeta:       w.MetaData.meta(w.ID, w.Status, w.PrivateNote, w.TotalAmt),
		CustomerID:       w.CustomerRef.value(),
		CustomerName:     w.CustomerRef.name(),
		TotalAmount:      w.TotalAmt,
		UnappliedAmount:  w.UnappliedAmt,
		TxnDate:          w.TxnDate,
		CurrencyCode:     w.CurrencyRef.value(),
		MethodID:         w.PaymentMethodRef.value(),
		MethodName:       w.PaymentMethodRef.name(),
		DepositAccountID: w.DepositToAccountRef.value(),
		ReferenceNumber:  w.PaymentRefNum,
		PrivateNote:      w.PrivateNote,
		Lines:            changedLinks(w.Line),
	}
}

func (w *wireChangeTxn) billPayment() ChangedBillPayment {
	out := ChangedBillPayment{
		ChangeMeta:   w.MetaData.meta(w.ID, w.Status, w.PrivateNote, w.TotalAmt),
		DocNumber:    w.DocNumber,
		VendorID:     w.VendorRef.value(),
		VendorName:   w.VendorRef.name(),
		TotalAmount:  w.TotalAmt,
		TxnDate:      w.TxnDate,
		CurrencyCode: w.CurrencyRef.value(),
		PayType:      w.PayType,
		PrivateNote:  w.PrivateNote,
		Lines:        changedLinks(w.Line),
	}
	switch {
	case w.CheckPayment != nil:
		out.AccountID = w.CheckPayment.BankAccountRef.value()
	case w.CreditCardPayment != nil:
		out.AccountID = w.CreditCardPayment.CCAccountRef.value()
	}
	return out
}

func (w *wireChangeEntity) reference(kind ReferenceKind) ChangedReference {
	return ChangedReference{
		ReferenceObject: w.toReference(kind),
		Deleted:         strings.EqualFold(w.Status, statusDeleted),
	}
}
