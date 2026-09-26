package accountingdriftservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type customerBalance struct {
	id       pulid.ID
	name     string
	currency string
	lines    []*repositories.AccountingDriftBalanceLine
}

func groupByCustomer(lines []*repositories.AccountingDriftBalanceLine) []*customerBalance {
	out := make([]*customerBalance, 0, len(lines))
	var current *customerBalance
	for _, line := range lines {
		if current == nil || current.id != line.CustomerID {
			current = &customerBalance{
				id:       line.CustomerID,
				name:     line.CustomerName,
				currency: line.CurrencyCode,
			}
			out = append(out, current)
		}
		if line.CurrencyCode == current.currency {
			current.lines = append(current.lines, line)
		}
	}
	return out
}

func (s *Service) ReconcileBalances(
	ctx context.Context,
	req *services.ReconcileAccountingDriftBalancesRequest,
) (*services.AccountingDriftBalanceResult, error) {
	result := &services.AccountingDriftBalanceResult{LastCustomerID: req.AfterCustomerID}
	sess, held, err := s.startRead(ctx, req.TenantInfo, req.ConnectionID)
	if err != nil || held {
		result.Held = held
		return result, err
	}

	datedFrom, err := s.source.ScopeStart(ctx, sess.tenant)
	if err != nil {
		return nil, err
	}
	limit := req.Customers
	if limit <= 0 {
		limit = defaultCustomers
	}
	limit = intutils.Clamp(limit, 1, maxCustomers)
	lines, err := s.source.ListBalances(ctx, &repositories.ListAccountingDriftBalancesRequest{
		TenantInfo:      sess.tenant,
		ConnectionID:    sess.conn.ID,
		DatedFrom:       datedFrom,
		AfterCustomerID: req.AfterCustomerID,
		Customers:       limit,
	})
	if err != nil {
		return nil, err
	}
	customers := groupByCustomer(lines)
	if len(customers) > 0 {
		result.LastCustomerID = customers[len(customers)-1].id
	}
	result.More = len(customers) == limit

	tally, err := s.compareBalances(ctx, sess, customers)
	if err != nil {
		return nil, err
	}
	result.Customers = tally.compared
	result.Skipped = tally.skipped
	result.Opened = len(tally.opened)
	result.Updated = tally.updated
	result.Resolved = tally.resolved
	result.Events = s.announce(ctx, sess.tenant, tally.opened, req.EventBudget)
	if result.Opened+result.Resolved+result.Updated > 0 {
		s.publishInvalidation(ctx, sess.tenant, pulid.Nil, sess.conn.ID)
	}
	return result, nil
}

func (s *Service) compareBalances(
	ctx context.Context,
	sess *readSession,
	customers []*customerBalance,
) (*compareTally, error) {
	tally := new(compareTally)
	if len(customers) == 0 {
		return tally, nil
	}
	ids := make([]pulid.ID, 0, len(customers))
	for _, customer := range customers {
		ids = append(ids, customer.id)
	}
	pending, err := s.source.ListPendingCustomers(
		ctx,
		&repositories.ListAccountingDriftPendingCustomersRequest{
			TenantInfo:   sess.tenant,
			ConnectionID: sess.conn.ID,
			CustomerIDs:  ids,
		},
	)
	if err != nil {
		return nil, err
	}
	waiting := make(map[pulid.ID]struct{}, len(pending))
	for _, id := range pending {
		waiting[id] = struct{}{}
	}

	ready := make([]*customerBalance, 0, len(customers))
	for _, customer := range customers {
		if _, skip := waiting[customer.id]; skip {
			tally.skipped++
			continue
		}
		ready = append(ready, customer)
	}
	provider, err := s.readBalances(ctx, sess, ready)
	if err != nil {
		return nil, err
	}

	at := s.now().Unix()
	observed := make(map[pulid.ID]*accountingsync.DriftObservation, len(ready))
	compared := make([]pulid.ID, 0, len(ready))
	for _, customer := range ready {
		obs := balanceObservation(sess, customer, provider, at)
		compared = append(compared, customer.id)
		observed[customer.id] = obs
	}
	tally.compared = len(compared)
	if len(compared) == 0 {
		return tally, nil
	}

	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		return s.applyObservations(txCtx, sess, compared, observed, at, tally)
	})
	if err != nil {
		return nil, err
	}
	return tally, nil
}

func (s *Service) readBalances(
	ctx context.Context,
	sess *readSession,
	customers []*customerBalance,
) (map[string]*services.AccountingDocumentState, error) {
	byType := make(map[accountingsync.SyncObjectType][]*accountingsync.AccountingSyncRecord, 2)
	for _, customer := range customers {
		for _, line := range customer.lines {
			byType[line.ObjectType] = append(
				byType[line.ObjectType],
				&accountingsync.AccountingSyncRecord{
					ObjectType: line.ObjectType,
					ObjectID:   line.ObjectID,
					ExternalID: line.ExternalID,
				},
			)
		}
	}
	out := make(map[string]*services.AccountingDocumentState, len(customers))
	for objectType, records := range byType {
		states, err := s.readProvider(ctx, sess, objectType, records)
		if err != nil {
			return nil, err
		}
		for id, state := range states {
			out[string(objectType)+":"+id] = state
		}
	}
	return out, nil
}

func balanceObservation(
	sess *readSession,
	customer *customerBalance,
	provider map[string]*services.AccountingDocumentState,
	at int64,
) *accountingsync.DriftObservation {
	var trenovaTotal, providerTotal int64
	detail := make([]accountingsync.DriftLine, 0, len(customer.lines))
	for _, line := range customer.lines {
		state := provider[string(line.ObjectType)+":"+line.ExternalID]
		if state == nil || !state.Found || state.Voided || state.Balance == nil {
			continue
		}
		providerMinor := money.MinorUnits(*state.Balance)
		trenovaTotal += line.OpenMinor
		providerTotal += providerMinor
		if providerMinor != line.OpenMinor {
			detail = append(detail, accountingsync.DriftLine{
				ObjectType:    line.ObjectType,
				ObjectID:      line.ObjectID,
				ObjectNumber:  line.Number,
				TrenovaMinor:  line.OpenMinor,
				ProviderMinor: providerMinor,
			})
		}
	}
	if trenovaTotal == providerTotal {
		return nil
	}
	return &accountingsync.DriftObservation{
		TenantInfo:    sess.tenant,
		ConnectionID:  sess.conn.ID,
		ObjectType:    accountingsync.SyncObjectCustomer,
		ObjectID:      customer.id,
		ObjectNumber:  customer.name,
		PartyID:       customer.id,
		PartyName:     customer.name,
		Kind:          accountingsync.DriftCustomerBalanceMismatch,
		CurrencyCode:  customer.currency,
		TrenovaMinor:  &trenovaTotal,
		ProviderMinor: &providerTotal,
		Detail:        detail,
		At:            at,
	}
}
