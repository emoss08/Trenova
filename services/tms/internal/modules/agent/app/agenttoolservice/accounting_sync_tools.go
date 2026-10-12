package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolProposalSelectionResolver = (*accountingtools.RetryAccountingSyncTool)(nil)
