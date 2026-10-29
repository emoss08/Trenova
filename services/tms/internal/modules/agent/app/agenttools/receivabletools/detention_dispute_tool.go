package receivabletools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/detentionpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
)

const ParamOccurrenceID = "occurrenceId"

const kindDetention = "detention occurrence"

func NewDisputeDetentionTool(
	detentions detentionpreview.DetentionDisputer,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:     "detention_open_dispute",
		Artifact: changepreview.DetentionRecordEntity,
		Description: "Record that the customer disputes a detention charge, with what " +
			"they said. The charge is kept as computed and held back from billing " +
			"until a person settles the dispute by approving or waiving it. Use it " +
			"when a customer's message rejects the charge; do not raise a dispute " +
			"on your own reading of the clock.",
		Resource:    permission.ResourceDetention,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Rationale: "Marks a detention charge disputed inside Trenova and holds it from " +
			"billing; nothing is sent, but the hold is not undone by a tool.",
		Properties: map[string]any{
			ParamOccurrenceID: agenttoolschema.KindID(
				"The occurrence, from detention_list_occurrences or "+
					"shipment_get. Never guess one.",
				permission.KindDetentionOccurrence,
			),
			toolparams.ParamNote: toolparams.StringProperty(
				"What the customer disputes, in their words where "+
					"you have them.",
				toolparams.MaxOperationNoteChars,
			),
		},
		Required: []string{ParamOccurrenceID, toolparams.ParamNote},
		Target: func(params map[string]any) (agenttoolports.ToolTarget, bool) {
			return toolparams.TargetOf(params, ParamOccurrenceID, permission.ResourceDetention)
		},
	}, toolspec.ReceivablePlan[*detentionservice.DisputeParams, *detentionservice.OccurrenceChange]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*detentionservice.DisputeParams, error) {
			occurrenceID, err := toolparams.RequirePulid(params.Params, ParamOccurrenceID)
			if err != nil {
				return nil, err
			}
			note, err := toolparams.RequireBoundedText(
				params.Params,
				toolparams.ParamNote,
				toolparams.MaxOperationNoteChars,
			)
			if err != nil {
				return nil, err
			}

			return &detentionservice.DisputeParams{
				OccurrenceID: occurrenceID,
				TenantInfo:   toolparams.TenantFrom(*params),
				Note:         note,
				UserID:       params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *detentionservice.DisputeParams,
			_ *agenttoolports.ToolExecuteParams,
		) (*detentionservice.OccurrenceChange, error) {
			return detentions.PreviewDispute(ctx, req)
		},
		Refused: func(*detentionservice.DisputeParams) string {
			return "Would mark the detention charge disputed."
		},
		Render: func(
			req *detentionservice.DisputeParams,
			change *detentionservice.OccurrenceChange,
		) (*agentpreview.ToolPreview, error) {
			summary := fmt.Sprintf(
				"Would mark the %s %s detention charge %s disputed and hold it from billing: %s",
				change.Before.BillableAmount.StringFixed(2),
				change.Before.Currency,
				detentionpreview.DetentionPlace(change.Before),
				req.Note,
			)

			return detentionpreview.DetentionDecisionPreview(summary, change, "disputedAt")
		},
		Run: func(
			ctx context.Context,
			req *detentionservice.DisputeParams,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			saved, err := detentions.Dispute(ctx, *req)
			if err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
				Action: "disputed",
				Kind:   kindDetention,
				Record: changepreview.RecordOf(changepreview.DetentionRecordEntity, saved.ID),
				IDs: map[string]string{
					ParamOccurrenceID:          saved.ID.String(),
					toolparams.ParamShipmentID: saved.ShipmentID.String(),
				},
			}, nil
		},
	})
}
