package aitrace

import (
	"context"
	"sync/atomic"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const defaultAISamplingRate = 1.0

type aiSampling struct {
	rate    float64
	sampler sdktrace.Sampler
}

var (
	defaultAISampling = &aiSampling{
		rate:    defaultAISamplingRate,
		sampler: RatioSampler(defaultAISamplingRate),
	}
	currentAISampling atomic.Pointer[aiSampling]
)

func loadAISampling() *aiSampling {
	if current := currentAISampling.Load(); current != nil {
		return current
	}

	return defaultAISampling
}

func SetSamplingRate(rate float64) {
	currentAISampling.Store(&aiSampling{rate: rate, sampler: RatioSampler(rate)})
}

func SamplingRate() float64 {
	return loadAISampling().rate
}

func RatioSampler(rate float64) sdktrace.Sampler {
	if rate >= 1 {
		return sdktrace.AlwaysSample()
	}
	if rate <= 0 {
		return sdktrace.NeverSample()
	}

	return sdktrace.TraceIDRatioBased(rate)
}

func NewSampler(defaultRate float64) sdktrace.Sampler {
	return sdktrace.ParentBased(rootSampler{fallback: RatioSampler(defaultRate)})
}

type rootSampler struct {
	fallback sdktrace.Sampler
}

//nolint:gocritic // hugeParam: the signature is fixed by sdktrace.Sampler
func (s rootSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	if forced, ok := forcedAnchor(p.ParentContext); ok && forced.TraceID == p.TraceID {
		decision := sdktrace.Drop
		if forced.Sampled {
			decision = sdktrace.RecordAndSample
		}

		return sdktrace.SamplingResult{
			Decision:   decision,
			Tracestate: trace.SpanContextFromContext(p.ParentContext).TraceState(),
		}
	}

	return s.fallback.ShouldSample(p)
}

func (s rootSampler) Description() string {
	return "AIAnchorAware{fallback:" + s.fallback.Description() +
		",anchored:" + loadAISampling().sampler.Description() + " by derivation}"
}

func sampledByAIRate(traceID trace.TraceID) bool {
	result := loadAISampling().sampler.ShouldSample(sdktrace.SamplingParameters{
		ParentContext: context.Background(),
		TraceID:       traceID,
	})

	return result.Decision == sdktrace.RecordAndSample
}
