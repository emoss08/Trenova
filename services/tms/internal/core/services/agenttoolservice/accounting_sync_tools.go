package agenttoolservice

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountingsyncservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	paramSyncRecordID        = "syncRecordId"
	paramSyncRecordIDs       = "syncRecordIds"
	paramSyncErrorCategories = "errorCategories"
	paramSyncReason          = "reason"
	paramBackfillFrom        = "from"
	paramBackfillTo          = "to"
	paramBackfillTypes       = "documentTypes"

	maxSyncRetryIDs     = 50
	maxSyncReasonRunes  = 500
	toolGetSyncRecord   = "get_accounting_sync_record"
	toolListSyncRecords = "list_accounting_sync_records"
	toolGetSyncStatus   = "get_accounting_sync_status"
	fieldSyncStatus     = "status"
	fieldSending        = "sending"
	sendingPausedLabel  = "Paused"
	sendingRunningLabel = "Sending"
	backfillDayLayout   = "2006-01-02"
	secondsInOneDay     = int64(24 * time.Hour / time.Second)
)

type accountingSyncOperator interface {
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
	Retry(ctx context.Context, req *serviceports.RetryAccountingSyncRequest) (int64, error)
	Skip(
		ctx context.Context,
		req *serviceports.SkipAccountingSyncRequest,
	) (*accountingsync.AccountingSyncRecord, error)
	PlanRedate(
		ctx context.Context,
		req *serviceports.RedateAccountingSyncRequest,
	) (*serviceports.AccountingSyncRedatePlan, error)
	Redate(
		ctx context.Context,
		req *serviceports.RedateAccountingSyncRequest,
	) (*accountingsync.AccountingSyncRecord, error)
	Pause(
		ctx context.Context,
		req *serviceports.PauseAccountingSyncRequest,
	) (*accountingsync.AccountingConnection, error)
	Resume(
		ctx context.Context,
		req *serviceports.PauseAccountingSyncRequest,
	) (*accountingsync.AccountingConnection, error)
	RequestBackfill(
		ctx context.Context,
		req *serviceports.RequestAccountingBackfillRequest,
	) (*accountingsync.AccountingBackfill, error)
}

type accountingSyncPolicySpec struct {
	resource   permission.Resource
	operation  permission.Operation
	tier       agent.AutonomyTier
	reversible bool
	idempotent bool
	rationale  string
}

func accountingSyncPolicy(name string, spec accountingSyncPolicySpec) serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          name,
		Kind:          agent.ToolKindAction,
		Resource:      spec.resource,
		Operation:     spec.operation,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   spec.tier,
		MaxTier:       spec.tier,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    spec.reversible,
		Idempotent:    spec.idempotent,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     spec.rationale,
	}
}

func syncDocumentLabel(record *accountingsync.AccountingSyncRecord) string {
	if record.ObjectNumber == "" {
		return string(record.ObjectType) + " " + record.ObjectID.String()
	}
	return string(record.ObjectType) + " " + record.ObjectNumber
}

func syncReasonFrom(params map[string]any, why string) (string, error) {
	raw, err := requireString(params, paramSyncReason)
	if err != nil {
		return "", errortypes.NewValidationError(paramSyncReason, errortypes.ErrRequired, why)
	}
	return stringutils.TruncateRunes(
		stringutils.OneLine(raw, maxSyncReasonRunes),
		maxSyncReasonRunes,
	), nil
}

