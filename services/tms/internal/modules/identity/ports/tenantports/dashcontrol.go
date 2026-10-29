package tenantports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/dashcontrol"
	"github.com/emoss08/trenova/pkg/pagination"
)

type DashControlRepository interface {
	GetOrCreate(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*dashcontrol.DashControl, error)
	Update(
		ctx context.Context,
		entity *dashcontrol.DashControl,
	) (*dashcontrol.DashControl, error)
}
