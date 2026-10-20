package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	paramSyncStatus        = "status"
	paramSyncDocumentType  = "documentType"
	paramSyncErrorCategory = "errorCategory"
	paramDocumentID        = "documentId"
	paramDocumentIDs       = "documentIds"
	paramSyncRecordID      = "syncRecordId"

	syncRecordsDefaultLimit = 25
	syncRecordsMaxLimit     = 50
	syncStateMaxDocuments   = 50

	absentNotSynced    = "not synced"
	absentNotScheduled = "not scheduled"

	syncDocumentIDSources = "invoice_list, invoice_get, receivable_list_customer_payments, " +
		"settlement_list_carrier or settlement_list_driver"
)

// syncDocumentKinds are the records a sync record can stand for: the invoices,
// payments, applications and settlements sent as documents, the customers,
// carriers and workers sent as the accounts they post to, and the journal
// entries posted one by one. A journal summary is keyed by its day, not by a
// record id, so it is not among them.
var syncDocumentKinds = []permission.RecordKind{
	permission.RecordKind(permission.ResourceInvoice),
	permission.RecordKind(permission.ResourceCustomerPayment),
	permission.KindCreditMemoApplication,
	permission.RecordKind(permission.ResourceCarrierSettlement),
	permission.RecordKind(permission.ResourceDriverSettlement),
	permission.RecordKind(permission.ResourceCustomer),
	permission.RecordKind(permission.ResourceCarrier),
	permission.RecordKind(permission.ResourceWorker),
	permission.RecordKind(permission.ResourceJournalEntry),
}

type accountingSyncLedgerReader interface {
	ListRecords(
		ctx context.Context,
		req *accountingsyncports.ListAccountingSyncRecordsRequest,
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
		req *accountingsyncports.AccountingSyncObjectStatesRequest,
	) (map[pulid.ID]*accountingsyncports.AccountingSyncObjectState, error)
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
		SyncedOn:       expectedDate(typeutils.ValueOrZero(record.SyncedAt), absentNotSynced),
		ExternalNumber: record.ExternalDocNumber,
		ExternalURL:    record.ExternalURL,
		SkippedReason:  record.SkippedReason,
		CanRetry:       record.Status.Retryable(),
		CanSkip: !record.Status.IsFinal() &&
			record.Status != accountingsync.SyncStatusInFlight,
	}
	if record.Status.Dispatchable() {
		row.NextAttemptOn = expectedDate(
			typeutils.ValueOrZero(record.NextAttemptAt),
			absentNotScheduled,
		)
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

type listAccountingSyncRecordsTool struct {
	ledger accountingSyncLedgerReader
}

func newListAccountingSyncRecordsTool(
	ledger accountingSyncLedgerReader,
) agentports.AgentQueryTool {
	return &listAccountingSyncRecordsTool{ledger: ledger}
}

func provideListAccountingSyncRecordsTool(
	ledger accountingsyncports.AccountingSyncService,
) agentports.AgentQueryTool {
	return newListAccountingSyncRecordsTool(ledger)
}

func (t *listAccountingSyncRecordsTool) Name() string { return "accounting_list_sync_records" }

func (t *listAccountingSyncRecordsTool) Description() string {
	return "List the documents Trenova sends to the accounting system and where each " +
		"stands, or say whether particular Trenova documents or parties reached it. " +
		"Filter one system's records, newest first, by status (Blocked and DeadLettered " +
		"did not reach the books), document type or error category, or search document " +
		"numbers. Or pass documentIds of invoices, memos, payments, customers, settlements, " +
		"carriers or drivers: each gets whether it is tracked, its current record and what " +
		"that means, across every syncing system unless you name one; a settlement shows " +
		"its bill or bill payment, whichever needs attention, else the latest, and no " +
		"record means it was never sent. A record has its status, error category, " +
		"resolution, tries and the accounting system's number once synced; pass its id " +
		"to accounting_get_sync_record for the tries or accounting_retry_sync_record once " +
		"the cause is fixed."
}

func (t *listAccountingSyncRecordsTool) SearchTerms() []string {
	return []string{"failed", "blocked", "ledger", "not synced", "synced", "did it sync"}
}

func (t *listAccountingSyncRecordsTool) ParamSchema() map[string]any {
	system := accountingSystemParam()
	system[toolschema.KeyDescription] = "The accounting system, such as \"QuickBooksOnline\" " +
		"or \"Xero\". Required to list records; with documentIds, only that system's " +
		"records answer, and without it every system that syncs does."

	return jsonschemautils.Object(map[string]any{
		paramAccountingSystem: system,
		paramDocumentIDs: agenttoolschema.OfKinds(jsonschemautils.DescribedArray(
			fmt.Sprintf(
				"Up to %d Trenova document ids, from %s, or the record on screen, to say "+
					"whether each reached the accounting system. The filters below list "+
					"records instead and are left out with it.",
				syncStateMaxDocuments,
				syncDocumentIDSources,
			),
			jsonschemautils.String(0),
			syncStateMaxDocuments,
		), syncDocumentKinds...),
		paramSyncStatus: jsonschemautils.DescribedArray(
			"Only records in these statuses. Blocked and DeadLettered need a person; "+
				"AwaitingApproval waits to be released.",
			agenttoolschema.Enum("A sync status.", agenttoolschema.SyncStatuses),
			len(accountingsync.AllSyncStatuses()),
		),
		paramSyncDocumentType: jsonschemautils.DescribedArray(
			"Only these kinds of document.",
			agenttoolschema.Enum("A document type.", agenttoolschema.SyncObjectTypes),
			len(accountingsync.AllSyncObjectTypes()),
		),
		paramSyncErrorCategory: jsonschemautils.DescribedArray(
			"Only records that last failed for these reasons.",
			agenttoolschema.Enum("An error category.", agenttoolschema.SyncErrorCategories),
			len(accountingsync.AllSyncErrorCategories()),
		),
		paramQuery: jsonschemautils.Text(
			"Words to find in Trenova or accounting system document numbers.",
		),
		paramLimit: limitSchema(syncRecordsDefaultLimit, syncRecordsMaxLimit),
		paramAfter: jsonschemautils.Text(
			"The nextCursor from a previous call of this tool, for the next page.",
		),
	})
}

func (t *listAccountingSyncRecordsTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceAccountingSync})
}

