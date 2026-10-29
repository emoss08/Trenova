package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/locationtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolResultReporter = (*locationtools.CreateLocationTool)(nil)

var _ agenttoolports.ToolValidator = (*locationtools.CreateLocationTool)(nil)

var _ agenttoolports.ToolPreviewer = (*locationtools.CreateLocationTool)(nil)
