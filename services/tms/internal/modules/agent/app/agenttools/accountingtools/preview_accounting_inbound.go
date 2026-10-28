package accountingtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/timeutils"
)

const fieldInboundNote = "note"

const fieldInboundDecidedByID = "decidedById"

const fieldInboundDecidedAt = "decidedAt"

func inboundChangeOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(
			fieldInboundStatus,
			fieldInboundNote,
			fieldInboundDecidedByID,
			fieldInboundDecidedAt,
		),
		toolpreview.Volatile(fieldInboundDecidedAt),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldInboundDecidedByID: permission.ResourceUser,
		}),
		toolpreview.Labels(map[string]string{
			fieldInboundNote:        "Why it stays out of Trenova",
			fieldInboundDecidedByID: "Decided by",
			fieldInboundDecidedAt:   "Decided at",
		}),
	}
}

func inboundChangeRecord(change *accountingsync.AccountingInboundChange) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceAccountingSync,
		ID:       change.ID,
		Label:    inboundLabel(change),
		Version:  common.PreviewVersion(change.Version),
	}
}

func inboundPostingSummary(preview *accountingsyncports.AccountingInboundApplyPreview) string {
	change := preview.Change
	currency := change.CurrencyCode
	pays := make([]string, 0, len(preview.Lines))
	for _, line := range preview.Lines {
		pays = append(pays, fmt.Sprintf(
			"%s %s of %s open",
			line.ObjectNumber,
			money.FormatMinor(line.AmountMinor, currency),
			money.FormatMinor(line.OpenMinor, currency),
		))
	}

	summary := fmt.Sprintf(
		"Would post %s for %s on %s, paying %s",
		inboundLabel(change),
		money.FormatMinor(change.AmountMinor, currency),
		timeutils.FormatCalendarDate(preview.PaidAt, nil),
		strings.Join(pays, "; "),
	)
	if preview.UnappliedMinor > 0 {
		summary += fmt.Sprintf(
			", leaving %s as unapplied cash",
			money.FormatMinor(preview.UnappliedMinor, currency),
		)
	}

	return summary + "."
}

func (t *ApplyAccountingInboundChangeTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	const refusedSummary = "Would bring a payment from the accounting system into Trenova."

	plan, err := t.PlanPreview(ctx, &params)
	if err != nil {
		return refusedSync(refusedSummary, err)
	}

	now := timeutils.NowUnix()
	change, err := toolpreview.Update(
		inboundChangeRecord(plan.Change),
		plan.Change,
		func(applied *accountingsync.AccountingInboundChange) error {
			if cErr := applied.CanApply(); cErr != nil {
				return cErr
			}
			applied.MarkApplied(nil, params.Actor.UserID, "", now)

			return nil
		},
		inboundChangeOptions()...,
	)
	if err != nil {
		return refusedSync(refusedSummary, err)
	}

	preview := toolpreview.Build(inboundPostingSummary(plan), change)
	preview.Partial = true

	return preview, nil
}

func (t *IgnoreAccountingInboundChangeTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	const refusedSummary = "Would keep a payment from the accounting system out of Trenova."

	req, change, err := t.request(ctx, &params)
	if err != nil {
		return refusedSync(refusedSummary, err)
	}

	now := timeutils.NowUnix()
	plan, err := common.PlanUpdate(
		inboundChangeRecord(change),
		change,
		func(ignored *accountingsync.AccountingInboundChange) error {
			return ignored.Dismiss(params.Actor.UserID, req.Note, now)
		},
		inboundChangeOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return plan.PlanPreview(fmt.Sprintf(
		"Would leave %s for %s out of Trenova for good: %s. It is never brought in, so the "+
			"documents it pays stay open here unless someone records the payment by hand.",
		inboundLabel(change),
		money.FormatMinor(change.AmountMinor, change.CurrencyCode),
		req.Note,
	)), nil
}
