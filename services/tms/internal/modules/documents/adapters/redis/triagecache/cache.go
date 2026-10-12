package triagecache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

const (
	prefix        = "triage"
	epochPart     = "epoch"
	rulesPart     = "rules"
	hintsPart     = "hints"
	EpochTTL      = 24 * time.Hour
	RulesTTL      = 10 * time.Minute
	SenderHintTTL = time.Hour
)

type Params struct {
	fx.In

	Client *redis.Client
}

type cache struct {
	client *redis.Client
}

func New(p Params) triageports.TriageCache {
	return &cache{client: p.Client}
}

func tenantParts(tenantInfo pagination.TenantInfo) []string {
	return []string{tenantInfo.OrgID.String(), tenantInfo.BuID.String()}
}

func EpochKey(tenantInfo pagination.TenantInfo) string {
	return strings.Join(append([]string{prefix, epochPart}, tenantParts(tenantInfo)...), ":")
}

func SectionKey(key *triageports.CacheKey) string {
	parts := make([]string, 0, 6)
	parts = append(parts, prefix, string(key.Section))
	parts = append(parts, tenantParts(key.TenantInfo)...)
	parts = append(parts, strconv.FormatInt(key.Epoch, 10))
	if key.Variant != "" {
		parts = append(parts, key.Variant)
	}

	return strings.Join(parts, ":")
}

func RulesKey(tenantInfo pagination.TenantInfo) string {
	return strings.Join(append([]string{prefix, rulesPart}, tenantParts(tenantInfo)...), ":")
}

func HintsKey(tenantInfo pagination.TenantInfo, senderDomain string) string {
	parts := append([]string{prefix, hintsPart}, tenantParts(tenantInfo)...)

	return strings.Join(append(parts, strings.ToLower(senderDomain)), ":")
}

func (c *cache) QueueEpoch(ctx context.Context, tenantInfo pagination.TenantInfo) (int64, error) {
	value, err := c.client.Get(ctx, EpochKey(tenantInfo)).Int64()
	if err != nil {
		if redishelpers.IsRedisNil(err) {
			return 0, nil
		}

		return 0, fmt.Errorf("read the triage epoch: %w", err)
	}

	return value, nil
}

func (c *cache) BumpQueueEpoch(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	key := EpochKey(tenantInfo)
	pipe := c.client.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, EpochTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("bump the triage epoch: %w", err)
	}

	return nil
}

func (c *cache) Get(ctx context.Context, key *triageports.CacheKey, dest any) (bool, error) {
	return c.read(ctx, SectionKey(key), dest)
}

func (c *cache) Set(
	ctx context.Context,
	key *triageports.CacheKey,
	value any,
	ttl time.Duration,
) error {
	if err := redishelpers.SetStringJSON(ctx, c.client, SectionKey(key), value, ttl); err != nil {
		return fmt.Errorf("write the triage %s: %w", key.Section, err)
	}

	return nil
}

func (c *cache) EnabledRules(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*triage.Rule, bool, error) {
	rules := make([]*triage.Rule, 0)
	hit, err := c.read(ctx, RulesKey(tenantInfo), &rules)
	if err != nil || !hit {
		return nil, false, err
	}

	return rules, true, nil
}

func (c *cache) SetEnabledRules(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rules []*triage.Rule,
) error {
	if err := redishelpers.SetStringJSON(
		ctx,
		c.client,
		RulesKey(tenantInfo),
		rules,
		RulesTTL,
	); err != nil {
		return fmt.Errorf("write the enabled triage rules: %w", err)
	}

	return nil
}

func (c *cache) DropEnabledRules(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	if err := c.client.Del(ctx, RulesKey(tenantInfo)).Err(); err != nil {
		return fmt.Errorf("drop the enabled triage rules: %w", err)
	}

	return nil
}

func (c *cache) SenderHints(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	senderDomain string,
) ([]*triage.SenderHint, bool, error) {
	hints := make([]*triage.SenderHint, 0)
	hit, err := c.read(ctx, HintsKey(tenantInfo, senderDomain), &hints)
	if err != nil || !hit {
		return nil, false, err
	}

	return hints, true, nil
}

func (c *cache) SetSenderHints(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	senderDomain string,
	hints []*triage.SenderHint,
) error {
	if err := redishelpers.SetStringJSON(
		ctx, c.client, HintsKey(tenantInfo, senderDomain), hints, SenderHintTTL,
	); err != nil {
		return fmt.Errorf("write the sender hints: %w", err)
	}

	return nil
}

func (c *cache) DropSenderHints(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	senderDomain string,
) error {
	if err := c.client.Del(ctx, HintsKey(tenantInfo, senderDomain)).Err(); err != nil {
		return fmt.Errorf("drop the sender hints: %w", err)
	}

	return nil
}

func (c *cache) read(ctx context.Context, key string, dest any) (bool, error) {
	err := redishelpers.GetStringJSON(ctx, c.client, key, dest)
	if err == nil {
		return true, nil
	}
	if redishelpers.IsRedisNil(err) || errors.Is(err, redis.Nil) {
		return false, nil
	}

	return false, fmt.Errorf("read %s: %w", key, err)
}
