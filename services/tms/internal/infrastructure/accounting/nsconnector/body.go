package nsconnector

import (
	"maps"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	moneyPlaces       = 2
	maxTranIDLength   = 45
	maxMemoLength     = 999
	maxLineTextLength = 999
	defaultLineText   = "Charges"
	dateLayout        = "2006-01-02"
)

type recordRef struct {
	ID string `json:"id"`
}

func refOf(id string) *recordRef {
	value := strings.TrimSpace(id)
	if value == "" {
		return nil
	}
	return &recordRef{ID: value}
}

type amount decimal.Decimal

func amountOf(value decimal.Decimal) *amount {
	out := amount(value)
	return &out
}

func (a amount) MarshalJSON() ([]byte, error) {
	return []byte(decimal.Decimal(a).String()), nil
}

type sublist[T any] struct {
	Items []T `json:"items"`
}

type itemLine struct {
	Item        recordRef `json:"item"`
	Description string    `json:"description,omitempty"`
	Quantity    amount    `json:"quantity"`
	Rate        amount    `json:"rate"`
	Amount      amount    `json:"amount"`
}

type expenseLine struct {
	Account recordRef `json:"account"`
	Amount  amount    `json:"amount"`
	Memo    string    `json:"memo,omitempty"`
}

type applyLine struct {
	Doc    recordRef `json:"doc"`
	Apply  bool      `json:"apply"`
	Amount *amount   `json:"amount,omitempty"`
}

type transactionBody struct {
	Entity       *recordRef            `json:"entity,omitempty"`
	Customer     *recordRef            `json:"customer,omitempty"`
	Subsidiary   *recordRef            `json:"subsidiary,omitempty"`
	TranDate     string                `json:"tranDate,omitempty"`
	DueDate      string                `json:"dueDate,omitempty"`
	TranID       string                `json:"tranId,omitempty"`
	OtherRefNum  string                `json:"otherRefNum,omitempty"`
	CheckNum     string                `json:"checkNum,omitempty"`
	Memo         string                `json:"memo,omitempty"`
	Message      string                `json:"message,omitempty"`
	Terms        *recordRef            `json:"terms,omitempty"`
	Currency     *recordRef            `json:"currency,omitempty"`
	ExchangeRate *amount               `json:"exchangeRate,omitempty"`
	Account      *recordRef            `json:"account,omitempty"`
	Payment      *amount               `json:"payment,omitempty"`
	Item         *sublist[itemLine]    `json:"item,omitempty"`
	Expense      *sublist[expenseLine] `json:"expense,omitempty"`
	Apply        *sublist[applyLine]   `json:"apply,omitempty"`
}

func (b *transactionBody) setCurrency(fields currencyFields) {
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

type recordItemLine struct {
	Item        refIn           `json:"item"`
	Description string          `json:"description"`
	Quantity    decimal.Decimal `json:"quantity"`
	Rate        decimal.Decimal `json:"rate"`
	Amount      decimal.Decimal `json:"amount"`
}

type recordExpenseLine struct {
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

type transactionRecord struct {
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
	Item          sublist[recordItemLine]    `json:"item"`
	Expense       sublist[recordExpenseLine] `json:"expense"`
	Apply         sublist[recordApplyLine]   `json:"apply"`
}

func (r *transactionRecord) party() *refIn {
	if r.Customer.ID.Value() != "" {
		return &r.Customer
	}
	return &r.Entity
}

func (r *transactionRecord) appliedDocs() []string {
	out := make([]string, 0, len(r.Apply.Items))
	for idx := range r.Apply.Items {
		line := &r.Apply.Items[idx]
		if line.Apply.Value() && string(line.Doc) != "" {
			out = append(out, string(line.Doc))
		}
	}
	return out
}

func unapplyLines(ids []string) *sublist[applyLine] {
	lines := make([]applyLine, 0, len(ids))
	for _, id := range ids {
		lines = append(lines, applyLine{Doc: recordRef{ID: id}, Apply: false})
	}
	return &sublist[applyLine]{Items: lines}
}

func cleanText(value string, limit int) string {
	stripped := strings.Map(func(r rune) rune {
		if r < ' ' || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			return ' '
		}
		return r
	}, value)
	return strings.TrimSpace(stringutils.OneLine(stripped, limit))
}

func lineText(description string) string {
	if text := strings.TrimSpace(description); text != "" {
		return text
	}
	return defaultLineText
}

func linePurpose(idx int, description string) string {
	return "Line " + strconv.Itoa(idx+1) + " (" + lineText(description) + ")"
}

func sameMoney(a, b decimal.Decimal) bool {
	return a.Round(moneyPlaces).Equal(b.Round(moneyPlaces))
}

func newResult(refs map[string]string) *accountingconnectorports.AccountingDocumentResult {
	copied := make(map[string]string, len(refs)+4)
	maps.Copy(copied, refs)
	return &accountingconnectorports.AccountingDocumentResult{Refs: copied}
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func unixOf(t time.Time) *int64 {
	if t.IsZero() {
		return nil
	}
	value := t.Unix()
	return &value
}
