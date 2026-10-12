package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ratetools"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateimportservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratesimulationservice"
)

var _ ratetools.RateImportKeeper = (*rateimportservice.Service)(nil)

var _ ratetools.RateSimulationKeeper = (*ratesimulationservice.Service)(nil)
