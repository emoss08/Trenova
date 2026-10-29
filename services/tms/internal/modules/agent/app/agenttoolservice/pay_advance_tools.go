package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*paytools.IssuePayAdvanceTool)(nil)

var _ agenttoolports.ToolValidator = (*paytools.IssuePayAdvanceTool)(nil)

var _ agenttoolports.ToolPreviewer = (*paytools.WriteOffPayAdvanceTool)(nil)

var _ agenttoolports.ToolValidator = (*paytools.WriteOffPayAdvanceTool)(nil)

var _ agenttoolports.TargetedTool = (*paytools.WriteOffPayAdvanceTool)(nil)
