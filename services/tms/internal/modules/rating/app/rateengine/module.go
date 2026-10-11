package rateengine

import (
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"go.uber.org/fx"
)

func asRateEngine(s *Service) ratequoteports.RateEngine {
	return s
}

// Module wires the rating engine as its own subsystem, alongside the formula
// engine it delegates to.
//
// It is a module rather than a handful of providers because the engine owns a
// small graph of its own — the resolver, the pricing paths, the matrix lookup
// and the trace — and grouping them keeps the composition root readable.
var Module = fx.Module(
	"rateengine",
	fx.Provide(
		New,
		asRateEngine,
	),
)
