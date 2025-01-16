package repositories

import (
	"context"

	"github.com/trenova-app/transport/internal/core/domain/hazardousmaterial"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/pkg/types/pulid"
)

type GetHazardousMaterialByIDOptions struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type HazardousMaterialRepository interface {
	List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*hazardousmaterial.HazardousMaterial], error)
	GetByID(ctx context.Context, opts GetHazardousMaterialByIDOptions) (*hazardousmaterial.HazardousMaterial, error)
	Create(ctx context.Context, hm *hazardousmaterial.HazardousMaterial) (*hazardousmaterial.HazardousMaterial, error)
	Update(ctx context.Context, hm *hazardousmaterial.HazardousMaterial) (*hazardousmaterial.HazardousMaterial, error)
}
