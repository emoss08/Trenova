package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*carriertools.CarrierInvoiceMatchTool)(nil)

var _ agenttoolports.ToolValidator = (*carriertools.CarrierInvoiceMatchTool)(nil)

var _ agenttoolports.TargetedTool = (*carriertools.CarrierInvoiceMatchTool)(nil)

var _ agenttoolports.ToolPreviewer = (*carriertools.CreateCarrierInvoiceMatchTool)(nil)

var _ agenttoolports.ToolValidator = (*carriertools.CreateCarrierInvoiceMatchTool)(nil)

var _ agenttoolports.ToolPreviewer = (*carriertools.LinkEDICarrierInvoiceTool)(nil)

var _ agenttoolports.ToolValidator = (*carriertools.LinkEDICarrierInvoiceTool)(nil)
