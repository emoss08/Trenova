package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ratetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*ratetools.GenerateRateConfirmationTool)(nil)

var _ agenttoolports.ToolPreviewer = (*ratetools.SendRateConfirmationTool)(nil)

var _ agenttoolports.ToolPreviewer = (*ratetools.VoidRateConfirmationTool)(nil)

var _ agenttoolports.ToolPreviewer = (*ratetools.RecordRateConfirmationConfirmedTool)(nil)

var _ agenttoolports.ToolValidator = (*ratetools.GenerateRateConfirmationTool)(nil)

var _ agenttoolports.ToolValidator = (*ratetools.SendRateConfirmationTool)(nil)

var _ agenttoolports.ToolValidator = (*ratetools.VoidRateConfirmationTool)(nil)

var _ agenttoolports.ToolValidator = (*ratetools.RecordRateConfirmationConfirmedTool)(nil)
