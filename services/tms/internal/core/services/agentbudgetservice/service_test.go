package agentbudgetservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeCost struct {
	cost  *repositories.AIUsageCost
	since int64
}

func (f *fakeCost) CostByDefinition(
	_ context.Context,
	req repositories.AIUsageCostRequest,
) (*repositories.AIUsageCost, error) {
	f.since = req.Since

	return f.cost, nil
}

type fakeRuns struct {
	started int
	since   int64
}

func (f *fakeRuns) CountSince(
	_ context.Context,
	req repositories.CountAgentRunsSinceRequest,
) (int, error) {
	f.since = req.Since

	return f.started, nil
}

type fakeTools struct {
	used map[string]int
}

func (f *fakeTools) CountExecutedTool(
	_ context.Context,
	req repositories.CountExecutedToolRequest,
) (int, error) {
	return f.used[req.ToolName], nil
}

type fixedZone string

func (z fixedZone) Timezone(context.Context, pagination.TenantInfo) string { return string(z) }

func money(value string) *decimal.Decimal {
	d := decimal.RequireFromString(value)

	return &d
}

func definition() *agentdefinition.Definition {
	return &agentdefinition.Definition{
		ID:             pulid.MustNew("agdef_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Night desk",
		ToolNames:      []string{"assign_move", "cancel_shipment"},
	}
}

// 2026-03-15 22:30 in Los Angeles, which is already the 16th in UTC. The
// windows must be drawn in the organization's zone or a Pacific desk's caps
// would reset at four in the afternoon.
const laEvening = 1773642600

func newService(cost *fakeCost, runs *fakeRuns, tools *fakeTools) *Service {
	return &Service{
		l:         zap.NewNop(),
		usage:     cost,
		runs:      runs,
		proposals: tools,
		zones:     fixedZone("America/Los_Angeles"),
		now:       func() int64 { return laEvening },
	}
}

func TestWindows_AreDrawnInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeCost{}, &fakeRuns{}, &fakeTools{})
	windows := svc.windows(t.Context(), pagination.TenantInfo{})

	loc, _ := time.LoadLocation("America/Los_Angeles")
	assert.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, loc).Unix(), windows.monthStart)
	assert.Equal(t, time.Date(2026, 3, 15, 0, 0, 0, 0, loc).Unix(), windows.dayStart)
	assert.Equal(t, time.Date(2026, 4, 1, 0, 0, 0, 0, loc).Unix(), windows.nextMonth)
	assert.Equal(t, time.Date(2026, 3, 16, 0, 0, 0, 0, loc).Unix(), windows.nextDay)
}

func TestCheckRun_RefusesWhenTheMonthlyBudgetIsSpent(t *testing.T) {
	t.Parallel()

	cost := &fakeCost{
		cost: &repositories.AIUsageCost{CostUSD: decimal.RequireFromString("25.50"), Calls: 40},
	}
	svc := newService(cost, &fakeRuns{}, &fakeTools{})
	d := definition()
	d.MonthlyBudgetUSD = money("25")

	refusal, err := svc.CheckRun(t.Context(), d)
	require.NoError(t, err)

	assert.Equal(t, services.BudgetCapMonthly, refusal.Cap)
	assert.Equal(t, "25.50", refusal.Spent)
	assert.Equal(t, "25.00", refusal.Limit)
	assert.Contains(
		t,
		refusal.Message("Night desk"),
		"spent its monthly budget (25.50 of 25.00 USD)",
	)
	assert.Equal(t, svc.windows(t.Context(), pagination.TenantInfo{}).monthStart, cost.since)
}

