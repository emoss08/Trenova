package dispatchplanner

import (
	"context"
	"math"
	"math/rand/v2"
	"slices"
)

const (
	checkEvery       = 16
	calibrationShare = 10
	maxCalibration   = 256
	coolingFloor     = 1e-3
	restartFloor     = 500
	restartPerTask   = 2

	rewardBest     = 5.0
	rewardImproved = 3.0
	rewardAccepted = 1.0
	segmentLength  = 100
	reaction       = 0.2
	minWeight      = 0.05
)

func (s *search) anneal(ctx context.Context) {
	limit := s.opt.Iterations
	if limit <= 0 || s.tasks == 0 || s.p.Resources == 0 {
		return
	}

	weights := newOperatorWeights(int(operatorCount))
	calibration := min(maxCalibration, max(limit/calibrationShare, 1))
	uphill := make([]int64, 0, calibration)
	restartAfter := max(restartFloor, restartPerTask*s.tasks)
	initial, temperature := 0.0, 0.0
	sinceBest := 0

	for iteration := range limit {
		if iteration == calibration {
			initial = initialTemperature(uphill)
			temperature = initial
		}
		if iteration%checkEvery == 0 {
			if s.expired(ctx) {
				break
			}
			if initial > 0 {
				temperature = initial * math.Pow(coolingFloor, s.progress(iteration))
			}
		}

		s.stats.Iterations++
		op := operator(weights.pick(s.rng))
		s.propose(op)
		if !s.choice.valid {
			weights.record(int(op), 0)
			continue
		}

		delta := s.choice.delta
		if iteration < calibration && delta > 0 {
			uphill = append(uphill, delta)
		}
		if !accepts(delta, temperature, s.rng) {
			weights.record(int(op), 0)
			sinceBest++
			s.maybeRestart(&sinceBest, restartAfter)
			continue
		}

		weights.record(int(op), s.accept(delta, &sinceBest))
		s.maybeRestart(&sinceBest, restartAfter)
	}
}

func (s *search) accept(delta int64, sinceBest *int) float64 {
	if delta > 0 && s.curIsBest {
		s.snapshot()
		s.curIsBest = false
	}
	s.apply(&s.choice)
	s.stats.Accepted++

	if s.cur.objective < s.bestObjective {
		s.bestObjective = s.cur.objective
		s.curIsBest = true
		s.stats.Improved++
		*sinceBest = 0
		return rewardBest
	}

	*sinceBest++
	if delta < 0 {
		return rewardImproved
	}

	return rewardAccepted
}

func (s *search) maybeRestart(sinceBest *int, after int) {
	if *sinceBest < after {
		return
	}
	*sinceBest = 0
	if s.curIsBest {
		return
	}
	s.restore()
	s.curIsBest = true
	s.stats.Restarts++
}

func accepts(delta int64, temperature float64, rng *rand.Rand) bool {
	if delta <= 0 {
		return true
	}
	if temperature <= 0 {
		return false
	}

	return rng.Float64() < math.Exp(-float64(delta)/temperature)
}

func initialTemperature(uphill []int64) float64 {
	if len(uphill) == 0 {
		return 1
	}
	sorted := slices.Clone(uphill)
	slices.Sort(sorted)

	return float64(sorted[len(sorted)/2]) / math.Ln2
}

type operatorWeights struct {
	weight []float64
	score  []float64
	uses   []int
	since  int
}

func newOperatorWeights(size int) *operatorWeights {
	w := &operatorWeights{
		weight: make([]float64, size),
		score:  make([]float64, size),
		uses:   make([]int, size),
	}
	for i := range w.weight {
		w.weight[i] = 1
	}

	return w
}

func (w *operatorWeights) pick(rng *rand.Rand) int {
	total := 0.0
	for _, weight := range w.weight {
		total += weight
	}

	roll := rng.Float64() * total
	for op, weight := range w.weight {
		roll -= weight
		if roll < 0 {
			return op
		}
	}

	return len(w.weight) - 1
}

func (w *operatorWeights) record(op int, reward float64) {
	w.score[op] += reward
	w.uses[op]++
	w.since++
	if w.since < segmentLength {
		return
	}

	for i := range w.weight {
		if w.uses[i] > 0 {
			w.weight[i] = (1-reaction)*w.weight[i] + reaction*w.score[i]/float64(w.uses[i])
		}
		w.weight[i] = max(w.weight[i], minWeight)
		w.score[i] = 0
		w.uses[i] = 0
	}
	w.since = 0
}
