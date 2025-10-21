package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ListHazardousMaterialRequest struct {
	Filter *pagination.QueryOptions `json:"filter" form:"filter"`
}

type GetHazardousMaterialByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type HazardousMaterialRepository interface {
	List(
		ctx context.Context,
		req *ListHazardousMaterialRequest,
	) (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error)
	GetByID(
		ctx context.Context,
		req GetHazardousMaterialByIDRequest,
	) (*hazardousmaterial.HazardousMaterial, error)
	Create(
		ctx context.Context,
		hm *hazardousmaterial.HazardousMaterial,
	) (*hazardousmaterial.HazardousMaterial, error)
	Update(
		ctx context.Context,
		hm *hazardousmaterial.HazardousMaterial,
	) (*hazardousmaterial.HazardousMaterial, error)
}
