package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/money"
)

const maxInvoiceLines = 100

type invoiceReader interface {
	GetByID(
		ctx context.Context,
		req repositories.GetInvoiceByIDRequest,
	) (*invoice.Invoice, error)
}

type getInvoiceTool struct {
	invoices invoiceReader
	access   fieldAccess
}

func newGetInvoiceTool(
	invoices repositories.InvoiceRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getInvoiceTool{invoices: invoices, access: newFieldAccess(permissions)}
}

func (t *getInvoiceTool) Name() string { return "get_invoice" }

func (t *getInvoiceTool) Description() string {
	return "Retrieve one invoice by id, with its line items, totals, payment and dispute " +
		"state. Use list_invoices first when you have a number or are looking for what is " +
		"unpaid. Amounts and the memo are left out, and named in withheldByAccess, when " +
		"your data access does not reach them."
}

func (t *getInvoiceTool) ParamSchema() map[string]any {
	return idSchema("invoiceId", "The invoice's id, from list_invoices, list_ar_open_items "+
		"or the page you are on.")
}

func (t *getInvoiceTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceInvoice})
}

func (t *getInvoiceTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	id, err := requirePulid(params.Params, "invoiceId")
	if err != nil {
		return nil, err
	}

	entity, err := t.invoices.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         id,
		TenantInfo: tenantOf(params),
	})
	if err != nil {
		return nil, err
	}

	return invoiceDetailFrom(entity, t.access.gate(ctx, params, permission.ResourceInvoice)), nil
}

type invoiceDetail struct {
	ID               string              `json:"id"`
	Number           string              `json:"number"`
	Status           string              `json:"status"`
	BillType         string              `json:"billType"`
	Scope            string              `json:"scope,omitempty"`
	SettlementStatus string              `json:"settlementStatus"`
	DisputeStatus    string              `json:"disputeStatus"`
	SendStatus       string              `json:"sendStatus"`
	EDISendStatus    string              `json:"ediSendStatus,omitempty"`
	PaymentTerm      string              `json:"paymentTerm"`
	Currency         string              `json:"currency"`
	CustomerID       string              `json:"customerId"`
	BillToName       string              `json:"billToName"`
	BillToCode       string              `json:"billToCode,omitempty"`
	BillToCity       string              `json:"billToCity,omitempty"`
	BillToState      string              `json:"billToState,omitempty"`
	ShipmentID       string              `json:"shipmentId,omitempty"`
	ProNumber        string              `json:"proNumber,omitempty"`
	BOL              string              `json:"bol,omitempty"`
	OrderID          string              `json:"orderId,omitempty"`
	OrderNumber      string              `json:"orderNumber,omitempty"`
	ShipmentCount    int                 `json:"shipmentCount,omitempty"`
	InvoiceDate      optionalDate        `json:"invoiceDate"`
	DueDate          optionalDate        `json:"dueDate"`
	ServiceDate      optionalDate        `json:"serviceDate"`
	PostedAt         optionalDate        `json:"postedAt"`
	SentAt           optionalDate        `json:"sentAt"`
	VoidedAt         optionalDate        `json:"voidedAt"`
	Adjustment       bool                `json:"isAdjustment"`
	Supersedes       string              `json:"supersedesInvoiceId,omitempty"`
	SupersededBy     string              `json:"supersededByInvoiceId,omitempty"`
	PDFDocumentID    string              `json:"pdfDocumentId,omitempty"`
	Subtotal         string              `json:"subtotalAmount,omitempty"`
	Other            string              `json:"otherAmount,omitempty"`
	Total            string              `json:"totalAmount,omitempty"`
	Applied          string              `json:"appliedAmount,omitempty"`
	BalanceDue       string              `json:"balanceDue,omitempty"`
	Memo             string              `json:"memo,omitempty"`
	Remittance       string              `json:"remittanceInstructions,omitempty"`
	VoidReason       string              `json:"voidReason,omitempty"`
	LastSendError    string              `json:"lastSendError,omitempty"`
	LastEDIError     string              `json:"lastEdiError,omitempty"`
	Lines            []invoiceLineDetail `json:"lines"`
	LineCount        int                 `json:"lineCount"`
	LinesTruncated   bool                `json:"linesTruncated,omitempty"`
	Withheld         []string            `json:"withheldByAccess,omitempty"`
}

type invoiceLineDetail struct {
	LineNumber  int    `json:"lineNumber"`
	Type        string `json:"type"`
	Description string `json:"description"`
	ChargeCode  string `json:"chargeCode,omitempty"`
	ProNumber   string `json:"proNumber,omitempty"`
	Quantity    string `json:"quantity,omitempty"`
	UnitPrice   string `json:"unitPrice,omitempty"`
	Amount      string `json:"amount,omitempty"`
}

