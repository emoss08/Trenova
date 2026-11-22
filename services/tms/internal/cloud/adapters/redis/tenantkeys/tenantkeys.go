package tenantkeys

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/shared/redis"
	"go.uber.org/fx"
)

const scanCount = 500

var ErrFragmentTooShort = errors.New(
	"a key purge needs an identifier long enough to name one tenant alone",
)

type Params struct {
	fx.In

	Client *redis.Client
}

type purge struct {
	client *redis.Client
}

func New(p Params) cloudtenantports.KeyPurge {
	return &purge{client: p.Client}
}

func (p *purge) DeleteMatching(ctx context.Context, fragment string, beat func()) (int64, error) {
	if len(fragment) < 20 || strings.ContainsAny(fragment, "*?[]\\") {
		return 0, ErrFragmentTooShort
	}
	pattern := "*" + fragment + "*"
	var (
		cursor  uint64
		deleted int64
	)
	for {
		keys, next, err := p.client.Scan(ctx, cursor, pattern, scanCount).Result()
		if err != nil {
			return deleted, fmt.Errorf("scan the keys of a tenant: %w", err)
		}
		if len(keys) > 0 {
			n, uerr := p.client.Unlink(ctx, keys...).Result()
			if uerr != nil {
				return deleted, fmt.Errorf("delete the keys of a tenant: %w", uerr)
			}
			deleted += n
		}
		beat()
		if next == 0 {
			return deleted, nil
		}
		cursor = next
	}
}
