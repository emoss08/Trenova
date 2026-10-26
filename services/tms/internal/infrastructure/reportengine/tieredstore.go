package reportengine

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/keel/analytics/cache"
)

type tieredStore struct {
	local  cache.Store
	shared cache.Store
	ttl    time.Duration
}

func (t *tieredStore) Get(ctx context.Context, key string) (value []byte, found bool, err error) {
	if local, ok, localErr := t.local.Get(ctx, key); localErr == nil && ok {
		return local, true, nil
	}
	value, found, err = t.shared.Get(ctx, key)
	if err != nil || !found {
		return nil, false, err
	}
	_ = t.local.Set(ctx, key, value, t.ttl)
	return value, true, nil
}

func (t *tieredStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	localErr := t.local.Set(ctx, key, value, ttl)
	if errors.Is(localErr, cache.ErrTooLarge) {
		localErr = nil
	}
	return errors.Join(localErr, t.shared.Set(ctx, key, value, ttl))
}
