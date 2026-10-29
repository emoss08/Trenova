package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*invoicetools.PostInvoicesTool)(nil)

var _ agenttoolports.ToolValidator = (*invoicetools.PostInvoicesTool)(nil)

var _ agenttoolports.ToolResultReporter = (*invoicetools.PostInvoicesTool)(nil)

var _ agenttoolports.ToolPreviewer = (*invoicetools.SendInvoicesTool)(nil)

var _ agenttoolports.ToolValidator = (*invoicetools.SendInvoicesTool)(nil)

var _ agenttoolports.ToolResultReporter = (*invoicetools.SendInvoicesTool)(nil)

var _ agenttoolports.ToolPreviewer = (*invoicetools.ApproveBillingQueueItemsTool)(nil)

var _ agenttoolports.ToolValidator = (*invoicetools.ApproveBillingQueueItemsTool)(nil)

var _ agenttoolports.ToolResultReporter = (*invoicetools.ApproveBillingQueueItemsTool)(nil)
