package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxDismissReasonChars = 500

// insightDismisser is the one write these tools make on an insight.
type insightDismisser interface {
	Dismiss(ctx context.Context, req serviceports.DismissInsightRequest) (*insight.Insight, error)
}

// dismissInsightTool records that a finding is not worth acting on. The
// finding stays on file with the reason, a person can restore it from the
// insights page, and the detector will not raise it again for a while.
type dismissInsightTool struct {
	insights insightDismisser
}

func newDismissInsightTool(insights insightDismisser) serviceports.AgentTool {
	return &dismissInsightTool{insights: insights}
}

func (t *dismissInsightTool) Name() string { return "dismiss_insight" }

func (t *dismissInsightTool) Description() string {
	return "Dismiss an insight that is not worth acting on: a known seasonal pattern, a " +
		"customer already being handled, a figure explained by something the detector " +
		"cannot see. Give the reason a person would want to read later. The finding is " +
		"kept with the reason and can be restored; the detector will not raise it again " +
		"for a while. Do not dismiss a finding because it is inconvenient or because you " +
		"could not resolve it; leave those active."
}

func (t *dismissInsightTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"insightId": map[string]any{
				"type":        "string",
				"description": "The insight's id, from list_insights or the run's subject.",
			},
			"reason": map[string]any{
				"type": "string",
				"description": fmt.Sprintf(
					"Why it is not worth acting on, in one or two sentences, at most %d characters.",
					maxDismissReasonChars,
				),
			},
		},
		"required":             []string{"insightId", "reason"},
		"additionalProperties": false,
	}
}

func (t *dismissInsightTool) Reversible() bool { return true }

func (t *dismissInsightTool) PermissionResource() permission.Resource {
	return permission.ResourceInsight
}

func (t *dismissInsightTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *dismissInsightTool) RequiresIdempotencyKey() bool { return false }

func (t *dismissInsightTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *dismissInsightTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	id, err := requirePulid(params, "insightId")
	if err != nil {
		return serviceports.ToolTarget{}, false
	}

	return serviceports.ToolTarget{Resource: permission.ResourceInsight, ID: id}, true
}

func (t *dismissInsightTool) Execute(ctx context.Context, params serviceports.ToolExecuteParams) error {
	id, reason, err := t.arguments(params)
	if err != nil {
		return err
	}

	_, err = t.insights.Dismiss(ctx, serviceports.DismissInsightRequest{
		ID:         id,
		UserID:     params.Actor.UserID,
		Reason:     reason,
		TenantInfo: tenantFrom(params),
	})

	return err
}

func (t *dismissInsightTool) Simulate(
	_ context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	id, reason, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	return &agent.ToolSimulation{
		Summary:   fmt.Sprintf("Would dismiss insight %s.", id),
		Previewed: true,
		Changes: []agent.FieldChange{
			{Field: "status", From: string(insight.StatusActive), To: string(insight.StatusDismissed)},
			{Field: "dismissReason", From: "", To: reason},
		},
	}, nil
}

func (t *dismissInsightTool) arguments(
	params serviceports.ToolExecuteParams,
) (pulid.ID, string, error) {
	if err := guardExecute(t, params); err != nil {
		return "", "", err
	}

	id, err := requirePulid(params.Params, "insightId")
	if err != nil {
		return "", "", err
	}

	reason, err := requireString(params.Params, "reason")
	if err != nil {
		return "", "", err
	}
	if len(reason) > maxDismissReasonChars {
		return "", "", fmt.Errorf("reason must be at most %d characters", maxDismissReasonChars)
	}

	return id, reason, nil
}
