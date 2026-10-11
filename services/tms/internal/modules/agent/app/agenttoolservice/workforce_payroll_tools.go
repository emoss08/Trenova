package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/services/timesheetservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
)

var _ workforcetools.PayrollExporter = (*timesheetservice.Service)(nil)
