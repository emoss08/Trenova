package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*invoicetools.PostInvoiceTool)(nil)

var _ agenttoolports.ToolValidator = (*invoicetools.PostInvoiceTool)(nil)

var _ agenttoolports.TargetedTool = (*invoicetools.PostInvoiceTool)(nil)

var _ agenttoolports.ToolPreviewer = (*invoicetools.SendInvoiceTool)(nil)

var _ agenttoolports.ToolValidator = (*invoicetools.SendInvoiceTool)(nil)

var _ agenttoolports.TargetedTool = (*invoicetools.SendInvoiceTool)(nil)
