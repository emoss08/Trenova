package nsdocsync

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsrefsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
	"github.com/shopspring/decimal"
)

const (
	reversalNote  = "Reverses "
	externalIDSQL = TransactionSQL + " WHERE t.type = ? AND t.externalid = ?"
)

func salesLocked(row *nsbinding.TransactionRow) bool {
	return row.Closed.Value() || row.AllLocked.Value() || row.ARLocked.Value()
}

func purchaseLocked(row *nsbinding.TransactionRow) bool {
	return row.Closed.Value() || row.AllLocked.Value() || row.APLocked.Value()
}

type Reversal struct {
	recordType  string
	reverseType string
	reverseCode string
	ID          string
	requestID   string
	locked      func(*nsbinding.TransactionRow) bool
	offset      bool
}

func (c *Documents) retireDocument(
	ctx context.Context,
	s *nsbinding.Session,
	r *Reversal,
) (string, error) {
	state, found, err := transactionState(ctx, s.Client, r.ID)
	if err != nil {
		if netsuiteerrors.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}
	if !found || state.Voided.Value() {
		return "", nil
	}
	locked := r.locked(state)
	if locked || state.Applied() {
		held, heldOK, findErr := findByExternalID(ctx, s.Client, r.reverseCode, r.requestID)
		if findErr != nil {
			return "", findErr
		}
		if heldOK {
			return held.IDValue(), nil
		}
		if state.Applied() {
			return "", paidConflict()
		}
	}
	if !locked {
		err = s.Client.Delete(ctx, r.recordType, r.ID)
		switch {
		case err == nil, netsuiteerrors.IsNotFound(err):
			return "", nil
		case !DeleteRefused(err):
			return "", err
		}
	}
	return c.reverse(ctx, s, r, state)
}

func findByExternalID(
	ctx context.Context,
	client *netsuite.Client,
	typeCode, externalID string,
) (*nsbinding.TransactionRow, bool, error) {
	if !validRequestID(externalID) {
		return nil, false, nil
	}
	var rows []nsbinding.TransactionRow
	if _, err := client.Query(ctx, netsuite.Query{
		SQL:    externalIDSQL,
		Params: []any{typeCode, externalID},
		Limit:  1,
	}, &rows); err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	return &rows[0], true, nil
}

func validRequestID(raw string) bool {
	if raw == "" || len(raw) > netsuite.MaxExternalIDLength {
		return false
	}
	for idx := range len(raw) {
		ch := raw[idx]
		valid := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '_'
		if !valid {
			return false
		}
	}
	return true
}

func (c *Documents) reverse(
	ctx context.Context,
	s *nsbinding.Session,
	r *Reversal,
	state *nsbinding.TransactionRow,
) (string, error) {
	var record nsbinding.TransactionRecord
	if err := s.Client.Get(ctx, netsuite.RecordRead{
		Type:               r.recordType,
		ID:                 r.ID,
		ExpandSubResources: true,
	}, &record); err != nil {
		return "", err
	}
	body, err := c.reversalBody(&record, state, r)
	if err != nil {
		return "", err
	}
	saved, err := writeRecord(ctx, s.Client, &recordWrite{
		recordType: r.reverseType,
		externalID: r.requestID,
		body:       body,
	})
	if saved == nil {
		return "", err
	}
	return saved.ID, err
}

func (c *Documents) reversalBody(
	record *nsbinding.TransactionRecord,
	state *nsbinding.TransactionRow,
	r *Reversal,
) (*nsbinding.TransactionBody, error) {
	body := &nsbinding.TransactionBody{
		Entity:     nsrefsync.RefOf(record.Party().ID.Value()),
		Subsidiary: nsrefsync.RefOf(record.Subsidiary.ID.Value()),
		Currency:   nsrefsync.RefOf(record.Currency.ID.Value()),
		TranDate:   c.Clock().UTC().Format(DateLayout),
		Memo:       nsrefsync.CleanText(reversalNote+record.TranID, maxMemoLength),
	}
	if r.offset {
		body.Account = nsrefsync.RefOf(record.Account.ID.Value())
	} else {
		body.Apply = &nsbinding.Sublist[nsbinding.ApplyLine]{Items: []nsbinding.ApplyLine{{
			Doc:    nsbinding.RecordRef{ID: r.ID},
			Apply:  true,
			Amount: nsbinding.AmountOf(state.AbsTotal()),
		}}}
	}
	if record.ExchangeRate.IsPositive() {
		body.ExchangeRate = nsbinding.AmountOf(record.ExchangeRate)
	}
	if items := reversedItems(record.Item.Items); len(items) > 0 {
		body.Item = &nsbinding.Sublist[nsbinding.ItemLine]{Items: items}
	}
	if expenses := reversedExpenses(record.Expense.Items); len(expenses) > 0 {
		body.Expense = &nsbinding.Sublist[nsbinding.ExpenseLine]{Items: expenses}
	}
	if body.Item == nil && body.Expense == nil {
		return nil, reversalLinesUnknown()
	}
	return body, nil
}

