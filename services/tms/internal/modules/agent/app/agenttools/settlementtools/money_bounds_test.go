package settlementtools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func settlementBoundsWith(
	t *testing.T,
	tool agentports.AgentTool,
	args map[string]any,
) agentports.MoneyBound {
	t.Helper()

	policy, ok := agentports.PolicyOf(tool)
	require.True(t, ok)
	require.True(t, policy.Bounded(), "%s proves its bounds", policy.Name)

	return policy.Condition.Bounds(t.Context(), agenttoolstest.ExecuteParams(args))
}

func TestAddDriverSettlementAdjustmentBounds(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		status driversettlement.Status
		within bool
	}{
		{status: driversettlement.StatusDraft, within: true},
		{status: driversettlement.StatusPendingApproval, within: true},
		{status: driversettlement.StatusApproved},
		{status: driversettlement.StatusPosted},
	} {
		t.Run(string(tc.status), func(t *testing.T) {
			t.Parallel()

			entity := &driversettlement.Settlement{
				ID: pulid.MustNew("dstl_"), Status: tc.status, CurrencyCode: "USD",
			}
			book := &fakeSettlementBook[driversettlement.Settlement]{
				plan: driverPlanner(entity, nil),
			}
			tool := settlementTool(t, provideAddDriverSettlementAdjustmentTool, book)

			bound := settlementBoundsWith(t, tool, map[string]any{
				"settlementId": entity.ID.String(),
				"description":  "Detention at the receiver",
				"amount":       "-75.25",
			})
			assert.Equal(t, tc.within, bound.Within)
			if tc.within {
				assert.Equal(t, int64(-7_525), bound.AmountMinor)
				assert.Equal(t, "USD", bound.Currency, "the settlement's currency is what moves")
				return
			}
			assert.Equal(t, agent.HeldByMoneyRecordState, bound.Held)
		})
	}
}

func TestRemoveSettlementAdjustmentBounds(t *testing.T) {
	t.Parallel()

	agentLine, personLine := pulid.MustNew("dstll_"), pulid.MustNew("dstll_")
	earning := pulid.MustNew("dstll_")
	settlement := func(status driversettlement.Status) *driversettlement.Settlement {
		return &driversettlement.Settlement{
			ID:           pulid.MustNew("dstl_"),
			Status:       status,
			CurrencyCode: "CAD",
			Lines: []*driversettlement.SettlementLine{
				{
					ID:           agentLine,
					Category:     driversettlement.LineCategoryAdjustment,
					AmountMinor:  12_500,
					AddedByAgent: true,
				},
				{ID: personLine, Category: driversettlement.LineCategoryAdjustment, AmountMinor: 500},
				{ID: earning, Category: driversettlement.LineCategoryEarning, AmountMinor: 90_000},
			},
		}
	}

	cases := []struct {
		name   string
		status driversettlement.Status
		line   pulid.ID
		within bool
		held   string
	}{
		{name: "a line an agent added", status: driversettlement.StatusDraft, line: agentLine, within: true},
		{name: "a line a person added", status: driversettlement.StatusDraft, line: personLine, held: agent.HeldByMoneyAgentLine},
		{name: "a computed line", status: driversettlement.StatusDraft, line: earning, held: agent.HeldByMoneyUnproven},
		{name: "a line not on the settlement", status: driversettlement.StatusDraft, line: pulid.MustNew("dstll_"), held: agent.HeldByMoneyUnproven},
		{name: "an approved settlement", status: driversettlement.StatusApproved, line: agentLine, held: agent.HeldByMoneyRecordState},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			entity := settlement(tc.status)
			book := &fakeSettlementBook[driversettlement.Settlement]{
				plan: driverPlanner(entity, nil),
			}
			tool := settlementTool(t, provideRemoveDriverSettlementAdjustmentTool, book)

			bound := settlementBoundsWith(t, tool, map[string]any{
				"settlementId": entity.ID.String(),
				"lineId":       tc.line.String(),
			})
			assert.Equal(t, tc.within, bound.Within)
			if tc.within {
				assert.Equal(t, int64(12_500), bound.AmountMinor)
				assert.Equal(t, "CAD", bound.Currency)
				return
			}
			assert.Equal(t, tc.held, bound.Held)
		})
	}
}

func TestRemoveCarrierSettlementAdjustmentBounds(t *testing.T) {
	t.Parallel()

	agentLine := pulid.MustNew("carstll_")
	entity := &carriersettlement.CarrierSettlement{
		ID:           pulid.MustNew("carstl_"),
		Status:       carriersettlement.StatusPendingApproval,
		CurrencyCode: "USD",
		Lines: []*carriersettlement.CarrierSettlementLine{{
			ID:           agentLine,
			EventType:    carriersettlement.CostEventTypeAdjustment,
			AmountMinor:  -3_000,
			AddedByAgent: true,
		}},
	}
	book := &fakeSettlementBook[carriersettlement.CarrierSettlement]{
		plan: func(*settlementshared.ActionRequest) *settlementshared.ActionPlan[*carriersettlement.CarrierSettlement] {
			return &settlementshared.ActionPlan[*carriersettlement.CarrierSettlement]{
				Before: entity,
				After:  entity,
			}
		},
	}
	tool := settlementTool(t, provideRemoveCarrierSettlementAdjustmentTool, book)

	bound := settlementBoundsWith(t, tool, map[string]any{
		"settlementId": entity.ID.String(),
		"lineId":       agentLine.String(),
	})
	require.True(t, bound.Within)
	assert.Equal(t, int64(-3_000), bound.AmountMinor)
	assert.Equal(t, "USD", bound.Currency)
}

func TestSettlementAdjustmentRefusedByItsPlanIsUnproven(t *testing.T) {
	t.Parallel()

	entity := &driversettlement.Settlement{ID: pulid.MustNew("dstl_"), Status: driversettlement.StatusDraft}
	book := &fakeSettlementBook[driversettlement.Settlement]{
		plan: func(*settlementshared.ActionRequest) *settlementshared.ActionPlan[*driversettlement.Settlement] {
			return &settlementshared.ActionPlan[*driversettlement.Settlement]{
				Before:  entity,
				After:   driversettlementservice.CloneSettlement(entity),
				Refusal: assert.AnError,
			}
		},
	}
	tool := settlementTool(t, provideAddDriverSettlementAdjustmentTool, book)

	bound := settlementBoundsWith(t, tool, map[string]any{
		"settlementId": entity.ID.String(),
		"description":  "Lumper reimbursement",
		"amount":       "40.00",
	})
	assert.Equal(t, entity.ID, book.planned.SettlementID)
	assert.False(t, bound.Within)
	assert.Equal(t, agent.HeldByMoneyUnproven, bound.Held)
}

func TestAddedAdjustmentsAreMarkedAsAnAgents(t *testing.T) {
	t.Parallel()

	entity := &driversettlement.Settlement{ID: pulid.MustNew("dstl_"), Status: driversettlement.StatusDraft}
	book := &fakeSettlementBook[driversettlement.Settlement]{plan: driverPlanner(entity, nil)}
	tool := settlementTool(t, provideAddDriverSettlementAdjustmentTool, book)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"settlementId": entity.ID.String(),
		"description":  "Lumper reimbursement",
		"amount":       "40.00",
	})))
	require.NotNil(t, book.performed)
	require.NotNil(t, book.performed.Adjustment)
	assert.True(t, book.performed.Adjustment.ByAgent)
}
