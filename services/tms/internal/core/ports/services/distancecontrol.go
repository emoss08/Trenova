package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distancecontrol"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DistanceControlService interface {
	Get(ctx context.Context, tenantInfo pagination.TenantInfo) (*distancecontrol.DistanceControl, error)
	Update(ctx context.Context, entity *distancecontrol.DistanceControl, userID pulid.ID) (*distancecontrol.DistanceControl, error)
}
