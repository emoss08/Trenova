package nsbinding

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/shopspring/decimal"
)

type RecordRef struct {
	ID string `json:"id"`
}

type Amount decimal.Decimal

func AmountOf(value decimal.Decimal) *Amount {
	out := Amount(value)
	return &out
}

func (a Amount) MarshalJSON() ([]byte, error) {
	return []byte(decimal.Decimal(a).String()), nil
}

type Sublist[T any] struct {
	Items []T `json:"items"`
}

type ItemLine struct {
	Item        RecordRef `json:"item"`
	Description string    `json:"description,omitempty"`
	Quantity    Amount    `json:"quantity"`
	Rate        Amount    `json:"rate"`
	Amount      Amount    `json:"amount"`
}

type ExpenseLine struct {
	Account RecordRef `json:"account"`
	Amount  Amount    `json:"amount"`
	Memo    string    `json:"memo,omitempty"`
}

type ApplyLine struct {
	Doc    RecordRef `json:"doc"`
	Apply  bool      `json:"apply"`
	Amount *Amount   `json:"amount,omitempty"`
}

type TransactionBody struct {
	Entity       *RecordRef            `json:"entity,omitempty"`
	Customer     *RecordRef            `json:"customer,omitempty"`
	Subsidiary   *RecordRef            `json:"subsidiary,omitempty"`
	TranDate     string                `json:"tranDate,omitempty"`
	DueDate      string                `json:"dueDate,omitempty"`
	TranID       string                `json:"tranId,omitempty"`
	OtherRefNum  string                `json:"otherRefNum,omitempty"`
	CheckNum     string                `json:"checkNum,omitempty"`
	Memo         string                `json:"memo,omitempty"`
	Message      string                `json:"message,omitempty"`
	Terms        *RecordRef            `json:"terms,omitempty"`
	Currency     *RecordRef            `json:"currency,omitempty"`
	ExchangeRate *Amount               `json:"exchangeRate,omitempty"`
	Account      *RecordRef            `json:"account,omitempty"`
	Payment      *Amount               `json:"payment,omitempty"`
	Item         *Sublist[ItemLine]    `json:"item,omitempty"`
	Expense      *Sublist[ExpenseLine] `json:"expense,omitempty"`
	Apply        *Sublist[ApplyLine]   `json:"apply,omitempty"`
}

func (b *TransactionBody) SetCurrency(fields CurrencyFields) {
	b.Currency = fields.currency
	b.ExchangeRate = fields.rate
}

type docID string

func (d *docID) UnmarshalJSON(data []byte) error {
	if jsonflex.KindOf(data) == jsonflex.KindObject {
		var ref struct {
			ID jsonflex.String `json:"id"`
		}
		if err := sonic.Unmarshal(data, &ref); err != nil {
			return err
		}
		*d = docID(ref.ID.Value())
		return nil
	}
	var value jsonflex.String
	if err := value.UnmarshalJSON(data); err != nil {
		return err
	}
	*d = docID(value.Value())
	return nil
}

type refIn struct {
	ID      jsonflex.String `json:"id"`
	RefName string          `json:"refName"`
}

type RecordItemLine struct {
	Item        refIn           `json:"item"`
	Description string          `json:"description"`
	Quantity    decimal.Decimal `json:"quantity"`
	Rate        decimal.Decimal `json:"rate"`
	Amount      decimal.Decimal `json:"amount"`
}

type RecordExpenseLine struct {
	Account refIn           `json:"account"`
	Amount  decimal.Decimal `json:"amount"`
	Memo    string          `json:"memo"`
}

type recordApplyLine struct {
	Doc    docID           `json:"doc"`
	Apply  jsonflex.Bool   `json:"apply"`
	Amount decimal.Decimal `json:"amount"`
	Type   string          `json:"type"`
	RefNum string          `json:"refNum"`
}

type TransactionRecord struct {
	ID            jsonflex.String            `json:"id"`
	TranID        string                     `json:"tranId"`
	TranDate      string                     `json:"tranDate"`
	Entity        refIn                      `json:"entity"`
	Customer      refIn                      `json:"customer"`
	Subsidiary    refIn                      `json:"subsidiary"`
	Currency      refIn                      `json:"currency"`
	Account       refIn                      `json:"account"`
	PaymentMethod refIn                      `json:"paymentMethod"`
	ExchangeRate  decimal.Decimal            `json:"exchangeRate"`
	Total         decimal.Decimal            `json:"total"`
	Payment       decimal.Decimal            `json:"payment"`
	Unapplied     decimal.Decimal            `json:"unapplied"`
	CheckNum      string                     `json:"checkNum"`
	Item          Sublist[RecordItemLine]    `json:"item"`
	Expense       Sublist[RecordExpenseLine] `json:"expense"`
	Apply         Sublist[recordApplyLine]   `json:"apply"`
}

func (r *TransactionRecord) Party() *refIn {
	if r.Customer.ID.Value() != "" {
		return &r.Customer
	}
	return &r.Entity
}

func (r *TransactionRecord) AppliedDocs() []string {
	out := make([]string, 0, len(r.Apply.Items))
	for idx := range r.Apply.Items {
		line := &r.Apply.Items[idx]
		if line.Apply.Value() && string(line.Doc) != "" {
			out = append(out, string(line.Doc))
		}
	}
	return out
}
