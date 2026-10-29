package nsdocsync

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsrefsync"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
)

const (
	TypeInvoice         = "CustInvc"
	TypeCreditMemo      = "CustCred"
	TypeCustomerPayment = "CustPymt"
	TypeVendorBill      = "VendBill"
	TypeVendorCredit    = "VendCred"
	TypeVendorPayment   = "VendPymt"
	findMatchLimit      = 50
	transactionColumns  = "SELECT t.id, t.type, t.tranid, t.externalid, t.otherrefnum, t.entity, " +
		"t.foreigntotal, t.foreignamountremaining, t.voided, cur.symbol AS currency, " +
		"TO_CHAR(t.lastmodifieddate, " + nsrefsync.TimeFormat + ") AS lastmodified"
	transactionFrom = " FROM transaction t LEFT JOIN currency cur ON cur.id = t.currency"
	TransactionSQL  = transactionColumns + transactionFrom
	stateSQL        = transactionColumns + ", ap.closed, ap.alllocked, ap.arlocked, ap.aplocked" +
		transactionFrom + " LEFT JOIN accountingperiod ap ON ap.id = t.postingperiod" +
		" WHERE t.id = ?"
)

func transactionState(
	ctx context.Context,
	client *netsuite.Client,
	id string,
) (*nsbinding.TransactionRow, bool, error) {
	internalID, ok := nsbinding.ParseInternalID(id)
	if !ok {
		return nil, false, netsuiteerrors.ErrInvalidRecordID
	}
	var rows []nsbinding.TransactionRow
	if _, err := client.Query(ctx, netsuite.Query{
		SQL:    stateSQL,
		Params: []any{internalID},
		Limit:  1,
	}, &rows); err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	return &rows[0], true, nil
}
