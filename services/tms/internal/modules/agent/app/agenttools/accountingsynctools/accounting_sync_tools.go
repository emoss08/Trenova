package accountingsynctools

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncbackfill"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingbackfill"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const paramSyncRecordID = "syncRecordId"

const ParamSyncRecordIDs = "syncRecordIds"

const paramSyncErrorCategories = "errorCategories"

const paramSyncReason = "reason"

const paramBackfillFrom = "from"

const paramBackfillTo = "to"

const paramBackfillTypes = "documentTypes"

const maxSyncRetryIDs = 50

const maxSyncReasonRunes = 500

const toolGetSyncRecord = "accounting_get_sync_record"

const toolListSyncRecords = "accounting_list_sync_records"

const toolGetSyncStatus = "accounting_get_connection"

const secondsInOneDay = int64(24 * time.Hour / time.Second)

func SyncDocumentLabel(record *accountingsync.AccountingSyncRecord) string {
	if record.ObjectNumber == "" {
		return string(record.ObjectType) + " " + record.ObjectID.String()
	}
	return string(record.ObjectType) + " " + record.ObjectNumber
}

func syncReasonFrom(params map[string]any, why string) (string, error) {
	raw, err := toolparams.RequireString(params, paramSyncReason)
	if err != nil {
		return "", errortypes.NewValidationError(paramSyncReason, errortypes.ErrRequired, why)
	}
	return stringutils.TruncateRunes(
		stringutils.OneLine(raw, maxSyncReasonRunes),
		maxSyncReasonRunes,
	), nil
}

type accountingSyncSummarizer interface {
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) (*accountingsyncports.AccountingSyncSummary, error)
}

func SyncingConnectionFor(
	ctx context.Context,
	sync accountingSyncSummarizer,
	tenant pagination.TenantInfo,
	system integration.Type,
) (*accountingsyncports.AccountingSyncSummary, error) {
	summary, err := sync.Summary(ctx, tenant, system)
	if err != nil {
		return nil, err
	}
	if summary.Connection == nil || !summary.Connection.IsSyncing() {
		return nil, errortypes.NewBusinessError(
			"{0} is not sending documents yet; a person finishes its setup first",
			summary.ProviderName,
		)
	}
	return summary, nil
}

func countWithStatus(
	summary *accountingsyncports.AccountingSyncSummary,
	statuses ...accountingsync.SyncStatus,
) int {
	total := 0
	for _, count := range summary.Counts {
		if slices.Contains(statuses, count.Status) {
			total += count.Count
		}
	}
	return total
}

func syncEnums[T ~string](
	params map[string]any,
	key string,
	valid func(T) bool,
) ([]T, error) {
	raw, ok := params[key]
	if !ok || raw == nil {
		return nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("parameter %q must be an array", key)
	}
	out := make([]T, 0, len(values))
	for index, value := range values {
		text, isText := value.(string)
		typed := T(strings.TrimSpace(text))
		if !isText || !valid(typed) {
			return nil, fmt.Errorf("parameter %q[%d] is not a known value", key, index)
		}
		out = append(out, typed)
	}
	return sliceutils.Dedupe(out), nil
}

var errUnpinnedRetry = errortypes.NewBusinessError(
	"accounting_retry_sync_record runs from an approval only on the records a person approved, and " +
		"this proposal names none; propose it again so the records its error categories " +
		"resolve to are listed",
)

type RetryAccountingSyncTool struct {
	reads   accountingsyncports.AccountingSyncReader
	actions accountingsyncports.AccountingSyncActionService
}

func newRetryAccountingSyncTool(
	reads accountingsyncports.AccountingSyncReader,
	actions accountingsyncports.AccountingSyncActionService,
) agenttoolports.AgentTool {
	return &RetryAccountingSyncTool{reads: reads, actions: actions}
}

func ProvideRetryAccountingSyncTool(
	reads accountingsyncports.AccountingSyncReader,
	actions accountingsyncports.AccountingSyncActionService,
) agenttoolports.AgentTool {
	return newRetryAccountingSyncTool(reads, actions)
}

func (t *RetryAccountingSyncTool) Name() string { return "accounting_retry_sync_record" }

