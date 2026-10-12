package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*common.ReceivableTool[struct{}, struct{}])(nil)

var _ agentports.ToolValidator = (*common.ReceivableTool[struct{}, struct{}])(nil)

var _ agentports.TargetedTool = (*common.ReceivableTool[struct{}, struct{}])(nil)

var _ agentports.SearchableTool = (*common.ReceivableTool[struct{}, struct{}])(nil)

var _ agentports.ToolResultReporter = common.ReportingReceivableTool[struct{}, struct{}]{}
