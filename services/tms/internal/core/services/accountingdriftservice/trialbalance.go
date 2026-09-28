package accountingdriftservice

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const retainedEarningsSubType = "RetainedEarnings"

type trialAccount struct {
	id       pulid.ID
	code     string
	label    string
	category string
	netMinor int64
}

type providerAccount struct {
	externalID string
	name       string
	accounts   []*trialAccount
}

type trialBalanceWindow struct {
	start      int64
	before     int64
	incomeFrom int64
	endDate    string
	fromDate   string
}

func isIncomeStatement(category string) bool {
	switch accounttype.Category(category) {
	case accounttype.CategoryRevenue,
		accounttype.CategoryCostOfRevenue,
		accounttype.CategoryExpense:
		return true
	case accounttype.CategoryAsset, accounttype.CategoryLiability, accounttype.CategoryEquity:
		return false
	default:
		return false
	}
}

func trialAccountLabel(code, name string) string {
	return strings.TrimSpace(strings.TrimSpace(code) + " " + strings.TrimSpace(name))
}

func (s *Service) pendingLedgerRecords(ctx context.Context, sess *readSession) (bool, error) {
	counts, err := s.records.CountByStatus(ctx, repositories.AccountingSyncConnectionRequest{
		TenantInfo:   sess.tenant,
		ConnectionID: sess.conn.ID,
	})
	if err != nil {
		return false, err
	}
	for _, count := range counts {
		switch count.Status {
		case accountingsync.SyncStatusQueued,
			accountingsync.SyncStatusInFlight,
			accountingsync.SyncStatusRetrying:
			if count.Count > 0 {
				return true, nil
			}
		case accountingsync.SyncStatusAwaitingApproval,
			accountingsync.SyncStatusSynced,
			accountingsync.SyncStatusBlocked,
			accountingsync.SyncStatusDeadLettered,
			accountingsync.SyncStatusSkipped,
			accountingsync.SyncStatusSuperseded:
		}
	}
	return false, nil
}

func (s *Service) location(ctx context.Context, sess *readSession) (*time.Location, error) {
	org, err := s.organizations.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: sess.tenant,
	})
	if err != nil {
		return nil, err
	}
	if org == nil {
		return time.UTC, nil
	}
	return timeutils.LoadLocation(org.Timezone), nil
}

func (s *Service) reconcileTrialBalance(
	ctx context.Context,
	sess *readSession,
	eventBudget int,
) (*services.AccountingDriftBalanceResult, error) {
	result := new(services.AccountingDriftBalanceResult)
	if sess.ledger == nil || sess.conn.SyncStartDate == nil {
		return result, nil
	}
	control, currency, skip, err := s.trialBalanceGate(ctx, sess)
	if err != nil {
		return nil, err
	}
	if skip {
		result.Skipped = 1
		return result, nil
	}

	loc, err := s.location(ctx, sess)
	if err != nil {
		return nil, err
	}
	now := s.now().Unix()
	window := s.trialWindow(sess.conn, now, loc)

	accounts, err := s.trenovaTrialBalance(ctx, sess, window)
	if err != nil {
		return nil, err
	}
	grouped, err := s.groupByProviderAccount(ctx, sess, control, accounts)
	if err != nil {
		return nil, err
	}
	if len(grouped) == 0 {
		return result, nil
	}

	rows, err := sess.ledger.ReadTrialBalance(ctx, &services.ReadTrialBalanceRequest{
		Auth:      sess.auth,
		StartDate: window.fromDate,
		EndDate:   window.endDate,
	})
	if err != nil {
		s.recordFailure(ctx, sess.conn, s.classify(sess, err))
		return nil, err
	}
	provider := make(map[string]services.AccountingTrialBalanceRow, len(rows))
	for idx := range rows {
		provider[rows[idx].AccountExternalID] = rows[idx]
	}

	compared := make([]pulid.ID, 0, len(accounts))
	for _, group := range grouped {
		for _, account := range group.accounts {
			compared = append(compared, account.id)
		}
	}
	observed := make(map[pulid.ID]*accountingsync.DriftObservation, len(grouped))
	for _, group := range grouped {
		row, found := provider[group.externalID]
		if obs := trialObservation(sess, group, row, found, currency, now); obs != nil {
			observed[obs.ObjectID] = obs
		}
	}

	tally := &compareTally{compared: len(grouped)}
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		return s.applyObservations(txCtx, sess, compared, observed, now, tally)
	})
	if err != nil {
		return nil, err
	}
	result.Customers = tally.compared
	result.Opened = len(tally.opened)
	result.Updated = tally.updated
	result.Resolved = tally.resolved
	result.Events = s.announce(ctx, sess.tenant, tally.opened, eventBudget)
	if result.Opened+result.Resolved+result.Updated > 0 {
		s.publishInvalidation(ctx, sess.tenant, pulid.Nil, sess.conn.ID)
	}
	return result, nil
}

