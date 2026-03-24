package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListServiceTypesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetServiceTypeByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
}

type BulkUpdateServiceTypeStatusRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	ServiceTypeIDs []pulid.ID            `json:"serviceTypeIds"`
	Status         domaintypes.Status    `json:"status"`
}

type GetServiceTypesByIDsRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	ServiceTypeIDs []pulid.ID            `json:"serviceTypeIds"`
}

type ServiceTypeSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	Classes            []string                       `json:"classes"`
}

type ServiceTypeRepository interface {
	List(
		ctx context.Context,
		req *ListServiceTypesRequest,
	) (*pagination.ListResult[*servicetype.ServiceType], error)
	GetByID(
		ctx context.Context,
		req GetServiceTypeByIDRequest,
	) (*servicetype.ServiceType, error)
	GetByIDs(
		ctx context.Context,
		req GetServiceTypesByIDsRequest,
	) ([]*servicetype.ServiceType, error)
	Create(
		ctx context.Context,
		entity *servicetype.ServiceType,
	) (*servicetype.ServiceType, error)
	Update(
		ctx context.Context,
		entity *servicetype.ServiceType,
	) (*servicetype.ServiceType, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateServiceTypeStatusRequest,
	) ([]*servicetype.ServiceType, error)
	SelectOptions(
		ctx context.Context,
		req *ServiceTypeSelectOptionsRequest,
	) (*pagination.ListResult[*servicetype.ServiceType], error)
}
