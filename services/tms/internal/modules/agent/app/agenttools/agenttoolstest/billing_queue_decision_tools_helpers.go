package agenttoolstest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
)

type FakeApprovalInvoices struct {
	Result *invoiceports.CreateInvoiceFromBillingQueueResult `json:"-"`
	err    error
}

func (f *FakeApprovalInvoices) PreviewApprovalInvoice(
	context.Context,
	*invoiceports.CreateInvoiceFromBillingQueueRequest,
) (*invoiceports.CreateInvoiceFromBillingQueueResult, error) {
	return f.Result, f.err
}

func ReviewedItem(status billingqueue.Status) *billingqueue.BillingQueueItem {
	billerID := pulid.MustNew("usr_")

	return &billingqueue.BillingQueueItem{
		ID:               pulid.MustNew("bqi_"),
		ShipmentID:       pulid.MustNew("shp_"),
		Number:           "INV-5001",
		Status:           status,
		AssignedBillerID: &billerID,
		Version:          3,
		Shipment:         &shipment.Shipment{ProNumber: "PRO-500"},
	}
}

func AgentParamsFor(params map[string]any) agentports.ToolExecuteParams {
	out := ExecuteParams(params)
	out.Actor = AgentActorFor(out.OrganizationID, out.BusinessUnitID)

	return out
}
