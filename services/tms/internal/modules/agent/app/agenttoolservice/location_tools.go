package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolResultReporter = (*masterdatatools.CreateLocationTool)(nil)

var _ agentports.ToolValidator = (*masterdatatools.CreateLocationTool)(nil)

var _ agentports.ToolPreviewer = (*masterdatatools.CreateLocationTool)(nil)
