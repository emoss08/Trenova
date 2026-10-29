package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*paytools.AssignPayProfileTool)(nil)

var _ agenttoolports.ToolValidator = (*paytools.AssignPayProfileTool)(nil)

var _ agenttoolports.ToolPreviewer = (*paytools.EndPayAssignmentTool)(nil)

var _ agenttoolports.ToolValidator = (*paytools.EndPayAssignmentTool)(nil)
