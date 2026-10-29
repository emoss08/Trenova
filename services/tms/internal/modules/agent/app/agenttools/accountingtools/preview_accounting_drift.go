package accountingtools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/textcase"
	"github.com/emoss08/trenova/shared/timeutils"
)

const fieldDriftStatus = "status"

const fieldDriftResolution = "resolution"

const fieldDriftNote = "resolutionNote"

const fieldDriftFixObject = "fixObjectType"

const fieldDriftResolvedByID = "resolvedById"

const fieldDriftResolvedAt = "resolvedAt"

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

func driftFindingRecord(finding *accountingdrift.AccountingDriftFinding) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceAccountingSync,
		ID:       finding.ID,
		Label:    driftLabel(finding),
		Version:  changepreview.PreviewVersion(finding.Version),
	}
}

func applyDriftFix(
	finding *accountingdrift.AccountingDriftFinding,
	fix *accountingdriftports.AccountingDriftFixPreview,
	actorID pulid.ID,
	now int64,
) error {
	if err := finding.CanFix(fix.Direction); err != nil {
		return err
	}
	if fix.Direction == accountingdrift.DriftPushTrenovaValue {
		finding.MarkPushed(pulid.Nil, actorID)
		return nil
	}
	finding.MarkAdjusted(fix.FixObject, pulid.Nil, actorID, now)
	return nil
}

func (t *ResolveAccountingDriftTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	const refusedSummary = "Would fix a difference between the accounting system and Trenova."

	_, fix, err := t.plan(ctx, &params)
	if err != nil {
		return refusedSync(refusedSummary, err)
	}

	now := timeutils.NowUnix()
	plan, err := changepreview.PlanUpdate(
		driftFindingRecord(fix.Finding),
		fix.Finding,
		func(fixed *accountingdrift.AccountingDriftFinding) error {
			return applyDriftFix(fixed, fix, params.Actor.UserID, now)
		},
		driftFindingOptions()...,
	)
	if err != nil {
		return nil, err
	}

	preview := plan.PlanPreview("Would " + textcase.LowerFirst(fix.Summary))
	preview.Partial = true
	return preview, nil
}

func (t *DismissAccountingDriftTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	const refusedSummary = "Would dismiss a difference between the accounting system and Trenova."

	req, fix, err := t.plan(ctx, &params)
	if err != nil {
		return refusedSync(refusedSummary, err)
	}

	now := timeutils.NowUnix()
	plan, err := changepreview.PlanUpdate(
		driftFindingRecord(fix.Finding),
		fix.Finding,
		func(dismissed *accountingdrift.AccountingDriftFinding) error {
			return dismissed.Dismiss(params.Actor.UserID, req.Note, now)
		},
		driftFindingOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return plan.PlanPreview(
		"Would " + textcase.LowerFirst(fix.Summary) + " Reason: " + req.Note,
	), nil
}

func (t *CheckAccountingDriftTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	const refusedSummary = "Would compare the accounting system with Trenova now."

	_, overview, err := t.setup(ctx, &params)
	if err != nil {
		return refusedSync(refusedSummary, err)
	}

	preview := toolpreview.Build(
		"Would read every document Trenova sent to "+overview.ProviderName+
			" back from it now, in the background, and raise or resolve differences. "+
			"Nothing changes in the books or in Trenova.",
		&agentpreview.RecordChange{
			Resource:  permission.ResourceAccountingIntegration,
			EntityID:  overview.ConnectionID,
			Label:     overview.ProviderName,
			Operation: agentpreview.PreviewOperationRun,
		},
	)
	preview.Partial = true
	return preview, nil
}
