package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*paytools.AssignPayProfileTool)(nil)

var _ agentports.ToolValidator = (*paytools.AssignPayProfileTool)(nil)

var _ agentports.ToolPreviewer = (*paytools.EndPayAssignmentTool)(nil)

var _ agentports.ToolValidator = (*paytools.EndPayAssignmentTool)(nil)
