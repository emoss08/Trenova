package repositories

import (
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const (
	defaultOrganizationTTL = 24 * time.Hour // * 24 hours
	orgKeyPrefix           = "org:"
	orgListKeyPrefix       = "orgs:"
)

type OrganizationRepositoryParams struct {
	fx.In

	Cache  *redis.Client
	Logger *logger.Logger
}

type organizationRepository struct {
	cache    *redis.Client
	l        *zerolog.Logger
	cacheTTL time.Duration
}

func NewOrganizationRepository(p OrganizationRepositoryParams) repositories.OrganizationRepository {
	log := p.Logger.With().
		Str("repository", "organization").
		Str("component", "redis").
		Logger()

	return &organizationRepository{
		cache:    p.Cache,
		l:        &log,
		cacheTTL: defaultOrganizationTTL,
	}
}
