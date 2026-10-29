package quickbookstxn

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/shopspring/decimal"
)

const (
	lineDetailAccountExpense = "AccountBasedExpenseLineDetail"
	payTypeCheck             = "Check"
)

var (
	ErrVendorRequired      = errors.New("quickbooks: a transaction needs a vendor")
	ErrAccountRequired     = errors.New("quickbooks: every purchase line needs an account")
	ErrBankAccountRequired = errors.New("quickbooks: a bill payment needs a bank account")
	ErrBillRequired        = errors.New("quickbooks: a bill payment needs the bill it pays")
	ErrNonPositiveAmount   = errors.New(
		"quickbooks: a bill payment amount must be greater than zero",
	)
)

type PurchaseLine struct {
	Description string
	AccountID   string
	Amount      decimal.Decimal
}

type PurchaseTxn struct {
	VendorID     string
	APAccountID  string
	DocNumber    string
	TxnDate      string
	DueDate      string
	CurrencyCode string
	ExchangeRate decimal.Decimal
	PrivateNote  string
	Lines        []PurchaseLine
}

type BillPaymentTxn struct {
	VendorID      string
	BankAccountID string
	BillID        string
	DocNumber     string
	TxnDate       string
	CurrencyCode  string
	ExchangeRate  decimal.Decimal
	PrivateNote   string
	Amount        decimal.Decimal
}

type accountExpenseDetail struct {
	AccountRef quickbooks.Ref `json:"AccountRef"`
}

type wirePurchaseLine struct {
	DetailType                    string               `json:"DetailType"`
	Amount                        money                `json:"Amount"`
	Description                   string               `json:"Description,omitempty"`
	AccountBasedExpenseLineDetail accountExpenseDetail `json:"AccountBasedExpenseLineDetail"`
}

type purchaseBody struct {
	ID           string             `json:"Id,omitempty"`
	SyncToken    string             `json:"SyncToken,omitempty"`
	VendorRef    quickbooks.Ref     `json:"VendorRef"`
	APAccountRef *quickbooks.Ref    `json:"APAccountRef,omitempty"`
	DocNumber    string             `json:"DocNumber,omitempty"`
	TxnDate      string             `json:"TxnDate,omitempty"`
	DueDate      string             `json:"DueDate,omitempty"`
	CurrencyRef  *quickbooks.Ref    `json:"CurrencyRef,omitempty"`
	ExchangeRate *exchangeRate      `json:"ExchangeRate,omitempty"`
	PrivateNote  string             `json:"PrivateNote,omitempty"`
	Line         []wirePurchaseLine `json:"Line"`
}

type checkPayment struct {
	BankAccountRef quickbooks.Ref `json:"BankAccountRef"`
}

type billPaymentBody struct {
	ID           string            `json:"Id,omitempty"`
	SyncToken    string            `json:"SyncToken,omitempty"`
	VendorRef    quickbooks.Ref    `json:"VendorRef"`
	PayType      string            `json:"PayType"`
	CheckPayment checkPayment      `json:"CheckPayment"`
	TotalAmt     money             `json:"TotalAmt"`
	TxnDate      string            `json:"TxnDate,omitempty"`
	DocNumber    string            `json:"DocNumber,omitempty"`
	PrivateNote  string            `json:"PrivateNote,omitempty"`
	CurrencyRef  *quickbooks.Ref   `json:"CurrencyRef,omitempty"`
	ExchangeRate *exchangeRate     `json:"ExchangeRate,omitempty"`
	Line         []wirePaymentLine `json:"Line"`
}

func CreateBill(
	ctx context.Context,
	c *quickbooks.Client,
	requestID string,
	txn *PurchaseTxn,
) (*TxnResult, error) {
	return createPurchase(ctx, c, requestID, TxnBill, txn)
}

func CreateVendorCredit(
	ctx context.Context,
	c *quickbooks.Client,
	requestID string,
	txn *PurchaseTxn,
) (*TxnResult, error) {
	return createPurchase(ctx, c, requestID, TxnVendorCredit, txn)
}

func DeleteBill(
	ctx context.Context,
	c *quickbooks.Client,
	requestID, id string,
) (*TxnResult, error) {
	return retire(ctx, c, requestID, TxnBill, id, "delete", "")
}

func DeleteVendorCredit(
	ctx context.Context,
	c *quickbooks.Client,
	requestID, id string,
) (*TxnResult, error) {
	return retire(ctx, c, requestID, TxnVendorCredit, id, "delete", "")
}

func VoidBillPayment(
	ctx context.Context,
	c *quickbooks.Client,
	requestID, id string,
) (*TxnResult, error) {
	return retire(ctx, c, requestID, TxnBillPayment, id, "update", "void")
}

