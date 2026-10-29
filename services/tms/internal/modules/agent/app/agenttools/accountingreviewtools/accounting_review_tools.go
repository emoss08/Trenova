package accountingreviewtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingbackfill"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingsynctools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const ParamMappingProposals = "proposals"

const ParamExternalID = "externalId"

const ParamBackfillAction = "action"

const maxMappingProposals = 50

const maxSyncReleaseIDs = 50

var backfillActions = agenttoolschema.Source(
	"accountingSync.backfillAction",
	accountingsyncports.AccountingBackfillActionValues(),
)

type accountingMappingReviewer interface {
	PlanConfirm(
		ctx context.Context,
		req *accountingmappingports.ConfirmAccountingMappingsRequest,
	) (*accountingmappingports.AccountingMappingConfirmPlan, error)
	Confirm(
		ctx context.Context,
		req *accountingmappingports.ConfirmAccountingMappingsRequest,
	) ([]*accountingmapping.AccountingMapping, error)
	PlanReject(
		ctx context.Context,
		req *accountingmappingports.AccountingMappingActionRequest,
	) (*accountingmappingports.AccountingMappingChange, error)
	Reject(
		ctx context.Context,
		req *accountingmappingports.AccountingMappingActionRequest,
	) (*accountingmapping.AccountingMapping, error)
}

type accountingSyncReleaser interface {
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) (*accountingsyncports.AccountingSyncSummary, error)
	GetRecord(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*accountingsync.AccountingSyncRecord, error)
	Release(
		ctx context.Context,
		req *accountingsyncports.ReleaseAccountingSyncRequest,
	) (int64, error)
}

type accountingBackfillChanger interface {
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) (*accountingsyncports.AccountingSyncSummary, error)
	ChangeBackfill(
		ctx context.Context,
		req *accountingsyncports.ChangeAccountingBackfillRequest,
	) (*accountingbackfill.AccountingBackfill, error)
}

type releasePlan struct {
	req      *accountingsyncports.ReleaseAccountingSyncRequest
	provider string
	records  []*accountingsync.AccountingSyncRecord
}

type backfillChange struct {
	req      *accountingsyncports.ChangeAccountingBackfillRequest
	provider string
	before   *accountingbackfill.AccountingBackfill
	after    *accountingbackfill.AccountingBackfill
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
				accountingtools.ParamMappingID: agenttoolschema.KindID("The mapping's id, from "+
					"accounting_list_mapping_gaps or accounting_get_mapping.",
					recordkind.AccountingMapping),
				ParamExternalID: toolparams.StringProperty("The proposed record's externalId, as "+
					"accounting_list_mapping_gaps shows it under mappedTo.", 0),
			},
			toolschema.KeyRequired: []string{
				accountingtools.ParamMappingID,
				ParamExternalID,
			},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func confirmMappingsRequest(
	params *agenttoolports.ToolExecuteParams,
) (*accountingmappingports.ConfirmAccountingMappingsRequest, error) {
	raw, ok := params.Params[ParamMappingProposals].([]any)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf(
			"parameter %q must list at least one proposal",
			ParamMappingProposals,
		)
	}
	if len(raw) > maxMappingProposals {
		return nil, fmt.Errorf(
			"parameter %q holds %d proposals; confirm at most %d at once",
			ParamMappingProposals, len(raw), maxMappingProposals,
		)
	}

	items := make([]accountingmappingports.AccountingMappingConfirmation, 0, len(raw))
	seen := make(map[pulid.ID]struct{}, len(raw))
	for idx, item := range raw {
		fields, isObject := item.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("%s[%d] must be an object", ParamMappingProposals, idx)
		}
		id, err := toolparams.RequirePulid(fields, accountingtools.ParamMappingID)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", ParamMappingProposals, idx, err)
		}
		if _, dup := seen[id]; dup {
			return nil, fmt.Errorf("%s[%d] repeats mapping %s", ParamMappingProposals, idx, id)
		}
		seen[id] = struct{}{}
		externalID, err := toolparams.RequireString(fields, ParamExternalID)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", ParamMappingProposals, idx, err)
		}
		items = append(items, accountingmappingports.AccountingMappingConfirmation{
			ID:         id,
			ExternalID: strings.TrimSpace(externalID),
		})
	}

	return &accountingmappingports.ConfirmAccountingMappingsRequest{
		TenantInfo: toolparams.TenantFrom(*params),
		UserID:     params.Actor.UserID,
		Items:      items,
		Source:     accountingmapping.MappingSourceAgent,
	}, nil
}