func (t *RetryAccountingSyncTool) Recipe() []string {
	return []string{
		toolGetSyncStatus,
		toolListSyncRecords,
		toolGetSyncRecord,
		"accounting_retry_sync_record",
	}
}

func (t *RetryAccountingSyncTool) Description() string {
	return "Send documents that did not reach the accounting system again. Name the records " +
		"by syncRecordIds, or retry every record that last failed for the given " +
		"errorCategories, such as Transient and RateLimited after an outage. Run on its own, " +
		"the categories are read when it runs; proposed, they become the records they name " +
		"then, at most 50, which the person approving may untick, and only those are " +
		"retried. Only Blocked, DeadLettered and Retrying records are retried; a document " +
		"waiting for a person to release it stays waiting. Retry only once the cause in the " +
		"record's resolution is fixed or the failure was temporary: a record whose cause " +
		"still stands is refused again. The accounting system recognizes a repeat, so " +
		"nothing is entered twice."
}

func (t *RetryAccountingSyncTool) SearchTerms() []string {
	return []string{"resend", "retry", "push again"}
}

func (t *RetryAccountingSyncTool) Prerequisites() []string {
	return []string{toolListSyncRecords, toolGetSyncRecord}
}

func (t *RetryAccountingSyncTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		accountingtools.ParamAccountingSystem: accountingtools.SystemSchema(),
		ParamSyncRecordIDs: toolschema.KeepEmpty(toolschema.RecordSubset(
			permission.ResourceAccountingSync.String(),
			agenttoolschema.OfKinds(jsonschemautils.DescribedArray(
				fmt.Sprintf(
					"Up to %d sync record ids, from accounting_list_sync_records.",
					maxSyncRetryIDs,
				),
				jsonschemautils.String(0),
				maxSyncRetryIDs,
			), recordkind.AccountingSyncRecord),
		)),
		paramSyncErrorCategories: jsonschemautils.DescribedArray(
			"Retry every retryable record that last failed for one of these reasons.",
			agenttoolschema.Enum("An error category.", agenttoolschema.SyncErrorCategories),
			len(accountingsync.AllSyncErrorCategories()),
		),
	}, accountingtools.ParamAccountingSystem)
}

func (t *RetryAccountingSyncTool) Policy() agenttoolports.ToolPolicy {
	return accountingtools.SyncPolicy(t.Name(), accountingtools.SyncPolicySpec{
		Resource:   permission.ResourceAccountingSync,
		Operation:  permission.OpUpdate,
		Tier:       agent.TierAutoExecute,
		Idempotent: true,
		Rationale: "Sends again, to the organization's own books, documents Trenova already " +
			"decided to send; the accounting system recognizes a repeat by its request id, so " +
			"nothing is entered twice, and a document held for release stays held.",
	})
}

type retryPlan struct {
	req     *accountingsyncports.RetryAccountingSyncRequest
	records []*accountingsync.AccountingSyncRecord
	summary *accountingsyncports.AccountingSyncSummary
}

func (t *RetryAccountingSyncTool) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*retryPlan, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}
	system, err := accountingtools.SystemFrom(params.Params)
	if err != nil {
		return nil, err
	}
	categories, err := syncEnums(
		params.Params,
		paramSyncErrorCategories,
		accountingsync.SyncErrorCategory.IsValid,
	)
	if err != nil {
		return nil, err
	}
	var ids []pulid.ID
	if _, named := params.Params[ParamSyncRecordIDs]; named {
		if ids, err = toolparams.RequirePulidSlice(
			params.Params,
			ParamSyncRecordIDs,
			maxSyncRetryIDs,
		); err != nil {
			return nil, err
		}
		ids = sliceutils.Dedupe(ids)
	}
	if len(ids) == 0 && len(categories) == 0 {
		return nil, errortypes.NewValidationError(
			ParamSyncRecordIDs,
			errortypes.ErrRequired,
			"Name the records to retry, or the error categories whose records to retry",
		)
	}
	if len(ids) == 0 && params.ApprovedFromProposal() {
		return nil, errUnpinnedRetry
	}

	tenant := toolparams.TenantFrom(params)
	summary, err := SyncingConnectionFor(ctx, t.reads, tenant, system)
	if err != nil {
		return nil, err
	}

	records := make([]*accountingsync.AccountingSyncRecord, 0, len(ids))
	for _, id := range ids {
		record, getErr := t.reads.GetRecord(ctx, tenant, id)
		if getErr != nil {
			return nil, getErr
		}
		if !record.Status.Retryable() {
			return nil, errortypes.NewBusinessError(
				"{0} is {1}; only Blocked, DeadLettered and Retrying records are retried",
				SyncDocumentLabel(record),
				string(record.Status),
			)
		}
		if len(categories) > 0 && !slices.Contains(categories, record.ErrorCategory) {
			continue
		}
		records = append(records, record)
	}

	return &retryPlan{
		req: &accountingsyncports.RetryAccountingSyncRequest{
			TenantInfo:      tenant,
			UserID:          params.Actor.UserID,
			IntegrationType: system,
			IDs:             ids,
			ErrorCategories: categories,
		},
		records: records,
		summary: summary,
	}, nil
}

