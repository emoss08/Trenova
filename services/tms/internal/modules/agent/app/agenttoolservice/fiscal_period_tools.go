package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ledgertools"
)

var _ ledgertools.FiscalPeriodKeeper = (*fiscalperiodservice.Service)(nil)
