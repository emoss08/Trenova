package insighttools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/timeutils"
)

var dismissedInsightFields = []string{changepreview.FieldStatus, "dismissReason", "dismissedAt"}

func (t *DismissInsightTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	detail, err := t.insights.GetDetail(ctx, insightports.GetInsightDetailRequest{
		ID:         request.ID,
		UserID:     request.UserID,
		Actor:      params.Actor,
		TenantInfo: request.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	finding := detail.Insight
	now := timeutils.NowUnix()
	plan, err := changepreview.PlanArchive(toolpreview.Record{
		Resource: permission.ResourceInsight,
		ID:       finding.ID,
		Label:    finding.Headline,
		Version:  changepreview.PreviewVersion(finding.Version),
	}, finding, func(dismissed *insight.Insight) error {
		return dismissed.Dismiss(request.UserID, request.Reason, now)
	}, toolpreview.Only(dismissedInsightFields...), toolpreview.Volatile("dismissedAt"))
	if err != nil {
		return nil, err
	}

	return plan.PlanPreview(fmt.Sprintf(
		"Would dismiss the insight %q. It stays on file with the reason and can be restored.",
		finding.Headline,
	)), nil
}
