package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/dispatchtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*dispatchtools.UnassignMovesTool)(nil)

var _ agentports.ToolPreviewer = (*dispatchtools.UpdateMoveStatusTool)(nil)

var _ agentports.ToolValidator = (*dispatchtools.UnassignMovesTool)(nil)

var _ agentports.ToolValidator = (*dispatchtools.UpdateMoveStatusTool)(nil)
