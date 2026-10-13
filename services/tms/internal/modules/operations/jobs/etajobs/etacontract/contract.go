package etacontract

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	RefreshWorkflowName     = "RefreshShipmentEtaWorkflow"
	SweepWorkflowName       = "ShipmentEtaSweepWorkflow"
	RefreshSignalName       = "shipment-eta-refresh"
	refreshWorkflowIDPrefix = "shipment-eta-refresh-"
)

type RefreshPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	ShipmentID     pulid.ID `json:"shipmentId"`
	Iterations     int      `json:"iterations"`
}

func (p *RefreshPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type RefreshSignal struct{}

func RefreshWorkflowID(shipmentID pulid.ID) string {
	return refreshWorkflowIDPrefix + shipmentID.String()
}

type RequesterParams struct {
	fx.In

	Signals kernelports.WorkflowSignalStarter `optional:"true"`
	Logger  *zap.Logger
}

type Requester struct {
	signals kernelports.WorkflowSignalStarter
	l       *zap.Logger
}

var _ stopetaports.RefreshRequester = (*Requester)(nil)

func NewRequester(p RequesterParams) stopetaports.RefreshRequester {
	return &Requester{signals: p.Signals, l: p.Logger.Named("shipment-eta-requester")}
}

func (r *Requester) RequestRefresh(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) {
	if r.signals == nil || shipmentID.IsNil() || tenantInfo.OrgID.IsNil() {
		return
	}

	_, err := r.signals.SignalWithStartWorkflow(
		ctx,
		RefreshWorkflowID(shipmentID),
		RefreshSignalName,
		RefreshSignal{},
		client.StartWorkflowOptions{
			ID:            RefreshWorkflowID(shipmentID),
			TaskQueue:     temporaltype.TaskQueueDispatch.String(),
			StaticSummary: "Project the shipment's arrival at each open stop",
			Priority:      temporal.Priority{FairnessKey: tenantInfo.OrgID.String()},
		},
		RefreshWorkflowName,
		&RefreshPayload{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			ShipmentID:     shipmentID,
		},
	)
	if err != nil {
		r.l.Warn("could not request a shipment eta refresh",
			zap.String("shipmentId", shipmentID.String()),
			zap.Error(err))
	}
}
