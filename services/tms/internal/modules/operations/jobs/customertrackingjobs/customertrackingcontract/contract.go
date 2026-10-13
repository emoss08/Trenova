package customertrackingcontract

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	EvaluateWorkflowName     = "EvaluateCustomerStatusWorkflow"
	SweepWorkflowName        = "CustomerStatusSweepWorkflow"
	PurgeWorkflowName        = "CustomerTrackingLinkPurgeWorkflow"
	EvaluateSignalName       = "customer-status-evaluate"
	evaluateWorkflowIDPrefix = "customer-status-evaluate-"
)

type EvaluatePayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	ShipmentID     pulid.ID `json:"shipmentId"`
	Iterations     int      `json:"iterations"`
}

func (p *EvaluatePayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type EvaluateSignal struct{}

func EvaluateWorkflowID(shipmentID pulid.ID) string {
	return evaluateWorkflowIDPrefix + shipmentID.String()
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

var _ customertrackingports.EvaluationRequester = (*Requester)(nil)

func NewRequester(p RequesterParams) customertrackingports.EvaluationRequester {
	return &Requester{signals: p.Signals, l: p.Logger.Named("customer-status-requester")}
}

// RequestEvaluation signals the shipment's evaluation workflow, starting it if
// it is not running, so a burst of changes to one shipment is evaluated once.
func (r *Requester) RequestEvaluation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) {
	if r.signals == nil || shipmentID.IsNil() || tenantInfo.OrgID.IsNil() {
		return
	}

	_, err := r.signals.SignalWithStartWorkflow(
		ctx,
		EvaluateWorkflowID(shipmentID),
		EvaluateSignalName,
		EvaluateSignal{},
		client.StartWorkflowOptions{
			ID:            EvaluateWorkflowID(shipmentID),
			TaskQueue:     temporaltype.TaskQueueDispatch.String(),
			StaticSummary: "Send the customer the status emails its rules ask for",
			Priority:      temporal.Priority{FairnessKey: tenantInfo.OrgID.String()},
		},
		EvaluateWorkflowName,
		&EvaluatePayload{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			ShipmentID:     shipmentID,
		},
	)
	if err != nil {
		r.l.Warn("could not request a customer status evaluation",
			zap.String("shipmentId", shipmentID.String()),
			zap.Error(err))
	}
}