func syncingConnectionFor(
	ctx context.Context,
	sync accountingSyncOperator,
	tenant pagination.TenantInfo,
	system integration.Type,
) (*serviceports.AccountingSyncSummary, error) {
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
	summary *serviceports.AccountingSyncSummary,
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

type retryAccountingSyncTool struct {
	sync accountingSyncOperator
}

func newRetryAccountingSyncTool(sync accountingSyncOperator) serviceports.AgentTool {
	return &retryAccountingSyncTool{sync: sync}
}

func provideRetryAccountingSyncTool(
	sync serviceports.AccountingSyncService,
) serviceports.AgentTool {
	return newRetryAccountingSyncTool(sync)
}

func (t *retryAccountingSyncTool) Name() string { return "retry_accounting_sync" }

func (t *retryAccountingSyncTool) Description() string {
	return "Send documents that did not reach the accounting system again. Name the records " +
		"by syncRecordIds, or retry every record that last failed for the given " +
		"errorCategories, such as Transient and RateLimited after an outage. Only Blocked, " +
		"DeadLettered and Retrying records are retried; a document waiting for a person to " +
		"release it stays waiting. Retry only once the cause in the record's resolution is " +
		"fixed or the failure was temporary: a record whose cause still stands is refused " +
		"again. The accounting system recognizes a repeat, so nothing is entered twice."
}

func (t *retryAccountingSyncTool) SearchTerms() []string {
	return []string{"resend", "retry", "push again"}
}

func (t *retryAccountingSyncTool) Prerequisites() []string {
	return []string{toolListSyncRecords, toolGetSyncRecord}
}

func (t *retryAccountingSyncTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramAccountingSystem: accountingSystemSchema(),
		paramSyncRecordIDs: jsonschemautils.DescribedArray(
			fmt.Sprintf(
				"Up to %d sync record ids, from list_accounting_sync_records or "+
					"get_record_accounting_sync_state.",
				maxSyncRetryIDs,
			),
			jsonschemautils.String(0),
			maxSyncRetryIDs,
		),
		paramSyncErrorCategories: jsonschemautils.DescribedArray(
			"Retry every retryable record that last failed for one of these reasons.",
			jsonschemautils.Enum(
				"An error category.",
				sliceutils.Strings(accountingsync.AllSyncErrorCategories())...,
			),
			len(accountingsync.AllSyncErrorCategories()),
		),
	}, paramAccountingSystem)
}

func (t *retryAccountingSyncTool) Policy() serviceports.ToolPolicy {
	return accountingSyncPolicy(t.Name(), accountingSyncPolicySpec{
		resource:   permission.ResourceAccountingSync,
		operation:  permission.OpUpdate,
		tier:       agent.TierAutoExecute,
		idempotent: true,
		rationale: "Sends again, to the organization's own books, documents Trenova already " +
			"decided to send; the accounting system recognizes a repeat by its request id, so " +
			"nothing is entered twice, and a document held for release stays held.",
	})
}

type retryPlan struct {
	req     *serviceports.RetryAccountingSyncRequest
	records []*accountingsync.AccountingSyncRecord
	summary *serviceports.AccountingSyncSummary
}

