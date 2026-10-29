package quickbookssync

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/emoss08/trenova/shared/quickbookstxn"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
)

const MaxQueryIDs = 200

var (
	ErrInvalidTxnID = errors.New("quickbooks: a transaction id is a number")
	ErrTooManyIDs   = errors.New("quickbooks: at most 200 ids may be read at once")
)

type DocumentState struct {
	ChangeMeta
	Kind         quickbookstxn.TxnKind
	DocNumber    string
	TotalAmount  decimal.Decimal
	Balance      *decimal.Decimal
	CurrencyCode string
}

type wireDocument struct {
	ID          string           `json:"Id"`
	Status      string           `json:"status"`
	DocNumber   string           `json:"DocNumber"`
	TotalAmt    decimal.Decimal  `json:"TotalAmt"`
	Balance     *decimal.Decimal `json:"Balance"`
	CurrencyRef *namedRef        `json:"CurrencyRef"`
	PrivateNote string           `json:"PrivateNote"`
	MetaData    *changeMetaData  `json:"MetaData"`
}

type documentQueryEnvelope struct {
	QueryResponse struct {
		Invoice      []wireDocument `json:"Invoice"`
		CreditMemo   []wireDocument `json:"CreditMemo"`
		Payment      []wireDocument `json:"Payment"`
		Bill         []wireDocument `json:"Bill"`
		VendorCredit []wireDocument `json:"VendorCredit"`
		BillPayment  []wireDocument `json:"BillPayment"`
		JournalEntry []wireDocument `json:"JournalEntry"`
	} `json:"QueryResponse"`
}

func (e *documentQueryEnvelope) documents(kind quickbookstxn.TxnKind) []wireDocument {
	switch kind {
	case quickbookstxn.TxnInvoice:
		return e.QueryResponse.Invoice
	case quickbookstxn.TxnCreditMemo:
		return e.QueryResponse.CreditMemo
	case quickbookstxn.TxnPayment:
		return e.QueryResponse.Payment
	case quickbookstxn.TxnBill:
		return e.QueryResponse.Bill
	case quickbookstxn.TxnVendorCredit:
		return e.QueryResponse.VendorCredit
	case quickbookstxn.TxnBillPayment:
		return e.QueryResponse.BillPayment
	case quickbookstxn.TxnJournalEntry:
		return e.QueryResponse.JournalEntry
	default:
		return nil
	}
}

func (w *wireDocument) state(kind quickbookstxn.TxnKind) DocumentState {
	return DocumentState{
		ChangeMeta:   w.MetaData.meta(w.ID, w.Status, w.PrivateNote, w.TotalAmt),
		Kind:         kind,
		DocNumber:    w.DocNumber,
		TotalAmount:  w.TotalAmt,
		Balance:      w.Balance,
		CurrencyCode: w.CurrencyRef.value(),
	}
}

func queryIDs(ids []string) ([]string, error) {
	out := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if _, err := strconv.ParseUint(id, 10, 64); err != nil {
			return nil, ErrInvalidTxnID
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) > MaxQueryIDs {
		return nil, ErrTooManyIDs
	}
	return out, nil
}

func QueryByIDs(
	ctx context.Context,
	c *quickbooks.Client,
	kind quickbookstxn.TxnKind,
	ids []string,
) ([]DocumentState, error) {
	if !kind.IsValid() {
		return nil, quickbookstxn.ErrUnknownTxnKind
	}
	unique, err := queryIDs(ids)
	if err != nil {
		return nil, err
	}
	if len(unique) == 0 {
		return []DocumentState{}, nil
	}

	query := c.Query()
	query.Set(quickbooks.QueryResource, "select * from "+string(kind)+" where Id in ('"+
		strings.Join(unique, "', '")+"') maxresults "+strconv.Itoa(len(unique)))

	var out documentQueryEnvelope
	if _, err = c.Do(ctx, &restx.Request{
		Endpoint: quickbooks.QueryResource,
		Method:   http.MethodGet,
		Path:     c.CompanyPath(quickbooks.QueryResource),
		Query:    query,
		Out:      &out,
	}); err != nil {
		return nil, err
	}

	wire := out.documents(kind)
	states := make([]DocumentState, 0, len(wire))
	for idx := range wire {
		states = append(states, wire[idx].state(kind))
	}
	return states, nil
}
