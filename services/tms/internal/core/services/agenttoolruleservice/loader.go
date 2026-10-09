package agenttoolruleservice

import (
	"context"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

const (
	loaderTTL        = 30 * time.Second
	maxCachedTenants = 4096
)

type LoaderParams struct {
	fx.In

	Repo repositories.AgentToolRuleOverrideRepository
}

type cachedRules struct {
	rules     map[string]*agent.ToolRuleOverride
	fetchedAt time.Time
}

type Loader struct {
	repo  repositories.AgentToolRuleOverrideRepository
	clock func() time.Time
	ttl   time.Duration

	mu     sync.Mutex
	cached map[pagination.TenantInfo]cachedRules
}

var _ services.ToolRuleOverrides = (*Loader)(nil)

func NewLoader(p LoaderParams) *Loader {
	return newLoader(p.Repo, time.Now)
}

func newLoader(repo repositories.AgentToolRuleOverrideRepository, clock func() time.Time) *Loader {
	return &Loader{
		repo:   repo,
		clock:  clock,
		ttl:    loaderTTL,
		cached: make(map[pagination.TenantInfo]cachedRules),
	}
}

func (l *Loader) For(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (map[string]*agent.ToolRuleOverride, error) {
	key := pagination.TenantInfo{OrgID: tenantInfo.OrgID, BuID: tenantInfo.BuID}
	now := l.clock()

	l.mu.Lock()
	cached, ok := l.cached[key]
	l.mu.Unlock()
	if ok && now.Sub(cached.fetchedAt) < l.ttl {
		return cached.rules, nil
	}

	overrides, err := l.repo.List(ctx, key)
	if err != nil {
		return nil, err
	}
	rules := make(map[string]*agent.ToolRuleOverride, len(overrides))
	for _, override := range overrides {
		rules[override.ToolName] = override
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.cached) >= maxCachedTenants {
		clear(l.cached)
	}
	l.cached[key] = cachedRules{rules: rules, fetchedAt: now}

	return rules, nil
}

func (l *Loader) Forget(tenantInfo pagination.TenantInfo) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.cached, pagination.TenantInfo{OrgID: tenantInfo.OrgID, BuID: tenantInfo.BuID})
}