func (t *RetryAccountingSyncTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	_, err := t.plan(ctx, params)
	return err
}

func (t *RetryAccountingSyncTool) ResolveProposalSelection(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (map[string]any, error) {
	if _, named := params.Params[ParamSyncRecordIDs]; named {
		return params.Params, nil
	}
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}
	system, err := accountingtools.SystemFrom(params.Params)
	if err != nil {
		return nil, err
	}
	categories, err := syncEnums(
		params.Params,
		paramSyncErrorCategories,
		accountingsync.SyncErrorCategory.IsValid,
	)
	if err != nil {
		return nil, err
	}
	if len(categories) == 0 {
		return params.Params, nil
	}

	candidates := retryCandidatesRequest(toolparams.TenantFrom(params), system, categories)
	candidates.Cursor.Limit = maxSyncRetryIDs
	page, err := t.reads.ListRecords(ctx, candidates)
	if err != nil {
		return nil, err
	}
	ids := make([]any, 0, len(page.Items))
	for _, record := range page.Items {
		if record != nil {
			ids = append(ids, record.ID.String())
		}
	}
	if len(ids) == 0 {
		return nil, errortypes.NewBusinessError(
			"No Blocked, DeadLettered or Retrying record last failed as {0}; "+
				"accounting_get_connection shows what is waiting",
			joinCategories(categories),
		)
	}

	pinned := make(map[string]any, len(params.Params)+1)
	maps.Copy(pinned, params.Params)
	pinned[ParamSyncRecordIDs] = ids

	return pinned, nil
}

func (t *RetryAccountingSyncTool) waitingCount(
	ctx context.Context,
	req *accountingsyncports.RetryAccountingSyncRequest,
) (count int, exact bool, err error) {
	candidates := retryCandidatesRequest(req.TenantInfo, req.IntegrationType, req.ErrorCategories)
	candidates.Cursor.Limit = 1
	candidates.Cursor.IncludeTotalCount = true
	page, err := t.reads.ListRecords(ctx, candidates)
	if err != nil {
		return 0, false, err
	}
	if page.TotalCount == nil {
		return len(page.Items), true, nil
	}

	return *page.TotalCount, page.TotalCountExact, nil
}

func retryCandidatesRequest(
	tenant pagination.TenantInfo,
	system integration.Type,
	categories []accountingsync.SyncErrorCategory,
) *accountingsyncports.ListAccountingSyncRecordsRequest {
	return &accountingsyncports.ListAccountingSyncRecordsRequest{
		TenantInfo:      tenant,
		IntegrationType: system,
		Statuses:        accountingsync.RetryableSyncStatuses(),
		ErrorCategories: categories,
	}
}

func joinCategories(categories []accountingsync.SyncErrorCategory) string {
	return strings.Join(sliceutils.Strings(categories), " or ")
}

func (t *RetryAccountingSyncTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	plan, err := t.plan(ctx, params)
	if err != nil {
		return err
	}
	_, err = t.actions.Retry(ctx, plan.req)
	return err
}