func (t *retryAccountingSyncTool) plan(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*retryPlan, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	system, err := accountingSystemFrom(params.Params)
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
	if _, named := params.Params[paramSyncRecordIDs]; named {
		if ids, err = requirePulidSlice(
			params.Params,
			paramSyncRecordIDs,
			maxSyncRetryIDs,
		); err != nil {
			return nil, err
		}
		ids = sliceutils.Dedupe(ids)
	}
	if len(ids) == 0 && len(categories) == 0 {
		return nil, errortypes.NewValidationError(
			paramSyncRecordIDs,
			errortypes.ErrRequired,
			"Name the records to retry, or the error categories whose records to retry",
		)
	}

	tenant := tenantFrom(*params)
	summary, err := syncingConnectionFor(ctx, t.sync, tenant, system)
	if err != nil {
		return nil, err
	}

	records := make([]*accountingsync.AccountingSyncRecord, 0, len(ids))
	for _, id := range ids {
		record, getErr := t.sync.GetRecord(ctx, tenant, id)
		if getErr != nil {
			return nil, getErr
		}
		if !record.Status.Retryable() {
			return nil, errortypes.NewBusinessError(
				"{0} is {1}; only Blocked, DeadLettered and Retrying records are retried",
				syncDocumentLabel(record),
				string(record.Status),
			)
		}
		records = append(records, record)
	}

	return &retryPlan{
		req: &serviceports.RetryAccountingSyncRequest{
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

func (t *retryAccountingSyncTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.plan(ctx, &params)
	return err
}

func (t *retryAccountingSyncTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.sync.Retry(ctx, plan.req)
	return err
}

type skipAccountingSyncTool struct {
	sync accountingSyncOperator
}

func newSkipAccountingSyncTool(sync accountingSyncOperator) serviceports.AgentTool {
	return &skipAccountingSyncTool{sync: sync}
}

func provideSkipAccountingSyncTool(sync serviceports.AccountingSyncService) serviceports.AgentTool {
	return newSkipAccountingSyncTool(sync)
}

func (t *skipAccountingSyncTool) Name() string { return "skip_accounting_sync" }

func (t *skipAccountingSyncTool) Description() string {
	return "Mark one document as intentionally not sent to the accounting system, with the " +
		"reason. Use it only when a person says the document must stay out of the books, " +
		"for example because they already entered it by hand. A skipped document is never " +
		"sent again, so the books will not have it unless someone enters it there. It " +
		"cannot skip a document already sent or being sent."
}

func (t *skipAccountingSyncTool) SearchTerms() []string {
	return []string{"skip", "exclude", "entered by hand"}
}

func (t *skipAccountingSyncTool) Prerequisites() []string {
	return []string{toolGetSyncRecord}
}

func (t *skipAccountingSyncTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramSyncRecordID: jsonschemautils.Text(
			"The sync record's id, from list_accounting_sync_records or " +
				"get_accounting_sync_record.",
		),
		paramSyncReason: jsonschemautils.Text(
			"Why the document stays out of the books, in one sentence a bookkeeper " +
				"would accept.",
		),
	}, paramSyncRecordID, paramSyncReason)
}

func (t *skipAccountingSyncTool) Policy() serviceports.ToolPolicy {
	return accountingSyncPolicy(t.Name(), accountingSyncPolicySpec{
		resource:  permission.ResourceAccountingSync,
		operation: permission.OpUpdate,
		tier:      agent.TierActWithApproval,
		rationale: "Leaves a document out of the organization's books for good; nothing sends " +
			"it again, so a person approves it.",
	})
}

// Target names the record this call would change, so a proposal to skip it is
// refused if the record moved on, such as being sent, before it is approved.
func (t *skipAccountingSyncTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramSyncRecordID, permission.ResourceAccountingSync)
}

func (t *skipAccountingSyncTool) request(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*serviceports.SkipAccountingSyncRequest, *accountingsync.AccountingSyncRecord, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, nil, err
	}
	id, err := requirePulid(params.Params, paramSyncRecordID)
	if err != nil {
		return nil, nil, err
	}
	reason, err := syncReasonFrom(params.Params, "Say why this document stays out of the books")
	if err != nil {
		return nil, nil, err
	}

	tenant := tenantFrom(*params)
	record, err := t.sync.GetRecord(ctx, tenant, id)
	if err != nil {
		return nil, nil, err
	}
	if record.Status.IsFinal() || record.Status == accountingsync.SyncStatusInFlight {
		return nil, nil, errortypes.NewBusinessError(
			"{0} is {1} and cannot be skipped",
			syncDocumentLabel(record),
			string(record.Status),
		)
	}

	return &serviceports.SkipAccountingSyncRequest{
		TenantInfo: tenant,
		UserID:     params.Actor.UserID,
		ID:         id,
		Reason:     reason,
	}, record, nil
}

func (t *skipAccountingSyncTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, _, err := t.request(ctx, &params)
	return err
}

