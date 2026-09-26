package quickbooks

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
)

type TxnKind string

const (
	TxnInvoice      = TxnKind("Invoice")
	TxnCreditMemo   = TxnKind("CreditMemo")
	TxnPayment      = TxnKind("Payment")
	TxnBill         = TxnKind("Bill")
	TxnVendorCredit = TxnKind("VendorCredit")
	TxnBillPayment  = TxnKind("BillPayment")
)

const (
	MaxDocNumberLength     = 21
	MaxPrivateNoteLength   = 4000
	MaxCustomerMemoLength  = 1000
	MaxLineDescription     = 4000
	lineDetailSalesItem    = "SalesItemLineDetail"
	faultStaleObject       = "5010"
	faultObjectNotFound    = "610"
	faultDuplicateDocument = "6140"
	faultClosedPeriod      = "6210"
	faultInvalidReference  = "2500"
)

var (
	ErrUnknownTxnKind   = errors.New("quickbooks: unknown transaction kind")
	ErrTxnIDRequired    = errors.New("quickbooks: a transaction id is required")
	ErrCustomerRequired = errors.New("quickbooks: a transaction needs a customer")
	ErrLinesRequired    = errors.New("quickbooks: a transaction needs at least one line")
	ErrItemRequired     = errors.New("quickbooks: every sales line needs an item")
	ErrDocNumberTooLong = errors.New(
		"quickbooks: a document number is at most 21 characters",
	)
	ErrNegativeAmount = errors.New("quickbooks: transaction amounts cannot be negative")
)

func (k TxnKind) IsValid() bool {
	switch k {
	case TxnInvoice, TxnCreditMemo, TxnPayment, TxnBill, TxnVendorCredit, TxnBillPayment:
		return true
	default:
		return false
	}
}

func (k TxnKind) resource() string {
	return strings.ToLower(string(k))
}

func (k TxnKind) AppPath() string {
	switch k {
	case TxnInvoice:
		return "/app/invoice?txnId="
	case TxnCreditMemo:
		return "/app/creditmemo?txnId="
	case TxnPayment:
		return "/app/recvpayment?txnId="
	case TxnBill:
		return "/app/bill?txnId="
	case TxnVendorCredit:
		return "/app/vendorcredit?txnId="
	case TxnBillPayment:
		return "/app/billpayment?txnId="
	default:
		return ""
	}
}

func (k TxnKind) numbered() bool {
	switch k {
	case TxnInvoice, TxnCreditMemo, TxnBill, TxnVendorCredit:
		return true
	case TxnPayment, TxnBillPayment:
		return false
	default:
		return false
	}
}

func CustomerAppPath() string {
	return "/app/customerdetail?nameId="
}

func VendorAppPath() string {
	return "/app/vendordetail?nameId="
}

type SalesLine struct {
	Description string
	ItemID      string
	Quantity    decimal.Decimal
	UnitPrice   decimal.Decimal
	Amount      decimal.Decimal
	ServiceDate string
}

type SalesTxn struct {
	CustomerID   string
	DocNumber    string
	TxnDate      string
	DueDate      string
	TermID       string
	CurrencyCode string
	ExchangeRate decimal.Decimal
	PrivateNote  string
	CustomerMemo string
	Lines        []SalesLine
}

type PaymentLink struct {
	TxnID   string
	TxnKind TxnKind
	Amount  decimal.Decimal
}

type PaymentTxn struct {
	CustomerID       string
	TxnDate          string
	CurrencyCode     string
	ExchangeRate     decimal.Decimal
	PaymentMethodID  string
	DepositAccountID string
	PaymentRefNum    string
	PrivateNote      string
	TotalAmount      decimal.Decimal
	Links            []PaymentLink
}

type TxnResult struct {
	Kind        TxnKind
	ID          string
	DocNumber   string
	SyncToken   string
	TotalAmount decimal.Decimal
	Status      string
}

type money decimal.Decimal

func (m money) MarshalJSON() ([]byte, error) {
	return []byte(decimal.Decimal(m).StringFixed(2)), nil
}

type exchangeRate decimal.Decimal

