package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	paramMappingProposals = "proposals"
	paramExternalID       = "externalId"
	paramBackfillAction   = "action"
	maxMappingProposals   = 50
	maxSyncReleaseIDs     = 50
)

var backfillActions = []serviceports.AccountingBackfillAction{
	serviceports.AccountingBackfillPause,
	serviceports.AccountingBackfillResume,
	serviceports.AccountingBackfillCancel,
}

type accountingMappingReviewer interface {
	PlanConfirm(
		ctx context.Context,
		req *serviceports.ConfirmAccountingMappingsRequest,
	) (*serviceports.AccountingMappingConfirmPlan, error)
	Confirm(
		ctx context.Context,
		req *serviceports.ConfirmAccountingMappingsRequest,
	) ([]*accountingsync.AccountingMapping, error)
	PlanReject(
		ctx context.Context,
		req *serviceports.AccountingMappingActionRequest,
	) (*serviceports.AccountingMappingChange, error)
	Reject(
		ctx context.Context,
		req *serviceports.AccountingMappingActionRequest,
	) (*accountingsync.AccountingMapping, error)
}

type accountingSyncReleaser interface {
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) (*serviceports.AccountingSyncSummary, error)
	GetRecord(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*accountingsync.AccountingSyncRecord, error)
	Release(ctx context.Context, req *serviceports.ReleaseAccountingSyncRequest) (int64, error)
}

type accountingBackfillChanger interface {
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) (*serviceports.AccountingSyncSummary, error)
	ChangeBackfill(
		ctx context.Context,
		req *serviceports.ChangeAccountingBackfillRequest,
	) (*accountingsync.AccountingBackfill, error)
}

type releasePlan struct {
	req      *serviceports.ReleaseAccountingSyncRequest
	provider string
	records  []*accountingsync.AccountingSyncRecord
}

type backfillChange struct {
	req      *serviceports.ChangeAccountingBackfillRequest
	provider string
	before   *accountingsync.AccountingBackfill
	after    *accountingsync.AccountingBackfill
}

func mappingProposalsProperty() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeArray,
		toolschema.KeyDescription: "The proposals to confirm, each as it was shown: the " +
			"mapping and the record Trenova proposed for it.",
		toolschema.KeyMinItems: 1,
		toolschema.KeyMaxItems: maxMappingProposals,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				paramMappingID: stringProperty("The mapping's id, from "+
					"list_accounting_mapping_gaps or get_accounting_mapping.", 0),
				paramExternalID: stringProperty("The proposed record's externalId, as "+
					"list_accounting_mapping_gaps shows it under mappedTo.", 0),
			},
			toolschema.KeyRequired:             []string{paramMappingID, paramExternalID},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func confirmMappingsRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.ConfirmAccountingMappingsRequest, error) {
	raw, ok := params.Params[paramMappingProposals].([]any)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf(
			"parameter %q must list at least one proposal",
			paramMappingProposals,
		)
	}
	if len(raw) > maxMappingProposals {
		return nil, fmt.Errorf(
			"parameter %q holds %d proposals; confirm at most %d at once",
			paramMappingProposals, len(raw), maxMappingProposals,
		)
	}

	items := make([]serviceports.AccountingMappingConfirmation, 0, len(raw))
	seen := make(map[pulid.ID]struct{}, len(raw))
	for idx, item := range raw {
		fields, isObject := item.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("%s[%d] must be an object", paramMappingProposals, idx)
		}
		id, err := requirePulid(fields, paramMappingID)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramMappingProposals, idx, err)
		}
		if _, dup := seen[id]; dup {
			return nil, fmt.Errorf("%s[%d] repeats mapping %s", paramMappingProposals, idx, id)
		}
		seen[id] = struct{}{}
		externalID, err := requireString(fields, paramExternalID)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramMappingProposals, idx, err)
		}
		items = append(items, serviceports.AccountingMappingConfirmation{
			ID:         id,
			ExternalID: strings.TrimSpace(externalID),
		})
	}

	return &serviceports.ConfirmAccountingMappingsRequest{
		TenantInfo: tenantFrom(*params),
		UserID:     params.Actor.UserID,
		Items:      items,
		Source:     accountingsync.MappingSourceAgent,
	}, nil
}

func mappingReviewSpec(spec *receivableSpec) *receivableSpec {
	spec.resource = permission.ResourceAccountingIntegration
	spec.operation = permission.OpUpdate
	spec.egress = agent.EgressInternal
	spec.defaultTier = agent.TierPropose
	spec.maxTier = agent.TierActWithApproval
	spec.reversible = true

	return spec
}

