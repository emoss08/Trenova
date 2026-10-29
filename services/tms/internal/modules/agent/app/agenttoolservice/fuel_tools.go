package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/fueltools"
	"github.com/emoss08/trenova/internal/modules/settlement/app/fuelpurchaseservice"
)

var _ fueltools.FuelPurchaseKeeper = (*fuelpurchaseservice.Service)(nil)

var _ fueltools.FuelCardKeeper = (*fuelpurchaseservice.Service)(nil)

var _ fueltools.FuelImportKeeper = (*fuelpurchaseservice.Service)(nil)