func reversedItems(lines []nsbinding.RecordItemLine) []nsbinding.ItemLine {
	out := make([]nsbinding.ItemLine, 0, len(lines))
	for idx := range lines {
		line := &lines[idx]
		item := line.Item.ID.Value()
		if item == "" {
			continue
		}
		quantity, rate := line.Quantity, line.Rate
		if !quantity.IsPositive() || !sameMoney(quantity.Mul(rate), line.Amount) {
			quantity, rate = decimal.NewFromInt(1), line.Amount
		}
		out = append(out, nsbinding.ItemLine{
			Item:        nsbinding.RecordRef{ID: item},
			Description: nsrefsync.CleanText(line.Description, nsrefsync.MaxLineTextLength),
			Quantity:    nsbinding.Amount(quantity),
			Rate:        nsbinding.Amount(rate),
			Amount:      nsbinding.Amount(line.Amount),
		})
	}
	return out
}

func reversedExpenses(lines []nsbinding.RecordExpenseLine) []nsbinding.ExpenseLine {
	out := make([]nsbinding.ExpenseLine, 0, len(lines))
	for idx := range lines {
		line := &lines[idx]
		account := line.Account.ID.Value()
		if account == "" {
			continue
		}
		out = append(out, nsbinding.ExpenseLine{
			Account: nsbinding.RecordRef{ID: account},
			Amount:  nsbinding.Amount(line.Amount),
			Memo:    nsrefsync.CleanText(line.Memo, nsrefsync.MaxLineTextLength),
		})
	}
	return out
}

func reversalLinesUnknown() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "reversal-lines",
		Message: nsbinding.ProviderName + " returned no lines for the document, so Trenova cannot " +
			"reverse it with a credit",
		Resolution: "Credit the document in " + nsbinding.ProviderName +
			", then mark this record resolved",
	}
}

type unwind struct {
	recordType string
	ID         string
	locked     func(*nsbinding.TransactionRow) bool
	refused    func() error
	offset     *creditOffset
}

type creditOffset struct {
	recordType string
	typeCode   string
	requestID  string
}

func (c *Documents) unwindRecord(
	ctx context.Context,
	s *nsbinding.Session,
	u *unwind,
) (string, error) {
	state, found, err := transactionState(ctx, s.Client, u.ID)
	if err != nil {
		if netsuiteerrors.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}
	if !found || state.Voided.Value() {
		return "", nil
	}
	if u.locked(state) {
		return c.offsetCredit(ctx, s, u, state)
	}
	if err = unapplyAll(ctx, s.Client, u.recordType, u.ID); err != nil {
		return "", u.settle(err)
	}
	return "", u.settle(s.Client.Delete(ctx, u.recordType, u.ID))
}

func invoiceOffset(requestID string) *creditOffset {
	return &creditOffset{recordType: RecordInvoice, typeCode: TypeInvoice, requestID: requestID}
}

func shortPayOffsetID(requestID, memo string) string {
	if requestID == "" {
		return ""
	}
	return requestID + "-" + memo
}

func shortPayCorrective(key string) string {
	return RefCorrective + "-" + strings.TrimPrefix(key, accountingsync.ExternalRefShortPayPrefix)
}

func (c *Documents) offsetCredit(
	ctx context.Context,
	s *nsbinding.Session,
	u *unwind,
	state *nsbinding.TransactionRow,
) (string, error) {
	if u.offset == nil || !validRequestID(u.offset.requestID) {
		return "", u.refused()
	}
	held, ok, err := findByExternalID(ctx, s.Client, u.offset.typeCode, u.offset.requestID)
	if err != nil {
		return "", err
	}
	if ok {
		return held.IDValue(), nil
	}
	if state.Applied() {
		return "", paidConflict()
	}
	return c.reverse(ctx, s, &Reversal{
		recordType:  u.recordType,
		reverseType: u.offset.recordType,
		reverseCode: u.offset.typeCode,
		ID:          u.ID,
		requestID:   u.offset.requestID,
		offset:      true,
	}, state)
}

func (u *unwind) settle(err error) error {
	switch {
	case err == nil, netsuiteerrors.IsNotFound(err):
		return nil
	case DeleteRefused(err):
		return errors.Join(u.refused(), err)
	default:
		return err
	}
}

func unapplyAll(
	ctx context.Context,
	client *netsuite.Client,
	recordType, id string,
) error {
	var record nsbinding.TransactionRecord
	if err := client.Get(ctx, netsuite.RecordRead{
		Type:               recordType,
		ID:                 id,
		ExpandSubResources: true,
	}, &record); err != nil {
		return err
	}
	docs := record.AppliedDocs()
	if len(docs) == 0 {
		return nil
	}
	return client.Patch(ctx, netsuite.RecordPatch{
		Type: recordType,
		ID:   id,
		Body: &nsbinding.TransactionBody{Apply: unapplyLines(docs)},
	})
}
