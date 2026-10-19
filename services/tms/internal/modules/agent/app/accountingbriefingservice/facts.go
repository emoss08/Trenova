package accountingbriefingservice

import (
	"context"
	"math"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/accountingbrief"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"golang.org/x/sync/errgroup"
)

const (
	secondsPerDay    = 24 * 60 * 60
	topCustomerLimit = 1
	worklistLimit    = 200
)

type ledgerSnapshot struct {
	kpis      *customerledgerports.ARDashboardKPIs
	stats     *customerledgerports.ARPaymentStats
	top       []*customerledgerports.ARTopOverdueCustomer
	openItems []*customerledgerports.AROpenItem
	worklist  *collectionports.WorklistPage
	drafts    *customerledgerports.ARDraftInvoices
	collected *customerledgerports.ARCollectionPerformance
}

func (s *Service) Facts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timezone string,
) (*accountingbrief.Facts, error) {
	now := s.now().Unix()
	snapshot, err := s.read(ctx, tenantInfo, now)
	if err != nil {
		return nil, err
	}

	facts := &accountingbrief.Facts{Timezone: timezone}
	foldBalances(facts, snapshot)
	foldDrafts(facts, snapshot.drafts, now)
	foldOpenItems(facts, snapshot.openItems, now)
	foldWorklist(facts, snapshot.worklist, now)

	return facts, nil
}

