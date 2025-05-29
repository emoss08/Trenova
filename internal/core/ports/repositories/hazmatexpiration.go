package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/compliance"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type HazmatExpirationRepository interface {
	GetHazmatExpirationByStateID(
		ctx context.Context,
		stateID pulid.ID,
	) (*compliance.HazmatExpiration, error)
}