func newConfirmAccountingMappingProposalsTool(
	mappings accountingMappingReviewer,
) serviceports.AgentTool {
	return newReceivableTool(mappingReviewSpec(&receivableSpec{
		name: "confirm_accounting_mapping_proposals",
		description: "Confirm Trenova's proposed accounting system matches exactly as " +
			"list_accounting_mapping_gaps shows them, several at once, so sync can use them. " +
			"A proposal that changed since it was read is refused; to choose a different " +
			"record, use set_accounting_mapping.",
		rationale: "Decides which accounts and records Trenova's documents post to in the " +
			"books, so a person approves it; clearing or changing a mapping undoes it.",
		properties: map[string]any{paramMappingProposals: mappingProposalsProperty()},
		required:   []string{paramMappingProposals},
	}), receivablePlan[*serviceports.ConfirmAccountingMappingsRequest, *serviceports.AccountingMappingConfirmPlan]{
		request: confirmMappingsRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.ConfirmAccountingMappingsRequest,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.AccountingMappingConfirmPlan, error) {
			return mappings.PlanConfirm(ctx, req)
		},
		refused: func(req *serviceports.ConfirmAccountingMappingsRequest) string {
			return fmt.Sprintf("Would confirm %s.",
				countOf(len(req.Items), "proposed accounting mapping"))
		},
		render: renderMappingConfirmations,
		run: func(
			ctx context.Context,
			req *serviceports.ConfirmAccountingMappingsRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := mappings.Confirm(ctx, req)

			return nil, err
		},
	})
}

func newRejectAccountingMappingProposalTool(
	mappings accountingMappingReviewer,
) serviceports.AgentTool {
	return newReceivableTool(mappingReviewSpec(&receivableSpec{
		name: "reject_accounting_mapping_proposal",
		description: "Turn down the wrong record Trenova proposed for an accounting mapping, " +
			"leaving it unmatched. That record is never proposed for it again. Use " +
			"set_accounting_mapping when the right record is known.",
		rationale: "Unmatches a proposal so it is not used; choosing a record for the mapping " +
			"undoes it.",
		properties: map[string]any{
			paramMappingID: stringProperty("The mapping's id, from "+
				"list_accounting_mapping_gaps or get_accounting_mapping. Never guess one.", 0),
		},
		required: []string{paramMappingID},
	}), receivablePlan[*serviceports.AccountingMappingActionRequest, *serviceports.AccountingMappingChange]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*serviceports.AccountingMappingActionRequest, error) {
			id, err := requirePulid(params.Params, paramMappingID)
			if err != nil {
				return nil, err
			}

			return &serviceports.AccountingMappingActionRequest{
				TenantInfo: tenantFrom(*params),
				UserID:     params.Actor.UserID,
				ID:         id,
				Source:     accountingsync.MappingSourceAgent,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *serviceports.AccountingMappingActionRequest,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.AccountingMappingChange, error) {
			return mappings.PlanReject(ctx, req)
		},
		refused: func(*serviceports.AccountingMappingActionRequest) string {
			return "Would turn down a proposed accounting mapping."
		},
		render: renderMappingRejection,
		run: func(
			ctx context.Context,
			req *serviceports.AccountingMappingActionRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := mappings.Reject(ctx, req)

			return nil, err
		},
	})
}

func releaseRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.ReleaseAccountingSyncRequest, error) {
	system, err := accountingSystemFrom(params.Params)
	if err != nil {
		return nil, err
	}
	ids, err := requirePulidSlice(params.Params, paramSyncRecordIDs, maxSyncReleaseIDs)
	if err != nil {
		return nil, err
	}

	return &serviceports.ReleaseAccountingSyncRequest{
		TenantInfo:      tenantFrom(*params),
		UserID:          params.Actor.UserID,
		IntegrationType: system,
		IDs:             sliceutils.Dedupe(ids),
	}, nil
}

func planRelease(
	ctx context.Context,
	sync accountingSyncReleaser,
	req *serviceports.ReleaseAccountingSyncRequest,
) (*releasePlan, error) {
	summary, err := syncingConnectionFor(ctx, sync, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}

	records := make([]*accountingsync.AccountingSyncRecord, 0, len(req.IDs))
	for _, id := range req.IDs {
		record, getErr := sync.GetRecord(ctx, req.TenantInfo, id)
		if getErr != nil {
			return nil, getErr
		}
		if record.Status != accountingsync.SyncStatusAwaitingApproval {
			return nil, errortypes.NewBusinessError(
				"{0} is {1}; only a document awaiting approval is released",
				syncDocumentLabel(record),
				string(record.Status),
			)
		}
		records = append(records, record)
	}

	return &releasePlan{req: req, provider: summary.ProviderName, records: records}, nil
}

