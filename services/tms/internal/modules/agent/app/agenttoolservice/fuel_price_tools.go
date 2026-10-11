package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/rating/app/fuelsurchargeservice"
)

var _ receivabletools.FuelIndexPriceKeeper = (*fuelsurchargeservice.Service)(nil)
