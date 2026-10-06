package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
)

type ProviderBreakerRepository interface {
	Rest(ctx context.Context, providerID pulid.ID, cooldown time.Duration) error
	Resting(ctx context.Context, providerIDs []pulid.ID) (map[pulid.ID]time.Duration, error)
}
