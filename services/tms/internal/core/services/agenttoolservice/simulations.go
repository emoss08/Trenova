package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// The previews below say what a write would change by reading the record
// as it stands. A tool whose current state cannot be read still answers,
// with the change it would make and no "from".

func (t *cancelShipmentTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	shipmentID, err := requirePulid(params.Params, "shipmentId")
	if err != nil {
		return nil, err
	}
	reason, err := requireString(params.Params, "cancelReason")
	if err != nil {
		return nil, err
	}

	entity, err := t.shipments.Get(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantFrom(params),
	})
	if err != nil {
		return nil, err
	}

	return &agent.ToolSimulation{
		Summary: fmt.Sprintf(
			"Would cancel shipment %s, releasing its assignments and stopping it being billed. Reason: %s",
			entity.ProNumber,
			reason,
		),
		Changes: []agent.FieldChange{
			{Field: "status", From: string(entity.Status), To: "Canceled"},
			{Field: "cancelReason", To: reason},
		},
	}, nil
}

func countOf(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}

	return fmt.Sprintf("%d %ss", n, noun)
}