func invoiceDetailFrom(entity *invoice.Invoice, gate *fieldGate) invoiceDetail {
	detail := invoiceDetail{
		ID:               entity.ID.String(),
		Number:           entity.Number,
		Status:           string(entity.Status),
		BillType:         string(entity.BillType),
		Scope:            string(entity.Scope),
		SettlementStatus: string(entity.SettlementStatus),
		DisputeStatus:    string(entity.DisputeStatus),
		SendStatus:       string(entity.SendStatus),
		EDISendStatus:    string(entity.EDISendStatus),
		PaymentTerm:      string(entity.PaymentTerm),
		Currency:         money.CurrencyCode(entity.CurrencyCode),
		CustomerID:       entity.CustomerID.String(),
		BillToName:       entity.BillToName,
		BillToCode:       entity.BillToCode,
		BillToCity:       entity.BillToCity,
		BillToState:      entity.BillToState,
		ShipmentID:       pulidString(entity.ShipmentID),
		ProNumber:        entity.ShipmentProNumber,
		BOL:              entity.ShipmentBOL,
		OrderID:          pulidString(entity.OrderID),
		OrderNumber:      entity.OrderNumber,
		ShipmentCount:    entity.ShipmentCount,
		InvoiceDate:      recordedDate(entity.InvoiceDate),
		DueDate:          pointerDate(entity.DueDate),
		ServiceDate:      pointerDate(entity.ServiceDate),
		PostedAt:         expectedDate(derefInt64(entity.PostedAt), "not posted"),
		SentAt:           expectedDate(derefInt64(entity.SentAt), "not sent"),
		VoidedAt:         expectedDate(derefInt64(entity.VoidedAt), "not voided"),
		Adjustment:       entity.IsAdjustmentArtifact,
		Supersedes:       pulidString(entity.SupersedesInvoiceID),
		SupersededBy:     pulidString(entity.SupersededByInvoiceID),
		PDFDocumentID:    pulidString(entity.PDFDocumentID),
		LineCount:        len(entity.Lines),
		Lines:            make([]invoiceLineDetail, 0, min(len(entity.Lines), maxInvoiceLines)),
	}

	if gate.show("subtotalAmount", "subtotalAmount") {
		detail.Subtotal = entity.SubtotalAmount.StringFixed(2)
	}
	if gate.show("otherAmount", "otherAmount") {
		detail.Other = entity.OtherAmount.StringFixed(2)
	}
	if gate.show("totalAmount", "totalAmount") {
		detail.Total = entity.TotalAmount.StringFixed(2)
		detail.BalanceDue = money.DecimalFromMinor(
			entity.TotalAmountMinor - entity.AppliedAmountMinor,
		).StringFixed(2)
	}
	if gate.show("appliedAmount", "appliedAmount") {
		detail.Applied = entity.AppliedAmount.StringFixed(2)
	}
	if entity.Memo != "" && gate.show("memo", "memo") {
		detail.Memo = entity.Memo
	}
	if entity.RemittanceInstructions != "" &&
		gate.show("remittanceInstructions", "remittanceInstructions") {
		detail.Remittance = entity.RemittanceInstructions
	}
	if entity.VoidReason != "" && gate.show("voidReason", "voidReason") {
		detail.VoidReason = entity.VoidReason
	}
	if entity.LastSendError != "" && gate.show("lastSendError", "lastSendError") {
		detail.LastSendError = entity.LastSendError
	}
	if entity.LastEDIError != "" && gate.show("lastEdiError", "lastEdiError") {
		detail.LastEDIError = entity.LastEDIError
	}

	showQuantity := gate.show("quantity", "lines.quantity")
	showPrice := gate.show("unitPrice", "lines.unitPrice")
	showAmount := gate.show("amount", "lines.amount")
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		if len(detail.Lines) == maxInvoiceLines {
			detail.LinesTruncated = true
			break
		}
		row := invoiceLineDetail{
			LineNumber:  line.LineNumber,
			Type:        string(line.Type),
			Description: line.Description,
			ChargeCode:  line.ChargeCode,
			ProNumber:   line.ShipmentProNumber,
		}
		if showQuantity {
			row.Quantity = line.Quantity.String()
		}
		if showPrice {
			row.UnitPrice = line.UnitPrice.StringFixed(2)
		}
		if showAmount {
			row.Amount = line.Amount.StringFixed(2)
		}
		detail.Lines = append(detail.Lines, row)
	}
	detail.Withheld = gate.Withheld()

	return detail
}
