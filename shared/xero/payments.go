package xero

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	PaymentTypeReceivable            = "ACCRECPAYMENT"
	PaymentTypePayable               = "ACCPAYPAYMENT"
	PaymentTypeReceivableCredit      = "ARCREDITPAYMENT"
	PaymentTypePayableCredit         = "APCREDITPAYMENT"
	PaymentTypeReceivableOverpayment = "AROVERPAYMENTPAYMENT"
	PaymentTypeReceivablePrepayment  = "ARPREPAYMENTPAYMENT"
	PaymentTypePayableOverpayment    = "APOVERPAYMENTPAYMENT"
	PaymentTypePayablePrepayment     = "APPREPAYMENTPAYMENT"
	paymentsResource                 = "Payments"
	batchPaymentsResource            = "BatchPayments"
	paymentListStatuses              = StatusAuthorised + "," + StatusDeleted
)

type PaymentInput struct {
	InvoiceID    string
	CreditNoteID string
	AccountID    string
	AccountCode  string
	Date         time.Time
	Amount       decimal.Decimal
	CurrencyRate decimal.Decimal
	Reference    string
}

type Payment struct {
	PaymentID      string
	BatchPaymentID string
	Status         string
	PaymentType    string
	Reference      string
	Amount         decimal.Decimal
	CurrencyRate   decimal.Decimal
	Date           time.Time
	UpdatedAt      time.Time
	InvoiceID      string
	InvoiceType    string
	InvoiceNumber  string
	CreditNoteID   string
	ContactID      string
	AccountID      string
	AccountCode    string
	IsReconciled   bool
}

type PaymentPage struct {
	Payments []Payment
	More     bool
}

type PaymentFilter struct {
	InvoiceIDs []string
	Reference  string
}

type BatchPaymentLine struct {
	InvoiceID string
	Amount    decimal.Decimal
}

type BatchPaymentInput struct {
	AccountID   string
	AccountCode string
	Date        time.Time
	Reference   string
	Payments    []BatchPaymentLine
}

type BatchPaymentItem struct {
	PaymentID string
	InvoiceID string
	Amount    decimal.Decimal
}

type BatchPayment struct {
	BatchPaymentID string
	Status         string
	Type           string
	Reference      string
	TotalAmount    decimal.Decimal
	Date           time.Time
	UpdatedAt      time.Time
	Payments       []BatchPaymentItem
}

type paymentBody struct {
	Invoice      *idRef  `json:"Invoice,omitempty"`
	CreditNote   *idRef  `json:"CreditNote,omitempty"`
	Account      *idRef  `json:"Account,omitempty"`
	Date         string  `json:"Date,omitempty"`
	Amount       *amount `json:"Amount,omitempty"`
	CurrencyRate *number `json:"CurrencyRate,omitempty"`
	Reference    string  `json:"Reference,omitempty"`
	Status       string  `json:"Status,omitempty"`
}

type paymentWriteEnvelope struct {
	Payments []paymentBody `json:"Payments"`
}

type wirePayment struct {
	PaymentID      string          `json:"PaymentID"`
	BatchPaymentID string          `json:"BatchPaymentID"`
	Status         string          `json:"Status"`
	PaymentType    string          `json:"PaymentType"`
	Reference      string          `json:"Reference"`
	Amount         decimal.Decimal `json:"Amount"`
	CurrencyRate   decimal.Decimal `json:"CurrencyRate"`
	Date           wireTime        `json:"Date"`
	UpdatedDateUTC wireTime        `json:"UpdatedDateUTC"`
	IsReconciled   bool            `json:"IsReconciled"`
	Invoice        *wireRef        `json:"Invoice"`
	CreditNote     *wireRef        `json:"CreditNote"`
	Account        *wireRef        `json:"Account"`
}

func (w *wirePayment) payment() Payment {
	out := Payment{
		PaymentID:      w.PaymentID,
		BatchPaymentID: w.BatchPaymentID,
		Status:         w.Status,
		PaymentType:    w.PaymentType,
		Reference:      w.Reference,
		Amount:         w.Amount,
		CurrencyRate:   w.CurrencyRate,
		Date:           w.Date.time(),
		UpdatedAt:      w.UpdatedDateUTC.time(),
		IsReconciled:   w.IsReconciled,
	}
	if w.Invoice != nil {
		out.InvoiceID = w.Invoice.InvoiceID
		out.InvoiceType = w.Invoice.Type
		out.InvoiceNumber = w.Invoice.InvoiceNumber
		out.ContactID = w.Invoice.contactID()
	}
	if w.CreditNote != nil {
		out.CreditNoteID = w.CreditNote.CreditNoteID
		if out.ContactID == "" {
			out.ContactID = w.CreditNote.contactID()
		}
	}
	if w.Account != nil {
		out.AccountID = w.Account.AccountID
		out.AccountCode = w.Account.Code
	}
	return out
}

type paymentsEnvelope struct {
	Payments   []wirePayment   `json:"Payments"`
	Pagination *wirePagination `json:"pagination"`
}

