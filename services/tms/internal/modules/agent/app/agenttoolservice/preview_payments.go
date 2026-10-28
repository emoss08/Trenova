package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*invoicetools.MatchBankReceiptTool)(nil)

var _ agenttoolports.ToolPreviewer = (*invoicetools.PostCustomerPaymentTool)(nil)
