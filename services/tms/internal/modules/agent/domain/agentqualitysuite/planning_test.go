package agentqualitysuite_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const day = int64(24 * 60 * 60)

func TestDecideSkip(t *testing.T) {
	t.Parallel()

	now := int64(1_800_000_000)
	last := &agentqualitysuite.LastRun{
		FingerprintHash: "fp",
		SuiteRevision:   "rev",
		FinishedAt:      now - 2*day,
	}

	tests := []struct {
		name string
		in   agentqualitysuite.SkipInput
		skip bool
	}{
		{
			name: "nothing changed and the last run is recent",
			in: agentqualitysuite.SkipInput{
				FingerprintHash: "fp",
				SuiteRevision:   "rev",
				Last:            last,
				Now:             now,
				ForceRerunDays:  7,
			},
			skip: true,
		},
		{
			name: "the agent changed",
			in: agentqualitysuite.SkipInput{
				FingerprintHash: "fp2",
				SuiteRevision:   "rev",
				Last:            last,
				Now:             now,
				ForceRerunDays:  7,
			},
		},
		{
			name: "the cases changed",
			in: agentqualitysuite.SkipInput{
				FingerprintHash: "fp",
				SuiteRevision:   "rev2",
				Last:            last,
				Now:             now,
				ForceRerunDays:  7,
			},
		},
		{
			name: "the last run is too old to stand for the agent",
			in: agentqualitysuite.SkipInput{
				FingerprintHash: "fp",
				SuiteRevision:   "rev",
				Last:            last,
				Now:             now,
				ForceRerunDays:  2,
			},
		},
		{
			name: "the agent has never completed a run",
			in: agentqualitysuite.SkipInput{
				FingerprintHash: "fp", SuiteRevision: "rev", Now: now, ForceRerunDays: 7,
			},
		},
		{
			name: "a person asked for the run",
			in: agentqualitysuite.SkipInput{
				FingerprintHash: "fp", SuiteRevision: "rev", Last: last, Now: now,
				ForceRerunDays: 7, Forced: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decision := agentqualitysuite.DecideSkip(tt.in)
			assert.Equal(t, tt.skip, decision.Skip)
			if tt.skip {
				assert.NotEmpty(t, decision.Reason)
			}
		})
	}
}

func samplingCases(
	sources map[agentquality.CaseSource]int,
	failing int,
) []agentqualitysuite.SamplingCase {
	cases := make([]agentqualitysuite.SamplingCase, 0, 64)
	failed := 0
	for _, source := range agentquality.AllCaseSources() {
		for range sources[source] {
			evalCase := agentqualitysuite.SamplingCase{
				ID:      pulid.MustNew("aec_"),
				Source:  source,
				Version: 1,
				Trigger: agent.RunTriggerChat,
			}
			if failed < failing {
				failed++
				evalCase.LastFailedAt = int64(1_000 + failed)
			}
			cases = append(cases, evalCase)
		}
	}

	return cases
}

func ids(cases []agentqualitysuite.SamplingCase) []pulid.ID {
	out := make([]pulid.ID, 0, len(cases))
	for _, evalCase := range cases {
		out = append(out, evalCase.ID)
	}

	return out
}

func TestSampleCases_IsDeterministicForARun(t *testing.T) {
	t.Parallel()

	cases := samplingCases(map[agentquality.CaseSource]int{
		agentquality.CaseSourceDecidedProposal: 40,
		agentquality.CaseSourceThumbsUp:        30,
		agentquality.CaseSourceCurated:         10,
	}, 0)
	runID := pulid.MustNew(agentqualitysuite.SuiteRunIDPrefix)
	seed := agentqualitysuite.SeedFromID(runID)
	assert.Equal(t, seed, agentqualitysuite.SeedFromID(runID), "a run's seed never changes")
	assert.GreaterOrEqual(t, seed, int64(0))

	first := agentqualitysuite.SampleCases(cases, seed, 20)
	reversed := make([]agentqualitysuite.SamplingCase, len(cases))
	for idx := range cases {
		reversed[len(cases)-1-idx] = cases[idx]
	}
	second := agentqualitysuite.SampleCases(reversed, seed, 20)

	require.Len(t, first, 20)
	assert.Equal(t, ids(first), ids(second))

	other := agentqualitysuite.SampleCases(cases, seed+1, 20)
	assert.NotEqual(t, ids(first), ids(other))
}

func TestSampleCases_PutsRecentFailuresFirstAndStratifiesTheRest(t *testing.T) {
	t.Parallel()

	cases := samplingCases(map[agentquality.CaseSource]int{
		agentquality.CaseSourceDecidedProposal: 50,
		agentquality.CaseSourceThumbsUp:        30,
		agentquality.CaseSourceCurated:         20,
	}, 3)

	sample := agentqualitysuite.SampleCases(cases, 42, 23)
	require.Len(t, sample, 23)

	for idx := range 3 {
		assert.True(t, sample[idx].Failing(), "failures come first")
	}
	assert.Greater(t, sample[0].LastFailedAt, sample[1].LastFailedAt, "newest failure first")

	counts := map[agentquality.CaseSource]int{}
	seen := map[pulid.ID]struct{}{}
	for _, evalCase := range sample[3:] {
		counts[evalCase.Source]++
		_, dup := seen[evalCase.ID]
		assert.False(t, dup, "a case is drawn once")
		seen[evalCase.ID] = struct{}{}
	}
	assert.Equal(t, 10, counts[agentquality.CaseSourceDecidedProposal])
	assert.Equal(t, 6, counts[agentquality.CaseSourceThumbsUp])
	assert.Equal(t, 4, counts[agentquality.CaseSourceCurated])
}

