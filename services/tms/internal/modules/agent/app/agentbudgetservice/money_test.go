package agentbudgetservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMoney struct {
	day, run agent.MoneyUse
	asked    []*agentports.SumAutomaticMoneyRequest
	err      error
}

func (f *fakeMoney) SumAutomaticMoney(
	_ context.Context,
	req *agentports.SumAutomaticMoneyRequest,
) (agent.MoneyUse, error) {
	f.asked = append(f.asked, req)
	if f.err != nil {
		return agent.MoneyUse{}, f.err
	}
	if req.RunID.IsNotNil() {
		return f.run, nil
	}

	return f.day, nil
}

type fixedBooks string

func (b fixedBooks) GetByOrgID(context.Context, pulid.ID) (*tenant.AccountingControl, error) {
	return &tenant.AccountingControl{FunctionalCurrencyCode: string(b)}, nil
}

type unreadableBooks struct{}

func (unreadableBooks) GetByOrgID(context.Context, pulid.ID) (*tenant.AccountingControl, error) {
	return nil, errors.New("books unavailable")
}

type fakeControls struct {
	control *tenant.AgentControl
	err     error
}

func (f fakeControls) GetOrCreate(context.Context, pagination.TenantInfo) (*tenant.AgentControl, error) {
	return f.control, f.err
}

func moneyControl() *tenant.AgentControl {
	control := &tenant.AgentControl{}
	control.ApplyAgentMoneyDefaults()

	return control
}

func TestMoneyAllowance_CountsTheRunTheDayAndWhatTheTurnHasNotRecorded(t *testing.T) {
	t.Parallel()

	counter := &fakeMoney{
		day: agent.MoneyUse{Calls: 7, AmountMinor: 70_000},
		run: agent.MoneyUse{Calls: 2, AmountMinor: 20_000},
	}
	svc := newService(nil, nil, nil)
	svc.money = counter
	svc.controls = fakeControls{control: moneyControl()}

	runID := pulid.MustNew("arun_")
	allowance, err := svc.MoneyAllowance(t.Context(), agentports.MoneyAllowanceRequest{
		Definition: definition(),
		RunID:      runID,
		Unrecorded: agent.MoneyUse{Calls: 1, AmountMinor: 5_000},
	})
	require.NoError(t, err)

	assert.Equal(t, agent.MoneyLimits{
		CallCeilingMinor: tenant.DefaultAgentMoneyCallCeilingMinor,
		RunCalls:         tenant.DefaultAgentMoneyRunCalls,
		RunAmountMinor:   tenant.DefaultAgentMoneyRunAmountMinor,
		DayCalls:         tenant.DefaultAgentMoneyDayCalls,
		DayAmountMinor:   tenant.DefaultAgentMoneyDayAmountMinor,
	}, allowance.Limits)
	assert.Equal(t, agent.MoneyUse{Calls: 3, AmountMinor: 25_000}, allowance.Run)
	assert.Equal(t, agent.MoneyUse{Calls: 8, AmountMinor: 75_000}, allowance.Day)
	assert.Equal(t, "USD", allowance.Currency)

	require.Len(t, counter.asked, 2)
	assert.Equal(t, "USD", counter.asked[0].Currency, "the day counts functional-currency writes")
	assert.Equal(t, "USD", counter.asked[1].Currency, "the run counts functional-currency writes")
	assert.False(t, counter.asked[1].Simulated)
	loc, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 3, 15, 0, 0, 0, 0, loc).Unix(), counter.asked[0].Since,
		"the day is the organization's own")
	assert.Equal(t, runID, counter.asked[1].RunID)
}

func TestMoneyAllowance_ATurnWithNoRunCountsOnlyItself(t *testing.T) {
	t.Parallel()

	counter := &fakeMoney{day: agent.MoneyUse{Calls: 1, AmountMinor: 100}}
	svc := newService(nil, nil, nil)
	svc.money = counter
	svc.controls = fakeControls{control: moneyControl()}

	allowance, err := svc.MoneyAllowance(t.Context(), agentports.MoneyAllowanceRequest{
		Definition: definition(),
		Unrecorded: agent.MoneyUse{Calls: 1, AmountMinor: 50},
	})
	require.NoError(t, err)
	assert.Equal(t, agent.MoneyUse{Calls: 1, AmountMinor: 50}, allowance.Run)
	assert.Len(t, counter.asked, 1)
}