func (r exchangeRate) MarshalJSON() ([]byte, error) {
	return []byte(decimal.Decimal(r).String()), nil
}

func exchangeRateOf(rate decimal.Decimal) *exchangeRate {
	if !rate.IsPositive() {
		return nil
	}
	value := exchangeRate(rate)
	return &value
}

type quantity decimal.Decimal

func (q quantity) MarshalJSON() ([]byte, error) {
	return []byte(decimal.Decimal(q).String()), nil
}

type memoValue struct {
	Value string `json:"value"`
}

type salesItemDetail struct {
	ItemRef     refValue  `json:"ItemRef"`
	Qty         *quantity `json:"Qty,omitempty"`
	UnitPrice   *quantity `json:"UnitPrice,omitempty"`
	ServiceDate string    `json:"ServiceDate,omitempty"`
}

type wireSalesLine struct {
	DetailType          string          `json:"DetailType"`
	Amount              money           `json:"Amount"`
	Description         string          `json:"Description,omitempty"`
	SalesItemLineDetail salesItemDetail `json:"SalesItemLineDetail"`
}

type salesBody struct {
	ID           string          `json:"Id,omitempty"`
	SyncToken    string          `json:"SyncToken,omitempty"`
	CustomerRef  refValue        `json:"CustomerRef"`
	DocNumber    string          `json:"DocNumber,omitempty"`
	TxnDate      string          `json:"TxnDate,omitempty"`
	DueDate      string          `json:"DueDate,omitempty"`
	SalesTermRef *refValue       `json:"SalesTermRef,omitempty"`
	CurrencyRef  *refValue       `json:"CurrencyRef,omitempty"`
	ExchangeRate *exchangeRate   `json:"ExchangeRate,omitempty"`
	PrivateNote  string          `json:"PrivateNote,omitempty"`
	CustomerMemo *memoValue      `json:"CustomerMemo,omitempty"`
	Line         []wireSalesLine `json:"Line"`
}

type linkedTxn struct {
	TxnID   string `json:"TxnId"`
	TxnType string `json:"TxnType"`
}

type wirePaymentLine struct {
	Amount    money       `json:"Amount"`
	LinkedTxn []linkedTxn `json:"LinkedTxn"`
}

type paymentBody struct {
	ID                  string            `json:"Id,omitempty"`
	SyncToken           string            `json:"SyncToken,omitempty"`
	CustomerRef         refValue          `json:"CustomerRef"`
	TotalAmt            money             `json:"TotalAmt"`
	TxnDate             string            `json:"TxnDate,omitempty"`
	CurrencyRef         *refValue         `json:"CurrencyRef,omitempty"`
	ExchangeRate        *exchangeRate     `json:"ExchangeRate,omitempty"`
	PaymentMethodRef    *refValue         `json:"PaymentMethodRef,omitempty"`
	DepositToAccountRef *refValue         `json:"DepositToAccountRef,omitempty"`
	PaymentRefNum       string            `json:"PaymentRefNum,omitempty"`
	PrivateNote         string            `json:"PrivateNote,omitempty"`
	Line                []wirePaymentLine `json:"Line"`
}

type entityRef struct {
	ID        string `json:"Id"`
	SyncToken string `json:"SyncToken"`
	Sparse    *bool  `json:"sparse,omitempty"`
}

type sparsePartyBody struct {
	entityRef
	partyBody
}

type wireTxn struct {
	ID        string          `json:"Id"`
	DocNumber string          `json:"DocNumber"`
	SyncToken string          `json:"SyncToken"`
	TotalAmt  decimal.Decimal `json:"TotalAmt"`
	Status    string          `json:"status"`
}

type txnEnvelope struct {
	Invoice      *wireTxn    `json:"Invoice"`
	CreditMemo   *wireTxn    `json:"CreditMemo"`
	Payment      *wireTxn    `json:"Payment"`
	Bill         *wireTxn    `json:"Bill"`
	VendorCredit *wireTxn    `json:"VendorCredit"`
	BillPayment  *wireTxn    `json:"BillPayment"`
	Customer     *wireEntity `json:"Customer"`
	Vendor       *wireEntity `json:"Vendor"`
}