func (e *paymentsEnvelope) payments() []Payment {
	out := make([]Payment, 0, len(e.Payments))
	for idx := range e.Payments {
		out = append(out, e.Payments[idx].payment())
	}
	return out
}

type batchPaymentLineBody struct {
	Invoice idRef  `json:"Invoice"`
	Amount  amount `json:"Amount"`
}

type batchPaymentBody struct {
	BatchPaymentID string                 `json:"BatchPaymentID,omitempty"`
	Account        *idRef                 `json:"Account,omitempty"`
	Date           string                 `json:"Date,omitempty"`
	Reference      string                 `json:"Reference,omitempty"`
	Status         string                 `json:"Status,omitempty"`
	Payments       []batchPaymentLineBody `json:"Payments,omitempty"`
}

type batchPaymentWriteEnvelope struct {
	BatchPayments []batchPaymentBody `json:"BatchPayments"`
}

type wireBatchPayment struct {
	BatchPaymentID string          `json:"BatchPaymentID"`
	Status         string          `json:"Status"`
	Type           string          `json:"Type"`
	Reference      string          `json:"Reference"`
	TotalAmount    decimal.Decimal `json:"TotalAmount"`
	Date           wireTime        `json:"Date"`
	UpdatedDateUTC wireTime        `json:"UpdatedDateUTC"`
	Payments       []wirePayment   `json:"Payments"`
}

type batchPaymentsEnvelope struct {
	BatchPayments []wireBatchPayment `json:"BatchPayments"`
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func (c *Client) CreatePayment(ctx context.Context, key string, in PaymentInput) (*Payment, error) {
	idem, err := idempotencyKey(key)
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}

	var out paymentsEnvelope
	if err = c.do(ctx, &call{
		endpoint: "payments-create",
		method:   http.MethodPut,
		path:     accountingPath(paymentsResource),
		key:      idem,
		body:     paymentWriteEnvelope{Payments: []paymentBody{body}},
		out:      &out,
	}); err != nil {
		return nil, err
	}
	if len(out.Payments) == 0 {
		return nil, ErrUnexpectedPayload
	}
	created := out.Payments[0].payment()
	return &created, nil
}

func (in *PaymentInput) body() (paymentBody, error) {
	invoiceID := strings.TrimSpace(in.InvoiceID)
	creditNoteID := strings.TrimSpace(in.CreditNoteID)
	if (invoiceID == "") == (creditNoteID == "") {
		return paymentBody{}, ErrTargetRequired
	}
	account, err := accountRef(in.AccountID, in.AccountCode)
	if err != nil {
		return paymentBody{}, err
	}
	if in.Date.IsZero() {
		return paymentBody{}, ErrDateRequired
	}
	if !in.Amount.IsPositive() {
		return paymentBody{}, ErrAmountNotPositive
	}
	if in.CurrencyRate.IsNegative() {
		return paymentBody{}, ErrNegativeRate
	}
	reference := strings.TrimSpace(in.Reference)
	if !textWithin(reference, MaxReferenceLength) {
		return paymentBody{}, ErrFieldTooLong
	}

	value := amount(in.Amount)
	body := paymentBody{
		Account:      account,
		Date:         formatDate(in.Date),
		Amount:       &value,
		CurrencyRate: numberOf(in.CurrencyRate),
		Reference:    reference,
	}
	if invoiceID != "" {
		id, idErr := guid(invoiceID)
		if idErr != nil {
			return paymentBody{}, idErr
		}
		body.Invoice = &idRef{InvoiceID: id}
		return body, nil
	}
	id, err := guid(creditNoteID)
	if err != nil {
		return paymentBody{}, err
	}
	body.CreditNote = &idRef{CreditNoteID: id}
	return body, nil
}

func accountRef(accountID, accountCode string) (*idRef, error) {
	id := strings.TrimSpace(accountID)
	code := strings.TrimSpace(accountCode)
	if (id == "") == (code == "") {
		return nil, ErrAccountRequired
	}
	if code != "" {
		if !textWithin(code, MaxItemCodeLength) || hasControl(code) {
			return nil, ErrFieldTooLong
		}
		return &idRef{Code: code}, nil
	}
	parsed, err := guid(id)
	if err != nil {
		return nil, err
	}
	return &idRef{AccountID: parsed}, nil
}

func (c *Client) DeletePayment(ctx context.Context, key, paymentID string) error {
	idem, err := idempotencyKey(key)
	if err != nil {
		return err
	}
	id, err := guid(paymentID)
	if err != nil {
		return err
	}
	return c.do(ctx, &call{
		endpoint: "payments-delete",
		method:   http.MethodPost,
		path:     accountingPath(paymentsResource, id),
		key:      idem,
		body:     paymentWriteEnvelope{Payments: []paymentBody{{Status: StatusDeleted}}},
	})
}

func (c *Client) Payments(
	ctx context.Context,
	page int,
	modifiedSince *time.Time,
) (*PaymentPage, error) {
	query, err := pageQuery(page)
	if err != nil {
		return nil, err
	}
	query.Set(statusesParam, paymentListStatuses)

	var out paymentsEnvelope
	if err = c.readPage(ctx, "payments", paymentsResource, query, modifiedSince, &out); err != nil {
		return nil, err
	}
	payments := out.payments()
	return &PaymentPage{
		Payments: payments,
		More:     hasMore(out.Pagination, page, len(payments)),
	}, nil
}

