package agentquerytoolservice

import (
	"context"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDriverPay struct {
	codes       []*driverpay.PayCode
	profiles    []*driverpay.PayProfile
	assignments []*driverpay.WorkerPayAssignment
	escrow      []*driverpay.EscrowAccount
	advances    []*driverpay.PayAdvance
	deductions  []*driverpay.RecurringDeduction
	earnings    []*driverpay.RecurringEarning

	codeRequest    repositories.ListActivePayCodesRequest
	escrowRequest  *repositories.ListEscrowAccountsRequest
	advanceRequest *repositories.ListPayAdvancesRequest
}

func (f *fakeDriverPay) ListActivePayCodes(
	_ context.Context,
	req repositories.ListActivePayCodesRequest,
) ([]*driverpay.PayCode, error) {
	f.codeRequest = req

	return f.codes, nil
}

func (f *fakeDriverPay) ListProfiles(
	context.Context,
	*repositories.ListPayProfilesRequest,
) (*pagination.ListResult[*driverpay.PayProfile], error) {
	return &pagination.ListResult[*driverpay.PayProfile]{Items: f.profiles}, nil
}

func (f *fakeDriverPay) ListWorkerAssignments(
	context.Context,
	repositories.ListWorkerPayAssignmentsRequest,
) ([]*driverpay.WorkerPayAssignment, error) {
	return f.assignments, nil
}

func (f *fakeDriverPay) ListEscrowAccounts(
	_ context.Context,
	req *repositories.ListEscrowAccountsRequest,
) (*pagination.ListResult[*driverpay.EscrowAccount], error) {
	f.escrowRequest = req

	return &pagination.ListResult[*driverpay.EscrowAccount]{Items: f.escrow}, nil
}

func (f *fakeDriverPay) ListAdvances(
	_ context.Context,
	req *repositories.ListPayAdvancesRequest,
) (*pagination.ListResult[*driverpay.PayAdvance], error) {
	f.advanceRequest = req
	if req.Status == "" {
		return &pagination.ListResult[*driverpay.PayAdvance]{Items: f.advances}, nil
	}

	matched := make([]*driverpay.PayAdvance, 0, len(f.advances))
	for _, advance := range f.advances {
		if advance.Status == req.Status {
			matched = append(matched, advance)
		}
	}

	return &pagination.ListResult[*driverpay.PayAdvance]{Items: matched}, nil
}

func (f *fakeDriverPay) ListDeductions(
	context.Context,
	*repositories.ListRecurringDeductionsRequest,
) (*pagination.ListResult[*driverpay.RecurringDeduction], error) {
	return &pagination.ListResult[*driverpay.RecurringDeduction]{Items: f.deductions}, nil
}

func (f *fakeDriverPay) ListEarnings(
	context.Context,
	*repositories.ListRecurringEarningsRequest,
) (*pagination.ListResult[*driverpay.RecurringEarning], error) {
	return &pagination.ListResult[*driverpay.RecurringEarning]{Items: f.earnings}, nil
}

func newFakeDriverPayTool(fake *fakeDriverPay) driverPayTool {
	return newDriverPayTool(fake, &fakePermissions{})
}

func TestListEscrowAccounts_WithholdsTheBalanceBelowRestricted(t *testing.T) {
	t.Parallel()

	worker := pulid.MustNew("wrk_")
	fake := &fakeDriverPay{escrow: []*driverpay.EscrowAccount{{
		ID:                 pulid.MustNew("escr_"),
		WorkerID:           worker,
		Status:             driverpay.EscrowAccountStatusActive,
		BalanceMinor:       125000,
		TargetAmountMinor:  250000,
		AnnualInterestRate: decimal.RequireFromString("2.5"),
		OpenedDate:         1_780_000_000,
	}}}
	tool := &listEscrowAccountsTool{newFakeDriverPayTool(fake)}

	withheld, err := tool.Query(t.Context(), agentParams(map[string]any{
		"workerId": worker.String(),
		"status":   "Active",
	}, ""))
	require.NoError(t, err)
	assert.Equal(t, worker, fake.escrowRequest.WorkerID)
	assert.Equal(t, driverpay.EscrowAccountStatusActive, fake.escrowRequest.Status)
	outcome := withheld.(*gatedOutcome)
	assert.Empty(t, outcome.Items.([]escrowAccountRow)[0].Balance)
	assert.Contains(t, outcome.Withheld, "balance")

	shown, err := tool.Query(t.Context(), agentParams(map[string]any{},
		permission.SensitivityRestricted))
	require.NoError(t, err)
	row := shown.(*gatedOutcome).Items.([]escrowAccountRow)[0]
	assert.Equal(t, "1250.00", row.Balance)
	assert.Equal(t, "2500.00", row.Target)
	assert.Equal(t, "2.5", row.InterestRate)

	_, err = tool.Query(t.Context(), agentParams(map[string]any{"status": "Frozen"}, ""))
	require.Error(t, err)
}

func TestListWorkerPayAssignments_SaysWhichIsInForce(t *testing.T) {
	t.Parallel()

	now := timeutils.NowUnix()
	ended := now - 100
	fake := &fakeDriverPay{assignments: []*driverpay.WorkerPayAssignment{
		{
			ID:            pulid.MustNew("wpa_"),
			PayProfileID:  pulid.MustNew("dpp_"),
			EffectiveFrom: now - 50,
			SplitPercent:  decimal.NewFromInt(100),
			PayProfile:    &driverpay.PayProfile{Name: "Team linehaul"},
		},
		{
			ID:            pulid.MustNew("wpa_"),
			PayProfileID:  pulid.MustNew("dpp_"),
			EffectiveFrom: now - 1000,
			EffectiveTo:   &ended,
			SplitPercent:  decimal.NewFromInt(50),
		},
	}}
	tool := &listWorkerPayAssignmentsTool{newFakeDriverPayTool(fake)}

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"workerId": pulid.MustNew("wrk_").String(),
	}, permission.SensitivityRestricted))
	require.NoError(t, err)

	view := result.(payAssignmentsView)
	require.Len(t, view.Assignments, 2)
	assert.True(t, view.Assignments[0].InForce)
	assert.Equal(t, "Team linehaul", view.Assignments[0].PayProfile)
	assert.False(t, view.Assignments[1].InForce)
	assert.Equal(t, "50", view.Assignments[1].SplitPercent)

	_, err = tool.Query(t.Context(), agentParams(map[string]any{}, ""))
	require.Error(t, err)
}

