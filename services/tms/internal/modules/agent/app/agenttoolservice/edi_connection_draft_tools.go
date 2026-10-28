package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/editools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*editools.ApplyConnectionDraftTool)(nil)

var _ agenttoolports.ToolValidator = (*editools.ApplyConnectionDraftTool)(nil)

var _ agenttoolports.ToolSelectionResolver = (*editools.ApplyConnectionDraftTool)(nil)

var _ agenttoolports.ToolResultReporter = (*editools.ApplyConnectionDraftTool)(nil)