func createPurchase(
	ctx context.Context,
	c *quickbooks.Client,
	requestID string,
	kind TxnKind,
	txn *PurchaseTxn,
) (*TxnResult, error) {
	body, err := purchaseBodyOf(requestID, kind, txn)
	if err != nil {
		return nil, err
	}
	return writeTxn(ctx, c, &txnWrite{requestID: requestID, kind: kind, body: body})
}

func purchaseBodyOf(requestID string, kind TxnKind, txn *PurchaseTxn) (*purchaseBody, error) {
	if err := quickbooks.ValidateRequestID(requestID); err != nil {
		return nil, err
	}
	if txn == nil || strings.TrimSpace(txn.VendorID) == "" {
		return nil, ErrVendorRequired
	}
	if len(txn.Lines) == 0 {
		return nil, ErrLinesRequired
	}
	docNumber := strings.TrimSpace(txn.DocNumber)
	if utf8.RuneCountInString(docNumber) > MaxDocNumberLength {
		return nil, ErrDocNumberTooLong
	}

	body := &purchaseBody{
		VendorRef:   quickbooks.Ref{Value: strings.TrimSpace(txn.VendorID)},
		DocNumber:   docNumber,
		TxnDate:     txn.TxnDate,
		PrivateNote: truncate(txn.PrivateNote, MaxPrivateNoteLength),
		Line:        make([]wirePurchaseLine, 0, len(txn.Lines)),
	}
	if kind == TxnBill {
		body.DueDate = txn.DueDate
	}
	if account := strings.TrimSpace(txn.APAccountID); account != "" {
		body.APAccountRef = &quickbooks.Ref{Value: account}
	}
	if currency := strings.TrimSpace(txn.CurrencyCode); currency != "" {
		body.CurrencyRef = &quickbooks.Ref{Value: strings.ToUpper(currency)}
		body.ExchangeRate = exchangeRateOf(txn.ExchangeRate)
	}

	total := decimal.Zero
	for idx := range txn.Lines {
		line := &txn.Lines[idx]
		account := strings.TrimSpace(line.AccountID)
		if account == "" {
			return nil, ErrAccountRequired
		}
		total = total.Add(line.Amount.Round(2))
		body.Line = append(body.Line, wirePurchaseLine{
			DetailType:  lineDetailAccountExpense,
			Amount:      money(line.Amount),
			Description: truncate(line.Description, MaxLineDescription),
			AccountBasedExpenseLineDetail: accountExpenseDetail{
				AccountRef: quickbooks.Ref{Value: account},
			},
		})
	}
	if total.IsNegative() {
		return nil, ErrNegativeAmount
	}

	return body, nil
}

func CreateBillPayment(
	ctx context.Context,
	c *quickbooks.Client,
	requestID string,
	txn *BillPaymentTxn,
) (*TxnResult, error) {
	body, err := billPaymentBodyOf(requestID, txn)
	if err != nil {
		return nil, err
	}
	return writeTxn(ctx, c, &txnWrite{requestID: requestID, kind: TxnBillPayment, body: body})
}

func billPaymentBodyOf(requestID string, txn *BillPaymentTxn) (*billPaymentBody, error) {
	if err := quickbooks.ValidateRequestID(requestID); err != nil {
		return nil, err
	}
	if txn == nil || strings.TrimSpace(txn.VendorID) == "" {
		return nil, ErrVendorRequired
	}
	bank := strings.TrimSpace(txn.BankAccountID)
	if bank == "" {
		return nil, ErrBankAccountRequired
	}
	bill := strings.TrimSpace(txn.BillID)
	if bill == "" {
		return nil, ErrBillRequired
	}
	if !txn.Amount.Round(2).IsPositive() {
		return nil, ErrNonPositiveAmount
	}

	body := &billPaymentBody{
		VendorRef:    quickbooks.Ref{Value: strings.TrimSpace(txn.VendorID)},
		PayType:      payTypeCheck,
		CheckPayment: checkPayment{BankAccountRef: quickbooks.Ref{Value: bank}},
		TotalAmt:     money(txn.Amount),
		TxnDate:      txn.TxnDate,
		DocNumber:    truncate(txn.DocNumber, MaxDocNumberLength),
		PrivateNote:  truncate(txn.PrivateNote, MaxPrivateNoteLength),
		Line: []wirePaymentLine{{
			Amount:    money(txn.Amount),
			LinkedTxn: []LinkedTxn{{TxnID: bill, TxnType: string(TxnBill)}},
		}},
	}
	if currency := strings.TrimSpace(txn.CurrencyCode); currency != "" {
		body.CurrencyRef = &quickbooks.Ref{Value: strings.ToUpper(currency)}
		body.ExchangeRate = exchangeRateOf(txn.ExchangeRate)
	}

	return body, nil
}
