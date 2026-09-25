package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	paramSyncStatus        = "status"
	paramSyncDocumentType  = "documentType"
	paramSyncErrorCategory = "errorCategory"
	paramSyncDocumentID    = "documentId"
	paramSyncDocumentIDs   = "documentIds"
	paramSyncRecordID      = "syncRecordId"
	paramSyncSearch        = "search"

	syncRecordsDefaultLimit = 25
	syncRecordsMaxLimit     = 50
	syncStateMaxDocuments   = 50

	absentNotSynced    = "not synced"
	absentNotScheduled = "not scheduled"

	syncDocumentIDSources = "list_invoices, get_invoice or list_customer_payments"
)

type accountingSyncLedgerReader interface {
	ListRecords(
		ctx context.Context,
		req *serviceports.ListAccountingSyncRecordsRequest,
	) (*pagination.CursorListResult[*accountingsync.AccountingSyncRecord], error)
	GetRecord(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*accountingsync.AccountingSyncRecord, error)
	ListAttempts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		recordID pulid.ID,
	) ([]*accountingsync.AccountingSyncAttempt, error)
	ObjectStates(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		objectIDs []pulid.ID,
	) (map[pulid.ID]*serviceports.AccountingSyncObjectState, error)
}

type accountingSyncRecordRow struct {
	ID             string       `json:"id"`
	DocumentType   string       `json:"documentType"`
	DocumentID     string       `json:"documentId"`
	DocumentNumber string       `json:"documentNumber,omitempty"`
	Operation      string       `json:"operation"`
	Source         string       `json:"source"`
	Status         string       `json:"status"`
	ErrorCategory  string       `json:"errorCategory,omitempty"`
	Resolution     string       `json:"resolution,omitempty"`
	Attempts       int          `json:"attempts"`
	DocumentDate   optionalDate `json:"documentDate"`
	QueuedOn       optionalDate `json:"queuedOn"`
	NextAttemptOn  optionalDate `json:"nextAttemptOn"`
	SyncedOn       optionalDate `json:"syncedOn"`
	ExternalNumber string       `json:"externalNumber,omitempty"`
	ExternalURL    string       `json:"externalUrl,omitempty"`
	SkippedReason  string       `json:"skippedReason,omitempty"`
	CanRetry       bool         `json:"canRetry"`
	CanSkip        bool         `json:"canSkip"`
}

type accountingSyncAttemptRow struct {
	Number        int          `json:"number"`
	Outcome       string       `json:"outcome"`
	ErrorCategory string       `json:"errorCategory,omitempty"`
	StartedOn     optionalDate `json:"startedOn"`
	DurationMs    int          `json:"durationMs"`
}

type accountingSyncRecordsResult struct {
	Records    []accountingSyncRecordRow `json:"records"`
	HasMore    bool                      `json:"hasMore"`
	NextCursor string                    `json:"nextCursor,omitempty"`
	LedgerPath string                    `json:"ledgerPath"`
}

type accountingSyncRecordDetail struct {
	Record          accountingSyncRecordRow    `json:"record"`
	WaitsOnRecordID string                     `json:"waitsOnRecordId,omitempty"`
	AttemptHistory  []accountingSyncAttemptRow `json:"attemptHistory"`
	WhatThisMeans   string                     `json:"whatThisMeans"`
}

type accountingSyncStateRow struct {
	DocumentID    string                   `json:"documentId"`
	Tracked       bool                     `json:"tracked"`
	Provider      string                   `json:"provider,omitempty"`
	Record        *accountingSyncRecordRow `json:"record,omitempty"`
	WhatThisMeans string                   `json:"whatThisMeans"`
}

type accountingSyncStatesResult struct {
	States []accountingSyncStateRow `json:"states"`
}

const accountingSyncLedgerPath = "/accounting/sync"

func accountingSyncRecordRowFrom(
	record *accountingsync.AccountingSyncRecord,
) accountingSyncRecordRow {
	row := accountingSyncRecordRow{
		ID:             record.ID.String(),
		DocumentType:   string(record.ObjectType),
		DocumentID:     record.ObjectID.String(),
		DocumentNumber: record.ObjectNumber,
		Operation:      string(record.Operation),
		Source:         string(record.SourceEvent),
		Status:         string(record.Status),
		ErrorCategory:  string(record.ErrorCategory),
		Resolution:     record.Resolution,
		Attempts:       record.AttemptCount,
		DocumentDate:   pointerDate(record.DocumentDate),
		QueuedOn:       recordedDate(record.QueuedAt),
		NextAttemptOn:  expectedDate(0, absentNotScheduled),
		SyncedOn:       expectedDate(derefInt64(record.SyncedAt), absentNotSynced),
		ExternalNumber: record.ExternalDocNumber,
		ExternalURL:    record.ExternalURL,
		SkippedReason:  record.SkippedReason,
		CanRetry:       record.Status.Retryable(),
		CanSkip: !record.Status.IsFinal() &&
			record.Status != accountingsync.SyncStatusInFlight,
	}
	if record.Status.Dispatchable() {
		row.NextAttemptOn = expectedDate(derefInt64(record.NextAttemptAt), absentNotScheduled)
	}

	return row
}

