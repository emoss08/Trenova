package deskbench

import (
	"math"
)

const (
	DefaultPassK = 4
	wilsonZ      = 1.959964
)

var tCritical95 = [...]float64{
	0, 12.706, 4.303, 3.182, 2.776, 2.571, 2.447, 2.365, 2.306, 2.262, 2.228,
	2.201, 2.179, 2.160, 2.145, 2.131, 2.120, 2.110, 2.101, 2.093, 2.086,
	2.080, 2.074, 2.069, 2.064, 2.060, 2.056, 2.052, 2.048, 2.045, 2.042,
}

type Stat struct {
	N    int     `json:"n"`
	Mean float64 `json:"mean"`
	Low  float64 `json:"low"`
	High float64 `json:"high"`
}

func statOf(values []float64) Stat {
	n := len(values)
	if n == 0 {
		return Stat{}
	}

	sum := 0.0
	for _, value := range values {
		sum += value
	}
	mean := sum / float64(n)
	stat := Stat{N: n, Mean: mean, Low: mean, High: mean}
	if n < 2 {
		return stat
	}

	squares := 0.0
	for _, value := range values {
		squares += (value - mean) * (value - mean)
	}
	stderr := math.Sqrt(squares/float64(n-1)) / math.Sqrt(float64(n))
	margin := tCritical(n-1) * stderr
	stat.Low = mean - margin
	stat.High = mean + margin

	return stat
}

func tCritical(df int) float64 {
	if df <= 0 {
		return 0
	}
	if df < len(tCritical95) {
		return tCritical95[df]
	}

	return wilsonZ
}

type Rate struct {
	Successes int     `json:"successes"`
	Trials    int     `json:"trials"`
	Value     float64 `json:"value"`
	Low       float64 `json:"low"`
	High      float64 `json:"high"`
}

func rateOf(successes, trials int) Rate {
	if trials <= 0 {
		return Rate{}
	}

	p := float64(successes) / float64(trials)
	n := float64(trials)
	z2 := wilsonZ * wilsonZ
	denominator := 1 + z2/n
	center := (p + z2/(2*n)) / denominator
	margin := wilsonZ * math.Sqrt(p*(1-p)/n+z2/(4*n*n)) / denominator

	return Rate{
		Successes: successes,
		Trials:    trials,
		Value:     p,
		Low:       math.Max(0, center-margin),
		High:      math.Min(1, center+margin),
	}
}

func passHatK(passed, trials, k int) *float64 {
	if k <= 0 || trials < k {
		return nil
	}

	estimate := 1.0
	for idx := range k {
		estimate *= float64(passed-idx) / float64(trials-idx)
		if estimate <= 0 {
			estimate = 0
			break
		}
	}

	return &estimate
}

func meanOf(values []float64) float64 {
	return statOf(values).Mean
}
