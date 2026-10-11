package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*paytools.PayWorkerNowTool)(nil)

var _ agentports.ToolValidator = (*paytools.PayWorkerNowTool)(nil)

var _ agentports.ToolResultReporter = (*paytools.PayWorkerNowTool)(nil)
