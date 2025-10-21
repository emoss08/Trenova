package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetServiceTypeByIDOptions struct {
	ID     pulid.ID `json:"id"     form:"id"`
	OrgID  pulid.ID `json:"orgId"  form:"orgId"`
	BuID   pulid.ID `json:"buId"   form:"buId"`
	UserID pulid.ID `json:"userId" form:"userId"`
}

type ListServiceTypeRequest struct {
	Filter *pagination.QueryOptions `form:"filter" json:"filter"`
	Status string                   `form:"status" json:"status"`
}

type ServiceTypeRepository interface {
	List(
		ctx context.Context,
		req *ListServiceTypeRequest,
	) (*pagination.ListResult[*servicetype.ServiceType], error)
	GetByID(ctx context.Context, opts GetServiceTypeByIDOptions) (*servicetype.ServiceType, error)
	Create(ctx context.Context, st *servicetype.ServiceType) (*servicetype.ServiceType, error)
	Update(ctx context.Context, st *servicetype.ServiceType) (*servicetype.ServiceType, error)
}
