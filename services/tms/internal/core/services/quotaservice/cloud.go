package quotaservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type CloudConfig struct {
	Plans         services.PlanService
	Counters      repositories.QuotaCounterRepository
	Subscriptions repositories.SubscriptionRepository
	DB            ports.DBConnection
	Logger        *zap.Logger
	Clock         func() time.Time
	InTransaction func(context.Context) bool
}

type CloudGuard struct {
	plans         services.PlanService
	counters      repositories.QuotaCounterRepository
	subscriptions repositories.SubscriptionRepository
	l             *zap.Logger
	now           func() time.Time
	inTransaction func(context.Context) bool
}

var _ services.QuotaGuard = (*CloudGuard)(nil)

type window struct {
	start int64
	end   int64
}

func NewCloud(cfg CloudConfig) *CloudGuard {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	inTransaction := cfg.InTransaction
	if inTransaction == nil {
		inTransaction = transactionDetector(cfg.DB)
	}

	return &CloudGuard{
		plans:         cfg.Plans,
		counters:      cfg.Counters,
		subscriptions: cfg.Subscriptions,
		l:             logger.Named("quota-guard"),
		now:           clock,
		inTransaction: inTransaction,
	}
}

func transactionDetector(db ports.DBConnection) func(context.Context) bool {
	return func(ctx context.Context) bool {
		if db == nil {
			return false
		}

		_, ok := db.DBForContext(ctx).(bun.Tx)
		return ok
	}
}

func (g *CloudGuard) Enforce(ctx context.Context, req *services.QuotaRequest) error {
	resolved, limit, limited, err := g.prepare(ctx, req)
	if err != nil || !limited || req.Quantity == 0 {
		return err
	}

	if limit.Window == platformplan.WindowPerItem {
		return perItemError(req, limit, resolved)
	}

	if !g.inTransaction(ctx) {
		return fmt.Errorf("%w: %s", ErrTransactionRequired, req.Meter)
	}

	if err = g.counters.Lock(ctx, req.TenantInfo, req.Meter); err != nil {
		return err
	}

	bounds, err := g.window(ctx, req.TenantInfo, limit.Window)
	if err != nil {
		return err
	}

	used, err := g.count(ctx, req.TenantInfo, req.Meter, bounds)
	if err != nil {
		return err
	}

	if used+req.Quantity > limit.Max {
		return errortypes.NewQuotaExceededError(
			string(req.Meter),
			limit.Max,
			used,
			resolved.Key().String(),
		)
	}

	if used+req.Quantity >= limit.Max {
		return g.endTrialOnExhaustion(ctx, req, resolved)
	}

	return nil
}

func (g *CloudGuard) endTrialOnExhaustion(
	ctx context.Context,
	req *services.QuotaRequest,
	resolved *platformplan.ResolvedPlan,
) error {
	if g.subscriptions == nil || !resolved.IsManaged() || !resolved.Plan.EndsTrial(req.Meter) {
		return nil
	}

	now := g.now().Unix()
	if resolved.Subscription.EffectiveStatus(now) != subscription.StatusTrialing {
		return nil
	}

	ended, err := g.subscriptions.EndTrial(ctx, &repositories.EndSubscriptionTrialRequest{
		TenantInfo: req.TenantInfo,
		ID:         resolved.Subscription.ID,
		EndedAt:    now,
	})
	if err != nil {
		return fmt.Errorf("end the trial when %s was used up: %w", req.Meter, err)
	}
	if !ended {
		return nil
	}

	g.l.Info("trial ended early because a trial-ending limit was used up",
		zap.String("organizationId", req.TenantInfo.OrgID.String()),
		zap.String("meter", string(req.Meter)),
	)
	orgID := req.TenantInfo.OrgID
	ports.AfterCommit(ctx, func(context.Context) {
		g.plans.Invalidate(orgID)
	})

	return nil
}

func (g *CloudGuard) Check(
	ctx context.Context,
	req *services.QuotaRequest,
) (*services.QuotaDecision, error) {
	resolved, limit, limited, err := g.prepare(ctx, req)
	if err != nil {
		return nil, err
	}

	decision := &services.QuotaDecision{
		Meter:     req.Meter,
		Plan:      resolved.Key(),
		Requested: req.Quantity,
		Allowed:   true,
		Unlimited: !limited,
	}
	if !limited {
		return decision, nil
	}

	decision.Window = limit.Window
	decision.Limit = limit.Max

	if limit.Window == platformplan.WindowPerItem {
		decision.Allowed = req.Quantity <= limit.Max
		decision.Used = req.Quantity
		decision.Remaining = max(limit.Max-req.Quantity, 0)
		return decision, nil
	}

	bounds, err := g.window(ctx, req.TenantInfo, limit.Window)
	if err != nil {
		return nil, err
	}

	used, err := g.count(ctx, req.TenantInfo, req.Meter, bounds)
	if err != nil {
		return nil, err
	}

	decision.Used = used
	decision.Remaining = max(limit.Max-used, 0)
	decision.Allowed = used+req.Quantity <= limit.Max
	decision.WindowStart = bounds.start
	decision.WindowEnd = bounds.end

	return decision, nil
}

