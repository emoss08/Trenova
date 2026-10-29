package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/bankreceipttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*bankreceipttools.ResolveBankReceiptWorkItemTool)(nil)
