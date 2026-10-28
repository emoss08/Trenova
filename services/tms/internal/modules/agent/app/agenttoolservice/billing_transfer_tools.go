package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*billingtools.TransferToBillingTool)(nil)

var _ agenttoolports.ToolValidator = (*billingtools.TransferToBillingTool)(nil)

var _ agenttoolports.ToolResultReporter = (*billingtools.TransferToBillingTool)(nil)

var _ agenttoolports.ToolSelectionResolver = (*billingtools.TransferToBillingTool)(nil)
