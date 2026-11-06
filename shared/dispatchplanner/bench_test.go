package dispatchplanner

import (
	"context"
	"fmt"
	"testing"
	"time"
)

const (
	benchCandidates = 40
	benchMaxChain   = 3
	benchRounds     = 25
	benchBudget     = 2 * time.Second
)

func BenchmarkSolve(b *testing.B) {
	for _, size := range []int{500, 2_000, 5_000} {
		board := newSynthBoard(uint64(size), size, size)
		problem, _ := board.problem(benchCandidates, Options{
			MaxPerResource: benchMaxChain,
			Iterations:     benchRounds * size,
			Budget:         benchBudget,
			RandomSeed:     1,
			Alternatives:   3,
		})
		b.Run(fmt.Sprintf("moves=%d_drivers=%d", size, size), func(b *testing.B) {
			var (
				result Result
				calls  int
			)

			b.ResetTimer()
			for range b.N {
				timeline := &synthTimeline{board: board}
				run := *problem
				run.Timeline = timeline

				var err error
				result, err = Solve(context.Background(), &run)
				if err != nil {
					b.Fatal(err)
				}
				calls = timeline.calls
			}
			b.StopTimer()

			metrics := board.measure(&result, benchMaxChain)
			if !metrics.feasible {
				b.Fatal("the planner produced an infeasible chain")
			}
			b.ReportMetric(100*float64(metrics.covered)/float64(size), "covered_%")
			b.ReportMetric(metrics.emptyMiles, "empty_mi")
			b.ReportMetric(float64(metrics.lateMinutes), "late_min")
			b.ReportMetric(float64(result.SeedObjective), "seed_cents")
			b.ReportMetric(float64(result.Objective), "plan_cents")
			b.ReportMetric(float64(result.Stats.Iterations), "iterations")
			b.ReportMetric(float64(calls), "oracle_calls")
		})
	}
}