type SkipAccountingSyncTool struct {
	reads   accountingsyncports.AccountingSyncReader
	actions accountingsyncports.AccountingSyncActionService
}

func newSkipAccountingSyncTool(
	reads accountingsyncports.AccountingSyncReader,
	actions accountingsyncports.AccountingSyncActionService,
) agenttoolports.AgentTool {
	return &SkipAccountingSyncTool{reads: reads, actions: actions}
}

func ProvideSkipAccountingSyncTool(
	reads accountingsyncports.AccountingSyncReader,
	actions accountingsyncports.AccountingSyncActionService,
) agenttoolports.AgentTool {
	return newSkipAccountingSyncTool(reads, actions)
}

func (t *SkipAccountingSyncTool) Name() string { return "accounting_skip_sync_record" }

func (t *SkipAccountingSyncTool) Description() string {
	return "Mark one document as intentionally not sent to the accounting system, with the " +
		"reason. Use it only when a person says the document must stay out of the books, " +
		"for example because they already entered it by hand. A skipped document is never " +
		"sent again, so the books will not have it unless someone enters it there. It " +
		"cannot skip a document already sent or being sent."
}

func (t *SkipAccountingSyncTool) SearchTerms() []string {
	return []string{"skip", "exclude", "entered by hand"}
}

func (t *SkipAccountingSyncTool) Prerequisites() []string {
	return []string{toolListSyncRecords, toolGetSyncRecord}
}

func (t *SkipAccountingSyncTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramSyncRecordID: agenttoolschema.KindID(
			"The sync record's id, from accounting_list_sync_records or "+
				"accounting_get_sync_record.",
			recordkind.AccountingSyncRecord,
		),
		paramSyncReason: jsonschemautils.Text(
			"Why the document stays out of the books, in one sentence a bookkeeper " +
				"would accept.",
		),
	}, paramSyncRecordID, paramSyncReason)
}

func (t *SkipAccountingSyncTool) Policy() agenttoolports.ToolPolicy {
	return accountingtools.SyncPolicy(t.Name(), accountingtools.SyncPolicySpec{
		Resource:  permission.ResourceAccountingSync,
		Operation: permission.OpUpdate,
		Tier:      agent.TierActWithApproval,
		Rationale: "Leaves a document out of the organization's books for good; nothing sends " +
			"it again, so a person approves it.",
	})
}

// Target names the record this call would change, so a proposal to skip it is
// refused if the record moved on, such as being sent, before it is approved.
func (t *SkipAccountingSyncTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramSyncRecordID, permission.ResourceAccountingSync)
}

func (t *SkipAccountingSyncTool) request(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*accountingsyncports.SkipAccountingSyncRequest, *accountingsync.AccountingSyncRecord, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, nil, err
	}
	id, err := toolparams.RequirePulid(params.Params, paramSyncRecordID)
	if err != nil {
		return nil, nil, err
	}
	reason, err := syncReasonFrom(params.Params, "Say why this document stays out of the books")
	if err != nil {
		return nil, nil, err
	}

	tenant := toolparams.TenantFrom(params)
	record, err := t.reads.GetRecord(ctx, tenant, id)
	if err != nil {
		return nil, nil, err
	}
	if record.Status.IsFinal() || record.Status == accountingsync.SyncStatusInFlight {
		return nil, nil, errortypes.NewBusinessError(
			"{0} is {1} and cannot be skipped",
			SyncDocumentLabel(record),
			string(record.Status),
		)
	}

	return &accountingsyncports.SkipAccountingSyncRequest{
		TenantInfo: tenant,
		UserID:     params.Actor.UserID,
		ID:         id,
		Reason:     reason,
	}, record, nil
}

func (t *SkipAccountingSyncTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	_, _, err := t.request(ctx, params)
	return err
}

func (t *SkipAccountingSyncTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	req, _, err := t.request(ctx, params)
	if err != nil {
		return err
	}
	_, err = t.actions.Skip(ctx, req)
	return err
}

type redateAccountingSyncTool struct {
	actions accountingsyncports.AccountingSyncActionService
}

