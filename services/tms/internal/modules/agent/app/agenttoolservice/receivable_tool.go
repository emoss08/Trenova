package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*toolspec.ReceivableTool[struct{}, struct{}])(nil)

var _ agenttoolports.ToolValidator = (*toolspec.ReceivableTool[struct{}, struct{}])(nil)

var _ agenttoolports.TargetedTool = (*toolspec.ReceivableTool[struct{}, struct{}])(nil)

var _ agenttoolports.SearchableTool = (*toolspec.ReceivableTool[struct{}, struct{}])(nil)

var _ agenttoolports.ToolResultReporter = toolspec.ReportingReceivableTool[struct{}, struct{}]{}
