package quickbookstxn

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/shared/quickbooks"
)

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

func updateTxn(ctx context.Context, c *quickbooks.Client, update *txnUpdate) (*TxnResult, error) {
	if strings.TrimSpace(update.id) == "" {
		return nil, ErrTxnIDRequired
	}
	current, err := ReadTransaction(ctx, c, update.kind, update.id)
	if err != nil {
		return nil, err
	}
	update.body.identify(current.ID, current.SyncToken)
	return writeTxn(ctx, c, &txnWrite{
		requestID: update.requestID,
		kind:      update.kind,
		body:      update.body,
	})
}

func UpdateInvoice(
	ctx context.Context,
	c *quickbooks.Client,
	requestID, id string,
	txn *SalesTxn,
) (*TxnResult, error) {
	return updateSales(ctx, c, requestID, TxnInvoice, id, txn)
}

func UpdateCreditMemo(
	ctx context.Context,
	c *quickbooks.Client,
	requestID, id string,
	txn *SalesTxn,
) (*TxnResult, error) {
	return updateSales(ctx, c, requestID, TxnCreditMemo, id, txn)
}

func updateSales(
	ctx context.Context,
	c *quickbooks.Client,
	requestID string,
	kind TxnKind,
	id string,
	txn *SalesTxn,
) (*TxnResult, error) {
	body, err := salesBodyOf(requestID, txn)
	if err != nil {
		return nil, err
	}
	return updateTxn(ctx, c, &txnUpdate{requestID: requestID, kind: kind, id: id, body: body})
}

func UpdateBill(
	ctx context.Context,
	c *quickbooks.Client,
	requestID, id string,
	txn *PurchaseTxn,
) (*TxnResult, error) {
	return updatePurchase(ctx, c, requestID, TxnBill, id, txn)
}

func UpdateVendorCredit(
	ctx context.Context,
	c *quickbooks.Client,
	requestID, id string,
	txn *PurchaseTxn,
) (*TxnResult, error) {
	return updatePurchase(ctx, c, requestID, TxnVendorCredit, id, txn)
}

func updatePurchase(
	ctx context.Context,
	c *quickbooks.Client,
	requestID string,
	kind TxnKind,
	id string,
	txn *PurchaseTxn,
) (*TxnResult, error) {
	body, err := purchaseBodyOf(requestID, kind, txn)
	if err != nil {
		return nil, err
	}
	return updateTxn(ctx, c, &txnUpdate{requestID: requestID, kind: kind, id: id, body: body})
}

func UpdateBillPayment(
	ctx context.Context,
	c *quickbooks.Client,
	requestID, id string,
	txn *BillPaymentTxn,
) (*TxnResult, error) {
	body, err := billPaymentBodyOf(requestID, txn)
	if err != nil {
		return nil, err
	}
	return updateTxn(ctx, c, &txnUpdate{
		requestID: requestID,
		kind:      TxnBillPayment,
		id:        id,
		body:      body,
	})
}
