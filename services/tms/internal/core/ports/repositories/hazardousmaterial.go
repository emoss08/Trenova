package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListHazardousMaterialsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetHazardousMaterialByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
}

type BulkUpdateHazardousMaterialStatusRequest struct {
	TenantInfo           pagination.TenantInfo `json:"-"`
	HazardousMaterialIDs []pulid.ID            `json:"hazardousMaterialIds"`
	Status               domaintypes.Status    `json:"status"`
}

type GetHazardousMaterialsByIDsRequest struct {
	TenantInfo           pagination.TenantInfo `json:"-"`
	HazardousMaterialIDs []pulid.ID            `json:"hazardousMaterialIds"`
}

type HazardousMaterialSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
}

type HazardousMaterialRepository interface {
	List(
		ctx context.Context,
		req *ListHazardousMaterialsRequest,
	) (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error)
	GetByID(
		ctx context.Context,
		req GetHazardousMaterialByIDRequest,
	) (*hazardousmaterial.HazardousMaterial, error)
	GetByIDs(
		ctx context.Context,
		req GetHazardousMaterialsByIDsRequest,
	) ([]*hazardousmaterial.HazardousMaterial, error)
	Create(
		ctx context.Context,
		entity *hazardousmaterial.HazardousMaterial,
	) (*hazardousmaterial.HazardousMaterial, error)
	Update(
		ctx context.Context,
		entity *hazardousmaterial.HazardousMaterial,
	) (*hazardousmaterial.HazardousMaterial, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateHazardousMaterialStatusRequest,
	) ([]*hazardousmaterial.HazardousMaterial, error)
	SelectOptions(
		ctx context.Context,
		req *HazardousMaterialSelectOptionsRequest,
	) (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error)
}
