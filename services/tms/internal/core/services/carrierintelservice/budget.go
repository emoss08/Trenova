package carrierintelservice

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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

type budgetCheck struct {
	tenant    pagination.TenantInfo
	control   *carrierintel.CarrierIntelControl
	bound     *boundProvider
	endpoint  carrierintel.Endpoint
	dotNumber string
	units     int
}

func (s *Service) marginalCost(ctx context.Context, check *budgetCheck) (decimal.Decimal, error) {
	price := check.bound.prices.Price(check.endpoint)
	if price.Model == carrierintel.BillingModelFree || !price.UnitCost.IsPositive() {
		return decimal.Zero, nil
	}
	units := max(check.units, 1)
	if price.Model == carrierintel.BillingModelPerDOTMonth {
		dedupe := carrierintel.BillingDedupeKey(check.endpoint, price, check.dotNumber, s.now())
		if dedupe != "" {
			exists, err := s.usageRepo.ExistsDedupeKey(
				ctx,
				check.tenant,
				check.bound.provider,
				dedupe,
			)
			if err != nil {
				return decimal.Zero, err
			}
			if exists {
				return decimal.Zero, nil
			}
		}
	}
	return price.UnitCost.Mul(decimalFromInt(units)), nil
}

func (s *Service) guardBudget(ctx context.Context, check *budgetCheck) error {
	control := check.control
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

	if control.DailyFullProfileCap != nil && check.endpoint == carrierintel.EndpointProfileFull {
		count, countErr := s.usageRepo.CountBillableSince(
			ctx,
			&repositories.CountBillableUsageRequest{
				TenantInfo: check.tenant,
				Provider:   check.bound.provider,
				Endpoint:   check.endpoint,
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

	mtd, err := s.monthToDateSpend(ctx, check.tenant)
	if err != nil {
		return err
	}
	projected := mtd.Add(marginal)
	limit := *control.MonthlySpendCap

	if projected.GreaterThan(limit) {
		s.notifySpend(ctx, check.tenant, mtd, limit, true)
		return fmt.Errorf("%w: %w", ErrSpendCapReached, errortypes.NewBusinessError(
			"The monthly carrier intelligence spend cap of ${0} has been reached",
			limit.StringFixed(2),
		))
	}

	softLimit := limit.Mul(decimal.NewFromInt(int64(control.SoftCapPercent))).
		Div(decimal.NewFromInt(100))
	if projected.GreaterThanOrEqual(softLimit) {
		s.notifySpend(ctx, check.tenant, projected, limit, false)
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

	s.sendNotification(ctx, &notificationRequest{
		tenant:      tenantInfo,
		eventType:   eventType,
		correlation: fmt.Sprintf("%s-%s", eventType, timeutils.MonthKeyUTC(s.now())),
		priority:    priority,
		title:       title,
		message:     message,
		link:        monitoringLink + "?tab=usage",
	})
}

func isSpendCapError(err error) bool {
	return errors.Is(err, ErrSpendCapReached)
}

func logBudgetDenied(l *zap.Logger, check *budgetCheck, err error) {
	l.Info("carrier intelligence call denied by budget",
		zap.String("provider", check.bound.provider.String()),
		zap.String("endpoint", check.endpoint.String()),
		zap.Error(err),
	)
}
