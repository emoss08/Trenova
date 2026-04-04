package repositories

import (
	"context"
	"fmt"
	"time"

	corerepositories "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const ssoLoginStatePrefix = "sso_login_state"

type SSOLoginStateRepositoryParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type ssoLoginStateRepository struct {
	client *redis.Client
	l      *zap.Logger
}

func NewSSOLoginStateRepository(
	p SSOLoginStateRepositoryParams,
) corerepositories.SSOLoginStateRepository {
	return &ssoLoginStateRepository{
		client: p.Client,
		l:      p.Logger.Named("redis.sso-login-state-repository"),
	}
}

func (r *ssoLoginStateRepository) Save(
	ctx context.Context,
	state *corerepositories.SSOLoginState,
	ttl time.Duration,
) error {
	return redishelpers.SetJSON(ctx, r.client, r.getKey(state.State), state, ttl)
}

func (r *ssoLoginStateRepository) Get(
	ctx context.Context,
	state string,
) (*corerepositories.SSOLoginState, error) {
	entity := new(corerepositories.SSOLoginState)
	if err := redishelpers.GetJSON(ctx, r.client, r.getKey(state), entity); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *ssoLoginStateRepository) Delete(ctx context.Context, state string) error {
	return r.client.Del(ctx, r.getKey(state)).Err()
}

func (r *ssoLoginStateRepository) getKey(state string) string {
	return fmt.Sprintf("%s:%s", ssoLoginStatePrefix, state)
}