func mappingReviewSpec(spec *toolspec.ReceivableSpec) *toolspec.ReceivableSpec {
	spec.Resource = permission.ResourceAccountingIntegration
	spec.Operation = permission.OpUpdate
	spec.Egress = agent.EgressInternal
	spec.DefaultTier = agent.TierPropose
	spec.MaxTier = agent.TierActWithApproval
	spec.Reversible = true

	return spec
}

func NewConfirmAccountingMappingProposalsTool(
	mappings accountingMappingReviewer,
) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(mappingReviewSpec(&toolspec.ReceivableSpec{
		Name:   "accounting_confirm_mapping",
		Recipe: []string{accountingtools.ToolListMappingGaps, "accounting_confirm_mapping"},
		Description: "Confirm Trenova's proposed accounting system matches exactly as " +
			"accounting_list_mapping_gaps shows them, several at once, so sync can use them. " +
			"A proposal that changed since it was read is refused; to choose a different " +
			"record, use accounting_set_mapping.",
		Rationale: "Decides which accounts and records Trenova's documents post to in the " +
			"books, so a person approves it; clearing or changing a mapping undoes it.",
		Properties: map[string]any{ParamMappingProposals: mappingProposalsProperty()},
		Required:   []string{ParamMappingProposals},
	}), toolspec.ReceivablePlan[*accountingmappingports.ConfirmAccountingMappingsRequest, *accountingmappingports.AccountingMappingConfirmPlan]{
		Request: confirmMappingsRequest,
		Plan: func(
			ctx context.Context,
			req *accountingmappingports.ConfirmAccountingMappingsRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*accountingmappingports.AccountingMappingConfirmPlan, error) {
			return mappings.PlanConfirm(ctx, req)
		},
		Refused: func(req *accountingmappingports.ConfirmAccountingMappingsRequest) string {
			return fmt.Sprintf("Would confirm %s.",
				changepreview.CountOf(len(req.Items), "proposed accounting mapping"))
		},
		Render: renderMappingConfirmations,
		Run: func(
			ctx context.Context,
			req *accountingmappingports.ConfirmAccountingMappingsRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := mappings.Confirm(ctx, req)

			return nil, err
		},
	})
}

func NewRejectAccountingMappingProposalTool(
	mappings accountingMappingReviewer,
) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(mappingReviewSpec(&toolspec.ReceivableSpec{
		Name: "accounting_reject_mapping",
		Description: "Turn down the wrong record Trenova proposed for an accounting mapping, " +
			"leaving it unmatched. That record is never proposed for it again. Use " +
			"accounting_set_mapping when the right record is known.",
		Rationale: "Unmatches a proposal so it is not used; choosing a record for the mapping " +
			"undoes it.",
		Properties: map[string]any{
			accountingtools.ParamMappingID: agenttoolschema.KindID("The mapping's id, from "+
				"accounting_list_mapping_gaps or accounting_get_mapping. Never guess one.",
				recordkind.AccountingMapping),
		},
		Required: []string{accountingtools.ParamMappingID},
	}), toolspec.ReceivablePlan[*accountingmappingports.AccountingMappingActionRequest, *accountingmappingports.AccountingMappingChange]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*accountingmappingports.AccountingMappingActionRequest, error) {
			id, err := toolparams.RequirePulid(params.Params, accountingtools.ParamMappingID)
			if err != nil {
				return nil, err
			}

			return &accountingmappingports.AccountingMappingActionRequest{
				TenantInfo: toolparams.TenantFrom(*params),
				UserID:     params.Actor.UserID,
				ID:         id,
				Source:     accountingmapping.MappingSourceAgent,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *accountingmappingports.AccountingMappingActionRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*accountingmappingports.AccountingMappingChange, error) {
			return mappings.PlanReject(ctx, req)
		},
		Refused: func(*accountingmappingports.AccountingMappingActionRequest) string {
			return "Would turn down a proposed accounting mapping."
		},
		Render: renderMappingRejection,
		Run: func(
			ctx context.Context,
			req *accountingmappingports.AccountingMappingActionRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := mappings.Reject(ctx, req)

			return nil, err
		},
	})
}

func releaseRequest(
	params *agenttoolports.ToolExecuteParams,
) (*accountingsyncports.ReleaseAccountingSyncRequest, error) {
	system, err := accountingtools.SystemFrom(params.Params)
	if err != nil {
		return nil, err
	}
	ids, err := toolparams.RequirePulidSlice(
		params.Params,
		accountingsynctools.ParamSyncRecordIDs,
		maxSyncReleaseIDs,
	)
	if err != nil {
		return nil, err
	}

	return &accountingsyncports.ReleaseAccountingSyncRequest{
		TenantInfo:      toolparams.TenantFrom(*params),
		UserID:          params.Actor.UserID,
		IntegrationType: system,
		IDs:             sliceutils.Dedupe(ids),
	}, nil
}

