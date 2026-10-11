package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/reporttools"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
)

var _ reporttools.ReportAdministrator = (*reporting.Service)(nil)