func (s *Service) trialBalanceGate(
	ctx context.Context,
	sess *readSession,
) (control *tenant.AccountingControl, currency string, skip bool, err error) {
	control, err = s.controls.GetByOrgID(ctx, sess.tenant.OrgID)
	if err != nil {
		return nil, "", false, err
	}
	functional := strings.ToUpper(strings.TrimSpace(control.FunctionalCurrencyCode))
	home := strings.ToUpper(strings.TrimSpace(sess.conn.ExternalHomeCurrency))
	if functional != "" && home != "" && functional != home {
		return control, "", true, nil
	}
	pending, err := s.pendingLedgerRecords(ctx, sess)
	if err != nil {
		return nil, "", false, err
	}
	if pending {
		return control, "", true, nil
	}
	if functional == "" {
		return control, home, false, nil
	}
	return control, functional, false, nil
}

func (s *Service) trialWindow(
	conn *accountingsync.AccountingConnection,
	now int64,
	loc *time.Location,
) trialBalanceWindow {
	start := *conn.SyncStartDate
	fiscalStart := conn.FiscalYearStart(now, loc)
	incomeFrom := fiscalStart
	if !conn.SentOpeningBalances() {
		incomeFrom = max(fiscalStart, start)
	}
	return trialBalanceWindow{
		start:      start,
		before:     timeutils.NextDayStart(now, loc),
		incomeFrom: incomeFrom,
		endDate:    timeutils.FormatCalendarDate(now, loc),
		fromDate:   timeutils.FormatCalendarDate(fiscalStart, loc),
	}
}

func (s *Service) trenovaTrialBalance(
	ctx context.Context,
	sess *readSession,
	window trialBalanceWindow,
) ([]*trialAccount, error) {
	byAccount := make(map[pulid.ID]*trialAccount, 32)
	accountOf := func(balance *repositories.LedgerAccountBalance) *trialAccount {
		account, ok := byAccount[balance.AccountID]
		if !ok {
			account = &trialAccount{
				id:       balance.AccountID,
				code:     balance.AccountCode,
				label:    trialAccountLabel(balance.AccountCode, balance.AccountName),
				category: balance.Category,
			}
			byAccount[balance.AccountID] = account
		}
		return account
	}
	add := func(balances []repositories.LedgerAccountBalance, income bool) {
		for idx := range balances {
			balance := &balances[idx]
			if isIncomeStatement(balance.Category) != income {
				continue
			}
			accountOf(balance).netMinor += balance.NetMinor()
		}
	}

	from := window.start
	during, err := s.ledger.SumLines(ctx, &repositories.SumLedgerRequest{
		TenantInfo: sess.tenant,
		From:       &from,
		Before:     window.before,
	})
	if err != nil {
		return nil, err
	}
	add(during, false)
	if sess.conn.SentOpeningBalances() {
		opening, openErr := s.ledger.SumLines(ctx, &repositories.SumLedgerRequest{
			TenantInfo:     sess.tenant,
			Before:         window.start,
			IncludeClosing: true,
		})
		if openErr != nil {
			return nil, openErr
		}
		add(opening, false)
	}
	incomeFrom := window.incomeFrom
	income, err := s.ledger.SumLines(ctx, &repositories.SumLedgerRequest{
		TenantInfo: sess.tenant,
		From:       &incomeFrom,
		Before:     window.before,
	})
	if err != nil {
		return nil, err
	}
	add(income, true)

	out := make([]*trialAccount, 0, len(byAccount))
	for _, account := range byAccount {
		out = append(out, account)
	}
	sortTrialAccounts(out)
	return out, nil
}

func uncoded(account *trialAccount) int {
	if account.code == "" {
		return 1
	}
	return 0
}

func sortTrialAccounts(accounts []*trialAccount) {
	slices.SortFunc(accounts, func(a, b *trialAccount) int {
		return cmp.Or(
			cmp.Compare(uncoded(a), uncoded(b)),
			cmp.Compare(a.code, b.code),
			cmp.Compare(a.label, b.label),
			cmp.Compare(a.id, b.id),
		)
	})
}

