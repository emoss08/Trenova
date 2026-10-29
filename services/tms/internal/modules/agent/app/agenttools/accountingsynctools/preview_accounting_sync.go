package accountingsynctools

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingbackfill"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const fieldAttemptCount = "attemptCount"

const fieldSkippedReason = "skippedReason"

const fieldSkippedByID = "skippedById"

const fieldPausedAt = "pausedAt"

const fieldPausedReason = "pausedReason"

const fieldPausedByID = "pausedById"

const fieldRangeStart = "rangeStart"

const fieldRangeEnd = "rangeEnd"

const fieldObjectTypes = "objectTypes"

const fieldRequestedByID = "requestedById"

const fieldRedatedTo = "redatedTo"

const fieldRedatedByID = "redatedById"

func SyncRecordOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(
			changepreview.FieldStatus,
			fieldAttemptCount,
			fieldSkippedReason,
			fieldSkippedByID,
			fieldRedatedTo,
			fieldRedatedByID,
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldSkippedByID: permission.ResourceUser,
			fieldRedatedByID: permission.ResourceUser,
		}),
		toolpreview.Labels(map[string]string{
			fieldAttemptCount:  "Attempts",
			fieldSkippedReason: "Why it is skipped",
			fieldSkippedByID:   "Skipped by",
			fieldRedatedTo:     "Sent dated",
			fieldRedatedByID:   "Re-dated by",
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

func BackfillOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(
			fieldRangeStart,
			fieldRangeEnd,
			fieldObjectTypes,
			changepreview.FieldStatus,
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

func SyncRecordRecord(record *accountingsync.AccountingSyncRecord) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceAccountingSync,
		ID:       record.ID,
		Label:    SyncDocumentLabel(record),
		Version:  changepreview.PreviewVersion(record.Version),
	}
}

func syncConnectionRecord(summary *accountingsyncports.AccountingSyncSummary) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceAccountingIntegration,
		ID:       summary.Connection.ID,
		Label:    summary.ProviderName,
		Version:  changepreview.PreviewVersion(summary.Connection.Version),
	}
}

func (t *RetryAccountingSyncTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	plan, err := t.plan(ctx, params)
	if err != nil {
		return accountingtools.RefusedSync(
			"Would send documents to the accounting system again.",
			err,
		)
	}

	if len(plan.req.IDs) == 0 {
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
			joinCategories(plan.req.ErrorCategories),
			waiting,
		))
		preview.Partial = true

		return preview, nil
	}

	now := timeutils.NowUnix()
	changes := make([]*agentpreview.RecordChange, 0, len(plan.records))
	for _, record := range plan.records {
		change, cErr := toolpreview.Update(
			SyncRecordRecord(record),
			record,
			func(retried *accountingsync.AccountingSyncRecord) error {
				return retried.Retry(now)
			},
			SyncRecordOptions()...,
		)
		if cErr != nil {
			return accountingtools.RefusedSync(
				"Would send documents to the accounting system again.",
				cErr,
			)
		}
		changes = append(changes, change)
	}

	summary := fmt.Sprintf(
		"Would send %s to %s again; the accounting system recognizes a repeat, so nothing "+
			"is entered twice.",
		changepreview.CountOf(len(plan.records), "document"),
		plan.summary.ProviderName,
	)
	left, exact := 0, true
	if len(plan.req.ErrorCategories) > 0 {
		waiting, waitingExact, wErr := t.waitingCount(ctx, plan.req)
		if wErr != nil {
			return nil, wErr
		}
		left, exact = max(waiting-len(plan.records), 0), waitingExact
	}
	if left > 0 {
		leaves := changepreview.CountOf(left, "other record")
		if !exact {
			leaves = "about " + leaves
		}
		summary += fmt.Sprintf(
			" It leaves %s that last failed as %s for another retry.",
			leaves,
			joinCategories(plan.req.ErrorCategories),
		)
	}
	preview := toolpreview.Build(summary, changes...)
	if left > 0 {
		preview.Partial = true
	}

	return preview, nil
}

