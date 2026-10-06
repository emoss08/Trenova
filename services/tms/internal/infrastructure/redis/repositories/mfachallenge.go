package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	corerepositories "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

const (
	mfaChallengePrefix         = "mfa_challenge"
	mfaChallengeFailuresPrefix = "mfa_challenge_failures"
)

type MFAChallengeRepositoryParams struct {
	fx.In

	Client *redis.Client
}

type mfaChallengeRepository struct {
	client *redis.Client
}

func NewMFAChallengeRepository(
	p MFAChallengeRepositoryParams,
) corerepositories.MFAChallengeRepository {
	return &mfaChallengeRepository{client: p.Client}
}

func (r *mfaChallengeRepository) Save(
	ctx context.Context,
	tokenHash string,
	challenge *corerepositories.MFAChallenge,
	ttl time.Duration,
) error {
	return redishelpers.SetJSON(ctx, r.client, challengeKey(tokenHash), challenge, ttl)
}

func (r *mfaChallengeRepository) Get(
	ctx context.Context,
	tokenHash string,
) (*corerepositories.MFAChallenge, error) {
	entity := new(corerepositories.MFAChallenge)
	if err := redishelpers.GetJSON(ctx, r.client, challengeKey(tokenHash), entity); err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errortypes.NewNotFoundError("Sign-in challenge not found")
		}
		return nil, err
	}

	return entity, nil
}

func (r *mfaChallengeRepository) Delete(ctx context.Context, tokenHash string) error {
	return r.client.Del(ctx, challengeKey(tokenHash), failuresKey(tokenHash)).Err()
}

func (r *mfaChallengeRepository) RecordFailure(
	ctx context.Context,
	tokenHash string,
	ttl time.Duration,
) (int64, error) {
	key := failuresKey(tokenHash)
	pipe := r.client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("record sign-in challenge failure: %w", err)
	}

	return incr.Val(), nil
}

func challengeKey(tokenHash string) string {
	return mfaChallengePrefix + ":" + tokenHash
}

func failuresKey(tokenHash string) string {
	return mfaChallengeFailuresPrefix + ":" + tokenHash
}
