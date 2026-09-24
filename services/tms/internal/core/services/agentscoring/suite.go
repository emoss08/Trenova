package agentscoring

import (
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DefaultRegressionThreshold = 0.05
	DefaultRegressionMinCases  = 10
	DefaultRegressionWindow    = 7
)

type CaseResult struct {
	CaseID      pulid.ID
	Source      agentquality.CaseSource
	Score       float64
	Passed      bool
	HardFailure bool
}

type Suite struct {
	Score        float64
	Cases        int
	Passed       int
	Failed       int
	HardFailures int
	Weight       float64
}

func ScoreSuite(results []CaseResult) Suite {
	suite := Suite{Cases: len(results)}
	weighted := 0.0
	for _, result := range results {
		weight := result.Source.Weight()
		suite.Weight += weight
		weighted += weight * result.Score
		if result.Passed {
			suite.Passed++
		} else {
			suite.Failed++
		}
		if result.HardFailure {
			suite.HardFailures++
		}
	}
	if suite.Weight > 0 {
		suite.Score = weighted / suite.Weight
	}

	return suite
}

type SuiteRun struct {
	Score       float64
	Cases       int
	CompletedAt int64
}

type RegressionPolicy struct {
	Threshold float64
	MinCases  int
	Window    int
}

func DefaultRegressionPolicy() RegressionPolicy {
	return RegressionPolicy{
		Threshold: DefaultRegressionThreshold,
		MinCases:  DefaultRegressionMinCases,
		Window:    DefaultRegressionWindow,
	}
}

type RegressionInput struct {
	Current  Suite
	Results  []CaseResult
	History  []SuiteRun
	Baseline []CaseResult
}

type Regression struct {
	Regressed       bool
	ScoreDropped    bool
	Median          float64
	Drop            float64
	Compared        int
	NewHardFailures []pulid.ID
}

func DetectRegression(in RegressionInput, policy RegressionPolicy) Regression {
	out := Regression{}

	window := recentRuns(in.History, policy.Window)
	out.Compared = len(window)
	if len(window) > 0 && in.Current.Cases >= policy.MinCases {
		scores := make([]float64, 0, len(window))
		for _, run := range window {
			scores = append(scores, run.Score)
		}
		out.Median = floatutils.Median(scores)
		out.Drop = out.Median - in.Current.Score
		out.ScoreDropped = out.Drop > policy.Threshold
	}

	passedAtBaseline := make(map[pulid.ID]struct{}, len(in.Baseline))
	for _, result := range in.Baseline {
		if result.Passed {
			passedAtBaseline[result.CaseID] = struct{}{}
		}
	}
	for _, result := range in.Results {
		if !result.HardFailure {
			continue
		}
		if _, passed := passedAtBaseline[result.CaseID]; passed {
			out.NewHardFailures = append(out.NewHardFailures, result.CaseID)
		}
	}

	out.Regressed = out.ScoreDropped || len(out.NewHardFailures) > 0

	return out
}

func recentRuns(history []SuiteRun, window int) []SuiteRun {
	if window <= 0 {
		window = DefaultRegressionWindow
	}
	sorted := slices.Clone(history)
	slices.SortStableFunc(sorted, func(a, b SuiteRun) int {
		switch {
		case a.CompletedAt > b.CompletedAt:
			return -1
		case a.CompletedAt < b.CompletedAt:
			return 1
		default:
			return 0
		}
	})
	if len(sorted) > window {
		sorted = sorted[:window]
	}

	return sorted
}
