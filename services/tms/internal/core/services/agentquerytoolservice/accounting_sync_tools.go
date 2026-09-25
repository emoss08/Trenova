package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/sliceutils"
)

type accountingStatusReader interface {
	Status(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		typ integration.Type,
	) (*serviceports.AccountingSyncStatus, error)
}

type accountingSyncSummaryReader interface {
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) (*serviceports.AccountingSyncSummary, error)
}

const (
	sendingNotStarted = "NotStarted"
	sendingPaused     = "Paused"
	sendingAutomatic  = "Automatic"
	sendingOnRelease  = "OnRelease"

	absentNotChosen = "not chosen"
	absentNotPaused = "not paused"

	queueWithheldReason = "Queue depth needs permission to read the accounting sync ledger."
)

type accountingSyncStatusCountRow struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type accountingSyncCauseRow struct {
	Status         string       `json:"status"`
	ErrorCategory  string       `json:"errorCategory,omitempty"`
	Resolution     string       `json:"resolution,omitempty"`
	Count          int          `json:"count"`
	OldestQueuedOn optionalDate `json:"oldestQueuedOn"`
	SampleRecordID string       `json:"sampleRecordId,omitempty"`
}

type accountingBackfillRow struct {
	ID            string       `json:"id"`
	Status        string       `json:"status"`
	From          optionalDate `json:"from"`
	To            optionalDate `json:"to"`
	DocumentTypes []string     `json:"documentTypes"`
	Enqueued      int          `json:"enqueued"`
	AlreadyQueued int          `json:"alreadyQueued"`
}

type accountingSyncQueueRow struct {
	ByStatus       []accountingSyncStatusCountRow `json:"byStatus"`
	NeedsAttention []accountingSyncCauseRow       `json:"needsAttention"`
	ActiveBackfill *accountingBackfillRow         `json:"activeBackfill,omitempty"`
}

type accountingSyncStatusRow struct {
	ID                  string                  `json:"id,omitempty"`
	System              string                  `json:"system"`
	Provider            string                  `json:"provider"`
	AvailableOnInstance bool                    `json:"availableOnInstance"`
	Connected           bool                    `json:"connected"`
	Status              string                  `json:"status"`
	Company             string                  `json:"company,omitempty"`
	HomeCurrency        string                  `json:"homeCurrency,omitempty"`
	MultiCurrency       bool                    `json:"multiCurrency"`
	BooksClosedThrough  optionalDate            `json:"booksClosedThrough"`
	LastCheckedOn       optionalDate            `json:"lastCheckedOn"`
	LastSuccessOn       optionalDate            `json:"lastSuccessOn"`
	ConsecutiveFailures int                     `json:"consecutiveFailures"`
	LastErrorCategory   string                  `json:"lastErrorCategory,omitempty"`
	LastError           string                  `json:"lastError,omitempty"`
	ReconnectBy         optionalDate            `json:"reconnectBy"`
	LastWebhookOn       optionalDate            `json:"lastWebhookOn"`
	Sending             string                  `json:"sending"`
	StartDate           optionalDate            `json:"startDate"`
	AutomaticSending    bool                    `json:"automaticSending"`
	PausedOn            optionalDate            `json:"pausedOn"`
	PausedReason        string                  `json:"pausedReason,omitempty"`
	SetupPath           string                  `json:"setupPath"`
	WhatThisMeans       string                  `json:"whatThisMeans"`
	Queue               *accountingSyncQueueRow `json:"queue,omitempty"`
	QueueWithheld       string                  `json:"queueWithheld,omitempty"`
}

type getAccountingSyncStatusTool struct {
	accounting accountingStatusReader
	sync       accountingSyncSummaryReader
	access     fieldAccess
}

func newGetAccountingSyncStatusTool(
	accounting accountingStatusReader,
	sync accountingSyncSummaryReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getAccountingSyncStatusTool{
		accounting: accounting,
		sync:       sync,
		access:     newFieldAccess(permissions),
	}
}

func provideGetAccountingSyncStatusTool(
	accounting serviceports.AccountingConnectionService,
	sync serviceports.AccountingSyncService,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetAccountingSyncStatusTool(accounting, sync, permissions)
}

func (t *getAccountingSyncStatusTool) Name() string { return "get_accounting_sync_status" }

func (t *getAccountingSyncStatusTool) Description() string {
	return "Get whether the accounting system is connected, to which company, and why it last failed. " +
		"It also says when it last answered, when the authorization must be renewed, and " +
		"whether documents are being sent: sending is NotStarted, Paused, Automatic, or " +
		"OnRelease when each document waits for a person to release it, with the start " +
		"date. Once sending has started, queue counts how many documents are in each sync " +
		"status and groups those needing attention by cause. Use it before explaining why " +
		"something did not reach the books. It returns status and dates, never credentials."
}

func (t *getAccountingSyncStatusTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		"system": jsonschemautils.Enum(
			"The accounting system. Example: \"QuickBooksOnline\".",
			string(integration.TypeQuickBooksOnline),
		),
	}, "system")
}

func (t *getAccountingSyncStatusTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceAccountingIntegration})
}