func planRelease(
	ctx context.Context,
	sync accountingSyncReleaser,
	req *accountingsyncports.ReleaseAccountingSyncRequest,
) (*releasePlan, error) {
	summary, err := accountingsynctools.SyncingConnectionFor(
		ctx,
		sync,
		req.TenantInfo,
		req.IntegrationType,
	)
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
				accountingsynctools.SyncDocumentLabel(record),
				string(record.Status),
			)
		}
		records = append(records, record)
	}

	return &releasePlan{req: req, provider: summary.ProviderName, records: records}, nil
}

func NewReleaseAccountingSyncTool(sync accountingSyncReleaser) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "accounting_release_sync_record",
		Recipe: []string{
			"accounting_list_sync_records",
			"accounting_get_sync_record",
			"accounting_release_sync_record",
		},
		Description: "Propose releasing documents the organization holds for approval before " +
			"they go to the accounting system, so they are sent. Name each by its sync record; " +
			"the person approving may untick some. What is sent cannot be called back, so a " +
			"person always decides.",
		Resource:    permission.ResourceAccountingSync,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Rationale: "Sends documents a review policy held for a person to the organization's " +
			"books; only a person releases them.",
		Properties: map[string]any{
			accountingtools.ParamAccountingSystem: accountingtools.SystemSchema(),
			accountingsynctools.ParamSyncRecordIDs: toolschema.RecordSubset(
				permission.ResourceAccountingSync.String(),
				agenttoolschema.KindIDs(fmt.Sprintf(
					"Up to %d sync records awaiting approval, from "+
						"accounting_list_sync_records or accounting_get_sync_record.",
					maxSyncReleaseIDs,
				), maxSyncReleaseIDs, recordkind.AccountingSyncRecord),
			),
		},
		Required: []string{
			accountingtools.ParamAccountingSystem,
			accountingsynctools.ParamSyncRecordIDs,
		},
	}, toolspec.ReceivablePlan[*accountingsyncports.ReleaseAccountingSyncRequest, *releasePlan]{
		Request: releaseRequest,
		Plan: func(
			ctx context.Context,
			req *accountingsyncports.ReleaseAccountingSyncRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*releasePlan, error) {
			return planRelease(ctx, sync, req)
		},
		Refused: func(req *accountingsyncports.ReleaseAccountingSyncRequest) string {
			return fmt.Sprintf("Would release %s to the accounting system.",
				changepreview.CountOf(len(req.IDs), "held document"))
		},
		Render: renderRelease,
		Run: func(
			ctx context.Context,
			req *accountingsyncports.ReleaseAccountingSyncRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			if _, err := planRelease(ctx, sync, req); err != nil {
				return nil, err
			}
			_, err := sync.Release(ctx, req)

			return nil, err
		},
	})
}

func backfillChangeRequest(
	params *agenttoolports.ToolExecuteParams,
) (*accountingsyncports.ChangeAccountingBackfillRequest, error) {
	if _, err := accountingtools.SystemFrom(params.Params); err != nil {
		return nil, err
	}
	action, err := toolparams.RequireEnum(
		params.Params,
		ParamBackfillAction,
		backfillActions.Values,
	)
	if err != nil {
		return nil, err
	}

	return &accountingsyncports.ChangeAccountingBackfillRequest{
		TenantInfo: toolparams.TenantFrom(*params),
		UserID:     params.Actor.UserID,
		Action:     action,
	}, nil
}

func planBackfillChange(
	ctx context.Context,
	sync accountingBackfillChanger,
	req *accountingsyncports.ChangeAccountingBackfillRequest,
	params *agenttoolports.ToolExecuteParams,
) (*backfillChange, error) {
	if !params.Actor.IsUser() || params.Actor.UserID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"A backfill is changed by a person who manages the accounting integration",
		)
	}
	system, err := accountingtools.SystemFrom(params.Params)
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
	case accountingsyncports.AccountingBackfillPause:
		changed = after.Pause()
	case accountingsyncports.AccountingBackfillResume:
		changed = after.Resume()
	case accountingsyncports.AccountingBackfillCancel:
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

