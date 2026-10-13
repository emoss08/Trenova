package stopetaports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ReplaceForShipmentsRequest struct {
	TenantInfo  pagination.TenantInfo
	ShipmentIDs []pulid.ID
	Rows        []*stopeta.StopETA
}

type ListTrackedShipmentIDsRequest struct {
	TenantInfo pagination.TenantInfo
	Statuses   []shipment.Status
	Limit      int
}

type ListAttentionRequest struct {
	TenantInfo pagination.TenantInfo
	Statuses   []shipment.Status
	Limit      int
}

type Repository interface {
	ReplaceForShipments(ctx context.Context, req *ReplaceForShipmentsRequest) error
	ListByShipmentIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentIDs []pulid.ID,
	) ([]*stopeta.StopETA, error)
	ListAttention(ctx context.Context, req *ListAttentionRequest) ([]*stopeta.StopETA, error)
	ListTrackedShipmentIDs(
		ctx context.Context,
		req *ListTrackedShipmentIDsRequest,
	) ([]pulid.ID, error)
	ListActiveTenants(
		ctx context.Context,
		statuses []shipment.Status,
	) ([]pagination.TenantInfo, error)
}

type RefreshResult struct {
	Shipments int
	Rows      int
	Attention int
}

type Refresher interface {
	RefreshShipments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentIDs []pulid.ID,
	) (*RefreshResult, error)
	RefreshTenant(ctx context.Context, tenantInfo pagination.TenantInfo) (*RefreshResult, error)
}

type RefreshRequester interface {
	RequestRefresh(ctx context.Context, tenantInfo pagination.TenantInfo, shipmentID pulid.ID)
}

type Observer interface {
	StopEtasRefreshed(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentIDs []pulid.ID,
	)
}
