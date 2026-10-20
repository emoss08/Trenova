package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/jsonschemautils"
)

const absentNoEntries = "no entries yet"

type factoringReserveBalanceRow struct {
	FactoringCompanyID string       `json:"factoringCompanyId"`
	ReserveHeld        string       `json:"reserveHeld,omitempty"`
	OwedToFactor       string       `json:"owedToFactor,omitempty"`
	Advanced           string       `json:"advanced,omitempty"`
	Released           string       `json:"released,omitempty"`
	ChargedBack        string       `json:"chargedBack,omitempty"`
	Fees               string       `json:"fees,omitempty"`
	EntryCount         int          `json:"entryCount"`
	LastMovement       optionalDate `json:"lastMovement"`
}

type getFactoringReserveBalanceTool struct {
	ledger factoringports.LedgerService
	access fieldAccess
}

func provideGetFactoringReserveBalanceTool(
	ledger factoringports.LedgerService,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &getFactoringReserveBalanceTool{ledger: ledger, access: newFieldAccess(permissions)}
}

func (t *getFactoringReserveBalanceTool) Name() string { return "factoring_get_reserve_balance" }

func (t *getFactoringReserveBalanceTool) Description() string {
	return "Read what a factor holds and has paid, from the factoring entries booked to the " +
		"ledger. It shows the reserve it still holds, what is owed it under secured borrowing, the " +
		"advances it paid, the reserves it released, what was paid back for chargebacks net " +
		"of reversals, the fees booked, and the day of the last movement. Use it before " +
		"factoring_release_reserve to see what the factor should remit."
}

func (t *getFactoringReserveBalanceTool) SearchTerms() []string {
	return []string{"factoring reserve", "reserve balance", "due from factor", "factor fees"}
}

func (t *getFactoringReserveBalanceTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramFactoringFactor: agenttoolschema.RecordIDText(
			permission.ResourceFactoringCompany,
			"The factor, from factoring_list_submissions or factoring_get_agreement.",
		),
	}, paramFactoringFactor)
}

func (t *getFactoringReserveBalanceTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFactoringReserve})
}

func (t *getFactoringReserveBalanceTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	factorID, err := requirePulid(params.Params, paramFactoringFactor)
	if err != nil {
		return nil, err
	}
	balance, err := t.ledger.GetBalance(ctx, factoringports.GetReserveBalanceRequest{
		TenantInfo:         tenantOf(params),
		FactoringCompanyID: factorID,
	})
	if err != nil {
		return nil, err
	}
	show, gate := amountsFor(ctx, t.access, params)
	row := factoringReserveBalanceRow{
		FactoringCompanyID: factorID.String(),
		ReserveHeld:        minorAmount(show, balance.ReserveMinor),
		OwedToFactor:       minorAmount(show, balance.LiabilityMinor),
		Advanced:           minorAmount(show, balance.AdvancedMinor),
		Released:           minorAmount(show, balance.ReleasedMinor),
		ChargedBack:        minorAmount(show, balance.ChargedBackMinor),
		Fees:               minorAmount(show, balance.FeeMinor),
		EntryCount:         balance.EntryCount,
		LastMovement:       pointerDateOr(balance.LastOccurredOn, absentNoEntries),
	}
	result := map[string]any{"balance": row}
	if gate != nil {
		result["withheldByAccess"] = gate.Withheld()
	}
	return result, nil
}
