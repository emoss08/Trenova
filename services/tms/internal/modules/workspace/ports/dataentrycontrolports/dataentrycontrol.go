package dataentrycontrolports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetDataEntryControlRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type DataEntryControlRepository interface {
	GetByOrgID(
		ctx context.Context,
		req GetDataEntryControlRequest,
	) (*dataentrycontrol.DataEntryControl, error)
	Create(
		ctx context.Context,
		entity *dataentrycontrol.DataEntryControl,
	) (*dataentrycontrol.DataEntryControl, error)
	Update(
		ctx context.Context,
		entity *dataentrycontrol.DataEntryControl,
	) (*dataentrycontrol.DataEntryControl, error)
	GetOrCreate(
		ctx context.Context,
		orgID, buID pulid.ID,
	) (*dataentrycontrol.DataEntryControl, error)
}
