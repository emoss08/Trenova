package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/rateconfirmationtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*rateconfirmationtools.GenerateRateConfirmationTool)(nil)

var _ agenttoolports.ToolPreviewer = (*rateconfirmationtools.SendRateConfirmationTool)(nil)

var _ agenttoolports.ToolPreviewer = (*rateconfirmationtools.VoidRateConfirmationTool)(nil)

var _ agenttoolports.ToolPreviewer = (*rateconfirmationtools.RecordRateConfirmationConfirmedTool)(
	nil,
)

var _ agenttoolports.ToolValidator = (*rateconfirmationtools.GenerateRateConfirmationTool)(nil)

var _ agenttoolports.ToolValidator = (*rateconfirmationtools.SendRateConfirmationTool)(nil)

var _ agenttoolports.ToolValidator = (*rateconfirmationtools.VoidRateConfirmationTool)(nil)

var _ agenttoolports.ToolValidator = (*rateconfirmationtools.RecordRateConfirmationConfirmedTool)(
	nil,
)
