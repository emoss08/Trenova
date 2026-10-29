package accountingtools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonschemautils"
)

type CheckAccountingConnectionTool struct {
	status accountingconnectionports.AccountingConnectionStatusReader
	health accountingconnectionports.AccountingHealthChecker
}

func newCheckAccountingConnectionTool(
	status accountingconnectionports.AccountingConnectionStatusReader,
	health accountingconnectionports.AccountingHealthChecker,
) agenttoolports.AgentTool {
	return &CheckAccountingConnectionTool{status: status, health: health}
}

func ProvideCheckAccountingConnectionTool(
	status accountingconnectionports.AccountingConnectionStatusReader,
	health accountingconnectionports.AccountingHealthChecker,
) agenttoolports.AgentTool {
	return newCheckAccountingConnectionTool(status, health)
}

func (t *CheckAccountingConnectionTool) Name() string { return "accounting_check_connection" }

func (t *CheckAccountingConnectionTool) Recipe() []string {
	return []string{"accounting_check_connection", "accounting_get_connection"}
}

func (t *CheckAccountingConnectionTool) Description() string {
	return "Check the accounting system connection now and record whether it answers. " +
		"It renews the authorization if it is about to expire and refreshes the company's " +
		"details. Use it after a person " +
		"says they fixed something on the accounting side, or before retrying work that " +
		"failed because the connection was down. It changes nothing in the books. It cannot " +
		"reconnect a revoked authorization; a person must do that from the integrations page."
}

func (t *CheckAccountingConnectionTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		"system": agenttoolschema.Enum(
			"The accounting system, such as \"QuickBooksOnline\" or \"Xero\".",
			agenttoolschema.AccountingSystems,
		),
	}, "system")
}

func (t *CheckAccountingConnectionTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceAccountingIntegration,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    false,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Asks the accounting system whether it answers and records the result in " +
			"Trenova; it writes nothing to the books.",
	}
}

func (t *CheckAccountingConnectionTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	_, err := t.connection(ctx, params)
	return err
}

func (t *CheckAccountingConnectionTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	conn, err := t.connection(ctx, params)
	if err != nil {
		return err
	}

	_, err = t.health.CheckHealth(ctx, toolparams.TenantFrom(params), conn.ID)
	return err
}

func (t *CheckAccountingConnectionTool) connection(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*accountingconnection.AccountingConnection, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}

	system, err := toolparams.RequireString(params.Params, "system")
	if err != nil {
		return nil, err
	}
	typ := integration.Type(system)
	if !accountingprovider.SupportsAccountingSync(typ) {
		return nil, agenttoolschema.ErrAccountingSystem()
	}

	status, err := t.status.Status(ctx, toolparams.TenantFrom(params), typ)
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