type txnQueryEnvelope struct {
	QueryResponse struct {
		Invoice      []wireTxn `json:"Invoice"`
		CreditMemo   []wireTxn `json:"CreditMemo"`
		Bill         []wireTxn `json:"Bill"`
		VendorCredit []wireTxn `json:"VendorCredit"`
	} `json:"QueryResponse"`
}

func (e *txnEnvelope) txn(kind TxnKind) *wireTxn {
	switch kind {
	case TxnInvoice:
		return e.Invoice
	case TxnCreditMemo:
		return e.CreditMemo
	case TxnPayment:
		return e.Payment
	case TxnBill:
		return e.Bill
	case TxnVendorCredit:
		return e.VendorCredit
	case TxnBillPayment:
		return e.BillPayment
	default:
		return nil
	}
}

func (e *txnEnvelope) party(kind ReferenceKind) *wireEntity {
	switch kind {
	case KindCustomer:
		return e.Customer
	case KindVendor:
		return e.Vendor
	case KindAccount, KindItem, KindTerm, KindPaymentMethod:
		return nil
	default:
		return nil
	}
}

func (e *txnQueryEnvelope) matches(kind TxnKind) []wireTxn {
	switch kind {
	case TxnInvoice:
		return e.QueryResponse.Invoice
	case TxnCreditMemo:
		return e.QueryResponse.CreditMemo
	case TxnBill:
		return e.QueryResponse.Bill
	case TxnVendorCredit:
		return e.QueryResponse.VendorCredit
	case TxnPayment, TxnBillPayment:
		return nil
	default:
		return nil
	}
}

func (w *wireTxn) result(kind TxnKind) *TxnResult {
	return &TxnResult{
		Kind:        kind,
		ID:          w.ID,
		DocNumber:   w.DocNumber,
		SyncToken:   w.SyncToken,
		TotalAmount: w.TotalAmt,
		Status:      w.Status,
	}
}

func (c *Client) CreateInvoice(
	ctx context.Context,
	requestID string,
	txn *SalesTxn,
) (*TxnResult, error) {
	return c.createSales(ctx, requestID, TxnInvoice, txn)
}

func (c *Client) CreateCreditMemo(
	ctx context.Context,
	requestID string,
	txn *SalesTxn,
) (*TxnResult, error) {
	return c.createSales(ctx, requestID, TxnCreditMemo, txn)
}

func (c *Client) createSales(
	ctx context.Context,
	requestID string,
	kind TxnKind,
	txn *SalesTxn,
) (*TxnResult, error) {
	body, err := salesBodyOf(requestID, txn)
	if err != nil {
		return nil, err
	}
	return c.writeTxn(ctx, &txnWrite{requestID: requestID, kind: kind, body: body})
}

func salesBodyOf(requestID string, txn *SalesTxn) (*salesBody, error) {
	if err := validateRequestID(requestID); err != nil {
		return nil, err
	}
	if txn == nil || strings.TrimSpace(txn.CustomerID) == "" {
		return nil, ErrCustomerRequired
	}
	if len(txn.Lines) == 0 {
		return nil, ErrLinesRequired
	}
	docNumber := strings.TrimSpace(txn.DocNumber)
	if utf8.RuneCountInString(docNumber) > MaxDocNumberLength {
		return nil, ErrDocNumberTooLong
	}

	body := &salesBody{
		CustomerRef: refValue{Value: strings.TrimSpace(txn.CustomerID)},
		DocNumber:   docNumber,
		TxnDate:     txn.TxnDate,
		DueDate:     txn.DueDate,
		PrivateNote: truncate(txn.PrivateNote, MaxPrivateNoteLength),
		Line:        make([]wireSalesLine, 0, len(txn.Lines)),
	}
	if term := strings.TrimSpace(txn.TermID); term != "" {
		body.SalesTermRef = &refValue{Value: term}
	}
	if currency := strings.TrimSpace(txn.CurrencyCode); currency != "" {
		body.CurrencyRef = &refValue{Value: strings.ToUpper(currency)}
		body.ExchangeRate = exchangeRateOf(txn.ExchangeRate)
	}
	if memo := truncate(txn.CustomerMemo, MaxCustomerMemoLength); memo != "" {
		body.CustomerMemo = &memoValue{Value: memo}
	}

	for idx := range txn.Lines {
		line := &txn.Lines[idx]
		if strings.TrimSpace(line.ItemID) == "" {
			return nil, ErrItemRequired
		}
		wire := wireSalesLine{
			DetailType:  lineDetailSalesItem,
			Amount:      money(line.Amount),
			Description: truncate(line.Description, MaxLineDescription),
			SalesItemLineDetail: salesItemDetail{
				ItemRef:     refValue{Value: strings.TrimSpace(line.ItemID)},
				ServiceDate: line.ServiceDate,
			},
		}
		if priced(line) {
			qty := quantity(line.Quantity)
			unit := quantity(line.UnitPrice)
			wire.SalesItemLineDetail.Qty = &qty
			wire.SalesItemLineDetail.UnitPrice = &unit
		}
		body.Line = append(body.Line, wire)
	}

	return body, nil
}

