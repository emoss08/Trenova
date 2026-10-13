package nsconnector

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/shopspring/decimal"
)

const (
	reversalNote  = "Reverses "
	externalIDSQL = transactionSQL + " WHERE t.type = ? AND t.externalid = ?"
)

func salesLocked(row *transactionRow) bool {
	return row.Closed.Value() || row.AllLocked.Value() || row.ARLocked.Value()
}

func purchaseLocked(row *transactionRow) bool {
	return row.Closed.Value() || row.AllLocked.Value() || row.APLocked.Value()
}

type reversal struct {
	recordType  string
	reverseType string
	reverseCode string
	id          string
	requestID   string
	locked      func(*transactionRow) bool
}

func (c *Connector) retireDocument(ctx context.Context, s *session, r *reversal) (string, error) {
	state, found, err := transactionState(ctx, s.client, r.id)
	if err != nil {
		if netsuite.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}
	if !found || state.Voided.Value() {
		return "", nil
	}
	locked := r.locked(state)
	if locked || state.applied() {
		held, heldOK, findErr := findByExternalID(ctx, s.client, r.reverseCode, r.requestID)
		if findErr != nil {
			return "", findErr
		}
		if heldOK {
			return held.id(), nil
		}
		if state.applied() {
			return "", paidConflict()
		}
	}
	if !locked {
		err = s.client.Delete(ctx, r.recordType, r.id)
		switch {
		case err == nil, netsuite.IsNotFound(err):
			return "", nil
		case !deleteRefused(err):
			return "", err
		}
	}
	return c.reverse(ctx, s, r, state)
}

func findByExternalID(
	ctx context.Context,
	client *netsuite.Client,
	typeCode, externalID string,
) (*transactionRow, bool, error) {
	if !validRequestID(externalID) {
		return nil, false, nil
	}
	var rows []transactionRow
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

func (c *Connector) reverse(
	ctx context.Context,
	s *session,
	r *reversal,
	state *transactionRow,
) (string, error) {
	var record transactionRecord
	if err := s.client.Get(ctx, netsuite.RecordRead{
		Type:               r.recordType,
		ID:                 r.id,
		ExpandSubResources: true,
	}, &record); err != nil {
		return "", err
	}
	body, err := c.reversalBody(&record, state, r.id)
	if err != nil {
		return "", err
	}
	saved, err := writeRecord(ctx, s.client, &recordWrite{
		recordType: r.reverseType,
		externalID: r.requestID,
		body:       body,
	})
	if saved == nil {
		return "", err
	}
	return saved.id, err
}

func (c *Connector) reversalBody(
	record *transactionRecord,
	state *transactionRow,
	id string,
) (*transactionBody, error) {
	body := &transactionBody{
		Entity:     refOf(record.party().ID.Value()),
		Subsidiary: refOf(record.Subsidiary.ID.Value()),
		Currency:   refOf(record.Currency.ID.Value()),
		TranDate:   c.now().UTC().Format(dateLayout),
		Memo:       cleanText(reversalNote+record.TranID, maxMemoLength),
		Apply: &sublist[applyLine]{Items: []applyLine{{
			Doc:    recordRef{ID: id},
			Apply:  true,
			Amount: amountOf(state.total()),
		}}},
	}
	if record.ExchangeRate.IsPositive() {
		body.ExchangeRate = amountOf(record.ExchangeRate)
	}
	if items := reversedItems(record.Item.Items); len(items) > 0 {
		body.Item = &sublist[itemLine]{Items: items}
	}
	if expenses := reversedExpenses(record.Expense.Items); len(expenses) > 0 {
		body.Expense = &sublist[expenseLine]{Items: expenses}
	}
	if body.Item == nil && body.Expense == nil {
		return nil, reversalLinesUnknown()
	}
	return body, nil
}

func reversedItems(lines []recordItemLine) []itemLine {
	out := make([]itemLine, 0, len(lines))
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
		out = append(out, itemLine{
			Item:        recordRef{ID: item},
			Description: cleanText(line.Description, maxLineTextLength),
			Quantity:    amount(quantity),
			Rate:        amount(rate),
			Amount:      amount(line.Amount),
		})
	}
	return out
}

func reversedExpenses(lines []recordExpenseLine) []expenseLine {
	out := make([]expenseLine, 0, len(lines))
	for idx := range lines {
		line := &lines[idx]
		account := line.Account.ID.Value()
		if account == "" {
			continue
		}
		out = append(out, expenseLine{
			Account: recordRef{ID: account},
			Amount:  amount(line.Amount),
			Memo:    cleanText(line.Memo, maxLineTextLength),
		})
	}
	return out
}

func reversalLinesUnknown() *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorConfiguration,
		Code:     "reversal-lines",
		Message: providerName + " returned no lines for the document, so Trenova cannot " +
			"reverse it with a credit",
		Resolution: "Credit the document in " + providerName +
			", then mark this record resolved",
	}
}

type unwind struct {
	recordType string
	id         string
	locked     func(*transactionRow) bool
	refused    func() error
}

func (c *Connector) unwindRecord(ctx context.Context, s *session, u *unwind) error {
	state, found, err := transactionState(ctx, s.client, u.id)
	if err != nil {
		if netsuite.IsNotFound(err) {
			return nil
		}
		return err
	}
	if !found || state.Voided.Value() {
		return nil
	}
	if u.locked(state) {
		return u.refused()
	}
	if err = unapplyAll(ctx, s.client, u.recordType, u.id); err != nil {
		return u.settle(err)
	}
	return u.settle(s.client.Delete(ctx, u.recordType, u.id))
}

func (u *unwind) settle(err error) error {
	switch {
	case err == nil, netsuite.IsNotFound(err):
		return nil
	case deleteRefused(err):
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
	var record transactionRecord
	if err := client.Get(ctx, netsuite.RecordRead{
		Type:               recordType,
		ID:                 id,
		ExpandSubResources: true,
	}, &record); err != nil {
		return err
	}
	docs := record.appliedDocs()
	if len(docs) == 0 {
		return nil
	}
	return client.Patch(ctx, netsuite.RecordPatch{
		Type: recordType,
		ID:   id,
		Body: &transactionBody{Apply: unapplyLines(docs)},
	})
}
