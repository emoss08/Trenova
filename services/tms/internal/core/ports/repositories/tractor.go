package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListTractorsRequest struct {
	Filter                  *pagination.QueryOptions `json:"filter"`
	IncludeEquipmentDetails bool                     `json:"includeEquipmentDetails"`
	IncludeFleetDetails     bool                     `json:"includeFleetDetails"`
	IncludeWorkerDetails    bool                     `json:"includeWorkerDetails"`
	Status                  string                   `json:"status"`
}

type GetTractorByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
}

type BulkUpdateTractorStatusRequest struct {
	TenantInfo pagination.TenantInfo       `json:"-"`
	TractorIDs []pulid.ID                  `json:"tractorIds"`
	Status     domaintypes.EquipmentStatus `json:"status"`
}

type GetTractorsByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TractorIDs []pulid.ID            `json:"tractorIds"`
}

type TractorSelectOptionsRequest struct {
	SelectOptionsRequest *pagination.SelectQueryRequest `json:"-"`
}

type TractorRepository interface {
	List(
		ctx context.Context,
		req *ListTractorsRequest,
	) (*pagination.ListResult[*tractor.Tractor], error)
	GetByID(
		ctx context.Context,
		req GetTractorByIDRequest,
	) (*tractor.Tractor, error)
	Create(
		ctx context.Context,
		entity *tractor.Tractor,
	) (*tractor.Tractor, error)
	Update(
		ctx context.Context,
		entity *tractor.Tractor,
	) (*tractor.Tractor, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateTractorStatusRequest,
	) ([]*tractor.Tractor, error)
	SelectOptions(
		ctx context.Context,
		req *TractorSelectOptionsRequest,
	) (*pagination.ListResult[*tractor.Tractor], error)
	GetByIDs(
		ctx context.Context,
		req GetTractorsByIDsRequest,
	) ([]*tractor.Tractor, error)
}
