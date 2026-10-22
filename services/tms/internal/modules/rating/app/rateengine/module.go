package rateengine

import (
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"go.uber.org/fx"
)

func asRateEngine(s *Service) ratequoteports.RateEngine {
	return s
}

func asRecords(s *tariff.RecordSource) Records {
	return s
}

var Module = fx.Module(
	"rateengine",
	fx.Provide(
		New,
		asRateEngine,
		asRecords,
	),
)
