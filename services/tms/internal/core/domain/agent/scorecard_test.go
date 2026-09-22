package agent_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func totals(tools ...agent.ToolOutcomeCount) *agent.ScorecardTotals {
	return &agent.ScorecardTotals{Runs: 10, CostUSD: decimal.NewFromFloat(1.25), ByTool: tools}
}

func card(tools ...agent.ToolOutcomeCount) *agent.Scorecard {
	return agent.NewScorecard("agd_1", agent.ScorecardWindow30d, 100, totals(tools...))
}

/*
An approval says a person agreed. Whether the change happened is the execution
count, and the two come apart exactly when it matters most — a tool that is
approved every time and fails every time. Pricing approvals would let that
tool report the best week of any agent in the org.
*/
func TestScorecard_TimeSavedCountsWhatWentThrough(t *testing.T) {
	t.Parallel()

	approvedButNeverRan := card(agent.ToolOutcomeCount{
		ToolName: "assign_move",
		Approved: 20,
		Executed: 0,
		Failed:   20,
	})

	assert.Zero(t, approvedButNeverRan.EstimatedMinutesSaved)

	ran := card(agent.ToolOutcomeCount{ToolName: "assign_move", Approved: 20, Executed: 20})
	assert.Equal(t, 20*agent.MinutesSavedFor("assign_move"), ran.EstimatedMinutesSaved)
}

// A read tool is worth nothing: looking something up is what the person
// asked for, not work taken off them. Otherwise a chatty read-only agent
// out-scores one that does the work.
func TestMinutesSavedFor_ReadsAreWorthNothing(t *testing.T) {
	t.Parallel()

	assert.Zero(t, agent.MinutesSavedFor("raise_exception"))
	assert.Positive(t, agent.MinutesSavedFor("assign_move"))
}

// An unpriced write is worth the default, not zero. Silence in the table is
// not a claim that the tool saves nobody anything.
func TestMinutesSavedFor_AnUnlistedToolTakesTheDefault(t *testing.T) {
	t.Parallel()

	assert.Equal(t, agent.DefaultMinutesSaved, agent.MinutesSavedFor("a_tool_nobody_priced"))
}

/*
Pending is not a rejection. An agent whose first proposal is still waiting has
an unknown record, not a zero one, and rendering 0% next to it would read as a
verdict nobody reached.
*/
func TestScorecard_ApprovalRateIsUnknownUntilSomebodyDecides(t *testing.T) {
	t.Parallel()

	assert.Nil(t, card(agent.ToolOutcomeCount{ToolName: "assign_move", Pending: 3}).ApprovalRate())

	rate := card(agent.ToolOutcomeCount{
		ToolName: "assign_move",
		Approved: 3,
		Rejected: 1,
		Pending:  5,
	}).ApprovalRate()
	require.NotNil(t, rate)
	assert.InDelta(t, 0.75, *rate, 0.0001)
}

// A modification is not an approval: the agent was nearly right, and a
// scorecard that counted it as agreement would hide the correcting.
func TestScorecard_ModificationIsNotApproval(t *testing.T) {
	t.Parallel()

	rate := card(agent.ToolOutcomeCount{ToolName: "assign_move", Approved: 1, Modified: 1}).
		ApprovalRate()
	require.NotNil(t, rate)
	assert.InDelta(t, 0.5, *rate, 0.0001)
}

func TestScorecard_SumsAcrossTools(t *testing.T) {
	t.Parallel()

	summed := card(
		agent.ToolOutcomeCount{ToolName: "assign_move", Approved: 2, Executed: 2},
		agent.ToolOutcomeCount{ToolName: "email_customer", Approved: 1, Rejected: 1, Executed: 1},
	)

	assert.Equal(t, 3, summed.Approved)
	assert.Equal(t, 1, summed.Rejected)
	assert.Equal(t, 4, summed.Proposals)
	assert.Equal(
		t,
		2*agent.MinutesSavedFor("assign_move")+agent.MinutesSavedFor("email_customer"),
		summed.EstimatedMinutesSaved,
	)
}

// Empty slices rather than nil, so a client that maps over them does not have
// to know the difference between "no tools" and "not loaded".
func TestNewScorecard_NeverHandsBackNilSlices(t *testing.T) {
	t.Parallel()

	empty := agent.NewScorecard("agd_1", agent.ScorecardWindow7d, 0, &agent.ScorecardTotals{})

	assert.NotNil(t, empty.ByTool)
	assert.NotNil(t, empty.Trend)
}

func TestScorecardWindow_Days(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 7, agent.ScorecardWindow7d.Days())
	assert.Equal(t, 90, agent.ScorecardWindow90d.Days())
	// An unknown window covers a month rather than no time at all.
	assert.Equal(t, 30, agent.ScorecardWindow("Whenever").Days())
}