func (c *Client) FindPayments(ctx context.Context, f PaymentFilter) ([]Payment, error) {
	invoiceIDs, err := guids(f.InvoiceIDs, MaxIDsPerRead)
	if err != nil {
		return nil, err
	}
	reference := strings.TrimSpace(f.Reference)
	if reference != "" && (!textWithin(reference, MaxReferenceLength) ||
		strings.ContainsAny(reference, "\"\\") || hasControl(reference)) {
		return nil, ErrInvalidFilter
	}
	if len(invoiceIDs) == 0 && reference == "" {
		return nil, ErrEmptyFilter
	}
	base := url.Values{whereParam: {whereJoin(
		whereAnyOf("Invoice.InvoiceID", invoiceIDs, whereGUID),
		whereAnyOf("Reference", optional(reference), whereString),
	)}}

	return collectPages(func(page int) ([]Payment, bool, error) {
		query, queryErr := withPage(base, page)
		if queryErr != nil {
			return nil, false, queryErr
		}
		var out paymentsEnvelope
		if readErr := c.readPage(
			ctx, "payments-search", paymentsResource, query, nil, &out,
		); readErr != nil {
			return nil, false, readErr
		}
		payments := out.payments()
		return payments, hasMore(out.Pagination, page, len(payments)), nil
	})
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func (c *Client) CreateBatchPayment(
	ctx context.Context,
	key string,
	in BatchPaymentInput,
) (*BatchPayment, error) {
	idem, err := idempotencyKey(key)
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}

	var out batchPaymentsEnvelope
	if err = c.do(ctx, &call{
		endpoint: "batch-payments-create",
		method:   http.MethodPut,
		path:     accountingPath(batchPaymentsResource),
		key:      idem,
		body:     batchPaymentWriteEnvelope{BatchPayments: []batchPaymentBody{body}},
		out:      &out,
	}); err != nil {
		return nil, err
	}
	if len(out.BatchPayments) == 0 {
		return nil, ErrUnexpectedPayload
	}
	return out.BatchPayments[0].batchPayment(), nil
}

func (in *BatchPaymentInput) body() (batchPaymentBody, error) {
	account, err := accountRef(in.AccountID, in.AccountCode)
	if err != nil {
		return batchPaymentBody{}, err
	}
	if in.Date.IsZero() {
		return batchPaymentBody{}, ErrDateRequired
	}
	reference := strings.TrimSpace(in.Reference)
	if !textWithin(reference, MaxReferenceLength) {
		return batchPaymentBody{}, ErrFieldTooLong
	}
	if len(in.Payments) == 0 {
		return batchPaymentBody{}, ErrPaymentsRequired
	}

	lines := make([]batchPaymentLineBody, 0, len(in.Payments))
	for idx := range in.Payments {
		line := &in.Payments[idx]
		invoiceID, idErr := guid(line.InvoiceID)
		if idErr != nil {
			return batchPaymentBody{}, idErr
		}
		if !line.Amount.IsPositive() {
			return batchPaymentBody{}, ErrAmountNotPositive
		}
		lines = append(lines, batchPaymentLineBody{
			Invoice: idRef{InvoiceID: invoiceID},
			Amount:  amount(line.Amount),
		})
	}
	return batchPaymentBody{
		Account:   account,
		Date:      formatDate(in.Date),
		Reference: reference,
		Payments:  lines,
	}, nil
}

func (w *wireBatchPayment) batchPayment() *BatchPayment {
	out := &BatchPayment{
		BatchPaymentID: w.BatchPaymentID,
		Status:         w.Status,
		Type:           w.Type,
		Reference:      w.Reference,
		TotalAmount:    w.TotalAmount,
		Date:           w.Date.time(),
		UpdatedAt:      w.UpdatedDateUTC.time(),
		Payments:       make([]BatchPaymentItem, 0, len(w.Payments)),
	}
	for idx := range w.Payments {
		item := BatchPaymentItem{
			PaymentID: w.Payments[idx].PaymentID,
			Amount:    w.Payments[idx].Amount,
		}
		if w.Payments[idx].Invoice != nil {
			item.InvoiceID = w.Payments[idx].Invoice.InvoiceID
		}
		out.Payments = append(out.Payments, item)
	}
	return out
}

func (c *Client) DeleteBatchPayment(ctx context.Context, key, batchID string) error {
	idem, err := idempotencyKey(key)
	if err != nil {
		return err
	}
	id, err := guid(batchID)
	if err != nil {
		return err
	}
	return c.do(ctx, &call{
		endpoint: "batch-payments-delete",
		method:   http.MethodPost,
		path:     accountingPath(batchPaymentsResource),
		key:      idem,
		body: batchPaymentWriteEnvelope{BatchPayments: []batchPaymentBody{{
			BatchPaymentID: id,
			Status:         StatusDeleted,
		}}},
	})
}