func priced(line *SalesLine) bool {
	if line.Quantity.IsZero() || line.Quantity.IsNegative() || line.UnitPrice.IsNegative() {
		return false
	}
	return line.Quantity.Mul(line.UnitPrice).Round(2).Equal(line.Amount.Round(2))
}

func (c *Client) CreatePayment(
	ctx context.Context,
	requestID string,
	txn *PaymentTxn,
) (*TxnResult, error) {
	body, err := paymentBodyOf(requestID, txn)
	if err != nil {
		return nil, err
	}
	return c.writeTxn(ctx, &txnWrite{requestID: requestID, kind: TxnPayment, body: body})
}

func (c *Client) UpdatePayment(
	ctx context.Context,
	requestID, id string,
	txn *PaymentTxn,
) (*TxnResult, error) {
	body, err := paymentBodyOf(requestID, txn)
	if err != nil {
		return nil, err
	}
	current, err := c.ReadTransaction(ctx, TxnPayment, id)
	if err != nil {
		return nil, err
	}
	body.ID = current.ID
	body.SyncToken = current.SyncToken
	return c.writeTxn(ctx, &txnWrite{requestID: requestID, kind: TxnPayment, body: body})
}

func paymentBodyOf(requestID string, txn *PaymentTxn) (*paymentBody, error) {
	if err := validateRequestID(requestID); err != nil {
		return nil, err
	}
	if txn == nil || strings.TrimSpace(txn.CustomerID) == "" {
		return nil, ErrCustomerRequired
	}
	if txn.TotalAmount.IsNegative() {
		return nil, ErrNegativeAmount
	}

	body := &paymentBody{
		CustomerRef:   refValue{Value: strings.TrimSpace(txn.CustomerID)},
		TotalAmt:      money(txn.TotalAmount),
		TxnDate:       txn.TxnDate,
		PaymentRefNum: truncate(txn.PaymentRefNum, MaxDocNumberLength),
		PrivateNote:   truncate(txn.PrivateNote, MaxPrivateNoteLength),
		Line:          make([]wirePaymentLine, 0, len(txn.Links)),
	}
	if currency := strings.TrimSpace(txn.CurrencyCode); currency != "" {
		body.CurrencyRef = &refValue{Value: strings.ToUpper(currency)}
		body.ExchangeRate = exchangeRateOf(txn.ExchangeRate)
	}
	if method := strings.TrimSpace(txn.PaymentMethodID); method != "" {
		body.PaymentMethodRef = &refValue{Value: method}
	}
	if deposit := strings.TrimSpace(txn.DepositAccountID); deposit != "" {
		body.DepositToAccountRef = &refValue{Value: deposit}
	}
	for idx := range txn.Links {
		link := &txn.Links[idx]
		if link.TxnKind != TxnInvoice && link.TxnKind != TxnCreditMemo {
			return nil, ErrUnknownTxnKind
		}
		if strings.TrimSpace(link.TxnID) == "" {
			return nil, ErrTxnIDRequired
		}
		if link.Amount.IsNegative() {
			return nil, ErrNegativeAmount
		}
		body.Line = append(body.Line, wirePaymentLine{
			Amount: money(link.Amount),
			LinkedTxn: []linkedTxn{{
				TxnID:   strings.TrimSpace(link.TxnID),
				TxnType: string(link.TxnKind),
			}},
		})
	}

	return body, nil
}