func TestSampleCases_TakesEveryCaseWhenThereIsRoom(t *testing.T) {
	t.Parallel()

	cases := samplingCases(map[agentquality.CaseSource]int{
		agentquality.CaseSourceCurated:  3,
		agentquality.CaseSourceThumbsUp: 2,
	}, 0)

	assert.Len(t, agentqualitysuite.SampleCases(cases, 7, 50), 5)
	assert.Empty(t, agentqualitysuite.SampleCases(cases, 7, 0))
	assert.Empty(t, agentqualitysuite.SampleCases(nil, 7, 10))
}

func TestSuiteRevision_MovesWithTheCases(t *testing.T) {
	t.Parallel()

	cases := samplingCases(map[agentquality.CaseSource]int{agentquality.CaseSourceCurated: 3}, 0)
	revision := agentqualitysuite.SuiteRevision(cases)

	reordered := []agentqualitysuite.SamplingCase{cases[2], cases[0], cases[1]}
	assert.Equal(t, revision, agentqualitysuite.SuiteRevision(reordered))

	edited := append([]agentqualitysuite.SamplingCase(nil), cases...)
	edited[1].Version = 2
	assert.NotEqual(t, revision, agentqualitysuite.SuiteRevision(edited))
	assert.NotEqual(t, revision, agentqualitysuite.SuiteRevision(cases[:2]))
}

func TestJudgeSample(t *testing.T) {
	t.Parallel()

	candidates := make([]pulid.ID, 0, 10)
	for range 10 {
		candidates = append(candidates, pulid.MustNew("aeval_"))
	}

	sample := agentqualitysuite.JudgeSample(candidates, 9, 0.2)
	assert.Len(t, sample, 2)
	assert.Equal(t, sample, agentqualitysuite.JudgeSample(candidates, 9, 0.2))
	assert.Len(t, agentqualitysuite.JudgeSample(candidates[:1], 9, 0.01), 1)
	assert.Empty(t, agentqualitysuite.JudgeSample(candidates, 9, 0))
}

func TestCheckBudget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		nightly string
		monthly string
		stop    bool
	}{
		{name: "room in both", nightly: "1.00", monthly: "10.00"},
		{name: "tonight's cap reached", nightly: "5.00", monthly: "10.00", stop: true},
		{name: "the month's cap reached", nightly: "1.00", monthly: "50.00", stop: true},
		{name: "over both", nightly: "7.50", monthly: "60.00", stop: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decision := agentqualitysuite.CheckBudget(agentqualitysuite.BudgetInput{
				NightlySpent: decimal.RequireFromString(tt.nightly),
				MonthlySpent: decimal.RequireFromString(tt.monthly),
				NightlyCap:   agentqualitycontrol.DefaultNightlyBudgetUSD,
				MonthlyCap:   agentqualitycontrol.DefaultMonthlyBudgetUSD,
			})
			assert.Equal(t, tt.stop, decision.Stop)
			if tt.stop {
				assert.NotEmpty(t, decision.Reason)
			}
		})
	}

	zero := agentqualitysuite.CheckBudget(agentqualitysuite.BudgetInput{
		NightlySpent: decimal.Zero,
		MonthlySpent: decimal.Zero,
		NightlyCap:   decimal.Zero,
		MonthlyCap:   decimal.Zero,
	})
	assert.True(t, zero.Stop, "a zero budget stops every replay")
}

func TestCheckBudget_HoldsWhatRunningJudgeBatchesWillCost(t *testing.T) {
	t.Parallel()

	room := agentqualitysuite.CheckBudget(agentqualitysuite.BudgetInput{
		NightlySpent: decimal.RequireFromString("3.00"),
		MonthlySpent: decimal.RequireFromString("10.00"),
		NightlyCap:   agentqualitycontrol.DefaultNightlyBudgetUSD,
		MonthlyCap:   agentqualitycontrol.DefaultMonthlyBudgetUSD,
		Committed:    decimal.RequireFromString("0.50"),
	})
	assert.False(t, room.Stop)
	assert.True(t, room.Room.Equal(decimal.RequireFromString("1.50")), room.Room.String())

	held := agentqualitysuite.CheckBudget(agentqualitysuite.BudgetInput{
		NightlySpent: decimal.RequireFromString("4.60"),
		MonthlySpent: decimal.RequireFromString("10.00"),
		NightlyCap:   agentqualitycontrol.DefaultNightlyBudgetUSD,
		MonthlyCap:   agentqualitycontrol.DefaultMonthlyBudgetUSD,
		Committed:    decimal.RequireFromString("0.40"),
	})
	assert.True(t, held.Stop, "spend and what running batches will cost reach the cap")
	assert.Contains(t, held.Reason, "held for judge batches")
}
