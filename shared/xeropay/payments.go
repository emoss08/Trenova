package xeropay

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/xero"
	"github.com/emoss08/trenova/shared/xerodocs"
	"github.com/emoss08/trenova/shared/xeroerrors"
	"github.com/emoss08/trenova/shared/xeroref"
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
	paymentListStatuses              = xerodocs.StatusAuthorised + "," + xerodocs.StatusDeleted
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
	Invoice      *xero.IDRef  `json:"Invoice,omitempty"`
	CreditNote   *xero.IDRef  `json:"CreditNote,omitempty"`
	Account      *xero.IDRef  `json:"Account,omitempty"`
	Date         string       `json:"Date,omitempty"`
	Amount       *xero.Amount `json:"Amount,omitempty"`
	CurrencyRate *xero.Number `json:"CurrencyRate,omitempty"`
	Reference    string       `json:"Reference,omitempty"`
	Status       string       `json:"Status,omitempty"`
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
	Date           xero.WireTime   `json:"Date"`
	UpdatedDateUTC xero.WireTime   `json:"UpdatedDateUTC"`
	IsReconciled   bool            `json:"IsReconciled"`
	Invoice        *xero.WireRef   `json:"Invoice"`
	CreditNote     *xero.WireRef   `json:"CreditNote"`
	Account        *xero.WireRef   `json:"Account"`
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
		Date:           w.Date.Time(),
		UpdatedAt:      w.UpdatedDateUTC.Time(),
		IsReconciled:   w.IsReconciled,
	}
	if w.Invoice != nil {
		out.InvoiceID = w.Invoice.InvoiceID
		out.InvoiceType = w.Invoice.Type
		out.InvoiceNumber = w.Invoice.InvoiceNumber
		out.ContactID = w.Invoice.NestedContactID()
	}
	if w.CreditNote != nil {
		out.CreditNoteID = w.CreditNote.CreditNoteID
		if out.ContactID == "" {
			out.ContactID = w.CreditNote.NestedContactID()
		}
	}
	if w.Account != nil {
		out.AccountID = w.Account.AccountID
		out.AccountCode = w.Account.Code
	}
	return out
}

type paymentsEnvelope struct {
	Payments   []wirePayment        `json:"Payments"`
	Pagination *xero.WirePagination `json:"pagination"`
}

func (e *paymentsEnvelope) payments() []Payment {
	out := make([]Payment, 0, len(e.Payments))
	for idx := range e.Payments {
		out = append(out, e.Payments[idx].payment())
	}
	return out
}

type batchPaymentLineBody struct {
	Invoice xero.IDRef  `json:"Invoice"`
	Amount  xero.Amount `json:"Amount"`
}

type batchPaymentBody struct {
	BatchPaymentID string                 `json:"BatchPaymentID,omitempty"`
	Account        *xero.IDRef            `json:"Account,omitempty"`
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
	Date           xero.WireTime   `json:"Date"`
	UpdatedDateUTC xero.WireTime   `json:"UpdatedDateUTC"`
	Payments       []wirePayment   `json:"Payments"`
}

type batchPaymentsEnvelope struct {
	BatchPayments []wireBatchPayment `json:"BatchPayments"`
}

//
//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func CreatePayment(
	ctx context.Context,
	c *xero.Client,
	key string,
	in PaymentInput,
) (*Payment, error) {
	idem, err := xero.IdempotencyKey(key)
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}

	var out paymentsEnvelope
	if err = c.Do(ctx, &xero.Call{
		Endpoint: "payments-create",
		Method:   http.MethodPut,
		Path:     xero.AccountingPath(paymentsResource),
		Key:      idem,
		Body:     paymentWriteEnvelope{Payments: []paymentBody{body}},
		Out:      &out,
	}); err != nil {
		return nil, err
	}
	if len(out.Payments) == 0 {
		return nil, xeroerrors.ErrUnexpectedPayload
	}
	created := out.Payments[0].payment()
	return &created, nil
}

