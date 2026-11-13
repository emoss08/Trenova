package demandforecast

import "math"

const (
	maxCountTerms    = 2_000
	poissonTolerance = 1e-9
	scoreTolerance   = 1e-6
)

type Count struct {
	Mean     float64
	Variance float64
	Offset   int
}

func GammaPoisson(alpha, beta float64) Count {
	if alpha <= 0 || beta <= 0 {
		return Count{}
	}
	mean := alpha / beta

	return Count{Mean: mean, Variance: mean + mean/beta}
}

func Nowcast(alpha, beta float64, booked int, unbookedShare float64) Count {
	if alpha <= 0 || beta <= 0 || unbookedShare <= 0 {
		return Count{}
	}
	share := math.Min(unbookedShare, 1)
	shape := alpha + float64(max(booked, 0))
	rate := (beta + 1 - share) / share
	mean := shape / rate

	return Count{Mean: mean, Variance: mean + mean/rate}
}

func (c Count) Plus(other Count) Count {
	return Count{
		Mean:     c.Mean + other.Mean,
		Variance: c.Variance + other.Variance,
		Offset:   c.Offset + other.Offset,
	}
}

func (c Count) Shift(by int) Count {
	c.Offset += by

	return c
}

func (c Count) Scale(by float64) Count {
	if by == 1 || c.Mean <= 0 {
		return c
	}
	excess := math.Max(c.Variance-c.Mean, 0)
	c.Mean *= by
	c.Variance = c.Mean + by*by*excess

	return c
}

func (c Count) Widen(dispersion float64) Count {
	if dispersion > 1 {
		c.Variance *= dispersion
	}

	return c
}

func (c Count) Score(actual int) float64 {
	k := actual - c.Offset
	if c.Mean <= 0 {
		return math.Abs(float64(k))
	}
	score := float64(max(-k, 0))
	cumulative, last := 0.0, -1
	c.walk(func(n int, p float64) bool {
		cumulative += p
		last = n
		if n < k {
			score += cumulative * cumulative
			return true
		}
		tail := math.Max(0, 1-cumulative)
		score += tail * tail

		return tail > scoreTolerance
	})
	if last < k {
		score += float64(k - last - 1)
	}

	return score
}

func (c Count) Exceeds(k int) float64 {
	k -= c.Offset
	if k < 0 {
		return 1
	}
	if c.Mean <= 0 {
		return 0
	}
	below := 0.0
	c.walk(func(n int, p float64) bool {
		below += p
		return n < k
	})

	return math.Max(0, math.Min(1, 1-below))
}

func (c Count) Quantile(q float64) int {
	if c.Mean <= 0 {
		return c.Offset
	}
	target := math.Max(0, math.Min(q, 1))
	cumulative := 0.0
	at := 0
	c.walk(func(n int, p float64) bool {
		cumulative += p
		at = n
		return cumulative < target
	})

	return c.Offset + at
}

func (c Count) walk(visit func(n int, p float64) bool) {
	mean := c.Mean
	excess := c.Variance - mean
	if excess <= poissonTolerance*math.Max(mean, 1) {
		p := math.Exp(-mean)
		for n := 0; n < maxCountTerms; n++ {
			if !visit(n, p) {
				return
			}
			p *= mean / float64(n+1)
		}
		return
	}

	r := mean * mean / excess
	ratio := mean / (r + mean)
	p := math.Exp(r * math.Log(r/(r+mean)))
	for n := 0; n < maxCountTerms; n++ {
		if !visit(n, p) {
			return
		}
		p *= (float64(n) + r) / float64(n+1) * ratio
	}
}
