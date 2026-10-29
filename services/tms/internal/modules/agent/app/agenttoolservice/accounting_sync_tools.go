package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingsynctools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolProposalSelectionResolver = (*accountingsynctools.RetryAccountingSyncTool)(
	nil,
)