func (s *Service) groupByProviderAccount(
	ctx context.Context,
	sess *readSession,
	control *tenant.AccountingControl,
	accounts []*trialAccount,
) ([]*providerAccount, error) {
	mappings, err := s.mappings.ListByConnection(ctx, &repositories.ListAccountingMappingsRequest{
		TenantInfo:   sess.tenant,
		ConnectionID: sess.conn.ID,
		TargetTypes: []accountingsync.MappingTargetType{
			accountingsync.TargetGLAccount,
			accountingsync.TargetAccountRole,
		},
	})
	if err != nil {
		return nil, err
	}
	byAccount := make(map[pulid.ID]*accountingsync.AccountingMapping, len(mappings))
	byRole := make(map[string]*accountingsync.AccountingMapping, 8)
	for _, mapping := range mappings {
		if mapping.State != accountingsync.MappingStateConfirmed || mapping.ExternalID == "" {
			continue
		}
		if mapping.TargetType == accountingsync.TargetGLAccount {
			byAccount[mapping.TrenovaObjectID] = mapping
			continue
		}
		byRole[mapping.TrenovaKey] = mapping
	}

	known := make(map[pulid.ID]*trialAccount, len(accounts))
	for _, account := range accounts {
		known[account.id] = account
	}
	for accountID, mapping := range byAccount {
		if _, ok := known[accountID]; ok {
			continue
		}
		account := &trialAccount{
			id:    accountID,
			label: strings.TrimSpace(strings.TrimPrefix(mapping.TargetLabel, "GL account ")),
		}
		known[accountID] = account
		accounts = append(accounts, account)
	}
	sortTrialAccounts(accounts)

	groups := make(map[string]*providerAccount, len(accounts))
	order := make([]*providerAccount, 0, len(accounts))
	for _, account := range accounts {
		mapping := byAccount[account.id]
		if mapping == nil {
			mapping = byRole[accountingsync.LedgerAccountRole(account.id, control)]
		}
		if mapping == nil {
			continue
		}
		group, ok := groups[mapping.ExternalID]
		if !ok {
			group = &providerAccount{externalID: mapping.ExternalID, name: mapping.ExternalName}
			groups[mapping.ExternalID] = group
			order = append(order, group)
		}
		group.accounts = append(group.accounts, account)
	}
	return s.withoutRetainedEarnings(ctx, sess, order)
}

func (s *Service) withoutRetainedEarnings(
	ctx context.Context,
	sess *readSession,
	groups []*providerAccount,
) ([]*providerAccount, error) {
	if len(groups) == 0 || s.references == nil {
		return groups, nil
	}
	ids := make([]string, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, group.externalID)
	}
	refs, err := s.references.GetByExternalIDs(
		ctx,
		&repositories.GetAccountingReferenceObjectsRequest{
			TenantInfo:   sess.tenant,
			ConnectionID: sess.conn.ID,
			Kind:         accountingsync.ReferenceKindAccount,
			ExternalIDs:  ids,
		},
	)
	if err != nil {
		return nil, err
	}
	retained := make(map[string]struct{}, 1)
	for _, ref := range refs {
		if ref.AccountSubType == retainedEarningsSubType {
			retained[ref.ExternalID] = struct{}{}
		}
	}
	return slices.DeleteFunc(groups, func(group *providerAccount) bool {
		_, skip := retained[group.externalID]
		return skip
	}), nil
}

func trialObservation(
	sess *readSession,
	group *providerAccount,
	row services.AccountingTrialBalanceRow,
	found bool,
	currency string,
	at int64,
) *accountingsync.DriftObservation {
	var trenovaMinor int64
	detail := make([]accountingsync.DriftLine, 0, len(group.accounts))
	for _, account := range group.accounts {
		trenovaMinor += account.netMinor
		detail = append(detail, accountingsync.DriftLine{
			ObjectType:   accountingsync.DriftObjectGLAccount,
			ObjectID:     account.id,
			ObjectNumber: account.label,
			TrenovaMinor: account.netMinor,
		})
	}
	var providerMinor int64
	name := group.name
	if found {
		providerMinor = money.MinorUnits(row.Net())
		if strings.TrimSpace(row.AccountName) != "" {
			name = strings.TrimSpace(row.AccountName)
		}
	}
	if trenovaMinor == providerMinor {
		return nil
	}
	if name == "" {
		name = group.externalID
	}
	return &accountingsync.DriftObservation{
		TenantInfo:    sess.tenant,
		ConnectionID:  sess.conn.ID,
		ObjectType:    accountingsync.DriftObjectGLAccount,
		ObjectID:      group.accounts[0].id,
		ObjectNumber:  name,
		ExternalID:    group.externalID,
		Kind:          accountingsync.DriftTrialBalanceMismatch,
		CurrencyCode:  currency,
		TrenovaMinor:  &trenovaMinor,
		ProviderMinor: &providerMinor,
		Detail:        detail,
		At:            at,
	}
}
