package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/timeutils"
)

var _ serviceports.ToolPreviewer = (*dismissInsightTool)(nil)

var dismissedInsightFields = []string{fieldStatus, "dismissReason", "dismissedAt"}

func (t *dismissInsightTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	detail, err := t.insights.GetDetail(ctx, serviceports.GetInsightDetailRequest{
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
	plan, err := planArchive(toolpreview.Record{
		Resource: permission.ResourceInsight,
		ID:       finding.ID,
		Label:    finding.Headline,
		Version:  previewVersion(finding.Version),
	}, finding, func(dismissed *insight.Insight) error {
		return dismissed.Dismiss(request.UserID, request.Reason, now)
	}, toolpreview.Only(dismissedInsightFields...), toolpreview.Volatile("dismissedAt"))
	if err != nil {
		return nil, err
	}

	return plan.preview(fmt.Sprintf(
		"Would dismiss the insight %q. It stays on file with the reason and can be restored.",
		finding.Headline,
	)), nil
}
