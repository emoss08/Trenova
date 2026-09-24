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
)

type accountingStatusReader interface {
	Status(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		typ integration.Type,
	) (*serviceports.AccountingSyncStatus, error)
}

type accountingSyncStatusRow struct {
	ID                  string       `json:"id,omitempty"`
	System              string       `json:"system"`
	Provider            string       `json:"provider"`
	AvailableOnInstance bool         `json:"availableOnInstance"`
	Connected           bool         `json:"connected"`
	Status              string       `json:"status"`
	Company             string       `json:"company,omitempty"`
	HomeCurrency        string       `json:"homeCurrency,omitempty"`
	MultiCurrency       bool         `json:"multiCurrency"`
	BooksClosedThrough  optionalDate `json:"booksClosedThrough"`
	LastCheckedOn       optionalDate `json:"lastCheckedOn"`
	LastSuccessOn       optionalDate `json:"lastSuccessOn"`
	ConsecutiveFailures int          `json:"consecutiveFailures"`
	LastErrorCategory   string       `json:"lastErrorCategory,omitempty"`
	LastError           string       `json:"lastError,omitempty"`
	ReconnectBy         optionalDate `json:"reconnectBy"`
	LastWebhookOn       optionalDate `json:"lastWebhookOn"`
	SetupPath           string       `json:"setupPath"`
	WhatThisMeans       string       `json:"whatThisMeans"`
}

type getAccountingSyncStatusTool struct {
	accounting accountingStatusReader
}

func newGetAccountingSyncStatusTool(accounting accountingStatusReader) serviceports.AgentQueryTool {
	return &getAccountingSyncStatusTool{accounting: accounting}
}

func provideGetAccountingSyncStatusTool(
	accounting serviceports.AccountingConnectionService,
) serviceports.AgentQueryTool {
	return newGetAccountingSyncStatusTool(accounting)
}

func (t *getAccountingSyncStatusTool) Name() string { return "get_accounting_sync_status" }

func (t *getAccountingSyncStatusTool) Description() string {
	return "Get whether the accounting system is connected, to which company, and why it last failed. " +
		"It also says when it last answered and when the authorization must be renewed. Use it " +
		"before explaining why something did not reach the books, or when asked whether " +
		"QuickBooks is connected. It returns status and dates, never credentials."
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

	status, err := t.accounting.Status(ctx, tenantOf(params), system)
	if err != nil {
		return nil, err
	}

	return accountingSyncStatusRowFrom(status), nil
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
