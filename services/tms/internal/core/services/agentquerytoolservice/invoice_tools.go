package agentquerytoolservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
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
		"state. Each line's id is what an invoice adjustment names. Use list_invoices first " +
		"when you have a number or are looking for what is unpaid, and get_invoices to check " +
		"several at once. Amounts and the memo are " +
		"left out, and named in withheldByAccess, when your data access does not reach them."
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
	ID          string `json:"id"`
	LineNumber  int    `json:"lineNumber"`
	Type        string `json:"type"`
	Description string `json:"description"`
	ChargeCode  string `json:"chargeCode,omitempty"`
	Accessorial string `json:"accessorialChargeId,omitempty"`
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
		PostedAt:         expectedDate(derefInt64(entity.PostedAt), absentNotPosted),
		SentAt:           expectedDate(derefInt64(entity.SentAt), "not sent"),
		VoidedAt:         expectedDate(derefInt64(entity.VoidedAt), absentNotVoided),
		Adjustment:       entity.IsAdjustmentArtifact,
		Supersedes:       pulidString(entity.SupersedesInvoiceID),
		SupersededBy:     pulidString(entity.SupersededByInvoiceID),
		PDFDocumentID:    pulidString(entity.PDFDocumentID),
		LineCount:        len(entity.Lines),
	}
	applyInvoiceAmounts(&detail, entity, gate)
	applyInvoiceText(&detail, entity, gate)
	detail.Lines, detail.LinesTruncated = invoiceLines(entity.Lines, gate)
	detail.Withheld = gate.Withheld()

	return detail
}

func applyInvoiceAmounts(detail *invoiceDetail, entity *invoice.Invoice, gate *fieldGate) {
	if gate.show("subtotalAmount", "subtotalAmount") {
		detail.Subtotal = entity.SubtotalAmount.StringFixed(2)
	}
	if gate.show("otherAmount", "otherAmount") {
		detail.Other = entity.OtherAmount.StringFixed(2)
	}
	if gate.show(fieldTotalAmount, fieldTotalAmount) {
		detail.Total = entity.TotalAmount.StringFixed(2)
		detail.BalanceDue = money.DecimalFromMinor(
			entity.TotalAmountMinor - entity.AppliedAmountMinor,
		).StringFixed(2)
	}
	if gate.show("appliedAmount", "appliedAmount") {
		detail.Applied = entity.AppliedAmount.StringFixed(2)
	}
}

func applyInvoiceText(detail *invoiceDetail, entity *invoice.Invoice, gate *fieldGate) {
	fields := []struct {
		field string
		value string
		into  *string
	}{
		{"memo", entity.Memo, &detail.Memo},
		{"remittanceInstructions", entity.RemittanceInstructions, &detail.Remittance},
		{"voidReason", entity.VoidReason, &detail.VoidReason},
		{"lastSendError", entity.LastSendError, &detail.LastSendError},
		{"lastEdiError", entity.LastEDIError, &detail.LastEDIError},
	}
	for _, entry := range fields {
		if entry.value != "" && gate.show(entry.field, entry.field) {
			*entry.into = entry.value
		}
	}
}

func invoiceLines(lines []*invoice.InvoiceLine, gate *fieldGate) ([]invoiceLineDetail, bool) {
	truncated := len(lines) > maxInvoiceLines
	if truncated {
		lines = lines[:maxInvoiceLines]
	}

	showQuantity := gate.show("quantity", "lines.quantity")
	showPrice := gate.show("unitPrice", "lines.unitPrice")
	showAmount := gate.show(fieldAmount, withheldLineAmount)
	rows := make([]invoiceLineDetail, 0, len(lines))
	for _, line := range lines {
		if line == nil {
			continue
		}
		row := invoiceLineDetail{
			ID:          line.ID.String(),
			LineNumber:  line.LineNumber,
			Type:        string(line.Type),
			Description: line.Description,
			ChargeCode:  line.ChargeCode,
			Accessorial: pulidString(line.AccessorialChargeID),
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
		rows = append(rows, row)
	}

	return rows, truncated
}

const (
	paramInvoiceIDs       = "invoiceIds"
	maxBatchInvoices      = 50
	maxBatchInvoiceLines  = 25
	batchInvoiceSearchFor = "invoices by id"
)

type invoiceBatchReader interface {
	GetByIDs(
		ctx context.Context,
		req repositories.GetInvoicesByIDsRequest,
	) ([]*invoice.Invoice, error)
}

type getInvoicesTool struct {
	invoices invoiceBatchReader
	access   fieldAccess
}

func newGetInvoicesTool(
	invoices repositories.InvoiceRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getInvoicesTool{invoices: invoices, access: newFieldAccess(permissions)}
}

func (t *getInvoicesTool) Name() string { return "get_invoices" }

func (t *getInvoicesTool) Description() string {
	return "Retrieve several invoices by id in one call, each with its status, totals, " +
		"payment state and line items. Use it instead of calling get_invoice once per " +
		"invoice, for example to check a batch of drafts before posting them. Up to 50; an " +
		"id that is not an invoice of this organization is named in the note. Amounts are " +
		"left out, and named in withheldByAccess, when your data access does not reach them."
}

func (t *getInvoicesTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			paramInvoiceIDs: idListProperty("The invoices' ids, from list_invoices, "+
				"list_ar_open_items or the page you are on.", maxBatchInvoices),
		},
		"required":             []string{paramInvoiceIDs},
		"additionalProperties": false,
	}
}