func newRedateAccountingSyncTool(
	actions accountingsyncports.AccountingSyncActionService,
) agenttoolports.AgentTool {
	return &redateAccountingSyncTool{actions: actions}
}

func ProvideRedateAccountingSyncTool(
	actions accountingsyncports.AccountingSyncActionService,
) agenttoolports.AgentTool {
	return newRedateAccountingSyncTool(actions)
}

func (t *redateAccountingSyncTool) Name() string { return "accounting_redate_sync_record" }

func (t *redateAccountingSyncTool) Description() string {
	return "Send a document the accounting system refused because its date falls in closed " +
		"books, dated instead on the first open day after the closing date. Use it when the " +
		"bookkeeper would rather keep the period closed than reopen it. The document keeps " +
		"its own date in Trenova, and its note in the books names that date. Trenova allows " +
		"this only when its closed-period posting policy is to post to the next open period; " +
		"otherwise the period must be reopened in the accounting system and the record retried."
}

func (t *redateAccountingSyncTool) SearchTerms() []string {
	return []string{"closed period", "books closed", "re-date", "first open day"}
}

func (t *redateAccountingSyncTool) Prerequisites() []string {
	return []string{toolListSyncRecords, toolGetSyncRecord}
}

func (t *redateAccountingSyncTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramSyncRecordID: agenttoolschema.KindID(
			"The sync record's id, from accounting_list_sync_records or "+
				"accounting_get_sync_record. It must be held for a closed period.",
			recordkind.AccountingSyncRecord,
		),
	}, paramSyncRecordID)
}

func (t *redateAccountingSyncTool) Policy() agenttoolports.ToolPolicy {
	return accountingtools.SyncPolicy(t.Name(), accountingtools.SyncPolicySpec{
		Resource:  permission.ResourceAccountingSync,
		Operation: permission.OpUpdate,
		Tier:      agent.TierActWithApproval,
		Rationale: "Changes the date the organization's books record for a document, " +
			"so a person approves it.",
	})
}

func (t *redateAccountingSyncTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramSyncRecordID, permission.ResourceAccountingSync)
}

func (t *redateAccountingSyncTool) request(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*accountingsyncports.RedateAccountingSyncRequest, *accountingsyncports.AccountingSyncRedatePlan, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, nil, err
	}
	id, err := toolparams.RequirePulid(params.Params, paramSyncRecordID)
	if err != nil {
		return nil, nil, err
	}
	req := &accountingsyncports.RedateAccountingSyncRequest{
		TenantInfo: toolparams.TenantFrom(params),
		UserID:     params.Actor.UserID,
		ID:         id,
	}
	plan, err := t.actions.PlanRedate(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	return req, plan, nil
}

func (t *redateAccountingSyncTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	_, _, err := t.request(ctx, params)
	return err
}

func (t *redateAccountingSyncTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	req, _, err := t.request(ctx, params)
	if err != nil {
		return err
	}
	_, err = t.actions.Redate(ctx, req)
	return err
}

type PauseAccountingSyncTool struct {
	reads accountingsyncports.AccountingSyncReader
	setup accountingsyncports.AccountingSyncSetupService
}

func newPauseAccountingSyncTool(
	reads accountingsyncports.AccountingSyncReader,
	setup accountingsyncports.AccountingSyncSetupService,
) agenttoolports.AgentTool {
	return &PauseAccountingSyncTool{reads: reads, setup: setup}
}

func ProvidePauseAccountingSyncTool(
	reads accountingsyncports.AccountingSyncReader,
	setup accountingsyncports.AccountingSyncSetupService,
) agenttoolports.AgentTool {
	return newPauseAccountingSyncTool(reads, setup)
}

func (t *PauseAccountingSyncTool) Name() string { return "accounting_pause_sync" }

func (t *PauseAccountingSyncTool) Description() string {
	return "Stop sending documents to the accounting system until someone resumes it. " +
		"Use it during an accounting system outage, or when a person asks to hold sending, " +
		"such as while the books are being closed. Nothing is lost: documents keep queueing " +
		"and go out when accounting_resume_sync runs. Give the reason so others know when " +
		"to resume."
}

func (t *PauseAccountingSyncTool) SearchTerms() []string {
	return []string{"pause", "hold sending", "outage"}
}

