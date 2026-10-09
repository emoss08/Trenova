package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
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

func newDisputeDetentionTool(detentions detentionDisputer) serviceports.AgentTool {
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
			paramOccurrenceID: agenttoolschema.IDText(
				"The occurrence, from list_detention_desk or " +
					"get_shipment. Never guess one.",
			),
			paramNote: stringProperty("What the customer disputes, in their words where "+
				"you have them.", maxOperationNoteChars),
		},
		required: []string{paramOccurrenceID, paramNote},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramOccurrenceID, permission.ResourceDetentionPolicy)
		},
	}, receivablePlan[*detentionservice.DisputeParams, *detentionservice.OccurrenceChange]{
		request: func(params *serviceports.ToolExecuteParams) (*detentionservice.DisputeParams, error) {
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
			_ *serviceports.ToolExecuteParams,
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
			_ *serviceports.ToolExecuteParams,
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
