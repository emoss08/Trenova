package repositories

import (
	"context"

	"github.com/trenova-app/transport/internal/core/domain/compliance"
	"github.com/trenova-app/transport/pkg/types/pulid"
)

type HazmatExpirationRepository interface {
	GetHazmatExpirationByStateID(ctx context.Context, stateID pulid.ID) (*compliance.HazmatExpiration, error)
}
