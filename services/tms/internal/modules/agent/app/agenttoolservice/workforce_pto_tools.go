package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
)

var _ workforcetools.PtoBalanceAdjuster = (*ptoledgerservice.Service)(nil)