func newReleaseAccountingSyncTool(sync accountingSyncReleaser) serviceports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name: "release_accounting_sync",
		description: "Propose releasing documents the organization holds for approval before " +
			"they go to the accounting system, so they are sent. Name each by its sync record; " +
			"the person approving may untick some. What is sent cannot be called back, so a " +
			"person always decides.",
		resource:    permission.ResourceAccountingSync,
		operation:   permission.OpUpdate,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		rationale: "Sends documents a review policy held for a person to the organization's " +
			"books; only a person releases them.",
		properties: map[string]any{
			paramAccountingSystem: accountingSystemSchema(),
			paramSyncRecordIDs: toolschema.RecordSubset(
				permission.ResourceAccountingSync.String(),
				idListProperty(fmt.Sprintf(
					"Up to %d sync records awaiting approval, from "+
						"list_accounting_sync_records or get_accounting_sync_record.",
					maxSyncReleaseIDs,
				), maxSyncReleaseIDs),
			),
		},
		required: []string{paramAccountingSystem, paramSyncRecordIDs},
	}, receivablePlan[*serviceports.ReleaseAccountingSyncRequest, *releasePlan]{
		request: releaseRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.ReleaseAccountingSyncRequest,
			_ *serviceports.ToolExecuteParams,
		) (*releasePlan, error) {
			return planRelease(ctx, sync, req)
		},
		refused: func(req *serviceports.ReleaseAccountingSyncRequest) string {
			return fmt.Sprintf("Would release %s to the accounting system.",
				countOf(len(req.IDs), "held document"))
		},
		render: renderRelease,
		run: func(
			ctx context.Context,
			req *serviceports.ReleaseAccountingSyncRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if _, err := planRelease(ctx, sync, req); err != nil {
				return nil, err
			}
			_, err := sync.Release(ctx, req)

			return nil, err
		},
	})
}

func backfillChangeRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.ChangeAccountingBackfillRequest, error) {
	if _, err := accountingSystemFrom(params.Params); err != nil {
		return nil, err
	}
	action, err := requireEnum(params.Params, paramBackfillAction, backfillActions)
	if err != nil {
		return nil, err
	}

	return &serviceports.ChangeAccountingBackfillRequest{
		TenantInfo: tenantFrom(*params),
		UserID:     params.Actor.UserID,
		Action:     action,
	}, nil
}

func planBackfillChange(
	ctx context.Context,
	sync accountingBackfillChanger,
	req *serviceports.ChangeAccountingBackfillRequest,
	params *serviceports.ToolExecuteParams,
) (*backfillChange, error) {
	if !params.Actor.IsUser() || params.Actor.UserID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"A backfill is changed by a person who manages the accounting integration",
		)
	}
	system, err := accountingSystemFrom(params.Params)
	if err != nil {
		return nil, err
	}
	summary, err := sync.Summary(ctx, req.TenantInfo, system)
	if err != nil {
		return nil, err
	}
	if summary.ActiveBackfill == nil {
		return nil, errortypes.NewBusinessError(
			"No backfill is in progress for {0}",
			summary.ProviderName,
		)
	}

	before := summary.ActiveBackfill
	after := *before
	changed := false
	switch req.Action {
	case serviceports.AccountingBackfillPause:
		changed = after.Pause()
	case serviceports.AccountingBackfillResume:
		changed = after.Resume()
	case serviceports.AccountingBackfillCancel:
		changed = after.Cancel(timeutils.NowUnix())
	}
	if !changed {
		return nil, errortypes.NewBusinessError(
			"A {0} backfill cannot be changed that way",
			strings.ToLower(string(before.Status)),
		)
	}

	req.ID = before.ID

	return &backfillChange{
		req:      req,
		provider: summary.ProviderName,
		before:   before,
		after:    &after,
	}, nil
}

