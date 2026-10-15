package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/editools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*editools.ApplyConnectionDraftTool)(nil)

var _ agentports.ToolValidator = (*editools.ApplyConnectionDraftTool)(nil)

var _ agentports.ToolSelectionResolver = (*editools.ApplyConnectionDraftTool)(nil)