func NewChangeAccountingBackfillTool(sync accountingBackfillChanger) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "accounting_update_backfill",
		Description: "Pause, resume or cancel the accounting backfill in progress when a " +
			"person asks, such as pausing it while the books are closed. Documents already " +
			"sent stay in the books; a cancelled backfill is requested again with " +
			"accounting_request_backfill.",
		Resource:    permission.ResourceAccountingIntegration,
		Operation:   permission.OpManage,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierActWithApproval,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Changes how much history reaches the organization's books; a person who " +
			"manages the integration approves it, and pausing or cancelling sends nothing.",
		Properties: map[string]any{
			accountingtools.ParamAccountingSystem: accountingtools.SystemSchema(),
			ParamBackfillAction: agenttoolschema.Enum("Pause holds what is left, Resume "+
				"carries on from where it stopped, Cancel ends it for good.", backfillActions),
		},
		Required: []string{accountingtools.ParamAccountingSystem, ParamBackfillAction},
	}, toolspec.ReceivablePlan[*accountingsyncports.ChangeAccountingBackfillRequest, *backfillChange]{
		Request: backfillChangeRequest,
		Plan: func(
			ctx context.Context,
			req *accountingsyncports.ChangeAccountingBackfillRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*backfillChange, error) {
			return planBackfillChange(ctx, sync, req, params)
		},
		Refused: func(req *accountingsyncports.ChangeAccountingBackfillRequest) string {
			return fmt.Sprintf("Would %s the accounting backfill.",
				strings.ToLower(string(req.Action)))
		},
		Render: renderBackfillChange,
		Run: func(
			ctx context.Context,
			req *accountingsyncports.ChangeAccountingBackfillRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
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
	req *accountingmappingports.ConfirmAccountingMappingsRequest,
	plan *accountingmappingports.AccountingMappingConfirmPlan,
) (*agentpreview.ToolPreview, error) {
	changes := make([]*agentpreview.RecordChange, 0, len(plan.Changes))
	for _, change := range plan.Changes {
		recorded, err := toolpreview.Changed(
			accountingtools.MappingRecord(change.Before),
			change.Before,
			change.After,
			accountingtools.MappingOptions()...,
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, recorded)
	}

	summary := fmt.Sprintf(
		"Would confirm %s as proposed, so sync sends documents with them.",
		changepreview.CountOf(len(plan.Changes), "accounting mapping"),
	)
	if already := len(req.Items) - len(plan.Changes); already > 0 {
		summary += fmt.Sprintf(" %s already confirmed.", changepreview.CountOf(already, "was"))
	}

	return toolpreview.Build(summary, changes...), nil
}

func renderMappingRejection(
	_ *accountingmappingports.AccountingMappingActionRequest,
	change *accountingmappingports.AccountingMappingChange,
) (*agentpreview.ToolPreview, error) {
	recorded, err := toolpreview.Changed(
		accountingtools.MappingRecord(change.Before),
		change.Before,
		change.After,
		accountingtools.MappingOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would turn down %s for %s; it is unmatched again and %s is never proposed for it "+
			"again.",
		accountingtools.MappedLabel(change.Before),
		change.Before.TargetLabel,
		accountingtools.MappedLabel(change.Before),
	), recorded), nil
}

func renderRelease(
	_ *accountingsyncports.ReleaseAccountingSyncRequest,
	plan *releasePlan,
) (*agentpreview.ToolPreview, error) {
	changes := make([]*agentpreview.RecordChange, 0, len(plan.records))
	for _, record := range plan.records {
		change, err := toolpreview.Update(
			accountingsynctools.SyncRecordRecord(record),
			record,
			func(released *accountingsync.AccountingSyncRecord) error {
				released.Status = accountingsync.SyncStatusQueued
				return nil
			},
			accountingsynctools.SyncRecordOptions()...,
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would release %s held for approval to %s, which sends them to the books; what is "+
			"sent cannot be called back.",
		changepreview.CountOf(len(plan.records), "document"),
		plan.provider,
	), changes...), nil
}

func renderBackfillChange(
	req *accountingsyncports.ChangeAccountingBackfillRequest,
	change *backfillChange,
) (*agentpreview.ToolPreview, error) {
	recorded, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceAccountingIntegration,
			ID:       change.before.ID,
			Label:    change.provider + " backfill",
		},
		change.before,
		change.after,
		accountingsynctools.BackfillOptions()...,
	)
	if err != nil {
		return nil, err
	}

	outcome := "what is left waits until someone resumes it"
	switch req.Action {
	case accountingsyncports.AccountingBackfillResume:
		outcome = "it carries on sending history to the books from where it stopped"
	case accountingsyncports.AccountingBackfillCancel:
		outcome = "what is left is never sent; documents already sent stay in the books"
	case accountingsyncports.AccountingBackfillPause:
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would %s the %s backfill (now %s); %s.",
		strings.ToLower(string(req.Action)),
		change.provider,
		strings.ToLower(string(change.before.Status)),
		outcome,
	), recorded), nil
}
