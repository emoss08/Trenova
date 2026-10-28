package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*common.ReceivableTool[struct{}, struct{}])(nil)

var _ agenttoolports.ToolValidator = (*common.ReceivableTool[struct{}, struct{}])(nil)

var _ agenttoolports.TargetedTool = (*common.ReceivableTool[struct{}, struct{}])(nil)

var _ agenttoolports.SearchableTool = (*common.ReceivableTool[struct{}, struct{}])(nil)

var _ agenttoolports.ToolResultReporter = common.ReportingReceivableTool[struct{}, struct{}]{}
