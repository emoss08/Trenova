package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*paytools.IssuePayAdvanceTool)(nil)

var _ agentports.ToolValidator = (*paytools.IssuePayAdvanceTool)(nil)

var _ agentports.ToolPreviewer = (*paytools.WriteOffPayAdvanceTool)(nil)

var _ agentports.ToolValidator = (*paytools.WriteOffPayAdvanceTool)(nil)

var _ agentports.TargetedTool = (*paytools.WriteOffPayAdvanceTool)(nil)
