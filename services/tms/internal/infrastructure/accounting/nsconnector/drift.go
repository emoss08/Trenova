package nsconnector

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsdocsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
)

var _ accountingconnectorports.AccountingDocumentReader = (*Connector)(nil)

func (c *Connector) DocumentReadLimits() accountingconnectorports.AccountingDocumentReadLimits {
	return accountingconnectorports.AccountingDocumentReadLimits{
		MaxPerRead: nsbinding.MaxIDsPerRead,
	}
}

func readable(kind accountingsync.SyncObjectType) (bool, error) {
	switch {
	case kind.IsSalesDocument(), kind.IsBill(), kind.IsBillPayment(),
		kind == accountingsync.SyncObjectCustomerPayment:
		return true, nil
	case kind == accountingsync.SyncObjectCreditApplication:
		return false, nil
	default:
		return false, nsbinding.ErrDocumentKind
	}
}

func (c *Connector) ReadDocuments(
	ctx context.Context,
	req *accountingconnectorports.ReadAccountingDocumentsRequest,
) ([]*accountingconnectorports.AccountingDocumentState, error) {
	ok, err := readable(req.Kind)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(req.Targets))
	for idx := range req.Targets {
		if id, valid := nsbinding.ParseInternalID(req.Targets[idx].ExternalID); valid && ok {
			ids = append(ids, id)
		}
	}
	found, editors, err := c.readFound(ctx, req.Auth, ids)
	if err != nil {
		return nil, err
	}
	out := make([]*accountingconnectorports.AccountingDocumentState, 0, len(req.Targets))
	for idx := range req.Targets {
		id := strings.TrimSpace(req.Targets[idx].ExternalID)
		state := &accountingconnectorports.AccountingDocumentState{ExternalID: id}
		if row, hit := found[id]; hit {
			balance := row.AbsRemaining()
			state.Found = true
			state.Voided = row.Voided.Value()
			state.DocNumber = row.TranID.Value()
			state.Total = row.AbsTotal()
			state.Balance = &balance
			state.CurrencyCode = strings.ToUpper(row.Currency.Value())
			state.ModifiedAt = unixOrZero(row.ModifiedAt())
			state.ModifiedBy = editors[id]
		}
		out = append(out, state)
	}
	return out, nil
}

func (c *Connector) readFound(
	ctx context.Context,
	auth accountingconnectorports.AccountingDocumentAuth,
	ids []int64,
) (found map[string]*nsbinding.TransactionRow, editors map[string]string, err error) {
	found = make(map[string]*nsbinding.TransactionRow, len(ids))
	if len(ids) == 0 {
		return found, map[string]string{}, nil
	}
	s, openErr := c.Open(auth)
	if openErr != nil {
		return nil, nil, openErr
	}
	hits := make([]int64, 0, len(ids))
	for start := 0; start < len(ids); start += nsbinding.MaxIDsPerRead {
		rows, readErr := readStates(
			ctx,
			s.Client,
			ids[start:min(start+nsbinding.MaxIDsPerRead, len(ids))],
		)
		if readErr != nil {
			return nil, nil, readErr
		}
		for idx := range rows {
			found[rows[idx].IDValue()] = &rows[idx]
			if id, ok := nsbinding.ParseInternalID(rows[idx].IDValue()); ok {
				hits = append(hits, id)
			}
		}
	}
	editors, err = (&nsbinding.EditorBook{Client: s.Client}).Names(ctx, hits)
	if err != nil {
		return nil, nil, err
	}
	return found, editors, nil
}

func readStates(
	ctx context.Context,
	client *netsuite.Client,
	ids []int64,
) ([]nsbinding.TransactionRow, error) {
	marks, params := nsbinding.IDList(ids)
	b := (&nsbinding.SQLBuilder{}).Where("t.id IN ("+marks+")", params...)
	var rows []nsbinding.TransactionRow
	if _, err := client.Query(ctx, netsuite.Query{
		SQL:    b.SQL(nsdocsync.TransactionSQL, "t.id"),
		Params: b.Params,
	}, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}
