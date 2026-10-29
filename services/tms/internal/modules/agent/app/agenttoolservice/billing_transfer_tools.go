package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtransfertools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*billingtransfertools.TransferToBillingTool)(nil)

var _ agenttoolports.ToolValidator = (*billingtransfertools.TransferToBillingTool)(nil)

var _ agenttoolports.ToolResultReporter = (*billingtransfertools.TransferToBillingTool)(nil)

var _ agenttoolports.ToolSelectionResolver = (*billingtransfertools.TransferToBillingTool)(nil)
