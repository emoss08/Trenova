package tendertools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
)

func (t *TenderToRoutingGuideTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	planned, err := t.tenders.PreviewWaterfall(ctx, request)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would offer the move down routing guide %s to %s, one at a time in this order.",
		planned.Guide.Name,
		changepreview.CountOf(len(planned.Tender.Offers), "carrier"),
	)
	if screening := planned.Screening; screening != nil {
		summary += masterdatatools.ScreeningSentence("Skipped as ineligible", screening.Skipped)
		summary += masterdatatools.ScreeningSentence("Offered despite a warning", screening.Warned)
	}

	return masterdatatools.TenderPreview(summary, planned)
}

func (t *TenderToCarriersTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	planned, err := t.tenders.PreviewSpot(ctx, request)
	if err != nil {
		return nil, err
	}

	order := "all at once"
	if request.Mode == tender.ModeSpotSequential {
		order = "one at a time in this order"
	}
	summary := fmt.Sprintf("Would offer the move to %s, %s.",
		changepreview.CountOf(len(planned.Tender.Offers), "carrier"), order)
	if len(planned.Warnings) > 0 {
		summary += " Eligibility warnings: " + strings.Join(planned.Warnings, "; ") + "."
	}

	preview, err := masterdatatools.TenderPreview(summary, planned)
	if err != nil {
		return nil, err
	}
	if planned.RefusalError != nil {
		toolpreview.Warn(preview, agent.PreviewWarningWouldFail, planned.RefusalError.Error())
	}

	return preview, nil
}
