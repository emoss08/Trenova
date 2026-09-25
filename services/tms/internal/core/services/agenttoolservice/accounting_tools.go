package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type accountingConnectionChecker interface {
	Status(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		typ integration.Type,
	) (*serviceports.AccountingSyncStatus, error)
	CheckHealth(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		connectionID pulid.ID,
	) (*accountingsync.AccountingConnection, error)
}

type checkAccountingConnectionTool struct {
	accounting accountingConnectionChecker
}

func newCheckAccountingConnectionTool(
	accounting accountingConnectionChecker,
) serviceports.AgentTool {
	return &checkAccountingConnectionTool{accounting: accounting}
}

func provideCheckAccountingConnectionTool(
	accounting serviceports.AccountingConnectionService,
) serviceports.AgentTool {
	return newCheckAccountingConnectionTool(accounting)
}

func (t *checkAccountingConnectionTool) Name() string { return "check_accounting_connection" }

func (t *checkAccountingConnectionTool) Description() string {
	return "Check the accounting system connection now and record whether it answers. " +
		"It renews the authorization if it is about to expire and refreshes the company's " +
		"details. Use it after a person " +
		"says they fixed something on the accounting side, or before retrying work that " +
		"failed because the connection was down. It changes nothing in the books. It cannot " +
		"reconnect a revoked authorization; a person must do that from the integrations page."
}

func (t *checkAccountingConnectionTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		"system": jsonschemautils.Enum(
			"The accounting system. Example: \"QuickBooksOnline\".",
			string(integration.TypeQuickBooksOnline),
		),
	}, "system")
}

func (t *checkAccountingConnectionTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceAccountingIntegration,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    false,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Asks the accounting system whether it answers and records the result in " +
			"Trenova; it writes nothing to the books.",
	}
}

func (t *checkAccountingConnectionTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.connection(ctx, &params)
	return err
}

func (t *checkAccountingConnectionTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	conn, err := t.connection(ctx, &params)
	if err != nil {
		return err
	}

	_, err = t.accounting.CheckHealth(ctx, tenantFrom(params), conn.ID)
	return err
}

func (t *checkAccountingConnectionTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) (*agent.ToolSimulation, error) {
	conn, err := t.connection(ctx, &params)
	if err != nil {
		return nil, err
	}

	provider := accountingsync.ProviderName(conn.IntegrationType)
	return &agent.ToolSimulation{
		Summary: fmt.Sprintf(
			"Would check %s for %s now and record whether it answers. It is %s at the moment.",
			provider,
			conn.ExternalCompanyName,
			conn.Status,
		),
		Previewed: true,
		Changes: []agent.FieldChange{
			{Field: "lastCheckedAt", From: lastCheckedLabel(conn), To: "now"},
		},
	}, nil
}

func (t *checkAccountingConnectionTool) connection(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*accountingsync.AccountingConnection, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, err
	}

	system, err := requireString(params.Params, "system")
	if err != nil {
		return nil, err
	}
	typ := integration.Type(system)
	if !accountingsync.SupportsAccountingSync(typ) {
		return nil, fmt.Errorf("system must be %q", integration.TypeQuickBooksOnline)
	}

	status, err := t.accounting.Status(ctx, tenantFrom(*params), typ)
	if err != nil {
		return nil, err
	}
	if status.Connection == nil || !status.Connection.IsActive() {
		return nil, errortypes.NewBusinessError(
			"{0} is not connected. A person must connect it from the integrations page.",
			status.ProviderName,
		)
	}

	return status.Connection, nil
}

func lastCheckedLabel(conn *accountingsync.AccountingConnection) string {
	if conn.LastCheckedAt == nil {
		return "never"
	}
	return timeutils.FormatInstantUTC(*conn.LastCheckedAt)
}
