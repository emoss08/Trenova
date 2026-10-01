package repositories

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const idempotencyPrefix = "idempotency:v1:"

var idempotencyClaimScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then
  redis.call('HSET', KEYS[1], 'state', 'processing', 'fp', ARGV[1], 'owner', ARGV[2])
  redis.call('PEXPIRE', KEYS[1], ARGV[3])
  return {1}
end
local record = redis.call('HMGET', KEYS[1], 'state', 'fp', 'status', 'ct', 'body', 'omitted')
return {0, record[1], record[2], record[3], record[4], record[5], record[6]}
`)

var idempotencyCompleteScript = redis.NewScript(`
if redis.call('HGET', KEYS[1], 'owner') ~= ARGV[1] then
  return 0
end
redis.call('HSET', KEYS[1], 'state', 'completed', 'status', ARGV[2], 'ct', ARGV[3], 'body', ARGV[4], 'omitted', ARGV[5])
redis.call('HDEL', KEYS[1], 'owner')
redis.call('PEXPIRE', KEYS[1], ARGV[6])
return 1
`)

var idempotencyReleaseScript = redis.NewScript(`
if redis.call('HGET', KEYS[1], 'owner') ~= ARGV[1] then
  return 0
end
return redis.call('DEL', KEYS[1])
`)

type IdempotencyStoreParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type idempotencyStore struct {
	client *redis.Client
	l      *zap.Logger
}

func NewIdempotencyStore(p IdempotencyStoreParams) repositories.IdempotencyStore {
	return &idempotencyStore{
		client: p.Client,
		l:      p.Logger.Named("redis.idempotency-store"),
	}
}

func (s *idempotencyStore) Claim(
	ctx context.Context,
	claim repositories.IdempotencyClaim,
) (repositories.IdempotencyClaimResult, error) {
	raw, err := idempotencyClaimScript.Run(
		ctx,
		s.client,
		[]string{idempotencyPrefix + claim.Key},
		claim.Fingerprint,
		claim.Owner,
		claim.LockTTL.Milliseconds(),
	).Slice()
	if err != nil {
		return repositories.IdempotencyClaimResult{}, fmt.Errorf("claim idempotency key: %w", err)
	}

	return decodeIdempotencyClaim(raw)
}

func (s *idempotencyStore) Complete(
	ctx context.Context,
	completion *repositories.IdempotencyCompletion,
) error {
	held, err := idempotencyCompleteScript.Run(
		ctx,
		s.client,
		[]string{idempotencyPrefix + completion.Key},
		completion.Owner,
		completion.Status,
		completion.ContentType,
		completion.Body,
		strconv.FormatBool(completion.BodyOmitted),
		completion.TTL.Milliseconds(),
	).Int()
	if err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}
	if held == 0 {
		return repositories.ErrIdempotencyClaimLost
	}
	return nil
}

func (s *idempotencyStore) Release(ctx context.Context, key, owner string) error {
	if err := idempotencyReleaseScript.Run(
		ctx,
		s.client,
		[]string{idempotencyPrefix + key},
		owner,
	).Err(); err != nil {
		return fmt.Errorf("release idempotency key: %w", err)
	}
	return nil
}

func decodeIdempotencyClaim(raw []any) (repositories.IdempotencyClaimResult, error) {
	if len(raw) == 0 {
		return repositories.IdempotencyClaimResult{}, errors.New(
			"claim idempotency key: empty script reply",
		)
	}
	if flag, _ := raw[0].(int64); flag == 1 {
		return repositories.IdempotencyClaimResult{Claimed: true}, nil
	}
	if len(raw) != 7 {
		return repositories.IdempotencyClaimResult{}, fmt.Errorf(
			"claim idempotency key: unexpected script reply of %d values",
			len(raw),
		)
	}

	record := &repositories.IdempotencyRecord{
		State:       repositories.IdempotencyState(replyString(raw[1])),
		Fingerprint: replyString(raw[2]),
		ContentType: replyString(raw[4]),
		BodyOmitted: replyString(raw[6]) == "true",
	}
	if status := replyString(raw[3]); status != "" {
		code, err := strconv.Atoi(status)
		if err != nil {
			return repositories.IdempotencyClaimResult{}, fmt.Errorf(
				"claim idempotency key: stored status %q: %w",
				status,
				err,
			)
		}
		record.Status = code
	}
	if body := replyString(raw[5]); body != "" {
		record.Body = []byte(body)
	}

	return repositories.IdempotencyClaimResult{Existing: record}, nil
}

func replyString(value any) string {
	text, _ := value.(string)
	return text
}
