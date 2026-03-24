package repositories

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	organizationKeyPrefix = "organization"
)

type OrganizationRepositoryParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type organizationRepository struct {
	client *redis.Client
	l      *zap.Logger
}

func NewOrganizationRepository(
	p OrganizationRepositoryParams,
) repositories.OrganizationCacheRepository {
	return &organizationRepository{
		client: p.Client,
		l:      p.Logger,
	}
}

func (r *organizationRepository) GetByID(
	ctx context.Context,
	orgID pulid.ID,
) (*tenant.Organization, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("orgID", orgID.String()),
	)

	key := fmt.Sprintf("%s:%s", organizationKeyPrefix, orgID.String())

	org := new(tenant.Organization)
	if err := redishelpers.GetJSON(ctx, r.client, key, org); err != nil {
		log.Error("failed to get organization from redis", zap.Error(err))
		return nil, err
	}

	return org, nil
}