func TestCheckRun_AllowsUnderBudgetAndWithoutOne(t *testing.T) {
	t.Parallel()

	cost := &fakeCost{cost: &repositories.AIUsageCost{CostUSD: decimal.RequireFromString("3")}}
	svc := newService(cost, &fakeRuns{started: 2}, &fakeTools{})
	d := definition()
	d.MonthlyBudgetUSD = money("25")
	d.DailyRunLimit = 3

	refusal, err := svc.CheckRun(t.Context(), d)
	require.NoError(t, err)
	assert.False(t, refusal.Refused())

	refusal, err = svc.CheckRun(t.Context(), definition())
	require.NoError(t, err)
	assert.False(t, refusal.Refused(), "no caps means nothing to refuse")
}

func TestCheckRun_RefusesTheRunPastTheDailyLimit(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeCost{}, &fakeRuns{started: 3}, &fakeTools{})
	d := definition()
	d.DailyRunLimit = 3

	refusal, err := svc.CheckRun(t.Context(), d)
	require.NoError(t, err)

	assert.Equal(t, services.BudgetCapDailyRuns, refusal.Cap)
	assert.Contains(t, refusal.Message("Night desk"), "started its 3 runs for today")
}

func TestCheckTool_CountsOnlyToolsWithACap(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeCost{}, &fakeRuns{}, &fakeTools{used: map[string]int{"assign_move": 5}})
	d := definition()
	d.ToolDailyLimits = map[string]int{"assign_move": 5}

	refusal, err := svc.CheckTool(t.Context(), services.CheckToolBudgetRequest{Definition: d, ToolName: "assign_move"})
	require.NoError(t, err)
	assert.Equal(t, services.BudgetCapTool, refusal.Cap)
	assert.Equal(t, "assign_move", refusal.Tool)

	refusal, err = svc.CheckTool(t.Context(), services.CheckToolBudgetRequest{Definition: d, ToolName: "cancel_shipment"})
	require.NoError(t, err)
	assert.False(t, refusal.Refused())
}

// A turn records its writes when it ends, so a cap counted only from
// recorded writes let one turn run a capped tool as often as it liked.
func TestCheckTool_CountsWritesThisTurnHasNotRecordedYet(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeCost{}, &fakeRuns{}, &fakeTools{used: map[string]int{"assign_move": 3}})
	d := definition()
	d.ToolDailyLimits = map[string]int{"assign_move": 5}

	refusal, err := svc.CheckTool(t.Context(), services.CheckToolBudgetRequest{
		Definition: d, ToolName: "assign_move", Unrecorded: 1,
	})
	require.NoError(t, err)
	assert.False(t, refusal.Refused(), "four of five")

	refusal, err = svc.CheckTool(t.Context(), services.CheckToolBudgetRequest{
		Definition: d, ToolName: "assign_move", Unrecorded: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, services.BudgetCapTool, refusal.Cap)
	assert.Equal(t, "5", refusal.Spent)
}

func TestStatus_ReportsEveryCapAndTheUnpricedCalls(t *testing.T) {
	t.Parallel()

	cost := &fakeCost{cost: &repositories.AIUsageCost{
		CostUSD: decimal.RequireFromString("12.345"), Calls: 20, UnpricedCalls: 4,
	}}
	svc := newService(
		cost,
		&fakeRuns{started: 1},
		&fakeTools{used: map[string]int{"assign_move": 2}},
	)
	d := definition()
	d.MonthlyBudgetUSD = money("50")
	d.DailyRunLimit = 10
	d.ToolDailyLimits = map[string]int{"assign_move": 8}
	d.SimulationMode = true

	status, err := svc.Status(t.Context(), d)
	require.NoError(t, err)

	assert.Equal(t, "12.35", status.SpentUSD)
	assert.Equal(t, "50.00", *status.MonthlyBudget)
	assert.Equal(t, 4, status.UnpricedCalls)
	assert.Equal(t, 1, status.RunsToday)
	assert.Equal(
		t,
		[]services.ToolBudgetUse{{Tool: "assign_move", Used: 2, Limit: 8}},
		status.Tools,
	)
	assert.True(t, status.SimulationMode)
}