func accountingSyncMeaning(record *accountingsync.AccountingSyncRecord, provider string) string {
	switch record.Status {
	case accountingsync.SyncStatusSynced:
		return "It is in " + provider + "."
	case accountingsync.SyncStatusQueued, accountingsync.SyncStatusInFlight:
		return "It is on its way to " + provider + "."
	case accountingsync.SyncStatusRetrying:
		return "The last try failed for a reason that usually passes; Trenova tries again " +
			"on its own."
	case accountingsync.SyncStatusAwaitingApproval:
		return "Sending is not automatic, so it waits for a person to release it."
	case accountingsync.SyncStatusBlocked:
		return provider + " refused it and it will not be tried again until the cause is " +
			"fixed and it is retried. The resolution says what to fix."
	case accountingsync.SyncStatusDeadLettered:
		return "Trenova gave up after repeated temporary failures. Retry it once " +
			provider + " answers again."
	case accountingsync.SyncStatusSkipped:
		return "A person chose not to send it to " + provider + "."
	case accountingsync.SyncStatusSuperseded:
		return "A later version of the document replaced this record."
	default:
		return "Its state is not known."
	}
}

func syncStatusValues() []string {
	return sliceutils.Strings(accountingsync.AllSyncStatuses())
}

func syncDocumentTypeValues() []string {
	return sliceutils.Strings(accountingsync.AllSyncObjectTypes())
}

func syncErrorCategoryValues() []string {
	return sliceutils.Strings(accountingsync.AllSyncErrorCategories())
}

type listAccountingSyncRecordsTool struct {
	ledger accountingSyncLedgerReader
}

func newListAccountingSyncRecordsTool(
	ledger accountingSyncLedgerReader,
) serviceports.AgentQueryTool {
	return &listAccountingSyncRecordsTool{ledger: ledger}
}

func provideListAccountingSyncRecordsTool(
	ledger serviceports.AccountingSyncService,
) serviceports.AgentQueryTool {
	return newListAccountingSyncRecordsTool(ledger)
}

func (t *listAccountingSyncRecordsTool) Name() string { return "list_accounting_sync_records" }

func (t *listAccountingSyncRecordsTool) Description() string {
	return "List the documents Trenova sends to the accounting system and where each stands, " +
		"newest first. Filter by status, such as Blocked and DeadLettered for what did not " +
		"reach the books, by document type, by error category, by one Trenova document, or " +
		"search document numbers. Each row has its status, the error category, the " +
		"plain-language resolution, how many tries it took and the accounting system's " +
		"document number once synced. Pass a row's id to get_accounting_sync_record for its " +
		"tries, or to retry_accounting_sync once the cause is fixed."
}

func (t *listAccountingSyncRecordsTool) SearchTerms() []string {
	return []string{"failed", "blocked", "ledger", "not synced"}
}

func (t *listAccountingSyncRecordsTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramAccountingSystem: accountingSystemParam(),
		paramSyncStatus: jsonschemautils.DescribedArray(
			"Only records in these statuses. Blocked and DeadLettered need a person; "+
				"AwaitingApproval waits to be released.",
			jsonschemautils.Enum("A sync status.", syncStatusValues()...),
			len(accountingsync.AllSyncStatuses()),
		),
		paramSyncDocumentType: jsonschemautils.DescribedArray(
			"Only these kinds of document.",
			jsonschemautils.Enum("A document type.", syncDocumentTypeValues()...),
			len(accountingsync.AllSyncObjectTypes()),
		),
		paramSyncErrorCategory: jsonschemautils.DescribedArray(
			"Only records that last failed for these reasons.",
			jsonschemautils.Enum("An error category.", syncErrorCategoryValues()...),
			len(accountingsync.AllSyncErrorCategories()),
		),
		paramSyncDocumentID: jsonschemautils.Text(
			"Only the records of one Trenova document, by its id from " +
				syncDocumentIDSources + ".",
		),
		paramSyncSearch: jsonschemautils.Text(
			"Words to find in Trenova or accounting system document numbers.",
		),
		paramLimit: jsonschemautils.Integer(fmt.Sprintf(
			"How many records to return, at most %d. Defaults to %d.",
			syncRecordsMaxLimit, syncRecordsDefaultLimit,
		)),
		paramAfter: jsonschemautils.Text(
			"The nextCursor from a previous call of this tool, for the next page.",
		),
	}, paramAccountingSystem)
}

