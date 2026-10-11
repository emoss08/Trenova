package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const paramInsightID = "insightId"

type insightRestorer interface {
	GetDetail(
		ctx context.Context,
		req insightports.GetInsightDetailRequest,
	) (*insightports.InsightDetail, error)
	PlanRestore(
		ctx context.Context,
		req insightports.RestoreInsightRequest,
	) (*kernelports.RecordChange[insight.Insight], error)
	Restore(ctx context.Context, req insightports.RestoreInsightRequest) (*insight.Insight, error)
}

func restoreInsightRequest(
	params *agentports.ToolExecuteParams,
) (*insightports.RestoreInsightRequest, error) {
	id, err := requirePulid(params.Params, paramInsightID)
	if err != nil {
		return nil, err
	}

	return &insightports.RestoreInsightRequest{ID: id, TenantInfo: tenantFrom(*params)}, nil
}

func newRestoreInsightTool(insights insightRestorer) agentports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name: "restore_insight",
		description: "Put a dismissed insight back on the list of findings to act on, when " +
			"the reason it was dismissed no longer holds. Find it with list_insights and " +
			"status Dismissed.",
		resource:    permission.ResourceInsight,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierActWithApproval,
		maxTier:     agent.TierAutoExecute,
		reversible:  true,
		rationale: "Returns a finding to the active list inside Trenova; nothing is sent and " +
			"dismiss_insight takes it off again.",
		properties: map[string]any{
			paramInsightID: agenttoolschema.RecordIDText(permission.ResourceInsight,
				"The dismissed insight, from list_insights. Never guess one."),
		},
		required:    []string{paramInsightID},
		searchTerms: []string{"restore insight", "undismiss", "reopen finding"},
		target: func(params map[string]any) (agentports.ToolTarget, bool) {
			return targetOf(params, paramInsightID, permission.ResourceInsight)
		},
	}, receivablePlan[*insightports.RestoreInsightRequest, *kernelports.RecordChange[insight.Insight]]{
		request: restoreInsightRequest,
		settle: func(
			ctx context.Context,
			req *insightports.RestoreInsightRequest,
			params *agentports.ToolExecuteParams,
		) error {
			_, err := insights.GetDetail(ctx, insightports.GetInsightDetailRequest{
				ID:         req.ID,
				UserID:     params.Actor.UserID,
				Actor:      params.Actor,
				TenantInfo: req.TenantInfo,
			})

			return err
		},
		plan: func(
			ctx context.Context,
			req *insightports.RestoreInsightRequest,
			_ *agentports.ToolExecuteParams,
		) (*kernelports.RecordChange[insight.Insight], error) {
			return insights.PlanRestore(ctx, *req)
		},
		refused: func(*insightports.RestoreInsightRequest) string {
			return "Would restore a dismissed insight."
		},
		render: func(
			_ *insightports.RestoreInsightRequest,
			change *kernelports.RecordChange[insight.Insight],
		) (*agent.ToolPreview, error) {
			built, err := toolpreview.Changed(toolpreview.Record{
				Resource: permission.ResourceInsight,
				ID:       change.Before.ID,
				Label:    change.Before.Headline,
				Version:  pinnedVersion(change.Before.Version),
			}, change.Before, change.After, toolpreview.Only(dismissedInsightFields...))
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would put the insight %q back on the active list.", change.Before.Headline,
			), built), nil
		},
		run: func(
			ctx context.Context,
			req *insightports.RestoreInsightRequest,
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			restored, err := insights.Restore(ctx, *req)
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: "restored",
				Kind:   "insight",
				Name:   restored.Headline,
				IDs:    map[string]string{paramInsightID: restored.ID.String()},
			}, nil
		},
	})
}