func (c *Client) VoidInvoice(ctx context.Context, requestID, id string) (*TxnResult, error) {
	return c.retire(ctx, requestID, TxnInvoice, id, "void", "")
}

func (c *Client) VoidPayment(ctx context.Context, requestID, id string) (*TxnResult, error) {
	return c.retire(ctx, requestID, TxnPayment, id, "update", "void")
}

func (c *Client) DeleteCreditMemo(ctx context.Context, requestID, id string) (*TxnResult, error) {
	return c.retire(ctx, requestID, TxnCreditMemo, id, "delete", "")
}

func (c *Client) retire(
	ctx context.Context,
	requestID string,
	kind TxnKind,
	id, operation, include string,
) (*TxnResult, error) {
	if err := validateRequestID(requestID); err != nil {
		return nil, err
	}
	current, err := c.ReadTransaction(ctx, kind, id)
	if err != nil {
		return nil, err
	}
	ref := entityRef{ID: current.ID, SyncToken: current.SyncToken}
	if include != "" {
		sparse := true
		ref.Sparse = &sparse
	}
	return c.writeTxn(ctx, &txnWrite{
		requestID: requestID,
		kind:      kind,
		body:      ref,
		operation: operation,
		include:   include,
	})
}

func (c *Client) ReadTransaction(ctx context.Context, kind TxnKind, id string) (*TxnResult, error) {
	if !kind.IsValid() {
		return nil, ErrUnknownTxnKind
	}
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return nil, ErrTxnIDRequired
	}

	var out txnEnvelope
	if _, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: "read-" + kind.resource(),
		Method:   http.MethodGet,
		Path:     c.companyPath(kind.resource(), trimmed),
		Query:    c.query(),
		Out:      &out,
	}); err != nil {
		return nil, err
	}
	wire := out.txn(kind)
	if wire == nil {
		return nil, ErrUnexpectedPayload
	}
	return wire.result(kind), nil
}

func (c *Client) FindTransactionByDocNumber(
	ctx context.Context,
	kind TxnKind,
	docNumber string,
) (*TxnResult, bool, error) {
	if !kind.numbered() {
		return nil, false, ErrUnknownTxnKind
	}
	number := strings.TrimSpace(docNumber)
	if number == "" {
		return nil, false, nil
	}
	if utf8.RuneCountInString(number) > MaxDocNumberLength {
		return nil, false, nil
	}

	query := c.query()
	query.Set(queryResource, "select * from "+string(kind)+" where DocNumber = '"+
		strings.ReplaceAll(number, "'", `\'`)+"'")

	var out txnQueryEnvelope
	if _, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: queryResource,
		Method:   http.MethodGet,
		Path:     c.companyPath(queryResource),
		Query:    query,
		Out:      &out,
	}); err != nil {
		return nil, false, err
	}

	matches := out.matches(kind)
	if len(matches) != 1 {
		return nil, false, nil
	}
	return matches[0].result(kind), true, nil
}

func (c *Client) UpdateCustomer(
	ctx context.Context,
	requestID, id string,
	draft *PartyDraft,
) (*ReferenceObject, error) {
	return c.updateParty(ctx, &partyUpdate{
		requestID: requestID,
		id:        id,
		kind:      KindCustomer,
		resource:  "customer",
		draft:     draft,
	})
}

func (c *Client) UpdateVendor(
	ctx context.Context,
	requestID, id string,
	draft *PartyDraft,
) (*ReferenceObject, error) {
	return c.updateParty(ctx, &partyUpdate{
		requestID: requestID,
		id:        id,
		kind:      KindVendor,
		resource:  "vendor",
		draft:     draft,
	})
}