func (t *listAccountingSyncRecordsTool) request(
	params *agentports.QueryToolParams,
) (*accountingsyncports.ListAccountingSyncRecordsRequest, error) {
	system, err := requireAccountingSystem(params.Params)
	if err != nil {
		return nil, fmt.Errorf(
			"%w to list sync records; to ask whether particular documents synced, pass %q",
			err, paramDocumentIDs,
		)
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

	return &accountingsyncports.ListAccountingSyncRecordsRequest{
		TenantInfo:      tenantOf(params),
		IntegrationType: system,
		Statuses:        statuses,
		ObjectTypes:     types,
		ErrorCategories: categories,
		Search:          optionalString(params.Params, paramQuery),
		Cursor:          cursor,
	}, nil
}

func (t *listAccountingSyncRecordsTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	ids, err := syncDocumentIDs(params.Params)
	if err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		return t.states(ctx, params, ids)
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

var syncListOnlyParams = []string{
	paramSyncStatus,
	paramSyncDocumentType,
	paramSyncErrorCategory,
	paramQuery,
	paramAfter,
}

func syncDocumentIDs(params map[string]any) ([]pulid.ID, error) {
	raw := sliceutils.DedupeStrings(optionalStrings(params, paramDocumentIDs))
	if len(raw) == 0 {
		return nil, nil
	}
	if len(raw) > syncStateMaxDocuments {
		return nil, fmt.Errorf(
			"parameter %q holds %d ids, more than the %d one call looks up; split it",
			paramDocumentIDs, len(raw), syncStateMaxDocuments,
		)
	}
	for _, key := range syncListOnlyParams {
		if len(optionalStrings(params, key)) > 0 {
			return nil, fmt.Errorf(
				"parameter %q lists records and does not apply to %q, which answers for "+
					"each document; leave one of them out",
				key, paramDocumentIDs,
			)
		}
	}

	ids := make([]pulid.ID, 0, len(raw))
	for _, value := range raw {
		id, err := pulid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf(
				"%q in %q is not a valid id: %w",
				value,
				paramDocumentIDs,
				err,
			)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (t *listAccountingSyncRecordsTool) states(
	ctx context.Context,
	params *agentports.QueryToolParams,
	ids []pulid.ID,
) (*accountingSyncStatesResult, error) {
	req := &accountingsyncports.AccountingSyncObjectStatesRequest{
		TenantInfo: tenantOf(params),
		ObjectIDs:  ids,
	}
	if optionalString(params.Params, paramAccountingSystem) != "" {
		system, err := requireAccountingSystem(params.Params)
		if err != nil {
			return nil, err
		}
		req.IntegrationType = system
	}

	states, err := t.ledger.ObjectStates(ctx, req)
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

type getAccountingSyncRecordTool struct {
	ledger accountingSyncLedgerReader
}

func newGetAccountingSyncRecordTool(ledger accountingSyncLedgerReader) agentports.AgentQueryTool {
	return &getAccountingSyncRecordTool{ledger: ledger}
}

func provideGetAccountingSyncRecordTool(
	ledger accountingsyncports.AccountingSyncService,
) agentports.AgentQueryTool {
	return newGetAccountingSyncRecordTool(ledger)
}

func (t *getAccountingSyncRecordTool) Name() string { return "accounting_get_sync_record" }

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
		paramSyncRecordID: agenttoolschema.OfKinds(jsonschemautils.Text(
			"The sync record's id, from accounting_list_sync_records or the run's subject.",
		), permission.KindAccountingSyncRecord),
	}, paramSyncRecordID)
}

func (t *getAccountingSyncRecordTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceAccountingSync})
}

func (t *getAccountingSyncRecordTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
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
