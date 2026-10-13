package nsconnector

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/shared/netsuite"
)

var _ accountingsyncports.AccountingDocumentReader = (*Connector)(nil)

const maxIDsPerRead = 500

func (c *Connector) DocumentReadLimits() accountingsyncports.AccountingDocumentReadLimits {
	return accountingsyncports.AccountingDocumentReadLimits{MaxPerRead: maxIDsPerRead}
}

func readable(kind accountingsync.SyncObjectType) (bool, error) {
	switch {
	case kind.IsSalesDocument(), kind.IsBill(), kind.IsBillPayment(),
		kind == accountingsync.SyncObjectCustomerPayment:
		return true, nil
	case kind == accountingsync.SyncObjectCreditApplication:
		return false, nil
	default:
		return false, errDocumentKind
	}
}

func (c *Connector) ReadDocuments(
	ctx context.Context,
	req *accountingsyncports.ReadAccountingDocumentsRequest,
) ([]*accountingsyncports.AccountingDocumentState, error) {
	ok, err := readable(req.Kind)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(req.Targets))
	for idx := range req.Targets {
		if id, valid := parseInternalID(req.Targets[idx].ExternalID); valid && ok {
			ids = append(ids, id)
		}
	}
	found := make(map[string]*transactionRow, len(ids))
	if len(ids) > 0 {
		s, openErr := c.open(req.Auth)
		if openErr != nil {
			return nil, openErr
		}
		for start := 0; start < len(ids); start += maxIDsPerRead {
			rows, readErr := readStates(
				ctx,
				s.client,
				ids[start:min(start+maxIDsPerRead, len(ids))],
			)
			if readErr != nil {
				return nil, readErr
			}
			for idx := range rows {
				found[rows[idx].id()] = &rows[idx]
			}
		}
	}
	out := make([]*accountingsyncports.AccountingDocumentState, 0, len(req.Targets))
	for idx := range req.Targets {
		id := strings.TrimSpace(req.Targets[idx].ExternalID)
		state := &accountingsyncports.AccountingDocumentState{ExternalID: id}
		if row, hit := found[id]; hit {
			balance := row.remaining()
			state.Found = true
			state.Voided = row.Voided.Value()
			state.DocNumber = row.TranID.Value()
			state.Total = row.total()
			state.Balance = &balance
			state.CurrencyCode = strings.ToUpper(row.Currency.Value())
			state.ModifiedAt = unixOrZero(row.modified())
		}
		out = append(out, state)
	}
	return out, nil
}

func readStates(
	ctx context.Context,
	client *netsuite.Client,
	ids []int64,
) ([]transactionRow, error) {
	params := make([]any, 0, len(ids))
	marks := make([]string, 0, len(ids))
	for _, id := range ids {
		params = append(params, id)
		marks = append(marks, "?")
	}
	b := (&sqlBuilder{}).where("t.id IN ("+strings.Join(marks, ", ")+")", params...)
	var rows []transactionRow
	if _, err := client.Query(ctx, netsuite.Query{
		SQL:    b.sql(transactionSQL, "t.id"),
		Params: b.params,
	}, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}
