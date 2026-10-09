package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
)

type AcquireProviderSlotRequest struct {
	ProviderID pulid.ID
	Token      string
	Limit      int
	TTL        time.Duration
}

type ProviderSlotRequest struct {
	ProviderID pulid.ID
	Token      string
	TTL        time.Duration
}

type ProviderSlotRepository interface {
	Acquire(ctx context.Context, req AcquireProviderSlotRequest) (bool, error)
	Refresh(ctx context.Context, req ProviderSlotRequest) (bool, error)
	Release(ctx context.Context, req ProviderSlotRequest) error
}
