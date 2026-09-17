package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
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
