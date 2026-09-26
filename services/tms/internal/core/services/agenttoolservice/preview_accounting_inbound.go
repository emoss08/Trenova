package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/timeutils"
)

var (
	_ serviceports.ToolPreviewer = (*applyAccountingInboundChangeTool)(nil)
	_ serviceports.ToolPreviewer = (*ignoreAccountingInboundChangeTool)(nil)
)

const (
	fieldInboundNote        = "note"
	fieldInboundDecidedByID = "decidedById"
	fieldInboundDecidedAt   = "decidedAt"
)

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
		Version:  previewVersion(change.Version),
	}
}

func inboundPostingSummary(preview *serviceports.AccountingInboundApplyPreview) string {
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

func (t *applyAccountingInboundChangeTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	const refusedSummary = "Would bring a payment from the accounting system into Trenova."

	plan, err := t.preview(ctx, &params)
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

func (t *ignoreAccountingInboundChangeTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	const refusedSummary = "Would keep a payment from the accounting system out of Trenova."

	req, change, err := t.request(ctx, &params)
	if err != nil {
		return refusedSync(refusedSummary, err)
	}

	now := timeutils.NowUnix()
	plan, err := planUpdate(
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

	return plan.preview(fmt.Sprintf(
		"Would leave %s for %s out of Trenova for good: %s. It is never brought in, so the "+
			"documents it pays stay open here unless someone records the payment by hand.",
		inboundLabel(change),
		money.FormatMinor(change.AmountMinor, change.CurrencyCode),
		req.Note,
	)), nil
}
