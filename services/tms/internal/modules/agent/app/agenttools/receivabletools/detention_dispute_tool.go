package receivabletools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
)

const ParamOccurrenceID = "occurrenceId"

const kindDetention = "detention occurrence"

func NewDisputeDetentionTool(detentions common.DetentionDisputer) agentports.AgentTool {
	return common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:     "detention_open_dispute",
		Artifact: common.DetentionRecordEntity,
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
			common.ParamNote: common.StringProperty(
				"What the customer disputes, in their words where "+
					"you have them.",
				common.MaxOperationNoteChars,
			),
		},
		Required: []string{ParamOccurrenceID, common.ParamNote},
		Target: func(params map[string]any) (agentports.ToolTarget, bool) {
			return common.TargetOf(params, ParamOccurrenceID, permission.ResourceDetention)
		},
	}, common.ReceivablePlan[*detentionservice.DisputeParams, *detentionservice.OccurrenceChange]{
		Request: func(params *agentports.ToolExecuteParams) (*detentionservice.DisputeParams, error) {
			occurrenceID, err := common.RequirePulid(params.Params, ParamOccurrenceID)
			if err != nil {
				return nil, err
			}
			note, err := common.RequireBoundedText(
				params.Params,
				common.ParamNote,
				common.MaxOperationNoteChars,
			)
			if err != nil {
				return nil, err
			}

			return &detentionservice.DisputeParams{
				OccurrenceID: occurrenceID,
				TenantInfo:   common.TenantFrom(*params),
				Note:         note,
				UserID:       params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *detentionservice.DisputeParams,
			_ *agentports.ToolExecuteParams,
		) (*detentionservice.OccurrenceChange, error) {
			return detentions.PreviewDispute(ctx, req)
		},
		Refused: func(*detentionservice.DisputeParams) string {
			return "Would mark the detention charge disputed."
		},
		Render: func(
			req *detentionservice.DisputeParams,
			change *detentionservice.OccurrenceChange,
		) (*agent.ToolPreview, error) {
			summary := fmt.Sprintf(
				"Would mark the %s %s detention charge %s disputed and hold it from billing: %s",
				change.Before.BillableAmount.StringFixed(2),
				change.Before.Currency,
				common.DetentionPlace(change.Before),
				req.Note,
			)

			return common.DetentionDecisionPreview(summary, change, "disputedAt")
		},
		Run: func(
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
				Record: common.RecordOf(common.DetentionRecordEntity, saved.ID),
				IDs: map[string]string{
					ParamOccurrenceID:      saved.ID.String(),
					common.ParamShipmentID: saved.ShipmentID.String(),
				},
			}, nil
		},
	})
}