func (t *skipAccountingSyncTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, _, err := t.request(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.sync.Skip(ctx, req)
	return err
}

type redateAccountingSyncTool struct {
	sync accountingSyncOperator
}

func newRedateAccountingSyncTool(sync accountingSyncOperator) serviceports.AgentTool {
	return &redateAccountingSyncTool{sync: sync}
}

func provideRedateAccountingSyncTool(
	sync serviceports.AccountingSyncService,
) serviceports.AgentTool {
	return newRedateAccountingSyncTool(sync)
}

func (t *redateAccountingSyncTool) Name() string { return "redate_accounting_sync" }

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
	return []string{toolGetSyncRecord}
}

func (t *redateAccountingSyncTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramSyncRecordID: jsonschemautils.Text(
			"The sync record's id, from list_accounting_sync_records or " +
				"get_accounting_sync_record. It must be held for a closed period.",
		),
	}, paramSyncRecordID)
}

func (t *redateAccountingSyncTool) Policy() serviceports.ToolPolicy {
	return accountingSyncPolicy(t.Name(), accountingSyncPolicySpec{
		resource:  permission.ResourceAccountingSync,
		operation: permission.OpUpdate,
		tier:      agent.TierActWithApproval,
		rationale: "Changes the date the organization's books record for a document, " +
			"so a person approves it.",
	})
}

func (t *redateAccountingSyncTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramSyncRecordID, permission.ResourceAccountingSync)
}

func (t *redateAccountingSyncTool) request(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*serviceports.RedateAccountingSyncRequest, *serviceports.AccountingSyncRedatePlan, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, nil, err
	}
	id, err := requirePulid(params.Params, paramSyncRecordID)
	if err != nil {
		return nil, nil, err
	}
	req := &serviceports.RedateAccountingSyncRequest{
		TenantInfo: tenantFrom(*params),
		UserID:     params.Actor.UserID,
		ID:         id,
	}
	plan, err := t.sync.PlanRedate(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	return req, plan, nil
}

func (t *redateAccountingSyncTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, _, err := t.request(ctx, &params)
	return err
}

func (t *redateAccountingSyncTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, _, err := t.request(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.sync.Redate(ctx, req)
	return err
}

type pauseAccountingSyncTool struct {
	sync accountingSyncOperator
}

func newPauseAccountingSyncTool(sync accountingSyncOperator) serviceports.AgentTool {
	return &pauseAccountingSyncTool{sync: sync}
}

func providePauseAccountingSyncTool(
	sync serviceports.AccountingSyncService,
) serviceports.AgentTool {
	return newPauseAccountingSyncTool(sync)
}

func (t *pauseAccountingSyncTool) Name() string { return "pause_accounting_sync" }

func (t *pauseAccountingSyncTool) Description() string {
	return "Stop sending documents to the accounting system until someone resumes it. " +
		"Use it during an accounting system outage, or when a person asks to hold sending, " +
		"such as while the books are being closed. Nothing is lost: documents keep queueing " +
		"and go out when resume_accounting_sync runs. Give the reason so others know when " +
		"to resume."
}

func (t *pauseAccountingSyncTool) SearchTerms() []string {
	return []string{"pause", "hold sending", "outage"}
}

func (t *pauseAccountingSyncTool) Prerequisites() []string {
	return []string{toolGetSyncStatus}
}

func (t *pauseAccountingSyncTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramAccountingSystem: accountingSystemSchema(),
		paramSyncReason: jsonschemautils.Text(
			"Why sending is paused and what would let it resume, in one sentence.",
		),
	}, paramAccountingSystem, paramSyncReason)
}

func (t *pauseAccountingSyncTool) Policy() serviceports.ToolPolicy {
	return accountingSyncPolicy(t.Name(), accountingSyncPolicySpec{
		resource:   permission.ResourceAccountingSync,
		operation:  permission.OpUpdate,
		tier:       agent.TierAutoExecute,
		reversible: true,
		rationale: "Holds what is waiting to go to the books; nothing is sent, changed or " +
			"lost, and resuming sends it on.",
	})
}