func TestMoneyAllowance_FailsWhenAnythingCannotBeRead(t *testing.T) {
	t.Parallel()

	unreadable := newService(nil, nil, nil)
	unreadable.money = &fakeMoney{}
	unreadable.controls = fakeControls{err: errors.New("down")}
	_, err := unreadable.MoneyAllowance(t.Context(), agentports.MoneyAllowanceRequest{
		Definition: definition(),
	})
	require.Error(t, err)

	uncounted := newService(nil, nil, nil)
	uncounted.money = &fakeMoney{err: errors.New("down")}
	uncounted.controls = fakeControls{control: moneyControl()}
	_, err = uncounted.MoneyAllowance(t.Context(), agentports.MoneyAllowanceRequest{
		Definition: definition(),
	})
	require.Error(t, err)

	_, err = uncounted.MoneyAllowance(t.Context(), agentports.MoneyAllowanceRequest{})
	require.Error(t, err)
}

func TestMoneyAllowance_ASimulatedRunCountsItsSimulatedWritesAgainstTheRunAndTheDay(t *testing.T) {
	t.Parallel()

	counter := &fakeMoney{
		day: agent.MoneyUse{Calls: 4, AmountMinor: 40_000},
		run: agent.MoneyUse{Calls: 2, AmountMinor: 3_000},
	}
	svc := newService(nil, nil, nil)
	svc.money = counter
	svc.controls = fakeControls{control: moneyControl()}

	simulated := definition()
	simulated.SimulationMode = true
	allowance, err := svc.MoneyAllowance(t.Context(), agentports.MoneyAllowanceRequest{
		Definition: simulated,
		RunID:      pulid.MustNew("arun_"),
		Unrecorded: agent.MoneyUse{Calls: 1, AmountMinor: 500},
	})
	require.NoError(t, err)

	require.Len(t, counter.asked, 2)
	assert.False(t, counter.asked[0].Simulated, "the day is the organization's real writes")
	assert.True(t, counter.asked[1].Simulated, "the run is its own simulated writes")
	assert.Equal(t, agent.MoneyUse{Calls: 3, AmountMinor: 3_500}, allowance.Run)
	assert.Equal(t, agent.MoneyUse{Calls: 7, AmountMinor: 43_500}, allowance.Day,
		"a simulation holds where the real run would")
}

func TestMoneyAllowance_FailsWhenTheDayOrTheCurrencyCannotBePlaced(t *testing.T) {
	t.Parallel()

	noZone := newService(nil, nil, nil)
	noZone.money = &fakeMoney{}
	noZone.controls = fakeControls{control: moneyControl()}
	noZone.zones = unreadableZone{}
	_, err := noZone.MoneyAllowance(t.Context(), agentports.MoneyAllowanceRequest{
		Definition: definition(),
	})
	require.ErrorIs(t, err, errNoTimezone, "a day drawn in UTC instead is not the organization's day")

	badZone := newService(nil, nil, nil)
	badZone.money = &fakeMoney{}
	badZone.controls = fakeControls{control: moneyControl()}
	badZone.zones = fixedZone("Not/AZone")
	_, err = badZone.MoneyAllowance(t.Context(), agentports.MoneyAllowanceRequest{
		Definition: definition(),
	})
	require.ErrorIs(t, err, errNoTimezone)

	noBooks := newService(nil, nil, nil)
	noBooks.money = &fakeMoney{}
	noBooks.controls = fakeControls{control: moneyControl()}
	noBooks.books = unreadableBooks{}
	_, err = noBooks.MoneyAllowance(t.Context(), agentports.MoneyAllowanceRequest{
		Definition: definition(),
	})
	require.Error(t, err)

	blankCurrency := newService(nil, nil, nil)
	blankCurrency.money = &fakeMoney{}
	blankCurrency.controls = fakeControls{control: moneyControl()}
	blankCurrency.books = fixedBooks(" ")
	_, err = blankCurrency.MoneyAllowance(t.Context(), agentports.MoneyAllowanceRequest{
		Definition: definition(),
	})
	require.ErrorIs(t, err, errNoCurrency)
}