func (t *listAccountingSyncRecordsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceAccountingSync})
}

func (t *listAccountingSyncRecordsTool) request(
	params *serviceports.QueryToolParams,
) (*serviceports.ListAccountingSyncRecordsRequest, error) {
	system, err := requireAccountingSystem(params.Params)
	if err != nil {
		return nil, err
	}
	statuses, err := optionalEnums(
		params.Params,
		paramSyncStatus,
		accountingsync.SyncStatus.IsValid,
	)
	if err != nil {
		return nil, err
	}
	types, err := optionalEnums(
		params.Params,
		paramSyncDocumentType,
		accountingsync.SyncObjectType.IsValid,
	)
	if err != nil {
		return nil, err
	}
	categories, err := optionalEnums(
		params.Params,
		paramSyncErrorCategory,
		accountingsync.SyncErrorCategory.IsValid,
	)
	if err != nil {
		return nil, err
	}
	documentID, err := optionalID(params.Params, paramSyncDocumentID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramSyncDocumentID, err)
	}
	limit := min(
		max(optionalInt(params.Params, paramLimit, syncRecordsDefaultLimit), 1),
		syncRecordsMaxLimit,
	)
	cursor, err := pagination.NewCursorInfo(limit, optionalString(params.Params, paramAfter))
	if err != nil {
		return nil, fmt.Errorf(
			"parameter %q is not a cursor this tool returned: %w",
			paramAfter,
			err,
		)
	}
	cursor.IncludeTotalCount = false

	return &serviceports.ListAccountingSyncRecordsRequest{
		TenantInfo:      tenantOf(params),
		IntegrationType: system,
		Statuses:        statuses,
		ObjectTypes:     types,
		ErrorCategories: categories,
		ObjectID:        documentID,
		Search:          optionalString(params.Params, paramSyncSearch),
		Cursor:          cursor,
	}, nil
}

func (t *listAccountingSyncRecordsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}

	page, err := t.ledger.ListRecords(ctx, req)
	if err != nil {
		return nil, err
	}

	result := &accountingSyncRecordsResult{
		Records:    make([]accountingSyncRecordRow, 0, len(page.Items)),
		HasMore:    page.HasNextPage,
		LedgerPath: accountingSyncLedgerPath,
	}
	for _, record := range page.Items {
		if record == nil {
			continue
		}
		result.Records = append(result.Records, accountingSyncRecordRowFrom(record))
	}
	if result.NextCursor, err = page.NextCursor(); err != nil {
		return nil, fmt.Errorf("encode the next page's cursor: %w", err)
	}

	return result, nil
}

type getAccountingSyncRecordTool struct {
	ledger accountingSyncLedgerReader
}

func newGetAccountingSyncRecordTool(ledger accountingSyncLedgerReader) serviceports.AgentQueryTool {
	return &getAccountingSyncRecordTool{ledger: ledger}
}

func provideGetAccountingSyncRecordTool(
	ledger serviceports.AccountingSyncService,
) serviceports.AgentQueryTool {
	return newGetAccountingSyncRecordTool(ledger)
}

func (t *getAccountingSyncRecordTool) Name() string { return "get_accounting_sync_record" }

func (t *getAccountingSyncRecordTool) Description() string {
	return "Get one accounting sync record with its status, why it last failed and every try. " +
		"It names the document it sends, gives the plain-language resolution, and the " +
		"accounting system's document number and link once synced. Use it to explain why a " +
		"document did not reach the books before retrying or skipping it. It never returns " +
		"what was sent or any credential."
}

func (t *getAccountingSyncRecordTool) SearchTerms() []string {
	return []string{"attempts", "tries", "sync error"}
}

func (t *getAccountingSyncRecordTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramSyncRecordID: jsonschemautils.Text(
			"The sync record's id, from list_accounting_sync_records, " +
				"get_record_accounting_sync_state, or the run's subject.",
		),
	}, paramSyncRecordID)
}

func (t *getAccountingSyncRecordTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceAccountingSync})
}