func (t *PauseAccountingSyncTool) Prerequisites() []string {
	return []string{toolGetSyncStatus}
}

func (t *PauseAccountingSyncTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		accountingtools.ParamAccountingSystem: accountingtools.SystemSchema(),
		paramSyncReason: jsonschemautils.Text(
			"Why sending is paused and what would let it resume, in one sentence.",
		),
	}, accountingtools.ParamAccountingSystem, paramSyncReason)
}

func (t *PauseAccountingSyncTool) Policy() agenttoolports.ToolPolicy {
	return accountingtools.SyncPolicy(t.Name(), accountingtools.SyncPolicySpec{
		Resource:   permission.ResourceAccountingSync,
		Operation:  permission.OpUpdate,
		Tier:       agent.TierAutoExecute,
		Reversible: true,
		Rationale: "Holds what is waiting to go to the books; nothing is sent, changed or " +
			"lost, and resuming sends it on.",
	})
}

func (t *PauseAccountingSyncTool) request(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*accountingsyncports.PauseAccountingSyncRequest, *accountingsyncports.AccountingSyncSummary, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, nil, err
	}
	system, err := accountingtools.SystemFrom(params.Params)
	if err != nil {
		return nil, nil, err
	}
	reason, err := syncReasonFrom(
		params.Params,
		"Say why sending is paused so others know when to resume it",
	)
	if err != nil {
		return nil, nil, err
	}

	tenant := toolparams.TenantFrom(params)
	summary, err := SyncingConnectionFor(ctx, t.reads, tenant, system)
	if err != nil {
		return nil, nil, err
	}
	if summary.Connection.IsPaused() {
		return nil, nil, errortypes.NewBusinessError(
			"Sending to {0} is already paused: {1}",
			summary.ProviderName,
			summary.Connection.PausedReason,
		)
	}

	return &accountingsyncports.PauseAccountingSyncRequest{
		TenantInfo:      tenant,
		UserID:          params.Actor.UserID,
		IntegrationType: system,
		Reason:          reason,
	}, summary, nil
}

func (t *PauseAccountingSyncTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	_, _, err := t.request(ctx, params)
	return err
}

func (t *PauseAccountingSyncTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	req, _, err := t.request(ctx, params)
	if err != nil {
		return err
	}
	_, err = t.setup.Pause(ctx, req)
	return err
}

type ResumeAccountingSyncTool struct {
	reads accountingsyncports.AccountingSyncReader
	setup accountingsyncports.AccountingSyncSetupService
}

func newResumeAccountingSyncTool(
	reads accountingsyncports.AccountingSyncReader,
	setup accountingsyncports.AccountingSyncSetupService,
) agenttoolports.AgentTool {
	return &ResumeAccountingSyncTool{reads: reads, setup: setup}
}

func ProvideResumeAccountingSyncTool(
	reads accountingsyncports.AccountingSyncReader,
	setup accountingsyncports.AccountingSyncSetupService,
) agenttoolports.AgentTool {
	return newResumeAccountingSyncTool(reads, setup)
}

func (t *ResumeAccountingSyncTool) Name() string { return "accounting_resume_sync" }

func (t *ResumeAccountingSyncTool) Recipe() []string {
	return []string{"accounting_get_connection", "accounting_resume_sync"}
}

func (t *ResumeAccountingSyncTool) Description() string {
	return "Start sending documents to the accounting system again after it was paused. " +
		"Everything queued while paused goes out at once, so resume only when a person says " +
		"the reason for the pause is over, or the outage it was paused for has passed and " +
		"accounting_get_connection shows the connection answering."
}

func (t *ResumeAccountingSyncTool) SearchTerms() []string {
	return []string{"resume", "unpause", "restart sending"}
}

func (t *ResumeAccountingSyncTool) Prerequisites() []string {
	return []string{toolGetSyncStatus}
}

func (t *ResumeAccountingSyncTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(
		map[string]any{accountingtools.ParamAccountingSystem: accountingtools.SystemSchema()},
		accountingtools.ParamAccountingSystem,
	)
}

