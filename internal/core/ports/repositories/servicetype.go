package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetServiceTypeByIDOptions struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type ServiceTypeRepository interface {
	List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*servicetype.ServiceType], error)
	GetByID(ctx context.Context, opts GetServiceTypeByIDOptions) (*servicetype.ServiceType, error)
	Create(ctx context.Context, st *servicetype.ServiceType) (*servicetype.ServiceType, error)
	Update(ctx context.Context, st *servicetype.ServiceType) (*servicetype.ServiceType, error)
}
