package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/editools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*editools.EdiTransferDecisionTool)(nil)

var _ agenttoolports.ToolValidator = (*editools.EdiTransferDecisionTool)(nil)

var _ agenttoolports.TargetedTool = (*editools.EdiTransferDecisionTool)(nil)
