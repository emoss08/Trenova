package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptworkitemservice"
)

var _ receivabletools.WorkItemRouter = (*bankreceiptworkitemservice.Service)(nil)
