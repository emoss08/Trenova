package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	accessPolicyKeyPrefix = "access-policies"
	accessPolicyCacheTTL  = 5 * time.Minute
)

type cachedAccessPolicies struct {
	Policies []*iam.AccessPolicy `json:"policies"`
}

type AccessPolicyCacheParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type accessPolicyCacheRepository struct {
	client *redis.Client
	l      *zap.Logger
}

func NewAccessPolicyCacheRepository(
	p AccessPolicyCacheParams,
) repositories.AccessPolicyCacheRepository {
	return &accessPolicyCacheRepository{
		client: p.Client,
		l:      p.Logger.Named("redis.access-policy-cache"),
	}
}

func accessPolicyCacheKey(req repositories.IAMTenantPolicyLookupRequest) string {
	return fmt.Sprintf(
		"%s:%s:%s",
		accessPolicyKeyPrefix,
		req.OrganizationID.String(),
		req.BusinessUnitID.String(),
	)
}

func (r *accessPolicyCacheRepository) GetEnabled(
	ctx context.Context,
	req repositories.IAMTenantPolicyLookupRequest,
) ([]*iam.AccessPolicy, bool, error) {
	cached := new(cachedAccessPolicies)
	if err := redishelpers.GetJSON(ctx, r.client, accessPolicyCacheKey(req), cached); err != nil {
		if redishelpers.IsRedisNil(err) {
			return nil, false, nil
		}
		r.l.Error("failed to read access policy cache", zap.Error(err))
		return nil, false, err
	}

	if cached.Policies == nil {
		cached.Policies = []*iam.AccessPolicy{}
	}
	return cached.Policies, true, nil
}

func (r *accessPolicyCacheRepository) SetEnabled(
	ctx context.Context,
	req repositories.IAMTenantPolicyLookupRequest,
	policies []*iam.AccessPolicy,
) error {
	if policies == nil {
		policies = []*iam.AccessPolicy{}
	}

	if err := redishelpers.SetJSON(
		ctx,
		r.client,
		accessPolicyCacheKey(req),
		&cachedAccessPolicies{Policies: policies},
		accessPolicyCacheTTL,
	); err != nil {
		r.l.Error("failed to write access policy cache", zap.Error(err))
		return err
	}

	return nil
}

func (r *accessPolicyCacheRepository) Invalidate(
	ctx context.Context,
	req repositories.IAMTenantPolicyLookupRequest,
) error {
	if err := r.client.Del(ctx, accessPolicyCacheKey(req)).Err(); err != nil {
		r.l.Error("failed to invalidate access policy cache", zap.Error(err))
		return err
	}

	return nil
}
