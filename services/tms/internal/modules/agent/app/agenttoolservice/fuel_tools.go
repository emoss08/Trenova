package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/settlement/app/fuelpurchaseservice"
)

var _ receivabletools.FuelPurchaseKeeper = (*fuelpurchaseservice.Service)(nil)

var _ receivabletools.FuelCardKeeper = (*fuelpurchaseservice.Service)(nil)

var _ receivabletools.FuelImportKeeper = (*fuelpurchaseservice.Service)(nil)
