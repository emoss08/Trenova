package heavyslots

import (
	"context"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/tenantlimitsconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/platformmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/redis/slotstore"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/redis"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	keyPrefix          = "heavy:slots:"
	storeTimeout       = 2 * time.Second
	firstPoll          = 250 * time.Millisecond
	maxPoll            = 2 * time.Second
	activityWaitMargin = 10 * time.Second
)

type Store interface {
	Acquire(ctx context.Context, claim slotstore.Claim) (bool, error)
	Refresh(ctx context.Context, claim slotstore.Claim) (bool, error)
	Release(ctx context.Context, claim slotstore.Claim) error
}

type Params struct {
	fx.In

	Client       *redis.Client
	Managed      *managedconfig.Config
	TenantLimits *tenantlimitsconfig.Config
	Metrics      *metrics.Registry
	Logger       *zap.Logger
}

type Gate struct {
	store         Store
	local         *localSlots
	limits        *tenantlimitsconfig.Config
	organizations bool
	metrics       *platformmetrics.Tenancy
	l             *zap.Logger
}

type localSlots struct {
	mu   sync.Mutex
	held map[string]int
}

func (l *localSlots) acquire(bounds []slotstore.Bound) (func(), bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, bound := range bounds {
		if l.held[bound.Key] >= bound.Limit {
			return nil, false
		}
	}
	for _, bound := range bounds {
		l.held[bound.Key]++
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			l.mu.Lock()
			defer l.mu.Unlock()
			for _, bound := range bounds {
				l.held[bound.Key]--
				if l.held[bound.Key] <= 0 {
					delete(l.held, bound.Key)
				}
			}
		})
	}, true
}

var _ kernelports.HeavyJobSlots = (*Gate)(nil)

func New(p Params) kernelports.HeavyJobSlots {
	return NewGate(
		slotstore.New(p.Client),
		p.TenantLimits,
		p.TenantLimits.HeavyJobs.Organizations.On(p.Managed.SharedTenancy()),
		p.Metrics.Tenancy,
		p.Logger,
	)
}

func NewGate(
	store Store,
	limits *tenantlimitsconfig.Config,
	organizations bool,
	tenancy *platformmetrics.Tenancy,
	logger *zap.Logger,
) *Gate {
	return &Gate{
		store:         store,
		local:         &localSlots{held: map[string]int{}},
		limits:        limits,
		organizations: organizations,
		metrics:       tenancy,
		l:             logger.Named("heavyslots"),
	}
}

func (g *Gate) claim(ctx context.Context, job kernelports.HeavyJob) slotstore.Claim {
	slot := g.limits.SlotFor(tenantlimitsconfig.Kind(job))
	bounds := []slotstore.Bound{{Key: keyPrefix + string(job), Limit: slot.Total}}
	if g.organizations {
		if tenant, ok := dbscope.TenantFrom(ctx); ok && tenant.OrganizationID.IsNotNil() {
			bounds = append(bounds, slotstore.Bound{
				Key:   keyPrefix + string(job) + ":org:" + tenant.OrganizationID.String(),
				Limit: slot.PerOrganization,
			})
		}
	}
	return slotstore.Claim{
		Bounds: bounds,
		Token:  pulid.MustNew("slot_").String(),
		TTL:    slot.Hold,
	}
}

func (g *Gate) TryHold(ctx context.Context, job kernelports.HeavyJob) (func(), bool) {
	claim := g.claim(ctx, job)
	acquired, err := g.acquire(ctx, claim)
	if err != nil {
		release, ok := g.fallback(job, claim, err)
		if !ok {
			g.metrics.RecordSlotWait(string(job), g.scope(claim))
		}
		return release, ok
	}
	if !acquired {
		g.metrics.RecordSlotWait(string(job), g.scope(claim))
		return nil, false
	}
	return g.hold(ctx, claim), true
}

func (g *Gate) Hold(ctx context.Context, job kernelports.HeavyJob) (func(), error) {
	claim := g.claim(ctx, job)
	deadline := g.waitUntil(ctx, job)
	poll := backoff.NewExponentialBackOff()
	poll.InitialInterval = firstPoll
	poll.MaxInterval = maxPoll
	waited := false
	for {
		acquired, err := g.acquire(ctx, claim)
		if err != nil {
			if release, ok := g.fallback(job, claim, err); ok {
				return release, nil
			}
		} else if acquired {
			return g.hold(ctx, claim), nil
		}
		if !waited {
			waited = true
			g.metrics.RecordSlotWait(string(job), g.scope(claim))
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			g.metrics.RecordLimitRefusal("heavy_"+string(job), g.scope(claim))
			return nil, g.busy(job)
		}
		next := min(poll.NextBackOff(), remaining)
		if activity.IsActivity(ctx) {
			activity.RecordHeartbeat(ctx, "waiting for a "+string(job)+" slot")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(next):
		}
	}
}

func (g *Gate) waitUntil(ctx context.Context, job kernelports.HeavyJob) time.Time {
	slot := g.limits.SlotFor(tenantlimitsconfig.Kind(job))
	until := time.Now().Add(slot.Wait)
	if !activity.IsActivity(ctx) {
		return until
	}
	until = time.Now().Add(slot.Hold)
	if deadline, ok := ctx.Deadline(); ok && deadline.Add(-activityWaitMargin).Before(until) {
		until = deadline.Add(-activityWaitMargin)
	}
	return until
}

func (g *Gate) busy(job kernelports.HeavyJob) error {
	slot := g.limits.SlotFor(tenantlimitsconfig.Kind(job))
	retryAfter := min(slot.Hold, 30*time.Second)
	return errortypes.NewRateLimitError(
		"job",
		"The server is busy with other work of this kind. Try again in a moment.",
	).WithRetryAfter(retryAfter)
}

func (g *Gate) scope(claim slotstore.Claim) string {
	if len(claim.Bounds) > 1 {
		return platformmetrics.LimitScopeOrganization
	}
	return platformmetrics.LimitScopeInstance
}

func (g *Gate) acquire(ctx context.Context, claim slotstore.Claim) (bool, error) {
	storeCtx, cancel := context.WithTimeout(ctx, storeTimeout)
	defer cancel()
	return g.store.Acquire(storeCtx, claim)
}

func (g *Gate) fallback(
	job kernelports.HeavyJob,
	claim slotstore.Claim,
	err error,
) (func(), bool) {
	g.l.Warn("could not reach the shared heavy job slots; holding one of this process's own",
		zap.String("job", string(job)), zap.Error(err))
	return g.local.acquire(claim.Bounds)
}

func (g *Gate) hold(ctx context.Context, claim slotstore.Claim) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(claim.TTL / 3)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				refreshCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), storeTimeout)
				if _, err := g.store.Refresh(refreshCtx, claim); err != nil {
					g.l.Debug("could not refresh a heavy job slot", zap.Error(err))
				}
				cancel()
			}
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			close(stop)
			<-done
			releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), storeTimeout)
			defer cancel()
			if err := g.store.Release(releaseCtx, claim); err != nil {
				g.l.Warn("could not release a heavy job slot; it lapses on its own", zap.Error(err))
			}
		})
	}
}
