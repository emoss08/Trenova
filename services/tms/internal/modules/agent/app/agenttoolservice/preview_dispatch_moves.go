package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/dispatchtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*dispatchtools.UnassignMovesTool)(nil)

var _ agenttoolports.ToolPreviewer = (*dispatchtools.UpdateMoveStatusTool)(nil)

var _ agenttoolports.ToolValidator = (*dispatchtools.UnassignMovesTool)(nil)

var _ agenttoolports.ToolValidator = (*dispatchtools.UpdateMoveStatusTool)(nil)
