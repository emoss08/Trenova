package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distancecontrol"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DistanceControlRepository interface {
	Get(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*distancecontrol.DistanceControl, error)
	EnsureDefault(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*distancecontrol.DistanceControl, error)
	Update(
		ctx context.Context,
		entity *distancecontrol.DistanceControl,
	) (*distancecontrol.DistanceControl, error)
	ResolveProfile(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		purpose string,
	) (pulid.ID, error)
}