func (t *pauseAccountingSyncTool) request(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*serviceports.PauseAccountingSyncRequest, *serviceports.AccountingSyncSummary, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, nil, err
	}
	system, err := accountingSystemFrom(params.Params)
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

	tenant := tenantFrom(*params)
	summary, err := syncingConnectionFor(ctx, t.sync, tenant, system)
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

	return &serviceports.PauseAccountingSyncRequest{
		TenantInfo:      tenant,
		UserID:          params.Actor.UserID,
		IntegrationType: system,
		Reason:          reason,
	}, summary, nil
}

func (t *pauseAccountingSyncTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, _, err := t.request(ctx, &params)
	return err
}

func (t *pauseAccountingSyncTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, _, err := t.request(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.sync.Pause(ctx, req)
	return err
}

type resumeAccountingSyncTool struct {
	sync accountingSyncOperator
}

func newResumeAccountingSyncTool(sync accountingSyncOperator) serviceports.AgentTool {
	return &resumeAccountingSyncTool{sync: sync}
}

func provideResumeAccountingSyncTool(
	sync serviceports.AccountingSyncService,
) serviceports.AgentTool {
	return newResumeAccountingSyncTool(sync)
}

func (t *resumeAccountingSyncTool) Name() string { return "resume_accounting_sync" }

func (t *resumeAccountingSyncTool) Description() string {
	return "Start sending documents to the accounting system again after it was paused. " +
		"Everything queued while paused goes out at once, so resume only when a person says " +
		"the reason for the pause is over, or the outage it was paused for has passed and " +
		"get_accounting_sync_status shows the connection answering."
}

func (t *resumeAccountingSyncTool) SearchTerms() []string {
	return []string{"resume", "unpause", "restart sending"}
}

func (t *resumeAccountingSyncTool) Prerequisites() []string {
	return []string{toolGetSyncStatus}
}

func (t *resumeAccountingSyncTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(
		map[string]any{paramAccountingSystem: accountingSystemSchema()},
		paramAccountingSystem,
	)
}

func (t *resumeAccountingSyncTool) Policy() serviceports.ToolPolicy {
	return accountingSyncPolicy(t.Name(), accountingSyncPolicySpec{
		resource:  permission.ResourceAccountingSync,
		operation: permission.OpUpdate,
		tier:      agent.TierActWithApproval,
		rationale: "Releases everything held to the organization's books at once, and a " +
			"person paused it for a reason, so a person approves it; what is sent cannot be " +
			"called back.",
	})
}

func (t *resumeAccountingSyncTool) request(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*serviceports.PauseAccountingSyncRequest, *serviceports.AccountingSyncSummary, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, nil, err
	}
	system, err := accountingSystemFrom(params.Params)
	if err != nil {
		return nil, nil, err
	}

	tenant := tenantFrom(*params)
	summary, err := syncingConnectionFor(ctx, t.sync, tenant, system)
	if err != nil {
		return nil, nil, err
	}
	if !summary.Connection.IsPaused() {
		return nil, nil, errortypes.NewBusinessError(
			"Sending to {0} is not paused",
			summary.ProviderName,
		)
	}

	return &serviceports.PauseAccountingSyncRequest{
		TenantInfo:      tenant,
		UserID:          params.Actor.UserID,
		IntegrationType: system,
	}, summary, nil
}

func (t *resumeAccountingSyncTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, _, err := t.request(ctx, &params)
	return err
}

