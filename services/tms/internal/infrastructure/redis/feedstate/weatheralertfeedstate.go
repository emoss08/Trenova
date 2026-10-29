package feedstate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/operations/ports/weatheralertports"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

const weatherAlertFeedStateKey = "weather-alerts:nws:feed-state:v1"

type WeatherAlertFeedStateStoreParams struct {
	fx.In

	Client *redis.Client
}

type weatherAlertFeedStateStore struct {
	client *redis.Client
}

func NewWeatherAlertFeedStateStore(
	p WeatherAlertFeedStateStoreParams,
) weatheralertports.WeatherAlertFeedStateStore {
	return &weatherAlertFeedStateStore{client: p.Client}
}

func (s *weatherAlertFeedStateStore) Get(
	ctx context.Context,
) (*weatheralertports.WeatherAlertFeedState, error) {
	raw, err := s.client.Get(ctx, weatherAlertFeedStateKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil //nolint:nilnil // nil is valid for redis.Nil
		}

		return nil, fmt.Errorf("read weather alert feed state: %w", err)
	}

	state := new(weatheralertports.WeatherAlertFeedState)
	if err = sonic.Unmarshal(raw, state); err != nil {
		_ = s.client.Del(ctx, weatherAlertFeedStateKey).Err()

		return nil, fmt.Errorf("decode weather alert feed state: %w", err)
	}

	return state, nil
}

func (s *weatherAlertFeedStateStore) Save(
	ctx context.Context,
	state *weatheralertports.WeatherAlertFeedState,
	ttl time.Duration,
) error {
	payload, err := sonic.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode weather alert feed state: %w", err)
	}
	if err = s.client.Set(ctx, weatherAlertFeedStateKey, payload, ttl).Err(); err != nil {
		return fmt.Errorf("write weather alert feed state: %w", err)
	}

	return nil
}
