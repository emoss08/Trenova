package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/editools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*editools.EdiTransferDecisionTool)(nil)

var _ agentports.ToolValidator = (*editools.EdiTransferDecisionTool)(nil)

var _ agentports.TargetedTool = (*editools.EdiTransferDecisionTool)(nil)
