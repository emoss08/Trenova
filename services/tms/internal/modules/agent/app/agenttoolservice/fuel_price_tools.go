package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/fueltools"
	"github.com/emoss08/trenova/internal/modules/rating/app/fuelsurchargeservice"
)

var _ fueltools.FuelIndexPriceKeeper = (*fuelsurchargeservice.Service)(nil)