type partyUpdate struct {
	requestID string
	id        string
	kind      ReferenceKind
	resource  string
	draft     *PartyDraft
}

func (c *Client) updateParty(ctx context.Context, update *partyUpdate) (*ReferenceObject, error) {
	body, err := partyBodyOf(update.requestID, update.draft, update.kind == KindVendor)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(update.id)
	if trimmed == "" {
		return nil, ErrTxnIDRequired
	}

	var current txnEnvelope
	if _, err = c.transport.Do(ctx, &restx.Request{
		Endpoint: "read-" + update.resource,
		Method:   http.MethodGet,
		Path:     c.companyPath(update.resource, trimmed),
		Query:    c.query(),
		Out:      &current,
	}); err != nil {
		return nil, err
	}
	existing := current.party(update.kind)
	if existing == nil {
		return nil, ErrUnexpectedPayload
	}

	sparse := true
	query := c.query()
	query.Set("requestid", update.requestID)
	var out txnEnvelope
	if _, err = c.transport.Do(ctx, &restx.Request{
		Endpoint: "update-" + update.resource,
		Method:   http.MethodPost,
		Path:     c.companyPath(update.resource),
		Query:    query,
		Body: sparsePartyBody{
			entityRef: entityRef{
				ID:        existing.ID,
				SyncToken: existing.SyncToken,
				Sparse:    &sparse,
			},
			partyBody: body,
		},
		Out: &out,
	}); err != nil {
		return nil, err
	}
	saved := out.party(update.kind)
	if saved == nil {
		return nil, ErrUnexpectedPayload
	}
	updated := saved.toReference(update.kind)
	return &updated, nil
}

type txnWrite struct {
	requestID string
	kind      TxnKind
	body      any
	operation string
	include   string
}

func (c *Client) writeTxn(ctx context.Context, write *txnWrite) (*TxnResult, error) {
	query := c.query()
	query.Set("requestid", write.requestID)
	if write.operation != "" {
		query.Set("operation", write.operation)
	}
	if write.include != "" {
		query.Set("include", write.include)
	}

	endpoint := "write-" + write.kind.resource()
	if write.operation != "" {
		endpoint += "-" + write.operation
	}

	var out txnEnvelope
	if _, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: endpoint,
		Method:   http.MethodPost,
		Path:     c.companyPath(write.kind.resource()),
		Query:    query,
		Body:     write.body,
		Out:      &out,
	}); err != nil {
		return nil, err
	}
	wire := out.txn(write.kind)
	if wire == nil {
		return nil, ErrUnexpectedPayload
	}
	return wire.result(write.kind), nil
}

func truncate(value string, maxLength int) string {
	trimmed := strings.TrimSpace(value)
	if utf8.RuneCountInString(trimmed) <= maxLength {
		return trimmed
	}
	return strings.TrimSpace(string([]rune(trimmed)[:maxLength]))
}

func FaultCodes(err error) []string {
	var fault *FaultError
	if !errors.As(err, &fault) {
		return nil
	}
	codes := make([]string, 0, len(fault.Errors))
	for idx := range fault.Errors {
		codes = append(codes, fault.Errors[idx].Code)
	}
	return codes
}

func hasFaultCode(err error, code string) bool {
	for _, candidate := range FaultCodes(err) {
		if candidate == code {
			return true
		}
	}
	return false
}

func IsStaleObject(err error) bool { return hasFaultCode(err, faultStaleObject) }

func IsObjectNotFound(err error) bool { return hasFaultCode(err, faultObjectNotFound) }

func IsDuplicateDocNumber(err error) bool { return hasFaultCode(err, faultDuplicateDocument) }

func IsClosedPeriod(err error) bool { return hasFaultCode(err, faultClosedPeriod) }

func IsInvalidReference(err error) bool { return hasFaultCode(err, faultInvalidReference) }

func IsValidationFault(err error) bool {
	var fault *FaultError
	if !errors.As(err, &fault) {
		return false
	}
	return fault.StatusCode == http.StatusBadRequest ||
		strings.EqualFold(fault.Type, "ValidationFault")
}