func newChangeAccountingBackfillTool(sync accountingBackfillChanger) serviceports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name: "change_accounting_backfill",
		description: "Pause, resume or cancel the accounting backfill in progress when a " +
			"person asks, such as pausing it while the books are closed. Documents already " +
			"sent stay in the books; a cancelled backfill is requested again with " +
			"request_accounting_backfill.",
		resource:    permission.ResourceAccountingIntegration,
		operation:   permission.OpManage,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierActWithApproval,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Changes how much history reaches the organization's books; a person who " +
			"manages the integration approves it, and pausing or cancelling sends nothing.",
		properties: map[string]any{
			paramAccountingSystem: accountingSystemSchema(),
			paramBackfillAction: enumProperty("Pause holds what is left, Resume carries on "+
				"from where it stopped, Cancel ends it for good.", backfillActions),
		},
		required: []string{paramAccountingSystem, paramBackfillAction},
	}, receivablePlan[*serviceports.ChangeAccountingBackfillRequest, *backfillChange]{
		request: backfillChangeRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.ChangeAccountingBackfillRequest,
			params *serviceports.ToolExecuteParams,
		) (*backfillChange, error) {
			return planBackfillChange(ctx, sync, req, params)
		},
		refused: func(req *serviceports.ChangeAccountingBackfillRequest) string {
			return fmt.Sprintf("Would %s the accounting backfill.",
				strings.ToLower(string(req.Action)))
		},
		render: renderBackfillChange,
		run: func(
			ctx context.Context,
			req *serviceports.ChangeAccountingBackfillRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			change, err := planBackfillChange(ctx, sync, req, params)
			if err != nil {
				return nil, err
			}
			_, err = sync.ChangeBackfill(ctx, change.req)

			return nil, err
		},
	})
}

func renderMappingConfirmations(
	req *serviceports.ConfirmAccountingMappingsRequest,
	plan *serviceports.AccountingMappingConfirmPlan,
) (*agent.ToolPreview, error) {
	changes := make([]*agent.RecordChange, 0, len(plan.Changes))
	for _, change := range plan.Changes {
		recorded, err := toolpreview.Changed(
			mappingRecord(change.Before),
			change.Before,
			change.After,
			mappingOptions()...,
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, recorded)
	}

	summary := fmt.Sprintf(
		"Would confirm %s as proposed, so sync sends documents with them.",
		countOf(len(plan.Changes), "accounting mapping"),
	)
	if already := len(req.Items) - len(plan.Changes); already > 0 {
		summary += fmt.Sprintf(" %s already confirmed.", countOf(already, "was"))
	}

	return toolpreview.Build(summary, changes...), nil
}

func renderMappingRejection(
	_ *serviceports.AccountingMappingActionRequest,
	change *serviceports.AccountingMappingChange,
) (*agent.ToolPreview, error) {
	recorded, err := toolpreview.Changed(
		mappingRecord(change.Before),
		change.Before,
		change.After,
		mappingOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would turn down %s for %s; it is unmatched again and %s is never proposed for it "+
			"again.",
		mappedLabel(change.Before),
		change.Before.TargetLabel,
		mappedLabel(change.Before),
	), recorded), nil
}

func renderRelease(
	_ *serviceports.ReleaseAccountingSyncRequest,
	plan *releasePlan,
) (*agent.ToolPreview, error) {
	changes := make([]*agent.RecordChange, 0, len(plan.records))
	for _, record := range plan.records {
		change, err := toolpreview.Update(
			syncRecordRecord(record),
			record,
			func(released *accountingsync.AccountingSyncRecord) error {
				released.Status = accountingsync.SyncStatusQueued
				return nil
			},
			syncRecordOptions()...,
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would release %s held for approval to %s, which sends them to the books; what is "+
			"sent cannot be called back.",
		countOf(len(plan.records), "document"),
		plan.provider,
	), changes...), nil
}

func renderBackfillChange(
	req *serviceports.ChangeAccountingBackfillRequest,
	change *backfillChange,
) (*agent.ToolPreview, error) {
	recorded, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceAccountingIntegration,
			ID:       change.before.ID,
			Label:    change.provider + " backfill",
		},
		change.before,
		change.after,
		backfillOptions()...,
	)
	if err != nil {
		return nil, err
	}

	outcome := "what is left waits until someone resumes it"
	switch req.Action {
	case serviceports.AccountingBackfillResume:
		outcome = "it carries on sending history to the books from where it stopped"
	case serviceports.AccountingBackfillCancel:
		outcome = "what is left is never sent; documents already sent stay in the books"
	case serviceports.AccountingBackfillPause:
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would %s the %s backfill (now %s); %s.",
		strings.ToLower(string(req.Action)),
		change.provider,
		strings.ToLower(string(change.before.Status)),
		outcome,
	), recorded), nil
}
