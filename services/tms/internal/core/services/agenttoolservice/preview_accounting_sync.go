package agenttoolservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

var (
	_ serviceports.ToolPreviewer = (*retryAccountingSyncTool)(nil)
	_ serviceports.ToolPreviewer = (*skipAccountingSyncTool)(nil)
	_ serviceports.ToolPreviewer = (*pauseAccountingSyncTool)(nil)
	_ serviceports.ToolPreviewer = (*resumeAccountingSyncTool)(nil)
	_ serviceports.ToolPreviewer = (*requestAccountingBackfillTool)(nil)
)

const (
	fieldAttemptCount  = "attemptCount"
	fieldSkippedReason = "skippedReason"
	fieldSkippedByID   = "skippedById"
	fieldPausedAt      = "pausedAt"
	fieldPausedReason  = "pausedReason"
	fieldPausedByID    = "pausedById"
	fieldRangeStart    = "rangeStart"
	fieldRangeEnd      = "rangeEnd"
	fieldObjectTypes   = "objectTypes"
	fieldRequestedByID = "requestedById"
)

func syncRecordOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(fieldStatus, fieldAttemptCount, fieldSkippedReason, fieldSkippedByID),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldSkippedByID: permission.ResourceUser,
		}),
		toolpreview.Labels(map[string]string{
			fieldAttemptCount:  "Attempts",
			fieldSkippedReason: "Why it is skipped",
			fieldSkippedByID:   "Skipped by",
		}),
	}
}

func syncPauseOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(fieldPausedAt, fieldPausedReason, fieldPausedByID),
		toolpreview.Volatile(fieldPausedAt),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldPausedByID: permission.ResourceUser,
		}),
		toolpreview.Labels(map[string]string{
			fieldPausedAt:     "Paused at",
			fieldPausedReason: "Why it is paused",
			fieldPausedByID:   "Paused by",
		}),
	}
}

func backfillOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(
			fieldRangeStart,
			fieldRangeEnd,
			fieldObjectTypes,
			fieldStatus,
			fieldRequestedByID,
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldRequestedByID: permission.ResourceUser,
		}),
		toolpreview.Labels(map[string]string{
			fieldRangeStart:    "Documents dated from",
			fieldRangeEnd:      "Documents dated to",
			fieldObjectTypes:   "Documents",
			fieldRequestedByID: "Requested by",
		}),
	}
}

func syncRecordRecord(record *accountingsync.AccountingSyncRecord) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceAccountingSync,
		ID:       record.ID,
		Label:    syncDocumentLabel(record),
		Version:  previewVersion(record.Version),
	}
}

func syncConnectionRecord(summary *serviceports.AccountingSyncSummary) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceAccountingIntegration,
		ID:       summary.Connection.ID,
		Label:    summary.ProviderName,
		Version:  previewVersion(summary.Connection.Version),
	}
}

func refusedSync(summary string, err error) (*agent.ToolPreview, error) {
	if isRefusal(err) {
		return warnWouldFail(toolpreview.Build(summary), err), nil
	}

	return nil, err
}

func (t *retryAccountingSyncTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return refusedSync("Would send documents to the accounting system again.", err)
	}

	if len(plan.records) == 0 {
		waiting := 0
		for _, group := range plan.summary.Attention {
			if slices.Contains(plan.req.ErrorCategories, group.ErrorCategory) {
				waiting += group.Count
			}
		}
		preview := toolpreview.Build(fmt.Sprintf(
			"Would send to %s again every record that last failed as %s; %d are blocked or "+
				"gave up that way now, plus any still retrying.",
			plan.summary.ProviderName,
			strings.Join(sliceutils.Strings(plan.req.ErrorCategories), " or "),
			waiting,
		))
		preview.Partial = true

		return preview, nil
	}

	now := timeutils.NowUnix()
	changes := make([]*agent.RecordChange, 0, len(plan.records))
	for _, record := range plan.records {
		change, cErr := toolpreview.Update(
			syncRecordRecord(record),
			record,
			func(retried *accountingsync.AccountingSyncRecord) error {
				return retried.Retry(now)
			},
			syncRecordOptions()...,
		)
		if cErr != nil {
			return refusedSync("Would send documents to the accounting system again.", cErr)
		}
		changes = append(changes, change)
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would send %s to %s again; the accounting system recognizes a repeat, so nothing "+
			"is entered twice.",
		countOf(len(plan.records), "document"),
		plan.summary.ProviderName,
	), changes...), nil
}

func (t *skipAccountingSyncTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, record, err := t.request(ctx, &params)
	if err != nil {
		return refusedSync("Would keep a document out of the accounting system.", err)
	}

	plan, err := planUpdate(
		syncRecordRecord(record),
		record,
		func(skipped *accountingsync.AccountingSyncRecord) error {
			return skipped.Skip(req.UserID, req.Reason)
		},
		syncRecordOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return plan.preview(fmt.Sprintf(
		"Would keep %s out of the accounting system for good: %s. Nothing sends it again, "+
			"so the books will not have it unless someone enters it there.",
		syncDocumentLabel(record),
		req.Reason,
	)), nil
}

func (t *pauseAccountingSyncTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, summary, err := t.request(ctx, &params)
	if err != nil {
		return refusedSync("Would pause sending to the accounting system.", err)
	}

	now := timeutils.NowUnix()
	change, err := toolpreview.Update(
		syncConnectionRecord(summary),
		summary.Connection,
		func(paused *accountingsync.AccountingConnection) error {
			paused.Pause(req.UserID, req.Reason, now)

			return nil
		},
		syncPauseOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would stop sending to %s; %s waiting now would wait until it resumes. Nothing is "+
			"lost.",
		summary.ProviderName,
		countOf(waitingToSend(summary), "document"),
	), change), nil
}

func (t *resumeAccountingSyncTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	_, summary, err := t.request(ctx, &params)
	if err != nil {
		return refusedSync("Would resume sending to the accounting system.", err)
	}

	change, err := toolpreview.Update(
		syncConnectionRecord(summary),
		summary.Connection,
		func(resumed *accountingsync.AccountingConnection) error {
			resumed.Resume()

			return nil
		},
		syncPauseOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would start sending to %s again; %s queued would go out now and cannot be called "+
			"back. It was paused because: %s",
		summary.ProviderName,
		countOf(waitingToSend(summary), "document"),
		summary.Connection.PausedReason,
	), change), nil
}

func waitingToSend(summary *serviceports.AccountingSyncSummary) int {
	return countWithStatus(
		summary,
		accountingsync.SyncStatusQueued,
		accountingsync.SyncStatusRetrying,
	)
}

func (p *backfillPlan) backfill() *accountingsync.AccountingBackfill {
	return accountingsync.NewAccountingBackfill(&accountingsync.NewBackfillParams{
		TenantInfo:    p.req.TenantInfo,
		ConnectionID:  p.connection.ID,
		RangeStart:    p.from,
		RangeEnd:      p.to,
		ObjectTypes:   p.types,
		RequestedByID: p.req.UserID,
	})
}

func (t *requestAccountingBackfillTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return refusedSync("Would send earlier documents to the accounting system.", err)
	}

	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceAccountingIntegration,
		Label:    "Backfill to " + plan.provider,
	}, plan.backfill(), backfillOptions()...)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would send every %s dated %s to %s that Trenova has not already sent to %s. "+
			"Anything already entered there by hand would arrive twice, and nothing sent can "+
			"be called back.",
		strings.Join(sliceutils.Strings(plan.types), ", "),
		dayLabel(plan.from),
		dayLabel(plan.to),
		plan.provider,
	), change), nil
}
