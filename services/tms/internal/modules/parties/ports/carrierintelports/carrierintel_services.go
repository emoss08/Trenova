package carrierintelports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CarrierIntelEnsureFreshRequest struct {
	TenantInfo pagination.TenantInfo
	CarrierIDs []pulid.ID
	Purpose    carrierintel.Purpose
	Budget     time.Duration
}

type CarrierIntelGate interface {
	GateFor(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		carrierIDs []pulid.ID,
	) (map[pulid.ID]*carrier.IntelGate, error)
	EnsureFresh(ctx context.Context, req *CarrierIntelEnsureFreshRequest)
}

type CarrierIntelEventReader interface {
	ListEvents(
		ctx context.Context,
		req *ListCarrierIntelEventsRequest,
	) (*pagination.CursorListResult[*carrierintelevent.CarrierIntelEvent], error)
}

type CarrierLifecycleEvent struct {
	TenantInfo     pagination.TenantInfo
	Carrier        *carrier.Carrier
	PreviousStatus carrier.Status
	Created        bool
}

type CarrierLifecycleObserver interface {
	CarrierSaved(ctx context.Context, event *CarrierLifecycleEvent)
	CarriersUsed(ctx context.Context, tenantInfo pagination.TenantInfo, carrierIDs []pulid.ID)
}
