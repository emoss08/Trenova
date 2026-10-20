package insighttools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxDismissReasonChars = 500

// insightDismisser is the one write these tools make on an insight.
type insightDismisser interface {
	GetDetail(
		ctx context.Context,
		req insightports.GetInsightDetailRequest,
	) (*insightports.InsightDetail, error)
	Dismiss(ctx context.Context, req insightports.DismissInsightRequest) (*insight.Insight, error)
}

// dismissInsightTool records that a finding is not worth acting on. The
// finding stays on file with the reason, a person can restore it from the
// insights page, and the detector will not raise it again for a while.
type DismissInsightTool struct {
	insights insightDismisser
}

func NewDismissInsightTool(insights insightDismisser) agentports.AgentTool {
	return &DismissInsightTool{insights: insights}
}

func (t *DismissInsightTool) Name() string { return "insight_dismiss" }

func (t *DismissInsightTool) Recipe() []string {
	return []string{"insight_list", "insight_get", "insight_dismiss"}
}

func (t *DismissInsightTool) Description() string {
	return "Dismiss an insight that is not worth acting on, with the reason a person would " +
		"want to read later. Use it for a known seasonal pattern, a customer already being " +
		"handled, a figure explained by something the detector cannot see. The finding is " +
		"kept with the reason and can be restored; the detector will not raise it again " +
		"for a while. Do not dismiss a finding because it is inconvenient or because you " +
		"could not resolve it; leave those active."
}

func (t *DismissInsightTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"insightId": agenttoolschema.RecordIDText(permission.ResourceInsight,
				"The insight's id, from insight_list or the run's subject."),
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

func (t *DismissInsightTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceInsight,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Dismisses an insight inside Trenova; it can be restored.",
	}
}

func (t *DismissInsightTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	id, err := common.RequirePulid(params, "insightId")
	if err != nil {
		return agentports.ToolTarget{}, false
	}

	return agentports.ToolTarget{Resource: permission.ResourceInsight, ID: id}, true
}

func (t *DismissInsightTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	request, err := t.request(params)
	if err != nil {
		return err
	}

	_, err = t.insights.Dismiss(ctx, request)

	return err
}

func (t *DismissInsightTool) request(
	params agentports.ToolExecuteParams,
) (insightports.DismissInsightRequest, error) {
	id, reason, err := t.arguments(params)
	if err != nil {
		return insightports.DismissInsightRequest{}, err
	}

	return insightports.DismissInsightRequest{
		ID:         id,
		UserID:     params.Actor.UserID,
		Reason:     reason,
		TenantInfo: common.TenantFrom(params),
	}, nil
}

func (t *DismissInsightTool) arguments(
	params agentports.ToolExecuteParams,
) (pulid.ID, string, error) {
	if err := common.GuardExecute(t, params); err != nil {
		return "", "", err
	}

	id, err := common.RequirePulid(params.Params, "insightId")
	if err != nil {
		return "", "", err
	}

	reason, err := common.RequireString(params.Params, "reason")
	if err != nil {
		return "", "", err
	}
	if len(reason) > maxDismissReasonChars {
		return "", "", fmt.Errorf("reason must be at most %d characters", maxDismissReasonChars)
	}

	return id, reason, nil
}
