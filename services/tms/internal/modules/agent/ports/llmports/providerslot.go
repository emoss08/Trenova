package llmports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
)

type AcquireProviderSlotRequest struct {
	ProviderID        pulid.ID
	Token             string
	Limit             int
	TTL               time.Duration
	OrganizationID    pulid.ID
	OrganizationLimit int
}

type ProviderSlotRequest struct {
	ProviderID     pulid.ID
	Token          string
	TTL            time.Duration
	OrganizationID pulid.ID
}

type ProviderSlotRepository interface {
	Acquire(ctx context.Context, req AcquireProviderSlotRequest) (bool, error)
	Refresh(ctx context.Context, req ProviderSlotRequest) (bool, error)
	Release(ctx context.Context, req ProviderSlotRequest) error
}