func (t *getAccountingSyncStatusTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	system := integration.Type(optionalString(params.Params, "system"))
	if !accountingsync.SupportsAccountingSync(system) {
		return nil, fmt.Errorf("system must be %q", integration.TypeQuickBooksOnline)
	}

	tenant := tenantOf(params)
	status, err := t.accounting.Status(ctx, tenant, system)
	if err != nil {
		return nil, err
	}

	row := accountingSyncStatusRowFrom(status)
	if status.Connection == nil || !status.Connection.IsSyncing() {
		return row, nil
	}
	if !t.access.mayRead(ctx, params, permission.ResourceAccountingSync) {
		row.QueueWithheld = queueWithheldReason
		return row, nil
	}

	summary, err := t.sync.Summary(ctx, tenant, system)
	if err != nil {
		return nil, err
	}
	row.Queue = accountingSyncQueueFrom(summary)

	return row, nil
}

func accountingSyncQueueFrom(summary *serviceports.AccountingSyncSummary) *accountingSyncQueueRow {
	queue := &accountingSyncQueueRow{
		ByStatus:       make([]accountingSyncStatusCountRow, 0, len(summary.Counts)),
		NeedsAttention: make([]accountingSyncCauseRow, 0, len(summary.Attention)),
	}
	for _, count := range summary.Counts {
		queue.ByStatus = append(queue.ByStatus, accountingSyncStatusCountRow{
			Status: string(count.Status),
			Count:  count.Count,
		})
	}
	for _, group := range summary.Attention {
		queue.NeedsAttention = append(queue.NeedsAttention, accountingSyncCauseRow{
			Status:         string(group.Status),
			ErrorCategory:  string(group.ErrorCategory),
			Resolution:     group.Resolution,
			Count:          group.Count,
			OldestQueuedOn: recordedDate(group.OldestQueuedAt),
			SampleRecordID: pulidString(group.SampleRecordID),
		})
	}
	if backfill := summary.ActiveBackfill; backfill != nil {
		queue.ActiveBackfill = &accountingBackfillRow{
			ID:            backfill.ID.String(),
			Status:        string(backfill.Status),
			From:          recordedDate(backfill.RangeStart),
			To:            recordedDate(backfill.RangeEnd),
			DocumentTypes: sliceutils.Strings(backfill.ObjectTypes),
			Enqueued:      backfill.EnqueuedCount,
			AlreadyQueued: backfill.AlreadyQueuedCount,
		}
	}

	return queue
}

func accountingSending(conn *accountingsync.AccountingConnection) string {
	switch {
	case !conn.IsSyncing():
		return sendingNotStarted
	case conn.IsPaused():
		return sendingPaused
	case conn.AutoSync:
		return sendingAutomatic
	default:
		return sendingOnRelease
	}
}

func accountingSyncStatusRowFrom(
	status *serviceports.AccountingSyncStatus,
) accountingSyncStatusRow {
	row := accountingSyncStatusRow{
		System:              string(status.IntegrationType),
		Provider:            status.ProviderName,
		AvailableOnInstance: status.Available,
		SetupPath:           accountingsync.SetupPath(status.IntegrationType),
	}

	conn := status.Connection
	if conn == nil {
		row.Status = "NeverConnected"
		row.Sending = sendingNotStarted
		row.StartDate = expectedDate(0, absentNotChosen)
		row.PausedOn = expectedDate(0, absentNotPaused)
		row.WhatThisMeans = accountingStatusMeaning(status, "")
		return row
	}

	row.ID = conn.ID.String()
	row.Connected = conn.IsActive()
	row.Status = string(conn.Status)
	row.Company = conn.ExternalCompanyName
	row.HomeCurrency = conn.ExternalHomeCurrency
	row.MultiCurrency = conn.ExternalMultiCurrencyEnabled
	row.BooksClosedThrough = pointerDate(conn.ExternalBooksClosedThrough)
	row.LastCheckedOn = pointerDate(conn.LastCheckedAt)
	row.LastSuccessOn = pointerDate(conn.LastSuccessAt)
	row.ConsecutiveFailures = conn.ConsecutiveFailures
	row.LastErrorCategory = string(conn.LastErrorCategory)
	row.LastError = conn.AgentErrorSummary()
	row.ReconnectBy = recordedDate(conn.RefreshTokenAbsoluteExpiresAt)
	row.LastWebhookOn = pointerDate(conn.LastWebhookAt)
	row.Sending = accountingSending(conn)
	row.StartDate = expectedDate(derefInt64(conn.SyncStartDate), absentNotChosen)
	row.AutomaticSending = conn.IsSyncing() && conn.AutoSync
	row.PausedOn = expectedDate(derefInt64(conn.PausedAt), absentNotPaused)
	row.PausedReason = conn.PausedReason
	row.WhatThisMeans = accountingStatusMeaning(status, conn.Status)

	return row
}

func accountingStatusMeaning(
	status *serviceports.AccountingSyncStatus,
	connection accountingsync.ConnectionStatus,
) string {
	provider := status.ProviderName
	switch connection {
	case accountingsync.ConnectionStatusConnected:
		return provider + " is connected and answering."
	case accountingsync.ConnectionStatusDegraded:
		return "The last check failed; Trenova retries automatically."
	case accountingsync.ConnectionStatusFailing:
		return "Several checks failed in a row; nothing reaches the books until it recovers."
	case accountingsync.ConnectionStatusRevoked:
		return "The authorization was revoked or expired. A person must reconnect " + provider + "."
	case accountingsync.ConnectionStatusDisconnected:
		return "Someone disconnected " + provider + " on purpose."
	default:
		if !status.Available {
			return provider + " is not set up on this Trenova instance; an administrator must add the app credentials."
		}
		return provider + " has never been connected. A person connects it from the integrations page."
	}
}
