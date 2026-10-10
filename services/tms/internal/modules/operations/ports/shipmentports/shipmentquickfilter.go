package shipmentports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

type ResolveShipmentQuickFilterBasisRequest struct {
	TenantInfo pagination.TenantInfo
	Timezone   string
	Margin     bool
	Detention  bool
}

type ShipmentQuickFilterBasisResolver interface {
	Resolve(
		ctx context.Context,
		req *ResolveShipmentQuickFilterBasisRequest,
	) (*ShipmentQuickFilterBasis, error)
	Prepare(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		opts *ShipmentOptions,
	) error
}
