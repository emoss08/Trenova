package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*billingtools.TransferToBillingTool)(nil)

var _ agentports.ToolValidator = (*billingtools.TransferToBillingTool)(nil)

var _ agentports.ToolResultReporter = (*billingtools.TransferToBillingTool)(nil)

var _ agentports.ToolSelectionResolver = (*billingtools.TransferToBillingTool)(nil)
