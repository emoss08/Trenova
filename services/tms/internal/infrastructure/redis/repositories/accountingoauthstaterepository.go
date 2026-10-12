package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const accountingOAuthStatePrefix = "accounting_oauth_state:"

type AccountingOAuthStateRepositoryParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type accountingOAuthStateRepository struct {
	client *redis.Client
	l      *zap.Logger
}

func NewAccountingOAuthStateRepository(
	p AccountingOAuthStateRepositoryParams,
) accountingsyncports.AccountingOAuthStateRepository {
	return &accountingOAuthStateRepository{
		client: p.Client,
		l:      p.Logger.Named("redis.accounting-oauth-state-repository"),
	}
}

func (r *accountingOAuthStateRepository) Save(
	ctx context.Context,
	state *accountingsyncports.AccountingOAuthState,
	ttl time.Duration,
) error {
	return redishelpers.SetStringJSON(
		ctx,
		r.client,
		accountingOAuthStatePrefix+state.State,
		state,
		ttl,
	)
}

func (r *accountingOAuthStateRepository) Take(
	ctx context.Context,
	state string,
) (*accountingsyncports.AccountingOAuthState, error) {
	entity := new(accountingsyncports.AccountingOAuthState)
	err := redishelpers.TakeStringJSON(ctx, r.client, accountingOAuthStatePrefix+state, entity)
	if errors.Is(err, redis.Nil) {
		return nil, errortypes.NewNotFoundError(
			"The connection request has expired or was already used",
		)
	}
	if err != nil {
		return nil, err
	}

	return entity, nil
}