func TestListPayCodes_NarrowsByDirection(t *testing.T) {
	t.Parallel()

	account := pulid.MustNew("gla_")
	fake := &fakeDriverPay{codes: []*driverpay.PayCode{{
		ID:          pulid.MustNew("payc_"),
		Code:        "BONUS",
		Name:        "Safety bonus",
		Direction:   driverpay.PayCodeDirectionEarning,
		GLAccountID: &account,
	}}}
	tool := &listPayCodesTool{newFakeDriverPayTool(fake)}

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"direction": "Earning",
	}, ""))
	require.NoError(t, err)
	assert.Equal(t, driverpay.PayCodeDirectionEarning, fake.codeRequest.Direction)
	rows := result.(*searchOutcome).Items.([]payCodeRow)
	require.Len(t, rows, 1)
	assert.Equal(t, account.String(), rows[0].GLAccountID)
}

func TestListRecurringDeductions_NeverShowsACourtOrder(t *testing.T) {
	t.Parallel()

	fake := &fakeDriverPay{deductions: []*driverpay.RecurringDeduction{{
		ID:               pulid.MustNew("rded_"),
		WorkerID:         pulid.MustNew("wrk_"),
		Kind:             driverpay.DeductionKindGarnishment,
		Status:           driverpay.DeductionStatusActive,
		Description:      "Child support",
		CourtOrderNumber: "CO-99",
		AmountMinor:      15000,
	}}}
	tool := &listRecurringDeductionsTool{newFakeDriverPayTool(fake)}

	result, err := tool.Query(t.Context(), agentParams(map[string]any{},
		permission.SensitivityRestricted))
	require.NoError(t, err)
	rows := result.(*gatedOutcome).Items.([]recurringRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "150.00", rows[0].Amount)
	assert.NotContains(t, rows[0].Description, "CO-99")
}

