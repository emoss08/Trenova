package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/hazmatexpiration"
	"github.com/emoss08/trenova/pkg/pulid"
)

type HazmatExpirationRepository interface {
	GetHazmatExpirationByStateID(
		ctx context.Context,
		stateID pulid.ID,
	) (*hazmatexpiration.HazmatExpiration, error)
}

type HazmatExpirationCacheRepository interface {
	GetHazmatExpirationByStateID(
		ctx context.Context,
		stateID pulid.ID,
	) (*hazmatexpiration.HazmatExpiration, error)
	Set(ctx context.Context, expiration *hazmatexpiration.HazmatExpiration) error
	Invalidate(ctx context.Context, stateID pulid.ID) error
}