func (t *getAccountingSyncRecordTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	id, err := requirePulid(params.Params, paramSyncRecordID)
	if err != nil {
		return nil, err
	}

	tenant := tenantOf(params)
	record, err := t.ledger.GetRecord(ctx, tenant, id)
	if err != nil {
		return nil, err
	}
	attempts, err := t.ledger.ListAttempts(ctx, tenant, id)
	if err != nil {
		return nil, err
	}

	detail := &accountingSyncRecordDetail{
		Record:          accountingSyncRecordRowFrom(record),
		WaitsOnRecordID: pulidString(record.DependsOnRecordID),
		AttemptHistory:  make([]accountingSyncAttemptRow, 0, len(attempts)),
		WhatThisMeans:   accountingSyncMeaning(record, "the accounting system"),
	}
	for _, attempt := range attempts {
		if attempt == nil {
			continue
		}
		detail.AttemptHistory = append(detail.AttemptHistory, accountingSyncAttemptRow{
			Number:        attempt.AttemptNumber,
			Outcome:       string(attempt.Outcome),
			ErrorCategory: string(attempt.ErrorCategory),
			StartedOn:     recordedDate(attempt.StartedAt),
			DurationMs:    attempt.DurationMs,
		})
	}

	return detail, nil
}

type getRecordAccountingSyncStateTool struct {
	ledger accountingSyncLedgerReader
}

func newGetRecordAccountingSyncStateTool(
	ledger accountingSyncLedgerReader,
) serviceports.AgentQueryTool {
	return &getRecordAccountingSyncStateTool{ledger: ledger}
}

func provideGetRecordAccountingSyncStateTool(
	ledger serviceports.AccountingSyncService,
) serviceports.AgentQueryTool {
	return newGetRecordAccountingSyncStateTool(ledger)
}

func (t *getRecordAccountingSyncStateTool) Name() string {
	return "get_record_accounting_sync_state"
}

func (t *getRecordAccountingSyncStateTool) Description() string {
	return "Get whether Trenova invoices, credit and debit memos, customer payments or " +
		"customers reached the accounting system, by their Trenova ids. Each answer says " +
		"whether the document is tracked, its current sync record with status and " +
		"resolution, and what that means. Use it when asked whether a particular document " +
		"synced; a document with no record was never sent, usually because sending had not " +
		"started or it is dated before the start date."
}

func (t *getRecordAccountingSyncStateTool) SearchTerms() []string {
	return []string{"reach", "synced", "did it sync"}
}

func (t *getRecordAccountingSyncStateTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramSyncDocumentIDs: jsonschemautils.DescribedArray(
			fmt.Sprintf(
				"Up to %d Trenova document ids, from %s, or the record on screen.",
				syncStateMaxDocuments,
				syncDocumentIDSources,
			),
			jsonschemautils.String(0),
			syncStateMaxDocuments,
		),
	}, paramSyncDocumentIDs)
}

func (t *getRecordAccountingSyncStateTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceAccountingSync})
}

func (t *getRecordAccountingSyncStateTool) documentIDs(
	params *serviceports.QueryToolParams,
) ([]pulid.ID, error) {
	raw := sliceutils.DedupeStrings(optionalStrings(params.Params, paramSyncDocumentIDs))
	if len(raw) == 0 {
		return nil, fmt.Errorf(
			"parameter %q must name at least one document id",
			paramSyncDocumentIDs,
		)
	}
	if len(raw) > syncStateMaxDocuments {
		return nil, fmt.Errorf(
			"parameter %q holds %d ids, more than the %d one call looks up; split it",
			paramSyncDocumentIDs, len(raw), syncStateMaxDocuments,
		)
	}

	ids := make([]pulid.ID, 0, len(raw))
	for _, value := range raw {
		id, err := pulid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf(
				"%q in %q is not a valid id: %w",
				value,
				paramSyncDocumentIDs,
				err,
			)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (t *getRecordAccountingSyncStateTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	ids, err := t.documentIDs(params)
	if err != nil {
		return nil, err
	}

	states, err := t.ledger.ObjectStates(ctx, tenantOf(params), ids)
	if err != nil {
		return nil, err
	}

	result := &accountingSyncStatesResult{States: make([]accountingSyncStateRow, 0, len(ids))}
	for _, id := range ids {
		state, ok := states[id]
		if !ok || state == nil || state.Record == nil {
			result.States = append(result.States, accountingSyncStateRow{
				DocumentID: id.String(),
				WhatThisMeans: "Nothing was sent for this document: sending has not started, " +
					"it is dated before the start date, or it is not a document that syncs.",
			})
			continue
		}
		row := accountingSyncRecordRowFrom(state.Record)
		result.States = append(result.States, accountingSyncStateRow{
			DocumentID:    id.String(),
			Tracked:       true,
			Provider:      state.ProviderName,
			Record:        &row,
			WhatThisMeans: accountingSyncMeaning(state.Record, state.ProviderName),
		})
	}

	return result, nil
}
