package quickbooks

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

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
	Kind         TxnKind
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
	} `json:"QueryResponse"`
}

func (e *documentQueryEnvelope) documents(kind TxnKind) []wireDocument {
	switch kind {
	case TxnInvoice:
		return e.QueryResponse.Invoice
	case TxnCreditMemo:
		return e.QueryResponse.CreditMemo
	case TxnPayment:
		return e.QueryResponse.Payment
	case TxnBill:
		return e.QueryResponse.Bill
	case TxnVendorCredit:
		return e.QueryResponse.VendorCredit
	case TxnBillPayment:
		return e.QueryResponse.BillPayment
	default:
		return nil
	}
}

func (w *wireDocument) state(kind TxnKind) DocumentState {
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

func (c *Client) QueryByIDs(
	ctx context.Context,
	kind TxnKind,
	ids []string,
) ([]DocumentState, error) {
	if !kind.IsValid() {
		return nil, ErrUnknownTxnKind
	}
	unique, err := queryIDs(ids)
	if err != nil {
		return nil, err
	}
	if len(unique) == 0 {
		return []DocumentState{}, nil
	}

	query := c.query()
	query.Set(queryResource, "select * from "+string(kind)+" where Id in ('"+
		strings.Join(unique, "', '")+"') maxresults "+strconv.Itoa(len(unique)))

	var out documentQueryEnvelope
	if _, err = c.transport.Do(ctx, &restx.Request{
		Endpoint: queryResource,
		Method:   http.MethodGet,
		Path:     c.companyPath(queryResource),
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

type identifiedBody interface {
	identify(id, syncToken string)
}

func (b *salesBody) identify(id, syncToken string) {
	b.ID = id
	b.SyncToken = syncToken
}

func (b *purchaseBody) identify(id, syncToken string) {
	b.ID = id
	b.SyncToken = syncToken
}

func (b *billPaymentBody) identify(id, syncToken string) {
	b.ID = id
	b.SyncToken = syncToken
}

type txnUpdate struct {
	requestID string
	kind      TxnKind
	id        string
	body      identifiedBody
}

func (c *Client) updateTxn(ctx context.Context, update *txnUpdate) (*TxnResult, error) {
	if strings.TrimSpace(update.id) == "" {
		return nil, ErrTxnIDRequired
	}
	current, err := c.ReadTransaction(ctx, update.kind, update.id)
	if err != nil {
		return nil, err
	}
	update.body.identify(current.ID, current.SyncToken)
	return c.writeTxn(ctx, &txnWrite{
		requestID: update.requestID,
		kind:      update.kind,
		body:      update.body,
	})
}

func (c *Client) UpdateInvoice(
	ctx context.Context,
	requestID, id string,
	txn *SalesTxn,
) (*TxnResult, error) {
	return c.updateSales(ctx, requestID, TxnInvoice, id, txn)
}

func (c *Client) UpdateCreditMemo(
	ctx context.Context,
	requestID, id string,
	txn *SalesTxn,
) (*TxnResult, error) {
	return c.updateSales(ctx, requestID, TxnCreditMemo, id, txn)
}

func (c *Client) updateSales(
	ctx context.Context,
	requestID string,
	kind TxnKind,
	id string,
	txn *SalesTxn,
) (*TxnResult, error) {
	body, err := salesBodyOf(requestID, txn)
	if err != nil {
		return nil, err
	}
	return c.updateTxn(ctx, &txnUpdate{requestID: requestID, kind: kind, id: id, body: body})
}

func (c *Client) UpdateBill(
	ctx context.Context,
	requestID, id string,
	txn *PurchaseTxn,
) (*TxnResult, error) {
	return c.updatePurchase(ctx, requestID, TxnBill, id, txn)
}

func (c *Client) UpdateVendorCredit(
	ctx context.Context,
	requestID, id string,
	txn *PurchaseTxn,
) (*TxnResult, error) {
	return c.updatePurchase(ctx, requestID, TxnVendorCredit, id, txn)
}

func (c *Client) updatePurchase(
	ctx context.Context,
	requestID string,
	kind TxnKind,
	id string,
	txn *PurchaseTxn,
) (*TxnResult, error) {
	body, err := purchaseBodyOf(requestID, kind, txn)
	if err != nil {
		return nil, err
	}
	return c.updateTxn(ctx, &txnUpdate{requestID: requestID, kind: kind, id: id, body: body})
}

func (c *Client) UpdateBillPayment(
	ctx context.Context,
	requestID, id string,
	txn *BillPaymentTxn,
) (*TxnResult, error) {
	body, err := billPaymentBodyOf(requestID, txn)
	if err != nil {
		return nil, err
	}
	return c.updateTxn(ctx, &txnUpdate{
		requestID: requestID,
		kind:      TxnBillPayment,
		id:        id,
		body:      body,
	})
}