func (t *getInvoicesTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceInvoice})
}

type invoiceBatchRow struct {
	ID               string              `json:"id"`
	Number           string              `json:"number"`
	Status           string              `json:"status"`
	SettlementStatus string              `json:"settlementStatus"`
	BillType         string              `json:"billType"`
	BillToName       string              `json:"billToName"`
	ProNumber        string              `json:"proNumber,omitempty"`
	TotalAmount      string              `json:"totalAmount,omitempty"`
	Currency         string              `json:"currency"`
	InvoiceDate      optionalDate        `json:"invoiceDate"`
	DueDate          optionalDate        `json:"dueDate"`
	SendStatus       string              `json:"sendStatus"`
	LineCount        int                 `json:"lineCount"`
	Lines            []invoiceLineDetail `json:"lines"`
	LinesTruncated   bool                `json:"linesTruncated,omitempty"`
}

func (t *getInvoicesTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	ids, err := requireIDList(params.Params, paramInvoiceIDs, maxBatchInvoices)
	if err != nil {
		return nil, err
	}
	ids = sliceutils.Dedupe(ids)

	entities, err := t.invoices.GetByIDs(ctx, repositories.GetInvoicesByIDsRequest{
		TenantInfo: tenantOf(params),
		InvoiceIDs: ids,
	})
	if err != nil {
		return nil, err
	}

	byID := make(map[pulid.ID]*invoice.Invoice, len(entities))
	for _, entity := range entities {
		if entity != nil {
			byID[entity.ID] = entity
		}
	}

	gate := t.access.gate(ctx, params, permission.ResourceInvoice)
	rows := make([]invoiceBatchRow, 0, len(byID))
	missing := make([]string, 0, len(ids)-len(byID))
	for _, id := range ids {
		entity, ok := byID[id]
		if !ok {
			missing = append(missing, id.String())

			continue
		}
		rows = append(rows, invoiceBatchRowFrom(entity, gate))
	}

	outcome := searchOutcome{
		Count:       len(rows),
		SearchedFor: []string{batchInvoiceSearchFor},
		Items:       rows,
		Columns:     columnsOf(rows),
	}
	if len(missing) > 0 {
		outcome.Note = "No invoice of this organization has these ids: " +
			strings.Join(missing, ", ")
	}

	return gatedResult(&outcome, gate), nil
}

func invoiceBatchRowFrom(entity *invoice.Invoice, gate *fieldGate) invoiceBatchRow {
	lines := entity.Lines
	truncated := len(lines) > maxBatchInvoiceLines
	if truncated {
		lines = lines[:maxBatchInvoiceLines]
	}
	details, _ := invoiceLines(lines, gate)

	row := invoiceBatchRow{
		ID:               entity.ID.String(),
		Number:           entity.Number,
		Status:           string(entity.Status),
		SettlementStatus: string(entity.SettlementStatus),
		BillType:         string(entity.BillType),
		BillToName:       entity.BillToName,
		ProNumber:        entity.ShipmentProNumber,
		Currency:         money.CurrencyCode(entity.CurrencyCode),
		InvoiceDate:      recordedDate(entity.InvoiceDate),
		DueDate:          pointerDate(entity.DueDate),
		SendStatus:       string(entity.SendStatus),
		LineCount:        len(entity.Lines),
		Lines:            details,
		LinesTruncated:   truncated,
	}
	if gate.show(fieldTotalAmount, fieldTotalAmount) {
		row.TotalAmount = entity.TotalAmount.StringFixed(2)
	}

	return row
}
