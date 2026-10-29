package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*paytools.PayWorkerNowTool)(nil)

var _ agenttoolports.ToolValidator = (*paytools.PayWorkerNowTool)(nil)

var _ agenttoolports.ToolResultReporter = (*paytools.PayWorkerNowTool)(nil)
