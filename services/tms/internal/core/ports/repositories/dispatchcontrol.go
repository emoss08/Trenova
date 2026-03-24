package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetDispatchControlRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type DispatchControlRepository interface {
	GetByOrgID(
		ctx context.Context,
		req GetDispatchControlRequest,
	) (*dispatchcontrol.DispatchControl, error)
	Create(
		ctx context.Context,
		entity *dispatchcontrol.DispatchControl,
	) (*dispatchcontrol.DispatchControl, error)
	Update(
		ctx context.Context,
		entity *dispatchcontrol.DispatchControl,
	) (*dispatchcontrol.DispatchControl, error)
	GetOrCreate(
		ctx context.Context,
		orgID, buID pulid.ID,
	) (*dispatchcontrol.DispatchControl, error)
}