func TestListPayAdvances_ShowsWhatIsStillOwed(t *testing.T) {
	t.Parallel()

	fake := &fakeDriverPay{advances: []*driverpay.PayAdvance{{
		ID:             pulid.MustNew("padv_"),
		WorkerID:       pulid.MustNew("wrk_"),
		Status:         driverpay.AdvanceStatusPartiallyRecovered,
		Source:         driverpay.AdvanceSourceEFSMoneyCode,
		AmountMinor:    30000,
		RecoveredMinor: 10000,
	}}}
	tool := &listPayAdvancesTool{newFakeDriverPayTool(fake)}

	result, err := tool.Query(t.Context(), agentParams(map[string]any{},
		permission.SensitivityRestricted))
	require.NoError(t, err)
	row := result.(*gatedOutcome).Items.([]payAdvanceRow)[0]
	assert.Equal(t, "200.00", row.StillOwed)

	withheld, err := tool.Query(t.Context(), agentParams(map[string]any{}, ""))
	require.NoError(t, err)
	assert.Empty(t, withheld.(*gatedOutcome).Items.([]payAdvanceRow)[0].Amount)
}

type fakeDriverNames struct{ workers []*worker.Worker }

func (f *fakeDriverNames) List(
	_ context.Context,
	req *repositories.ListWorkersRequest,
) (*pagination.CursorListResult[*worker.Worker], error) {
	matched := make([]*worker.Worker, 0, len(f.workers))
	for _, item := range f.workers {
		full := strings.ToLower(item.FirstName + " " + item.LastName)
		if strings.Contains(full, strings.ToLower(req.Filter.Query)) {
			matched = append(matched, item)
		}
	}

	return &pagination.CursorListResult[*worker.Worker]{Items: matched}, nil
}

/*
"take 150 out of emily chen's settlement for the fuel advance" sent the
advances list her name, which it did not take, and then sent it as query,
the name every other list takes it by. A name that fits one driver
narrows the list to them; one that fits nobody or several is refused with who
it fits.
*/
func TestListPayAdvances_NarrowsToTheDriverNamed(t *testing.T) {
	t.Parallel()

	emily := &worker.Worker{ID: pulid.MustNew("wrk_"), FirstName: "Emily", LastName: "Chen"}
	emil := &worker.Worker{ID: pulid.MustNew("wrk_"), FirstName: "Emil", LastName: "Chenko"}
	fake := &fakeDriverPay{}
	tool := &listPayAdvancesTool{newFakeDriverPayTool(fake)}
	tool.workers = &fakeDriverNames{workers: []*worker.Worker{emily, emil}}

	_, err := tool.Query(t.Context(), agentParams(map[string]any{"query": "emily chen"},
		permission.SensitivityRestricted))
	require.NoError(t, err)
	require.NotNil(t, fake.advanceRequest)
	assert.Equal(t, emily.ID, fake.advanceRequest.WorkerID)

	_, err = tool.Query(t.Context(), agentParams(map[string]any{"query": "chen"},
		permission.SensitivityRestricted))
	require.Error(t, err)
	assert.Contains(t, err.Error(), emily.ID.String())
	assert.Contains(t, err.Error(), emil.ID.String())

	_, err = tool.Query(t.Context(), agentParams(map[string]any{"query": "nobody"},
		permission.SensitivityRestricted))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search_worker")
}

/*
gpt-6-luna asked for David Park's outstanding advances, was told none matched
and that others "may exist", and asked again without the status. An empty
status-filtered list says what the driver has in any status, or that there
is none at all, so the next call is the proposal.
*/
func TestListPayAdvances_AnEmptyStatusSaysWhatOtherStatusesHold(t *testing.T) {
	t.Parallel()

	worker := pulid.MustNew("wrk_")
	recovered := &driverpay.PayAdvance{
		ID: pulid.MustNew("padv_"), WorkerID: worker, Status: driverpay.AdvanceStatusRecovered,
		Reference: "FUEL-0412", IssuedDate: 1790000000, CurrencyCode: "USD",
	}
	fake := &fakeDriverPay{advances: []*driverpay.PayAdvance{recovered}}
	tool := &listPayAdvancesTool{newFakeDriverPayTool(fake)}

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"workerId": worker.String(), "status": "Outstanding",
	}, permission.SensitivityRestricted))
	require.NoError(t, err)
	note := result.(*gatedOutcome).Note
	assert.Contains(t, note, "FUEL-0412 (Recovered")
	assert.NotContains(t, note, "may exist")

	fake.advances = nil
	result, err = tool.Query(t.Context(), agentParams(map[string]any{
		"workerId": worker.String(), "status": "Outstanding",
	}, permission.SensitivityRestricted))
	require.NoError(t, err)
	assert.Contains(t, result.(*gatedOutcome).Note, "no pay advances in any status")
}
