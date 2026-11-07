package driverpositioningports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const MaxLive = 2_000

type GetRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type LiveRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerIDs  []pulid.ID
	Now        int64
}

type ReplaceRequest struct {
	TenantInfo  pagination.TenantInfo
	Positioning *driverpositioning.Positioning
}

type Replaced struct {
	Positioning *driverpositioning.Positioning
	Previous    *driverpositioning.Positioning
}

type CancelRequest struct {
	TenantInfo   pagination.TenantInfo
	ID           pulid.ID
	CanceledByID pulid.ID
	CanceledAt   int64
}

type Repository interface {
	Get(ctx context.Context, req *GetRequest) (*driverpositioning.Positioning, error)
	ListLive(ctx context.Context, req *LiveRequest) ([]*driverpositioning.Positioning, error)
	Replace(ctx context.Context, req *ReplaceRequest) (*Replaced, error)
	Cancel(ctx context.Context, req *CancelRequest) (*driverpositioning.Positioning, error)
}

type LiveReader interface {
	ListLive(ctx context.Context, req *LiveRequest) ([]*driverpositioning.Positioning, error)
}

type CreateRequest struct {
	TenantInfo pagination.TenantInfo
	Actor      *permissionports.RequestActor
	WorkerID   pulid.ID
	ToCell     int64
	SlotStart  int64
}

type CallOffRequest struct {
	TenantInfo pagination.TenantInfo
	Actor      *permissionports.RequestActor
	ID         pulid.ID
}

type Service interface {
	Create(ctx context.Context, req *CreateRequest) (*driverpositioning.Positioning, error)
	CallOff(ctx context.Context, req *CallOffRequest) (*driverpositioning.Positioning, error)
	ListLive(ctx context.Context, req *LiveRequest) ([]*driverpositioning.Positioning, error)
}
