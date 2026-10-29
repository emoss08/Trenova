package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/reporttools"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportdashboardservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportscheduleservice"
)

var (
	_ reporttools.ReportAdministrator    = (*reporting.Service)(nil)
	_ reporttools.DashboardAdministrator = (*reportdashboardservice.Service)(nil)
	_ reporttools.ScheduleAdministrator  = (*reportscheduleservice.Service)(nil)
)
