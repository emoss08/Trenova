package carrierintelservice

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelcontrol"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelusage"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const spendCacheTTL = 60 * time.Second

var ErrSpendCapReached = errors.New("carrier intelligence spend cap reached")

func decimalFromInt(v int) decimal.Decimal { return decimal.NewFromInt(int64(v)) }

func decimalZero() decimal.Decimal { return decimal.Zero }

type spendEntry struct {
	month   string
	cost    decimal.Decimal
	fetched time.Time
}

type spendCache struct {
	mu      sync.Mutex
	entries map[string]spendEntry
}

var spend = &spendCache{entries: make(map[string]spendEntry)}

func tenantKey(tenantInfo pagination.TenantInfo) string {
	return tenantInfo.OrgID.String() + ":" + tenantInfo.BuID.String()
}

func invalidateSpendCache(tenantInfo pagination.TenantInfo) {
	spend.mu.Lock()
	delete(spend.entries, tenantKey(tenantInfo))
	spend.mu.Unlock()
}

func (s *Service) monthToDateSpend(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (decimal.Decimal, error) {
	now := s.now()
	month := timeutils.MonthKeyUTC(now)
	key := tenantKey(tenantInfo)

	spend.mu.Lock()
	entry, ok := spend.entries[key]
	spend.mu.Unlock()
	if ok && entry.month == month && time.Since(entry.fetched) < spendCacheTTL {
		return entry.cost, nil
	}

	cost, err := s.usageRepo.CostSince(ctx, tenantInfo, timeutils.MonthStartUTC(now))
	if err != nil {
		return decimal.Zero, err
	}

	spend.mu.Lock()
	spend.entries[key] = spendEntry{month: month, cost: cost, fetched: time.Now()}
	spend.mu.Unlock()
	return cost, nil
}

type BudgetCheck struct {
	Tenant    pagination.TenantInfo
	Control   *carrierintelcontrol.CarrierIntelControl
	Bound     *BoundProvider
	Endpoint  carrierintelusage.Endpoint
	DOTNumber string
	Units     int
}

func (s *Service) marginalCost(ctx context.Context, check *BudgetCheck) (decimal.Decimal, error) {
	price := check.Bound.prices.Price(check.Endpoint)
	if price.Model == carrierintelusage.BillingModelFree || !price.UnitCost.IsPositive() {
		return decimal.Zero, nil
	}
	units := max(check.Units, 1)
	if price.Model == carrierintelusage.BillingModelPerDOTMonth {
		billed, err := s.alreadyBilledThisMonth(ctx, check, price)
		if err != nil {
			return decimal.Zero, err
		}
		if billed {
			return decimal.Zero, nil
		}
	}
	return price.UnitCost.Mul(decimalFromInt(units)), nil
}

func (s *Service) alreadyBilledThisMonth(
	ctx context.Context,
	check *BudgetCheck,
	price carrierintelusage.EndpointPrice,
) (bool, error) {
	dedupe := carrierintelusage.BillingDedupeKey(check.Endpoint, price, check.DOTNumber, s.now())
	if dedupe == "" {
		return false, nil
	}
	return s.usageRepo.ExistsDedupeKey(ctx, check.Tenant, check.Bound.provider, dedupe)
}

func (s *Service) GuardBudget(ctx context.Context, check *BudgetCheck) error {
	control := check.Control
	if control.MonthlySpendCap == nil && control.DailyFullProfileCap == nil {
		return nil
	}

	marginal, err := s.marginalCost(ctx, check)
	if err != nil {
		return err
	}
	if !marginal.IsPositive() {
		return nil
	}

	if control.DailyFullProfileCap != nil &&
		check.Endpoint == carrierintelusage.EndpointProfileFull {
		count, countErr := s.usageRepo.CountBillableSince(
			ctx,
			&carrierintelports.CountBillableUsageRequest{
				TenantInfo: check.Tenant,
				Provider:   check.Bound.provider,
				Endpoint:   check.Endpoint,
				Since:      timeutils.DayStartUTC(s.now()),
			},
		)
		if countErr != nil {
			return countErr
		}
		if count >= *control.DailyFullProfileCap {
			return fmt.Errorf("%w: %w", ErrSpendCapReached, errortypes.NewBusinessError(
				"The daily limit of {0} full carrier profiles has been reached. Lighter lookups remain available",
				*control.DailyFullProfileCap,
			))
		}
	}

	if control.MonthlySpendCap == nil {
		return nil
	}

	mtd, err := s.monthToDateSpend(ctx, check.Tenant)
	if err != nil {
		return err
	}
	projected := mtd.Add(marginal)
	limit := *control.MonthlySpendCap

	if projected.GreaterThan(limit) {
		s.notifySpend(ctx, check.Tenant, mtd, limit, true)
		return fmt.Errorf("%w: %w", ErrSpendCapReached, errortypes.NewBusinessError(
			"The monthly carrier intelligence spend cap of ${0} has been reached",
			limit.StringFixed(2),
		))
	}

	softLimit := limit.Mul(decimal.NewFromInt(int64(control.SoftCapPercent))).
		Div(decimal.NewFromInt(100))
	if projected.GreaterThanOrEqual(softLimit) {
		s.notifySpend(ctx, check.Tenant, projected, limit, false)
	}
	return nil
}

func (s *Service) notifySpend(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	spent, limit decimal.Decimal,
	hard bool,
) {
	eventType := eventSpendSoftCap
	priority := notification.PriorityHigh
	title := "Carrier intelligence spend approaching cap"
	message := fmt.Sprintf(
		"Estimated carrier intelligence spend this month is $%s of the $%s cap.",
		spent.StringFixed(2), limit.StringFixed(2),
	)
	if hard {
		eventType = eventSpendHardCap
		priority = notification.PriorityCritical
		title = "Carrier intelligence spend cap reached"
		message = fmt.Sprintf(
			"The $%s monthly cap has been reached. Background monitoring is paused until next month or until the cap is raised.",
			limit.StringFixed(2),
		)
	}

	s.SendNotification(ctx, &NotificationRequest{
		Tenant:      tenantInfo,
		EventType:   eventType,
		Correlation: fmt.Sprintf("%s-%s", eventType, timeutils.MonthKeyUTC(s.now())),
		Priority:    priority,
		Title:       title,
		Message:     message,
		Link:        MonitoringLink + "?tab=usage",
	})
}

func IsSpendCapError(err error) bool {
	return errors.Is(err, ErrSpendCapReached)
}

func logBudgetDenied(l *zap.Logger, check *BudgetCheck, err error) {
	l.Info("carrier intelligence call denied by budget",
		zap.String("provider", check.Bound.provider.String()),
		zap.String("endpoint", check.Endpoint.String()),
		zap.Error(err),
	)
}