func (t *ResumeAccountingSyncTool) Policy() agenttoolports.ToolPolicy {
	return accountingtools.SyncPolicy(t.Name(), accountingtools.SyncPolicySpec{
		Resource:  permission.ResourceAccountingSync,
		Operation: permission.OpUpdate,
		Tier:      agent.TierActWithApproval,
		Rationale: "Releases everything held to the organization's books at once, and a " +
			"person paused it for a reason, so a person approves it; what is sent cannot be " +
			"called back.",
	})
}

func (t *ResumeAccountingSyncTool) request(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*accountingsyncports.PauseAccountingSyncRequest, *accountingsyncports.AccountingSyncSummary, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, nil, err
	}
	system, err := accountingtools.SystemFrom(params.Params)
	if err != nil {
		return nil, nil, err
	}

	tenant := toolparams.TenantFrom(params)
	summary, err := SyncingConnectionFor(ctx, t.reads, tenant, system)
	if err != nil {
		return nil, nil, err
	}
	if !summary.Connection.IsPaused() {
		return nil, nil, errortypes.NewBusinessError(
			"Sending to {0} is not paused",
			summary.ProviderName,
		)
	}

	return &accountingsyncports.PauseAccountingSyncRequest{
		TenantInfo:      tenant,
		UserID:          params.Actor.UserID,
		IntegrationType: system,
	}, summary, nil
}

func (t *ResumeAccountingSyncTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	_, _, err := t.request(ctx, params)
	return err
}

func (t *ResumeAccountingSyncTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	req, _, err := t.request(ctx, params)
	if err != nil {
		return err
	}
	_, err = t.setup.Resume(ctx, req)
	return err
}

type RequestAccountingBackfillTool struct {
	reads     accountingsyncports.AccountingSyncReader
	backfills accountingsyncports.AccountingBackfillService
}

func newRequestAccountingBackfillTool(
	reads accountingsyncports.AccountingSyncReader,
	backfills accountingsyncports.AccountingBackfillService,
) agenttoolports.AgentTool {
	return &RequestAccountingBackfillTool{reads: reads, backfills: backfills}
}

func ProvideRequestAccountingBackfillTool(
	reads accountingsyncports.AccountingSyncReader,
	backfills accountingsyncports.AccountingBackfillService,
) agenttoolports.AgentTool {
	return newRequestAccountingBackfillTool(reads, backfills)
}

func (t *RequestAccountingBackfillTool) Name() string { return "accounting_request_backfill" }

func (t *RequestAccountingBackfillTool) Description() string {
	return "Backfill the accounting system with older invoices, memos, payments and " +
		"settlements. It sends those " +
		"dated from the start date up to the day sync began, which no live posting sent. Narrow " +
		"it by from and to dates and documentTypes. Use it only when a person asks for " +
		"history in the books, and ask whether any of it was already entered there by hand, " +
		"because those would arrive twice. One backfill runs at a time."
}

func (t *RequestAccountingBackfillTool) SearchTerms() []string {
	return []string{"backfill", "historical", "history"}
}

func (t *RequestAccountingBackfillTool) Prerequisites() []string {
	return []string{toolGetSyncStatus}
}

func (t *RequestAccountingBackfillTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		accountingtools.ParamAccountingSystem: accountingtools.SystemSchema(),
		paramBackfillFrom: agenttoolschema.Date(
			"The first document date to send. Defaults to the start date.",
		),
		paramBackfillTo: agenttoolschema.Date(
			"The last document date to send. Defaults to the day sending began.",
		),
		paramBackfillTypes: jsonschemautils.DescribedArray(
			"Only these kinds of document. Defaults to all of them.",
			agenttoolschema.Enum("A document type.", agenttoolschema.BackfillObjectTypes),
			len(accountingbackfill.BackfillObjectTypes()),
		),
	}, accountingtools.ParamAccountingSystem)
}

func (t *RequestAccountingBackfillTool) Policy() agenttoolports.ToolPolicy {
	return accountingtools.SyncPolicy(t.Name(), accountingtools.SyncPolicySpec{
		Resource:  permission.ResourceAccountingIntegration,
		Operation: permission.OpManage,
		Tier:      agent.TierActWithApproval,
		Rationale: "Sends historical documents to the organization's books, some of which may " +
			"already be there by hand, and cannot be called back; a person who manages the " +
			"integration approves it.",
	})
}

