package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/timeofftools"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
)

var _ timeofftools.PtoBalanceAdjuster = (*ptoledgerservice.Service)(nil)