func (t *SkipAccountingSyncTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	req, record, err := t.request(ctx, params)
	if err != nil {
		return accountingtools.RefusedSync(
			"Would keep a document out of the accounting system.",
			err,
		)
	}

	plan, err := changepreview.PlanUpdate(
		SyncRecordRecord(record),
		record,
		func(skipped *accountingsync.AccountingSyncRecord) error {
			return skipped.Skip(req.UserID, req.Reason)
		},
		SyncRecordOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return plan.PlanPreview(fmt.Sprintf(
		"Would keep %s out of the accounting system for good: %s. Nothing sends it again, "+
			"so the books will not have it unless someone enters it there.",
		SyncDocumentLabel(record),
		req.Reason,
	)), nil
}

func (t *redateAccountingSyncTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	req, redatePlan, err := t.request(ctx, params)
	if err != nil {
		return accountingtools.RefusedSync(
			"Would send a document to the accounting system dated on the first open day.",
			err,
		)
	}

	now := timeutils.NowUnix()
	plan, err := changepreview.PlanUpdate(
		SyncRecordRecord(redatePlan.Before),
		redatePlan.Before,
		func(redated *accountingsync.AccountingSyncRecord) error {
			return redated.Redate(req.UserID, redatePlan.SentDate, now)
		},
		SyncRecordOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return plan.PlanPreview(fmt.Sprintf(
		"Would send %s to the accounting system dated %s, the first open day after its books "+
			"closed. Trenova keeps the document's own date, and the note in the books names it.",
		SyncDocumentLabel(redatePlan.Before),
		redatePlan.SentDay,
	)), nil
}

func (t *PauseAccountingSyncTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	req, summary, err := t.request(ctx, params)
	if err != nil {
		return accountingtools.RefusedSync("Would pause sending to the accounting system.", err)
	}

	now := timeutils.NowUnix()
	change, err := toolpreview.Update(
		syncConnectionRecord(summary),
		summary.Connection,
		func(paused *accountingconnection.AccountingConnection) error {
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
		changepreview.CountOf(waitingToSend(summary), "document"),
	), change), nil
}

func (t *ResumeAccountingSyncTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	_, summary, err := t.request(ctx, params)
	if err != nil {
		return accountingtools.RefusedSync("Would resume sending to the accounting system.", err)
	}

	change, err := toolpreview.Update(
		syncConnectionRecord(summary),
		summary.Connection,
		func(resumed *accountingconnection.AccountingConnection) error {
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
		changepreview.CountOf(waitingToSend(summary), "document"),
		summary.Connection.PausedReason,
	), change), nil
}

func waitingToSend(summary *accountingsyncports.AccountingSyncSummary) int {
	return countWithStatus(
		summary,
		accountingsync.SyncStatusQueued,
		accountingsync.SyncStatusRetrying,
	)
}

func (p *backfillPlan) backfill() *accountingbackfill.AccountingBackfill {
	return accountingbackfill.NewAccountingBackfill(&accountingbackfill.NewBackfillParams{
		TenantInfo:    p.req.TenantInfo,
		ConnectionID:  p.connection.ID,
		RangeStart:    p.from,
		RangeEnd:      p.to,
		ObjectTypes:   p.types,
		RequestedByID: p.req.UserID,
	})
}

func (t *RequestAccountingBackfillTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	plan, err := t.plan(ctx, params)
	if err != nil {
		return accountingtools.RefusedSync(
			"Would send earlier documents to the accounting system.",
			err,
		)
	}

	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceAccountingIntegration,
		Label:    "Backfill to " + plan.provider,
	}, plan.backfill(), BackfillOptions()...)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would send every %s dated %s to %s that Trenova has not already sent to %s. "+
			"Anything already entered there by hand would arrive twice, and nothing sent can "+
			"be called back.",
		strings.Join(sliceutils.Strings(plan.types), ", "),
		changepreview.DayLabel(plan.from),
		changepreview.DayLabel(plan.to),
		plan.provider,
	), change), nil
}