func optionalBackfillDay(
	params map[string]any,
	key string,
) (seconds int64, present bool, err error) {
	raw := strings.TrimSpace(toolparams.OptionalString(params, key))
	if raw == "" {
		return 0, false, nil
	}
	day, parseErr := time.Parse(changepreview.BackfillDayLayout, raw)
	if parseErr != nil {
		return 0, false, errortypes.NewValidationError(
			key,
			errortypes.ErrInvalid,
			"{0} must be a date as YYYY-MM-DD",
			key,
		)
	}
	return day.Unix(), true, nil
}

type backfillPlan struct {
	req        *accountingsyncports.RequestAccountingBackfillRequest
	connection *accountingconnection.AccountingConnection
	provider   string
	from       int64
	to         int64
	types      []accountingsync.SyncObjectType
}

func (t *RequestAccountingBackfillTool) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*backfillPlan, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}
	if !params.Actor.IsUser() || params.Actor.UserID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"A backfill is started by a person who manages the accounting integration",
		)
	}
	system, err := accountingtools.SystemFrom(params.Params)
	if err != nil {
		return nil, err
	}
	from, hasFrom, err := optionalBackfillDay(params.Params, paramBackfillFrom)
	if err != nil {
		return nil, err
	}
	to, hasTo, err := optionalBackfillDay(params.Params, paramBackfillTo)
	if err != nil {
		return nil, err
	}
	types, err := syncEnums(
		params.Params,
		paramBackfillTypes,
		func(typ accountingsync.SyncObjectType) bool {
			return slices.Contains(accountingbackfill.BackfillObjectTypes(), typ)
		},
	)
	if err != nil {
		return nil, err
	}

	tenant := toolparams.TenantFrom(params)
	summary, err := SyncingConnectionFor(ctx, t.reads, tenant, system)
	if err != nil {
		return nil, err
	}
	if types, err = accountingsyncbackfill.ObjectTypes(summary.Connection, types); err != nil {
		return nil, err
	}
	if summary.ActiveBackfill != nil {
		return nil, errortypes.NewBusinessError(
			"A backfill is already {0} for {1}; it must finish or be cancelled first",
			strings.ToLower(string(summary.ActiveBackfill.Status)),
			summary.ProviderName,
		)
	}

	conn := summary.Connection
	start, began := *conn.SyncStartDate, *conn.SyncEnabledAt
	rangeStart, rangeEnd := start, began
	if hasFrom {
		rangeStart = from
	}
	if hasTo {
		rangeEnd = min(to+secondsInOneDay-1, began)
	}
	if rangeStart < start {
		return nil, errortypes.NewValidationError(
			paramBackfillFrom,
			errortypes.ErrInvalid,
			"A backfill cannot reach before the start date, {0}",
			changepreview.DayLabel(start),
		)
	}
	if rangeStart > rangeEnd {
		return nil, errortypes.NewValidationError(
			paramBackfillTo,
			errortypes.ErrInvalid,
			"A backfill covers documents dated from {0} to {1}, the day sending began",
			changepreview.DayLabel(start),
			changepreview.DayLabel(began),
		)
	}

	return &backfillPlan{
		req: &accountingsyncports.RequestAccountingBackfillRequest{
			TenantInfo:      tenant,
			UserID:          params.Actor.UserID,
			IntegrationType: system,
			RangeStart:      &rangeStart,
			RangeEnd:        &rangeEnd,
			ObjectTypes:     types,
		},
		connection: conn,
		provider:   summary.ProviderName,
		from:       rangeStart,
		to:         rangeEnd,
		types:      types,
	}, nil
}

func (t *RequestAccountingBackfillTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	_, err := t.plan(ctx, params)
	return err
}

func (t *RequestAccountingBackfillTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	plan, err := t.plan(ctx, params)
	if err != nil {
		return err
	}
	_, err = t.backfills.RequestBackfill(ctx, plan.req)
	return err
}