func (in *PaymentInput) body() (paymentBody, error) {
	invoiceID := strings.TrimSpace(in.InvoiceID)
	creditNoteID := strings.TrimSpace(in.CreditNoteID)
	if (invoiceID == "") == (creditNoteID == "") {
		return paymentBody{}, xeroerrors.ErrTargetRequired
	}
	account, err := accountRef(in.AccountID, in.AccountCode)
	if err != nil {
		return paymentBody{}, err
	}
	if in.Date.IsZero() {
		return paymentBody{}, xeroerrors.ErrDateRequired
	}
	if !in.Amount.IsPositive() {
		return paymentBody{}, xeroerrors.ErrAmountNotPositive
	}
	if in.CurrencyRate.IsNegative() {
		return paymentBody{}, xeroerrors.ErrNegativeRate
	}
	reference := strings.TrimSpace(in.Reference)
	if !xero.TextWithin(reference, xerodocs.MaxReferenceLength) {
		return paymentBody{}, xeroerrors.ErrFieldTooLong
	}

	value := xero.Amount(in.Amount)
	body := paymentBody{
		Account:      account,
		Date:         xero.FormatDate(in.Date),
		Amount:       &value,
		CurrencyRate: xero.NumberOf(in.CurrencyRate),
		Reference:    reference,
	}
	if invoiceID != "" {
		id, idErr := xero.GUID(invoiceID)
		if idErr != nil {
			return paymentBody{}, idErr
		}
		body.Invoice = &xero.IDRef{InvoiceID: id}
		return body, nil
	}
	id, err := xero.GUID(creditNoteID)
	if err != nil {
		return paymentBody{}, err
	}
	body.CreditNote = &xero.IDRef{CreditNoteID: id}
	return body, nil
}

func accountRef(accountID, accountCode string) (*xero.IDRef, error) {
	id := strings.TrimSpace(accountID)
	code := strings.TrimSpace(accountCode)
	if (id == "") == (code == "") {
		return nil, xeroerrors.ErrAccountRequired
	}
	if code != "" {
		if !xero.TextWithin(code, xeroref.MaxItemCodeLength) || xero.HasControl(code) {
			return nil, xeroerrors.ErrFieldTooLong
		}
		return &xero.IDRef{Code: code}, nil
	}
	parsed, err := xero.GUID(id)
	if err != nil {
		return nil, err
	}
	return &xero.IDRef{AccountID: parsed}, nil
}

func DeletePayment(ctx context.Context, c *xero.Client, key, paymentID string) error {
	idem, err := xero.IdempotencyKey(key)
	if err != nil {
		return err
	}
	id, err := xero.GUID(paymentID)
	if err != nil {
		return err
	}
	return c.Do(ctx, &xero.Call{
		Endpoint: "payments-delete",
		Method:   http.MethodPost,
		Path:     xero.AccountingPath(paymentsResource, id),
		Key:      idem,
		Body:     paymentWriteEnvelope{Payments: []paymentBody{{Status: xerodocs.StatusDeleted}}},
	})

}

func Payments(
	ctx context.Context,
	c *xero.Client,
	page int,
	modifiedSince *time.Time,
) (*PaymentPage, error) {
	query, err := xero.PageQuery(page)
	if err != nil {
		return nil, err
	}
	query.Set(xero.StatusesParam, paymentListStatuses)

	var out paymentsEnvelope
	if err = xero.ReadPage(ctx, c, "payments", paymentsResource, query, modifiedSince, &out); err != nil {
		return nil, err
	}
	payments := out.payments()
	return &PaymentPage{
		Payments: payments,
		More:     xero.HasMore(out.Pagination, page, len(payments)),
	}, nil
}