func (g *CloudGuard) Usage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*services.QuotaUsageSummary, error) {
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return nil, ErrTenantRequired
	}

	resolved, err := g.plans.Resolve(ctx, tenantInfo.OrgID, tenantInfo.BuID)
	if err != nil {
		return nil, err
	}

	summary := newSummary(resolved, g.now().Unix())
	meters := resolved.Plan.MeterKeys()
	summary.Meters = make([]services.QuotaMeterUsage, 0, len(meters))

	var monthly *window
	for _, meter := range meters {
		limit, _ := resolved.Limit(meter)
		usage := services.QuotaMeterUsage{
			Meter:     meter,
			Window:    limit.Window,
			Limit:     limit.Max,
			Remaining: limit.Max,
		}

		if limit.Window != platformplan.WindowPerItem {
			bounds := window{}
			if limit.Window == platformplan.WindowMonthly {
				if monthly == nil {
					computed, windowErr := g.window(ctx, tenantInfo, platformplan.WindowMonthly)
					if windowErr != nil {
						return nil, windowErr
					}
					monthly = &computed
				}
				bounds = *monthly
			}

			used, countErr := g.count(ctx, tenantInfo, meter, bounds)
			if countErr != nil {
				return nil, countErr
			}

			usage.Used = used
			usage.Remaining = max(limit.Max-used, 0)
			usage.WindowStart = bounds.start
			usage.WindowEnd = bounds.end
		}

		summary.Meters = append(summary.Meters, usage)
	}

	return summary, nil
}

func (g *CloudGuard) prepare(
	ctx context.Context,
	req *services.QuotaRequest,
) (resolved *platformplan.ResolvedPlan, limit platformplan.Limit, limited bool, err error) {
	if req == nil {
		return nil, platformplan.Limit{}, false, ErrRequestRequired
	}
	if req.Quantity < 0 {
		return nil, platformplan.Limit{}, false, fmt.Errorf(
			"%w: %s requested %d",
			ErrInvalidQuantity,
			req.Meter,
			req.Quantity,
		)
	}
	if req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
		return nil, platformplan.Limit{}, false, ErrTenantRequired
	}

	resolved, err = g.plans.Resolve(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID)
	if err != nil {
		return nil, platformplan.Limit{}, false, err
	}

	if resolved.IsManaged() && !resolved.AllowsWrites() {
		reason := platformplan.ReasonSubscriptionReadOnly
		if !resolved.AllowsLogin() {
			reason = platformplan.ReasonSubscriptionExpired
		}
		return nil, platformplan.Limit{}, false, errortypes.NewPlanRestrictionError(
			"",
			reason,
			resolved.Key().String(),
		)
	}

	limit, limited = resolved.Limit(req.Meter)
	if !limited {
		return resolved, limit, false, nil
	}

	if limit.Window != platformplan.WindowPerItem && !g.counters.Supports(req.Meter) {
		g.l.Error("plan limits a meter that has no quota counter",
			zap.String("meter", string(req.Meter)),
			zap.String("plan", resolved.Key().String()),
		)
		return nil, platformplan.Limit{}, false, fmt.Errorf("%w: %s", ErrCounterMissing, req.Meter)
	}

	return resolved, limit, true, nil
}

func (g *CloudGuard) window(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	kind platformplan.Window,
) (window, error) {
	if kind != platformplan.WindowMonthly {
		return window{}, nil
	}

	timezone, err := g.counters.OrganizationTimezone(ctx, tenantInfo)
	if err != nil {
		return window{}, fmt.Errorf("read the organization's timezone: %w", err)
	}

	loc := timeutils.LoadLocation(timezone)
	now := g.now().Unix()

	return window{
		start: timeutils.MonthStart(now, loc),
		end:   timeutils.NextMonthStart(now, loc),
	}, nil
}

func (g *CloudGuard) count(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	meter platformcatalog.MeterKey,
	bounds window,
) (int64, error) {
	return g.counters.Count(ctx, &repositories.QuotaCountRequest{
		TenantInfo:  tenantInfo,
		Meter:       meter,
		WindowStart: bounds.start,
		WindowEnd:   bounds.end,
	})
}

func perItemError(
	req *services.QuotaRequest,
	limit platformplan.Limit,
	resolved *platformplan.ResolvedPlan,
) error {
	if req.Quantity <= limit.Max {
		return nil
	}

	return errortypes.NewQuotaExceededError(
		string(req.Meter),
		limit.Max,
		req.Quantity,
		resolved.Key().String(),
	)
}

func newSummary(resolved *platformplan.ResolvedPlan, checkedAt int64) *services.QuotaUsageSummary {
	summary := &services.QuotaUsageSummary{
		OrganizationID:         resolved.OrganizationID,
		BusinessUnitID:         resolved.BusinessUnitID,
		Plan:                   resolved.Key(),
		Origin:                 resolved.Origin,
		Unlimited:              resolved.Plan.IsUnlimited(),
		RestrictedCapabilities: []platformplan.Capability{},
		CheckedAt:              checkedAt,
	}

	if resolved.Plan != nil {
		summary.PlanName = resolved.Plan.Name
		summary.RestrictedCapabilities = append(
			summary.RestrictedCapabilities,
			resolved.Plan.RestrictedCapabilities...,
		)
	}

	if resolved.IsManaged() {
		summary.SubscriptionID = resolved.Subscription.ID
		summary.Status = resolved.Status
		summary.TrialEndsAt = resolved.Subscription.TrialEndsAt
		summary.ReadOnlyUntil = resolved.Subscription.ReadOnlyUntil
		summary.SubscribedAt = resolved.Subscription.CreatedAt
	}

	return summary
}