func (t *resumeAccountingSyncTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, _, err := t.request(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.sync.Resume(ctx, req)
	return err
}

type requestAccountingBackfillTool struct {
	sync accountingSyncOperator
}

func newRequestAccountingBackfillTool(sync accountingSyncOperator) serviceports.AgentTool {
	return &requestAccountingBackfillTool{sync: sync}
}

func provideRequestAccountingBackfillTool(
	sync serviceports.AccountingSyncService,
) serviceports.AgentTool {
	return newRequestAccountingBackfillTool(sync)
}

func (t *requestAccountingBackfillTool) Name() string { return "request_accounting_backfill" }

func (t *requestAccountingBackfillTool) Description() string {
	return "Backfill the accounting system with older invoices, memos, payments and " +
		"settlements. It sends those " +
		"dated from the start date up to the day sync began, which no live posting sent. Narrow " +
		"it by from and to dates and documentTypes. Use it only when a person asks for " +
		"history in the books, and ask whether any of it was already entered there by hand, " +
		"because those would arrive twice. One backfill runs at a time."
}

func (t *requestAccountingBackfillTool) SearchTerms() []string {
	return []string{"backfill", "historical", "history"}
}

func (t *requestAccountingBackfillTool) Prerequisites() []string {
	return []string{toolGetSyncStatus}
}

func (t *requestAccountingBackfillTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramAccountingSystem: accountingSystemSchema(),
		paramBackfillFrom: jsonschemautils.Text(
			"The first document date to send, as YYYY-MM-DD. Defaults to the start date.",
		),
		paramBackfillTo: jsonschemautils.Text(
			"The last document date to send, as YYYY-MM-DD. Defaults to the day sending began.",
		),
		paramBackfillTypes: jsonschemautils.DescribedArray(
			"Only these kinds of document. Defaults to all of them.",
			jsonschemautils.Enum(
				"A document type.",
				sliceutils.Strings(accountingsync.BackfillObjectTypes())...,
			),
			len(accountingsync.BackfillObjectTypes()),
		),
	}, paramAccountingSystem)
}

func (t *requestAccountingBackfillTool) Policy() serviceports.ToolPolicy {
	return accountingSyncPolicy(t.Name(), accountingSyncPolicySpec{
		resource:  permission.ResourceAccountingIntegration,
		operation: permission.OpManage,
		tier:      agent.TierActWithApproval,
		rationale: "Sends historical documents to the organization's books, some of which may " +
			"already be there by hand, and cannot be called back; a person who manages the " +
			"integration approves it.",
	})
}

func optionalBackfillDay(
	params map[string]any,
	key string,
) (seconds int64, present bool, err error) {
	raw := strings.TrimSpace(optionalString(params, key))
	if raw == "" {
		return 0, false, nil
	}
	day, parseErr := time.Parse(backfillDayLayout, raw)
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

func dayLabel(seconds int64) string {
	return time.Unix(seconds, 0).UTC().Format(backfillDayLayout)
}

type backfillPlan struct {
	req        *serviceports.RequestAccountingBackfillRequest
	connection *accountingsync.AccountingConnection
	provider   string
	from       int64
	to         int64
	types      []accountingsync.SyncObjectType
}

func (t *requestAccountingBackfillTool) plan(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*backfillPlan, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, err
	}
	if !params.Actor.IsUser() || params.Actor.UserID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"A backfill is started by a person who manages the accounting integration",
		)
	}
	system, err := accountingSystemFrom(params.Params)
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
			return slices.Contains(accountingsync.BackfillObjectTypes(), typ)
		},
	)
	if err != nil {
		return nil, err
	}

	tenant := tenantFrom(*params)
	summary, err := syncingConnectionFor(ctx, t.sync, tenant, system)
	if err != nil {
		return nil, err
	}
	if types, err = accountingsyncservice.BackfillTypes(summary.Connection, types); err != nil {
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
			dayLabel(start),
		)
	}
	if rangeStart > rangeEnd {
		return nil, errortypes.NewValidationError(
			paramBackfillTo,
			errortypes.ErrInvalid,
			"A backfill covers documents dated from {0} to {1}, the day sending began",
			dayLabel(start),
			dayLabel(began),
		)
	}

	return &backfillPlan{
		req: &serviceports.RequestAccountingBackfillRequest{
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

func (t *requestAccountingBackfillTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.plan(ctx, &params)
	return err
}

func (t *requestAccountingBackfillTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.sync.RequestBackfill(ctx, plan.req)
	return err
}