func FindPayments(ctx context.Context, c *xero.Client, f PaymentFilter) ([]Payment, error) {
	invoiceIDs, err := xero.GUIDs(f.InvoiceIDs, xero.MaxIDsPerRead)
	if err != nil {
		return nil, err
	}
	reference := strings.TrimSpace(f.Reference)
	if reference != "" && (!xero.TextWithin(reference, xerodocs.MaxReferenceLength) ||
		strings.ContainsAny(reference, "\"\\") || xero.HasControl(reference)) {
		return nil, xeroerrors.ErrInvalidFilter
	}
	if len(invoiceIDs) == 0 && reference == "" {
		return nil, xeroerrors.ErrEmptyFilter
	}
	base := url.Values{whereParam: {xero.WhereJoin(
		xero.WhereAnyOf("Invoice.InvoiceID", invoiceIDs, xero.WhereGUID),
		xero.WhereAnyOf("Reference", xero.Optional(reference), xero.WhereString),
	)}}

	return xero.CollectPages(func(page int) ([]Payment, bool, error) {
		query, queryErr := xero.WithPage(base, page)
		if queryErr != nil {
			return nil, false, queryErr
		}
		var out paymentsEnvelope
		if readErr := xero.ReadPage(
			ctx,
			c, "payments-search", paymentsResource, query, nil, &out,
		); readErr != nil {
			return nil, false, readErr
		}
		payments := out.payments()
		return payments, xero.HasMore(out.Pagination, page, len(payments)), nil
	})
}

//nolint:gocritic // value inputs are the package contract the accounting adapter is written against.
func CreateBatchPayment(
	ctx context.Context,
	c *xero.Client,
	key string,
	in BatchPaymentInput,
) (*BatchPayment, error) {
	idem, err := xero.IdempotencyKey(key)
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}

	var out batchPaymentsEnvelope
	if err = c.Do(ctx, &xero.Call{
		Endpoint: "batch-payments-create",
		Method:   http.MethodPut,
		Path:     xero.AccountingPath(batchPaymentsResource),
		Key:      idem,
		Body:     batchPaymentWriteEnvelope{BatchPayments: []batchPaymentBody{body}},
		Out:      &out,
	}); err != nil {
		return nil, err
	}
	if len(out.BatchPayments) == 0 {
		return nil, xeroerrors.ErrUnexpectedPayload
	}
	return out.BatchPayments[0].batchPayment(), nil
}

func (in *BatchPaymentInput) body() (batchPaymentBody, error) {
	account, err := accountRef(in.AccountID, in.AccountCode)
	if err != nil {
		return batchPaymentBody{}, err
	}
	if in.Date.IsZero() {
		return batchPaymentBody{}, xeroerrors.ErrDateRequired
	}
	reference := strings.TrimSpace(in.Reference)
	if !xero.TextWithin(reference, xerodocs.MaxReferenceLength) {
		return batchPaymentBody{}, xeroerrors.ErrFieldTooLong
	}
	if len(in.Payments) == 0 {
		return batchPaymentBody{}, xeroerrors.ErrPaymentsRequired
	}

	lines := make([]batchPaymentLineBody, 0, len(in.Payments))
	for idx := range in.Payments {
		line := &in.Payments[idx]
		invoiceID, idErr := xero.GUID(line.InvoiceID)
		if idErr != nil {
			return batchPaymentBody{}, idErr
		}
		if !line.Amount.IsPositive() {
			return batchPaymentBody{}, xeroerrors.ErrAmountNotPositive
		}
		lines = append(lines, batchPaymentLineBody{
			Invoice: xero.IDRef{InvoiceID: invoiceID},
			Amount:  xero.Amount(line.Amount),
		})
	}
	return batchPaymentBody{
		Account:   account,
		Date:      xero.FormatDate(in.Date),
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
		Date:           w.Date.Time(),
		UpdatedAt:      w.UpdatedDateUTC.Time(),
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

func DeleteBatchPayment(ctx context.Context, c *xero.Client, key, batchID string) error {
	idem, err := xero.IdempotencyKey(key)
	if err != nil {
		return err
	}
	id, err := xero.GUID(batchID)
	if err != nil {
		return err
	}
	return c.Do(ctx, &xero.Call{
		Endpoint: "batch-payments-delete",
		Method:   http.MethodPost,
		Path:     xero.AccountingPath(batchPaymentsResource),
		Key:      idem,
		Body: batchPaymentWriteEnvelope{BatchPayments: []batchPaymentBody{{
			BatchPaymentID: id,
			Status:         xerodocs.StatusDeleted,
		}}},
	})

}
