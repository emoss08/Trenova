package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
)

var _ receivabletools.FiscalPeriodKeeper = (*fiscalperiodservice.Service)(nil)
