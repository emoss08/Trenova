package assistanttest

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/pulid"
)

type RecordedAnchors struct {
	Asked []*agentports.RecordAnchorRequest
}

func (r *RecordedAnchors) Anchor(
	_ context.Context,
	req *agentports.RecordAnchorRequest,
) []agentturn.RuntimeAnchor {
	r.Asked = append(r.Asked, req)
	out := make([]agentturn.RuntimeAnchor, 0, len(req.Records))
	for _, record := range req.Records {
		out = append(out, agentturn.RuntimeAnchor{
			Kind:  "shipment",
			ID:    record.ID,
			Label: "SEED-BENCH-CM-01",
			Facts: []agentturn.AnchorFact{
				{Name: "status", Value: "Canceled for Range Logistics"},
				{Name: "canceled", Value: "by Admin: Range pulled the load by phone"},
			},
		})
	}

	return out
}

func RefusedProposal() *agent.AgentProposal {
	return &agent.AgentProposal{
		ID:             pulid.MustNew("ap_"),
		ToolName:       "shipment_place_hold",
		Status:         agent.ProposalStatusExecutionFailed,
		TargetResource: "shipment",
		TargetID:       pulid.MustNew("shp_"),
		TargetVersion:  4,
		ExecutionError: fmt.Errorf(
			"%w: the shipment SEED-BENCH-CM-01 was at version 4 when this was proposed and is "+
				"at version 5 now, so nothing was changed",
			agentplan.ErrTargetChanged,
		).Error(),
	}
}
