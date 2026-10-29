package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ledgertools"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptworkitemservice"
)

var _ ledgertools.WorkItemRouter = (*bankreceiptworkitemservice.Service)(nil)
