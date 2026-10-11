package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
)

const (
	paramOccurrenceID = "occurrenceId"
	paramNote         = "note"
	kindDetention     = "detention occurrence"
)

type detentionDisputer interface {
	Dispute(
		ctx context.Context,
		params detentionservice.DisputeParams,
	) (*detention.DetentionOccurrence, error)
	PreviewDispute(
		ctx context.Context,
		params *detentionservice.DisputeParams,
	) (*detentionservice.OccurrenceChange, error)
}

func newDisputeDetentionTool(detentions detentionDisputer) agentports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "dispute_detention",
		artifact: detentionRecordEntity,
		description: "Record that the customer disputes a detention charge, with what " +
			"they said. The charge is kept as computed and held back from billing " +
			"until a person settles the dispute by approving or waiving it. Use it " +
			"when a customer's message rejects the charge; do not raise a dispute " +
			"on your own reading of the clock.",
		resource:    permission.ResourceDetentionPolicy,
		operation:   permission.OpUpdate,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		rationale: "Marks a detention charge disputed inside Trenova and holds it from " +
			"billing; nothing is sent, but the hold is not undone by a tool.",
		properties: map[string]any{
			paramOccurrenceID: agenttoolschema.KindID(
				"The occurrence, from list_detention_desk or "+
					"get_shipment. Never guess one.",
				permission.KindDetentionOccurrence,
			),
			paramNote: stringProperty("What the customer disputes, in their words where "+
				"you have them.", maxOperationNoteChars),
		},
		required: []string{paramOccurrenceID, paramNote},
		target: func(params map[string]any) (agentports.ToolTarget, bool) {
			return targetOf(params, paramOccurrenceID, permission.ResourceDetentionPolicy)
		},
	}, receivablePlan[*detentionservice.DisputeParams, *detentionservice.OccurrenceChange]{
		request: func(params *agentports.ToolExecuteParams) (*detentionservice.DisputeParams, error) {
			occurrenceID, err := requirePulid(params.Params, paramOccurrenceID)
			if err != nil {
				return nil, err
			}
			note, err := requireBoundedText(params.Params, paramNote, maxOperationNoteChars)
			if err != nil {
				return nil, err
			}

			return &detentionservice.DisputeParams{
				OccurrenceID: occurrenceID,
				TenantInfo:   tenantFrom(*params),
				Note:         note,
				UserID:       params.Actor.UserID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *detentionservice.DisputeParams,
			_ *agentports.ToolExecuteParams,
		) (*detentionservice.OccurrenceChange, error) {
			return detentions.PreviewDispute(ctx, req)
		},
		refused: func(*detentionservice.DisputeParams) string {
			return "Would mark the detention charge disputed."
		},
		render: func(
			req *detentionservice.DisputeParams,
			change *detentionservice.OccurrenceChange,
		) (*agent.ToolPreview, error) {
			summary := fmt.Sprintf(
				"Would mark the %s %s detention charge %s disputed and hold it from billing: %s",
				change.Before.BillableAmount.StringFixed(2),
				change.Before.Currency,
				detentionPlace(change.Before),
				req.Note,
			)

			return detentionDecisionPreview(summary, change, "disputedAt")
		},
		run: func(
			ctx context.Context,
			req *detentionservice.DisputeParams,
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			saved, err := detentions.Dispute(ctx, *req)
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: "disputed",
				Kind:   kindDetention,
				Record: recordOf(detentionRecordEntity, saved.ID),
				IDs: map[string]string{
					paramOccurrenceID: saved.ID.String(),
					paramShipmentID:   saved.ShipmentID.String(),
				},
			}, nil
		},
	})
}
