package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

var (
	_ agentports.ToolPreviewer = (*resolveAccountingDriftTool)(nil)
	_ agentports.ToolPreviewer = (*dismissAccountingDriftTool)(nil)
	_ agentports.ToolPreviewer = (*checkAccountingDriftTool)(nil)
)

const (
	fieldDriftStatus       = "status"
	fieldDriftResolution   = "resolution"
	fieldDriftNote         = "resolutionNote"
	fieldDriftFixObject    = "fixObjectType"
	fieldDriftResolvedByID = "resolvedById"
	fieldDriftResolvedAt   = "resolvedAt"
)

func driftFindingOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(
			fieldDriftStatus,
			fieldDriftResolution,
			fieldDriftNote,
			fieldDriftFixObject,
			fieldDriftResolvedByID,
			fieldDriftResolvedAt,
		),
		toolpreview.Volatile(fieldDriftResolvedAt),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldDriftResolvedByID: permission.ResourceUser,
		}),
		toolpreview.Labels(map[string]string{
			fieldDriftNote:         "Why both sides stay as they are",
			fieldDriftFixObject:    "Fixed with",
			fieldDriftResolvedByID: "Decided by",
			fieldDriftResolvedAt:   "Decided at",
		}),
	}
}

func driftFindingRecord(finding *accountingsync.AccountingDriftFinding) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceAccountingSync,
		ID:       finding.ID,
		Label:    driftLabel(finding),
		Version:  previewVersion(finding.Version),
	}
}

func applyDriftFix(
	finding *accountingsync.AccountingDriftFinding,
	fix *accountingsyncports.AccountingDriftFixPreview,
	actorID pulid.ID,
	now int64,
) error {
	if err := finding.CanFix(fix.Direction); err != nil {
		return err
	}
	if fix.Direction == accountingsync.DriftPushTrenovaValue {
		finding.MarkPushed(pulid.Nil, actorID)
		return nil
	}
	finding.MarkAdjusted(fix.FixObject, pulid.Nil, actorID, now)
	return nil
}

func (t *resolveAccountingDriftTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	const refusedSummary = "Would fix a difference between the accounting system and Trenova."

	_, fix, err := t.plan(ctx, &params)
	if err != nil {
		return refusedSync(refusedSummary, err)
	}

	now := timeutils.NowUnix()
	plan, err := planUpdate(
		driftFindingRecord(fix.Finding),
		fix.Finding,
		func(fixed *accountingsync.AccountingDriftFinding) error {
			return applyDriftFix(fixed, fix, params.Actor.UserID, now)
		},
		driftFindingOptions()...,
	)
	if err != nil {
		return nil, err
	}

	preview := plan.preview("Would " + stringutils.LowerFirst(fix.Summary))
	preview.Partial = true
	return preview, nil
}

func (t *dismissAccountingDriftTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	const refusedSummary = "Would dismiss a difference between the accounting system and Trenova."

	req, fix, err := t.plan(ctx, &params)
	if err != nil {
		return refusedSync(refusedSummary, err)
	}

	now := timeutils.NowUnix()
	plan, err := planUpdate(
		driftFindingRecord(fix.Finding),
		fix.Finding,
		func(dismissed *accountingsync.AccountingDriftFinding) error {
			return dismissed.Dismiss(params.Actor.UserID, req.Note, now)
		},
		driftFindingOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return plan.preview(
		"Would " + stringutils.LowerFirst(fix.Summary) + " Reason: " + req.Note,
	), nil
}

func (t *checkAccountingDriftTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	const refusedSummary = "Would compare the accounting system with Trenova now."

	_, overview, err := t.setup(ctx, &params)
	if err != nil {
		return refusedSync(refusedSummary, err)
	}

	preview := toolpreview.Build(
		"Would read every document Trenova sent to "+overview.ProviderName+
			" back from it now, in the background, and raise or resolve differences. "+
			"Nothing changes in the books or in Trenova.",
		&agent.RecordChange{
			Resource:  permission.ResourceAccountingIntegration,
			EntityID:  overview.ConnectionID,
			Label:     overview.ProviderName,
			Operation: agent.PreviewOperationRun,
		},
	)
	preview.Partial = true
	return preview, nil
}
