package agentruntime

import (
	"context"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/ttlcache"
	"go.uber.org/zap"
)

const (
	matchPoolTTL     = 30 * time.Second
	matchPoolEntries = 4096
	matchEmbedRate   = 30
	matchEmbedBurst  = 10
	matchEmbedPeriod = time.Minute
)

type matchPool struct {
	agents    []*agentdefinition.Definition
	held      []map[string]struct{}
	permitted []string
	allowed   map[string]struct{}
}

type matchPoolKey struct {
	organizationID pulid.ID
	businessUnitID pulid.ID
	userID         pulid.ID
}

type matchPools interface {
	get(key matchPoolKey) (*matchPool, bool)
	put(key matchPoolKey, pool *matchPool)
}

type cachedMatchPools struct {
	cache *ttlcache.Cache[matchPoolKey, *matchPool]
}

func newMatchPools() matchPools {
	return cachedMatchPools{cache: ttlcache.New[matchPoolKey, *matchPool](ttlcache.Options{
		TTL:        matchPoolTTL,
		MaxEntries: matchPoolEntries,
	})}
}

func (p cachedMatchPools) get(key matchPoolKey) (*matchPool, bool) {
	return p.cache.Get(key)
}

func (p cachedMatchPools) put(key matchPoolKey, pool *matchPool) {
	p.cache.Set(key, pool)
}

func (s *Service) matchPool(
	ctx context.Context,
	actor *permissionports.RequestActor,
) (*matchPool, error) {
	key := matchPoolKey{
		organizationID: actor.OrganizationID,
		businessUnitID: actor.BusinessUnitID,
		userID:         actor.UserID,
	}
	if s.pools != nil {
		if pool, ok := s.pools.get(key); ok {
			return pool, nil
		}
	}

	agents, err := OtherUsableAgents(ctx, &OtherUsableAgentsRequest{
		Permissions: s.permissions,
		Definitions: s.definitions,
		Actor:       actor,
	})
	if err != nil {
		return nil, err
	}

	pool := &matchPool{agents: agents}
	if len(agents) > 0 {
		var union []string
		pool.held, union = s.recommendableTools(agents)
		pool.permitted = s.permittedWith(ctx, actor, union, nil)
	}
	pool.allowed = make(map[string]struct{}, len(pool.permitted))
	for _, name := range pool.permitted {
		pool.allowed[name] = struct{}{}
	}
	if s.pools != nil && ctx.Err() == nil {
		s.pools.put(key, pool)
	}

	return pool, nil
}

func (s *Service) recommendableTools(
	agents []*agentdefinition.Definition,
) (held []map[string]struct{}, union []string) {
	held = make([]map[string]struct{}, len(agents))
	seen := make(map[string]struct{}, agentdefinition.MaxTools)
	union = make([]string, 0, agentdefinition.MaxTools)
	for idx, definition := range agents {
		names := agentdefinition.WithoutCoreTools(s.heldTools(definition))
		held[idx] = make(map[string]struct{}, len(names))
		for _, name := range names {
			held[idx][name] = struct{}{}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			union = append(union, name)
		}
	}

	return held, union
}

func matchableTools(catalog *agenttoolcatalog.Catalog) func() []string {
	return sync.OnceValue(func() []string {
		if catalog == nil {
			return nil
		}

		return agentdefinition.WithoutCoreTools(catalog.Names())
	})
}

func (p *matchPool) permitsWithin(matches []agenttoolcatalog.Match, limit int) bool {
	if len(matches) == 0 || limit <= 0 {
		return false
	}
	floor := matches[min(limit, len(matches))-1].Score
	for _, match := range matches {
		if match.Score < floor {
			return false
		}
		if _, ok := p.allowed[match.Name]; ok {
			return true
		}
	}

	return false
}

func (p *matchPool) delegateHolding(
	currentID pulid.ID,
	ranked []agenttoolcatalog.Match,
) (*agentdefinition.Definition, []string) {
	if currentID.IsNil() || len(ranked) == 0 {
		return nil, nil
	}
	var current *agentdefinition.Definition
	for _, definition := range p.agents {
		if definition.ID == currentID {
			current = definition
			break
		}
	}
	if current == nil || !current.Delegates() {
		return nil, nil
	}

	top := topTier(ranked)
	for idx, delegate := range p.agents {
		if !current.MayDelegateTo(delegate.ID) || current.DelegateRefusal(delegate) != "" {
			continue
		}
		matched := make([]string, 0, len(top))
		for _, name := range top {
			if _, ok := p.held[idx][name]; ok {
				matched = append(matched, name)
			}
		}
		if len(matched) > 0 {
			return current, matched
		}
	}

	return nil, nil
}

func (s *Service) mayEmbedMatch(ctx context.Context, actor *permissionports.RequestActor) bool {
	if s.rateLimits == nil {
		return true
	}

	decisions, err := s.rateLimits.Check(ctx, []kernelports.RateLimitRequest{{
		Key: "{" + actor.OrganizationID.String() + "}:" + agentports.AgentMatchRateScope + ":" +
			actor.UserID.String(),
		Policy: kernelports.RateLimitPolicy{
			Rate:   matchEmbedRate,
			Period: matchEmbedPeriod,
			Burst:  matchEmbedBurst,
		},
		Cost: 1,
	}})
	if err != nil {
		s.logger.Warn("could not check how often this person's questions were matched; embedding anyway",
			zap.Error(err),
		)

		return true
	}

	return len(decisions) == 0 || decisions[0].Allowed
}
