package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/payrolltools"
	"github.com/emoss08/trenova/internal/modules/workforce/app/timesheetservice"
)

var _ payrolltools.PayrollExporter = (*timesheetservice.Service)(nil)