func (s *Service) read(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	now int64,
) (*ledgerSnapshot, error) {
	snapshot := new(ledgerSnapshot)
	analytics := customerledgerports.GetARAnalyticsRequest{TenantInfo: tenantInfo}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		snapshot.kpis, err = s.receivables.GetDashboardKPIs(gctx, analytics)
		return err
	})
	g.Go(func() error {
		var err error
		snapshot.stats, err = s.receivables.GetPaymentStats(gctx, analytics)
		return err
	})
	g.Go(func() error {
		var err error
		snapshot.top, err = s.receivables.ListTopOverdueCustomers(
			gctx,
			customerledgerports.ListARTopOverdueCustomersRequest{
				TenantInfo: tenantInfo,
				Limit:      topCustomerLimit,
			},
		)
		return err
	})
	g.Go(func() error {
		var err error
		snapshot.openItems, err = s.receivables.ListOpenItems(gctx, tenantInfo, pulid.Nil, now)
		return err
	})
	g.Go(func() error {
		var err error
		snapshot.worklist, err = s.collections.Worklist(gctx, &collectionports.WorklistRequest{
			TenantInfo: tenantInfo,
			Scope:      collectionports.CollectorScopeAll,
			Limit:      worklistLimit,
		})
		return err
	})
	g.Go(func() error {
		var err error
		snapshot.drafts, err = s.receivables.GetDraftInvoices(gctx, analytics)
		return err
	})
	g.Go(func() error {
		var err error
		snapshot.collected, err = s.receivables.GetCollectionPerformance(
			gctx,
			customerledgerports.GetARCollectionMetricsRequest{
				TenantInfo: tenantInfo,
				PeriodDays: accountingbrief.CollectionWindowDays,
			},
		)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return snapshot, nil
}

func foldBalances(facts *accountingbrief.Facts, snapshot *ledgerSnapshot) {
	if kpis := snapshot.kpis; kpis != nil {
		facts.DSOTenths = toTenths(kpis.CurrentDSODays)
		facts.DSODeltaTenths = toTenths(kpis.DSODeltaDays)
		facts.CEIPercent = int(math.Round(kpis.CEI))
		if overview := kpis.Overview; overview != nil {
			facts.OpenMinor = overview.TotalOpenMinor
			facts.OpenInvoices = overview.OpenInvoiceCount
			facts.OverdueMinor = overview.OverdueMinor
			facts.OverdueInvoices = overview.OverdueInvoiceCount
			facts.PastNinetyMinor = overview.Buckets.DaysOver90Minor
			facts.DisputedMinor = overview.DisputedOpenMinor
			facts.DisputedInvoices = overview.DisputedInvoiceCount
			facts.UnappliedCashMinor = overview.UnappliedCashMinor
		}
	}
	if stats := snapshot.stats; stats != nil {
		facts.UnappliedPayments = stats.UnappliedPaymentCount
		facts.CollectedTodayMinor = stats.PostedTodayMinor
		facts.PaymentsToday = stats.PostedTodayCount
	}
	if collected := snapshot.collected; collected != nil && collected.Totals != nil {
		facts.CollectedWindowMinor = collected.Totals.CollectedMinor
		facts.BilledWindowMinor = collected.Totals.CreditSalesMinor
	}
	if len(snapshot.top) > 0 && snapshot.top[0] != nil && snapshot.top[0].OverdueMinor > 0 {
		facts.TopCustomerID = snapshot.top[0].CustomerID
		facts.TopCustomerName = strings.TrimSpace(snapshot.top[0].CustomerName)
		facts.TopCustomerOverdueMinor = snapshot.top[0].OverdueMinor
	}
}

func foldDrafts(
	facts *accountingbrief.Facts,
	drafts *customerledgerports.ARDraftInvoices,
	now int64,
) {
	if drafts == nil || drafts.Count <= 0 {
		return
	}
	facts.DraftInvoices = drafts.Count
	facts.DraftMinor = drafts.AmountMinor
	if drafts.OldestCreatedAt > 0 && drafts.OldestCreatedAt <= now {
		facts.OldestDraftDays = int((now - drafts.OldestCreatedAt) / secondsPerDay)
	}
}

func daysUntil(at, now int64) int {
	return int((at - now + secondsPerDay - 1) / secondsPerDay)
}

func foldOpenItems(
	facts *accountingbrief.Facts,
	items []*customerledgerports.AROpenItem,
	now int64,
) {
	crossingFrom := accountingbrief.PastDueThresholdDays - accountingbrief.CrossingWindowDays
	dueBy := now + accountingbrief.DueSoonDays*secondsPerDay
	dueThisMonthBy := now + accountingbrief.DueThisMonthDays*secondsPerDay

	for _, item := range items {
		if item == nil || item.OpenAmountMinor <= 0 {
			continue
		}
		if item.DaysPastDue >= crossingFrom &&
			item.DaysPastDue < accountingbrief.PastDueThresholdDays {
			facts.CrossingNinetyInvoices++
			facts.CrossingNinetyMinor += item.OpenAmountMinor
		}
		if item.DaysPastDue == 0 && item.DueDate > now {
			if item.DueDate <= dueBy {
				facts.DueSoonInvoices++
				facts.DueSoonMinor += item.OpenAmountMinor
			}
			if item.DueDate <= dueThisMonthBy {
				facts.DueThisMonthInvoices++
				facts.DueThisMonthMinor += item.OpenAmountMinor
			}
			if next := daysUntil(item.DueDate, now); facts.NextDueInDays == 0 ||
				next < facts.NextDueInDays {
				facts.NextDueInDays = next
			}
		}
		if item.HasShortPay && item.OpenAmountMinor <= accountingbrief.ShortPayRemainderMaxMinor {
			facts.ShortPayRemainders++
			facts.ShortPayRemainderMinor += item.OpenAmountMinor
		}
	}
}

func foldWorklist(facts *accountingbrief.Facts, page *collectionports.WorklistPage, now int64) {
	if page == nil {
		return
	}

	facts.FollowUpsDue = page.Summary.FollowUpsDue
	promiseBy := now + accountingbrief.PromiseLookaheadDays*secondsPerDay
	quietSince := now - accountingbrief.UncontactedDays*secondsPerDay
	var largestBroken int64

	for _, row := range page.Rows {
		if row == nil {
			continue
		}

		promise := row.Promise
		if promise != nil {
			unpaid := max(promise.AmountMinor-promise.PaidSinceMinor, 0)
			switch promise.Status {
			case collection.PromiseBroken:
				facts.BrokenPromises++
				facts.BrokenPromiseMinor += unpaid
				if unpaid > largestBroken {
					largestBroken = unpaid
					facts.BrokenPromiseCustomerID = row.CustomerID
					facts.BrokenPromiseCustomerName = strings.TrimSpace(row.CustomerName)
				}
			case collection.PromiseOpen:
				if promise.PromisedOn <= promiseBy {
					facts.PromisesDue++
					facts.PromisesDueMinor += unpaid
				}
			case collection.PromiseKept:
			}
		}

		promised := promise != nil && promise.Status == collection.PromiseOpen
		quiet := row.LastActivityAt == nil || *row.LastActivityAt < quietSince
		if row.OverdueMinor > 0 && !promised && quiet {
			facts.UncontactedAccounts++
			facts.UncontactedOverdueMinor += row.OverdueMinor
		}
	}
}

func toTenths(value float64) int64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return int64(math.Round(value * 10))
}
