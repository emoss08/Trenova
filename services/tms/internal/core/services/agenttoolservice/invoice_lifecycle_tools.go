package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/shopspring/decimal"
)

const (
	paramOrderID               = "orderId"
	paramOffCycleNote          = "offCycleReason"
	paramVoidDisposition       = "disposition"
	paramMemoQuantity          = "quantity"
	paramAccessorialChargeID   = "accessorialChargeId"
	paramReferenceInvoiceID    = "referenceInvoiceId"
	paramInvoiceMemo           = "memo"
	paramForceResend           = "force"
	paramRemittance            = "remittanceInstructions"
	paramEmailSubject          = "emailSubject"
	paramEmailBody             = "emailBody"
	paramEmailTo               = "emailTo"
	paramEmailCc               = "emailCc"
	paramEmailBcc              = "emailBcc"
	paramAttachmentDocumentIDs = "attachmentDocumentIds"
	maxInvoiceTextChars        = 4000
	maxInvoiceEmailBodyChars   = 20000
	maxInvoiceRecipients       = 25
	maxInvoiceAttachments      = 50
	maxInvoiceShipments        = 100
	maxInvoiceReasonChars      = 1000
	maxMemoLineCount           = 200
	maxMemoLineTextChars       = 500
)

var (
	errNothingToUpdate = errors.New(
		"name at least one thing to change on the draft: memo, remittanceInstructions, " +
			"emailSubject, emailBody, emailTo, emailCc, emailBcc or attachmentDocumentIds",
	)
	errPDFWouldSend = errortypes.NewBusinessError(
		"rendering this invoice's PDF would email it to the customer at once, as their " +
			"billing profile sends invoices when the PDF is made; use send_invoice, which a " +
			"person approves, instead",
	)
	errOrderOrShipments = errors.New(
		"name the freight to invoice: an orderId, shipmentIds, or an orderId with the " +
			"shipmentIds of it to bill",
	)
	memoBillTypes    = []billingqueue.BillType{billingqueue.BillTypeCreditMemo, billingqueue.BillTypeDebitMemo}
	voidDispositions = []invoice.VoidDisposition{
		invoice.VoidDispositionRebill,
		invoice.VoidDispositionDoNotRebill,
	}
	recipientParams = []string{paramEmailTo, paramEmailCc, paramEmailBcc}
)

type invoiceDraftEditor interface {
	UpdateDraft(
		ctx context.Context,
		req *serviceports.UpdateInvoiceDraftRequest,
		actor *serviceports.RequestActor,
	) (*invoice.Invoice, error)
	PreviewUpdateDraft(
		ctx context.Context,
		req *serviceports.UpdateInvoiceDraftRequest,
	) (*serviceports.InvoiceDraftUpdatePreview, error)
}

type invoicePDFGenerator interface {
	GeneratePDF(
		ctx context.Context,
		req *serviceports.InvoicePreviewRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.GenerateInvoicePDFResult, error)
	PlanPDFGeneration(
		ctx context.Context,
		req *serviceports.InvoicePreviewRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.InvoicePDFGenerationPlan, error)
}

type invoiceCreator interface {
	PlanCreateInvoices(
		ctx context.Context,
		req *serviceports.PlanCreateInvoicesRequest,
	) (*serviceports.CreateInvoicesPlan, error)
	CreateInvoicesFromOrder(
		ctx context.Context,
		req *serviceports.CreateInvoiceFromOrderRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.CreateInvoicesResult, error)
	CreateInvoicesFromShipments(
		ctx context.Context,
		req *serviceports.CreateInvoiceFromShipmentsRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.CreateInvoicesResult, error)
}

type invoiceMemoRaiser interface {
	CreateMemo(
		ctx context.Context,
		req *serviceports.CreateMemoRequest,
		actor *serviceports.RequestActor,
	) (*invoice.Invoice, error)
	PreviewMemo(
		ctx context.Context,
		req *serviceports.CreateMemoRequest,
	) (*invoice.Invoice, error)
}

type invoiceVoider interface {
	VoidInvoice(
		ctx context.Context,
		req *serviceports.VoidInvoiceRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.VoidInvoiceResult, error)
	PreviewVoid(
		ctx context.Context,
		req *serviceports.VoidInvoiceRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.InvoiceVoidPreview, error)
}

type invoiceEDISender interface {
	SendEDI(
		ctx context.Context,
		req *serviceports.SendInvoiceEDIRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.InvoiceEDISendResult, error)
	PreviewSendEDI(
		ctx context.Context,
		req *serviceports.SendInvoiceEDIRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.InvoiceEDISendPreview, error)
}

func invoiceIDProperty() map[string]any {
	return stringProperty("The invoice, from list_invoices or get_invoice. Never guess one.", 0)
}

func recipientsProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: description,
		toolschema.KeyItems:       map[string]any{toolschema.KeyType: toolschema.TypeString},
		toolschema.KeyMaxItems:    maxInvoiceRecipients,
	}
}

func newUpdateInvoiceDraftTool(invoices invoiceDraftEditor) serviceports.AgentTool {
	return newReceivableTool(receivableSpec{
		name: "update_invoice_draft",
		description: "Change what a draft invoice says and where it goes: its memo, remittance " +
			"instructions, email subject and body, recipients and attached documents. Only " +
			"the fields you pass change; an empty string or list clears one. A posted " +
			"invoice is changed with an invoice adjustment instead.",
		resource:    permission.ResourceInvoice,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		taintHold: "Outside content could put its own wording or recipients on an invoice " +
			"the customer receives.",
		condition: &serviceports.TierCondition{
			Description: "Changing who the invoice is emailed to is a proposal a person " +
				"decides; any other change runs once a person approves it.",
			Limit: draftUpdateTierLimit,
		},
		rationale: "Edits a draft nobody outside has seen; the customer sees it only when a " +
			"person sends it, but the recipients decide where it goes then.",
		properties: map[string]any{
			paramInvoiceID: invoiceIDProperty(),
			paramInvoiceMemo: stringProperty("The note printed on the invoice.",
				maxInvoiceTextChars),
			paramRemittance: stringProperty("How the customer should pay, printed on the "+
				"invoice.", maxInvoiceTextChars),
			paramEmailSubject: stringProperty("The subject of the email the invoice goes in.",
				invoice.MaxEmailSubjectLength),
			paramEmailBody: stringProperty("The body of the email the invoice goes in.",
				maxInvoiceEmailBodyChars),
			paramEmailTo:  recipientsProperty("Who the invoice is emailed to; replaces the list."),
			paramEmailCc:  recipientsProperty("Who is copied; replaces the list."),
			paramEmailBcc: recipientsProperty("Who is blind-copied; replaces the list."),
			paramAttachmentDocumentIDs: map[string]any{
				toolschema.KeyType: toolschema.TypeArray,
				toolschema.KeyDescription: "The documents attached to the email, from " +
					"search_documents; replaces the list.",
				toolschema.KeyItems:    map[string]any{toolschema.KeyType: toolschema.TypeString},
				toolschema.KeyMaxItems: maxInvoiceAttachments,
			},
		},
		required: []string{paramInvoiceID},
		target:   targetInvoice,
	}, receivablePlan[*serviceports.UpdateInvoiceDraftRequest, *serviceports.InvoiceDraftUpdatePreview]{
		request: draftUpdateRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.UpdateInvoiceDraftRequest,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.InvoiceDraftUpdatePreview, error) {
			return invoices.PreviewUpdateDraft(ctx, req)
		},
		refused: func(*serviceports.UpdateInvoiceDraftRequest) string {
			return "Would update the draft invoice."
		},
		render: renderDraftUpdate,
		run: func(
			ctx context.Context,
			req *serviceports.UpdateInvoiceDraftRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := invoices.UpdateDraft(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func draftUpdateTierLimit(
	_ context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // a TierCondition passes params by value
) agent.AutonomyTier {
	for _, key := range recipientParams {
		if _, ok := params.Params[key]; ok {
			return agent.TierPropose
		}
	}

	return agent.TierActWithApproval
}

func draftUpdateRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.UpdateInvoiceDraftRequest, error) {
	invoiceID, err := requirePulid(params.Params, paramInvoiceID)
	if err != nil {
		return nil, err
	}
	req := &serviceports.UpdateInvoiceDraftRequest{
		InvoiceID:  invoiceID,
		TenantInfo: tenantFrom(*params),
	}

	texts := []struct {
		key   string
		limit int
		into  **string
	}{
		{paramInvoiceMemo, maxInvoiceTextChars, &req.Memo},
		{paramRemittance, maxInvoiceTextChars, &req.RemittanceInstructions},
		{paramEmailSubject, invoice.MaxEmailSubjectLength, &req.EmailSubject},
		{paramEmailBody, maxInvoiceEmailBodyChars, &req.EmailBody},
	}
	for _, text := range texts {
		if *text.into, err = presentText(params.Params, text.key, text.limit); err != nil {
			return nil, err
		}
	}
	recipients := []struct {
		key  string
		into **[]string
	}{
		{paramEmailTo, &req.EmailTo},
		{paramEmailCc, &req.EmailCC},
		{paramEmailBcc, &req.EmailBCC},
	}
	for _, list := range recipients {
		if *list.into, err = presentRecipients(params.Params, list.key); err != nil {
			return nil, err
		}
	}
	if req.AttachmentIDs, err = presentIDs(
		params.Params, paramAttachmentDocumentIDs, maxInvoiceAttachments,
	); err != nil {
		return nil, err
	}

	if req.Memo == nil && req.RemittanceInstructions == nil && req.EmailSubject == nil &&
		req.EmailBody == nil && req.EmailTo == nil && req.EmailCC == nil &&
		req.EmailBCC == nil && req.AttachmentIDs == nil {
		return nil, errNothingToUpdate
	}

	return req, nil
}

func presentText(params map[string]any, key string, limit int) (*string, error) {
	raw, ok := params[key]
	if !ok {
		return nil, nil
	}
	if _, isString := raw.(string); !isString {
		return nil, fmt.Errorf("parameter %q must be a string", key)
	}
	text, err := boundedText(params, key, limit)
	if err != nil {
		return nil, err
	}

	return &text, nil
}

func presentList(params map[string]any, key string, limit int) ([]string, bool, error) {
	raw, ok := params[key]
	if !ok {
		return nil, false, nil
	}
	entries, isList := raw.([]any)
	if !isList {
		return nil, false, fmt.Errorf("parameter %q must be a list", key)
	}
	if len(entries) > limit {
		return nil, false, fmt.Errorf("parameter %q lists %d entries; at most %d",
			key, len(entries), limit)
	}

	values := make([]string, 0, len(entries))
	for idx, entry := range entries {
		text, isString := entry.(string)
		if !isString || strings.TrimSpace(text) == "" {
			return nil, false, fmt.Errorf("parameter %q[%d] must be a non-empty string", key, idx)
		}
		values = append(values, strings.TrimSpace(text))
	}

	return values, true, nil
}

func presentRecipients(params map[string]any, key string) (*[]string, error) {
	values, ok, err := presentList(params, key, maxInvoiceRecipients)
	if err != nil || !ok {
		return nil, err
	}

	addresses := stringutils.NormalizeEmailAddresses(values)
	for _, address := range addresses {
		if err = is.EmailFormat.Validate(address); err != nil {
			return nil, fmt.Errorf("parameter %q: %q is not an email address", key, address)
		}
	}

	return &addresses, nil
}

func presentIDs(params map[string]any, key string, limit int) (*[]pulid.ID, error) {
	values, ok, err := presentList(params, key, limit)
	if err != nil || !ok {
		return nil, err
	}

	ids := make([]pulid.ID, 0, len(values))
	for idx, value := range values {
		id, parseErr := pulid.Parse(value)
		if parseErr != nil {
			return nil, fmt.Errorf("parameter %q[%d] is not a valid id: %w", key, idx, parseErr)
		}
		ids = append(ids, id)
	}

	return &ids, nil
}

func newGenerateInvoicePDFTool(invoices invoicePDFGenerator) serviceports.AgentTool {
	plan := func(
		ctx context.Context,
		req *serviceports.InvoicePreviewRequest,
		params *serviceports.ToolExecuteParams,
	) (*serviceports.InvoicePDFGenerationPlan, error) {
		generation, err := invoices.PlanPDFGeneration(ctx, req, params.Actor)
		if err != nil {
			return nil, err
		}
		if generation.AutoSends {
			return nil, errPDFWouldSend
		}
		return generation, nil
	}

	return newReceivableTool(receivableSpec{
		name: "generate_invoice_pdf",
		description: "Render an invoice's PDF from what it says now and file it on the " +
			"invoice, replacing the one there. Refused when the customer's billing profile " +
			"would email the invoice the moment its PDF is made; send it with send_invoice.",
		resource:    permission.ResourceInvoice,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierAutoExecute,
		rationale: "Renders the invoice as it stands into its filed PDF; nothing is sent, and " +
			"it is refused whenever rendering would email the customer.",
		properties: map[string]any{paramInvoiceID: invoiceIDProperty()},
		required:   []string{paramInvoiceID},
		target:     targetInvoice,
	}, receivablePlan[*serviceports.InvoicePreviewRequest, *serviceports.InvoicePDFGenerationPlan]{
		request: func(params *serviceports.ToolExecuteParams) (*serviceports.InvoicePreviewRequest, error) {
			invoiceID, err := requirePulid(params.Params, paramInvoiceID)
			if err != nil {
				return nil, err
			}
			return &serviceports.InvoicePreviewRequest{
				InvoiceID:  invoiceID,
				TenantInfo: tenantFrom(*params),
			}, nil
		},
		plan: plan,
		refused: func(*serviceports.InvoicePreviewRequest) string {
			return "Would render the invoice's PDF."
		},
		render: renderPDFGeneration,
		run: func(
			ctx context.Context,
			req *serviceports.InvoicePreviewRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if _, err := plan(ctx, req, params); err != nil {
				return nil, err
			}
			_, err := invoices.GeneratePDF(ctx, req, params.Actor)
			return nil, err
		},
	})
}

type createInvoicesRequest struct {
	plan *serviceports.PlanCreateInvoicesRequest
}

func newCreateInvoiceTool(invoices invoiceCreator) serviceports.AgentTool {
	return newReportingReceivableTool(receivableSpec{
		name: "create_invoice",
		description: "Propose drafting invoices for completed freight: an order's billable " +
			"shipments as one grouped invoice, or shipments on their own. A split-billed " +
			"shipment makes one draft per payer. Drafts are posted with post_invoice. A " +
			"person always decides.",
		resource:    permission.ResourceInvoice,
		operation:   permission.OpCreate,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		artifact:    invoiceRecordEntity,
		rationale: "Turns freight into receivables and takes it off the billing queue and any " +
			"statement; only a person decides what is billed and when.",
		properties: map[string]any{
			paramOrderID: stringProperty("The order to bill as one grouped invoice, from "+
				"get_order or get_shipment.", 0),
			paramShipmentIDs: idListProperty("The shipments to bill, from search_shipments or "+
				"get_shipment. With orderId, only these of the order's shipments.",
				maxInvoiceShipments),
			paramOffCycleNote: stringProperty("Why this customer's freight is billed now "+
				"rather than on their periodic statement. Required only for statement "+
				"customers.", maxInvoiceReasonChars),
		},
	}, receivablePlan[*createInvoicesRequest, *serviceports.CreateInvoicesPlan]{
		request: createInvoicesRequestFrom,
		plan: func(
			ctx context.Context,
			req *createInvoicesRequest,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.CreateInvoicesPlan, error) {
			return invoices.PlanCreateInvoices(ctx, req.plan)
		},
		refused: func(req *createInvoicesRequest) string {
			if req.plan.OrderID.IsNotNil() {
				return "Would invoice the order."
			}
			return fmt.Sprintf("Would invoice %s.", countOf(len(req.plan.ShipmentIDs), "shipment"))
		},
		render: renderCreateInvoices,
		run: func(
			ctx context.Context,
			req *createInvoicesRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			var (
				made *serviceports.CreateInvoicesResult
				err  error
			)
			if req.plan.OrderID.IsNotNil() {
				made, err = invoices.CreateInvoicesFromOrder(ctx,
					&serviceports.CreateInvoiceFromOrderRequest{
						OrderID:        req.plan.OrderID,
						ShipmentIDs:    req.plan.ShipmentIDs,
						TenantInfo:     req.plan.TenantInfo,
						OffCycleReason: req.plan.OffCycleReason,
					}, params.Actor)
			} else {
				made, err = invoices.CreateInvoicesFromShipments(ctx,
					&serviceports.CreateInvoiceFromShipmentsRequest{
						ShipmentIDs:    req.plan.ShipmentIDs,
						TenantInfo:     req.plan.TenantInfo,
						OffCycleReason: req.plan.OffCycleReason,
					}, params.Actor)
			}
			if err != nil {
				return nil, err
			}
			return invoiceResult("created", made.Primary), nil
		},
	})
}

func createInvoicesRequestFrom(params *serviceports.ToolExecuteParams) (*createInvoicesRequest, error) {
	orderID, hasOrder, err := optionalPulid(params.Params, paramOrderID)
	if err != nil {
		return nil, err
	}
	req := &serviceports.PlanCreateInvoicesRequest{
		OrderID:    orderID,
		TenantInfo: tenantFrom(*params),
	}
	if _, ok := params.Params[paramShipmentIDs]; ok {
		if req.ShipmentIDs, err = requirePulidSlice(
			params.Params, paramShipmentIDs, maxInvoiceShipments,
		); err != nil {
			return nil, err
		}
	}
	if !hasOrder && len(req.ShipmentIDs) == 0 {
		return nil, errOrderOrShipments
	}
	if req.OffCycleReason, err = boundedText(
		params.Params, paramOffCycleNote, maxInvoiceReasonChars,
	); err != nil {
		return nil, err
	}

	return &createInvoicesRequest{plan: req}, nil
}

func invoiceResult(action string, entity *invoice.Invoice) *agent.ToolExecutionResult {
	if entity == nil || entity.ID.IsNil() {
		return &agent.ToolExecutionResult{Action: action, Kind: "invoice"}
	}

	return &agent.ToolExecutionResult{
		Action: action,
		Kind:   strings.ToLower(string(entity.BillType)),
		IDs: map[string]string{
			paramInvoiceID: entity.ID.String(),
			"number":       entity.Number,
		},
		Record: &agent.RecordRef{EntityType: invoiceRecordEntity, ID: entity.ID.String()},
	}
}

func newCreateInvoiceMemoTool(invoices invoiceMemoRaiser) serviceports.AgentTool {
	return newReportingReceivableTool(receivableSpec{
		name: "create_invoice_memo",
		description: "Propose a draft credit or debit memo to a customer with no shipment " +
			"behind it, such as a goodwill credit or a returned-check fee. It stays a draft " +
			"until post_invoice posts it; apply a posted credit with apply_credit_memo. A " +
			"person always decides.",
		resource:    permission.ResourceInvoice,
		operation:   permission.OpCreate,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		artifact:    invoiceRecordEntity,
		rationale: "Raises what a customer owes or is owed outside any shipment; only a " +
			"person decides it, and it is never posted as it is made.",
		properties: map[string]any{
			paramCustomerID: stringProperty("The customer, from list_customers or "+
				"get_invoice.", 0),
			paramBillType: enumProperty("CreditMemo lowers what the customer owes; "+
				"DebitMemo raises it.", enumNames(memoBillTypes)),
			paramReason: stringProperty("Why the memo is raised; the customer sees it.",
				maxInvoiceReasonChars),
			paramAdjustmentLines: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "The memo's lines, each a positive amount.",
				toolschema.KeyMinItems:    1,
				toolschema.KeyMaxItems:    maxMemoLineCount,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeObject,
					toolschema.KeyProperties: map[string]any{
						paramLineDescription: stringProperty("What the line is for.",
							maxMemoLineTextChars),
						paramAmount: stringProperty("The line's unit amount in major units, "+
							"such as 125.00.", 0),
						paramMemoQuantity: stringProperty("How many; defaults to 1.", 0),
						paramAccessorialChargeID: stringProperty("The accessorial the line "+
							"corrects, from get_invoice's lines.", 0),
					},
					toolschema.KeyRequired: []string{paramLineDescription, paramAmount},
				},
			},
			paramReferenceInvoiceID: stringProperty("The posted invoice this memo relates to, "+
				"from list_invoices or get_invoice.", 0),
			paramInvoiceDate: dayProperty("The memo's date; defaults to today."),
			paramInvoiceMemo: stringProperty("A note printed on the memo.", maxInvoiceTextChars),
		},
		required: []string{paramCustomerID, paramBillType, paramReason, paramAdjustmentLines},
	}, receivablePlan[*serviceports.CreateMemoRequest, *invoice.Invoice]{
		request: memoRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.CreateMemoRequest,
			_ *serviceports.ToolExecuteParams,
		) (*invoice.Invoice, error) {
			return invoices.PreviewMemo(ctx, req)
		},
		refused: func(req *serviceports.CreateMemoRequest) string {
			return fmt.Sprintf("Would raise a draft %s.", req.BillType)
		},
		render: renderMemo,
		run: func(
			ctx context.Context,
			req *serviceports.CreateMemoRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := invoices.CreateMemo(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			return invoiceResult("created", created), nil
		},
	})
}

func memoRequest(params *serviceports.ToolExecuteParams) (*serviceports.CreateMemoRequest, error) {
	customerID, err := requirePulid(params.Params, paramCustomerID)
	if err != nil {
		return nil, err
	}
	billType, err := requireEnum(params.Params, paramBillType, memoBillTypes)
	if err != nil {
		return nil, err
	}
	reason, err := requireBoundedText(params.Params, paramReason, maxInvoiceReasonChars)
	if err != nil {
		return nil, err
	}
	note, err := boundedText(params.Params, paramInvoiceMemo, maxInvoiceTextChars)
	if err != nil {
		return nil, err
	}
	referenceID, _, err := optionalPulid(params.Params, paramReferenceInvoiceID)
	if err != nil {
		return nil, err
	}
	lines, err := memoLines(params.Params)
	if err != nil {
		return nil, err
	}

	req := &serviceports.CreateMemoRequest{
		TenantInfo:         tenantFrom(*params),
		CustomerID:         customerID,
		BillType:           billType,
		Lines:              lines,
		ReferenceInvoiceID: referenceID,
		Reason:             reason,
		Memo:               note,
		MemoKind:           invoice.MemoKindManual,
	}
	if strings.TrimSpace(optionalString(params.Params, paramInvoiceDate)) != "" {
		day, dayErr := requireUTCDay(params.Params, paramInvoiceDate)
		if dayErr != nil {
			return nil, dayErr
		}
		req.InvoiceDate = day.Unix()
	}

	return req, nil
}

func memoLines(params map[string]any) ([]*serviceports.CreateMemoLineInput, error) {
	entries, err := readObjects(params, paramAdjustmentLines, maxMemoLineCount)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("%s needs at least one line", paramAdjustmentLines)
	}

	lines := make([]*serviceports.CreateMemoLineInput, 0, len(entries))
	for idx, fields := range entries {
		line, lineErr := memoLine(fields)
		if lineErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramAdjustmentLines, idx, lineErr)
		}
		lines = append(lines, line)
	}

	return lines, nil
}

func memoLine(fields map[string]any) (*serviceports.CreateMemoLineInput, error) {
	description, err := requireBoundedText(fields, paramLineDescription, maxMemoLineTextChars)
	if err != nil {
		return nil, err
	}
	amount, err := requireAmount(fields, paramAmount)
	if err != nil {
		return nil, err
	}
	line := &serviceports.CreateMemoLineInput{Description: description, Amount: amount}
	if raw := strings.TrimSpace(optionalString(fields, paramMemoQuantity)); raw != "" {
		quantity, parseErr := decimal.NewFromString(raw)
		if parseErr != nil || !quantity.IsPositive() {
			return nil, fmt.Errorf("%s must be a positive number", paramMemoQuantity)
		}
		line.Quantity = quantity
	}
	if line.AccessorialChargeID, _, err = optionalPulid(fields, paramAccessorialChargeID); err != nil {
		return nil, err
	}

	return line, nil
}

func newVoidInvoiceTool(invoices invoiceVoider) serviceports.AgentTool {
	return newReportingReceivableTool(receivableSpec{
		name: "void_invoice",
		description: "Propose voiding an invoice. A draft is voided in place; a posted one " +
			"through a full-reversal adjustment that credits it, which may wait for an " +
			"approver. Its freight is rebilled or canceled as you say. Unapply payments and " +
			"credits first. A person always decides.",
		resource:    permission.ResourceInvoice,
		operation:   permission.OpCancel,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		artifact:    invoiceRecordEntity,
		rationale: "Takes an invoice out of circulation and reverses what it booked; it " +
			"cannot be undone, so only a person voids one.",
		properties: map[string]any{
			paramInvoiceID: invoiceIDProperty(),
			paramReason: stringProperty("Why it is voided; kept on the invoice.",
				maxInvoiceReasonChars),
			paramVoidDisposition: enumProperty("Rebill puts its freight back on the billing "+
				"queue to be billed again; DoNotRebill cancels it.", enumNames(voidDispositions)),
		},
		required: []string{paramInvoiceID, paramReason, paramVoidDisposition},
		target:   targetInvoice,
	}, receivablePlan[*serviceports.VoidInvoiceRequest, *serviceports.InvoiceVoidPreview]{
		request: voidRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.VoidInvoiceRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.InvoiceVoidPreview, error) {
			return invoices.PreviewVoid(ctx, req, params.Actor)
		},
		refused: func(*serviceports.VoidInvoiceRequest) string {
			return "Would void the invoice."
		},
		render: renderVoid,
		run: func(
			ctx context.Context,
			req *serviceports.VoidInvoiceRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			result, err := invoices.VoidInvoice(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			action := "voided"
			if result.PendingApproval {
				action = "void awaiting approval"
			}
			return invoiceResult(action, result.Invoice), nil
		},
	})
}

func voidRequest(params *serviceports.ToolExecuteParams) (*serviceports.VoidInvoiceRequest, error) {
	invoiceID, err := requirePulid(params.Params, paramInvoiceID)
	if err != nil {
		return nil, err
	}
	reason, err := requireBoundedText(params.Params, paramReason, maxInvoiceReasonChars)
	if err != nil {
		return nil, err
	}
	disposition, err := requireEnum(params.Params, paramVoidDisposition, voidDispositions)
	if err != nil {
		return nil, err
	}

	return &serviceports.VoidInvoiceRequest{
		InvoiceID:   invoiceID,
		TenantInfo:  tenantFrom(*params),
		Reason:      reason,
		Disposition: disposition,
	}, nil
}

func newSendInvoiceEDITool(invoices invoiceEDISender) serviceports.AgentTool {
	return newReceivableTool(receivableSpec{
		name: "send_invoice_edi",
		description: "Propose sending a posted invoice to the customer's EDI partner as a 210. " +
			"Set force only to resend one that already went, when the partner asks for it " +
			"again. A person always decides.",
		resource:    permission.ResourceInvoice,
		operation:   permission.OpSubmit,
		egress:      agent.EgressExternalRecipient,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		rationale: "Transmits the invoice to the customer's EDI trading partner, which " +
			"cannot be called back; only a person sends it.",
		properties: map[string]any{
			paramInvoiceID: invoiceIDProperty(),
			paramForceResend: map[string]any{
				toolschema.KeyType:        toolschema.TypeBoolean,
				toolschema.KeyDescription: "Resend an invoice that was already sent by EDI.",
			},
		},
		required: []string{paramInvoiceID},
		target:   targetInvoice,
	}, receivablePlan[*serviceports.SendInvoiceEDIRequest, *serviceports.InvoiceEDISendPreview]{
		request: func(params *serviceports.ToolExecuteParams) (*serviceports.SendInvoiceEDIRequest, error) {
			invoiceID, err := requirePulid(params.Params, paramInvoiceID)
			if err != nil {
				return nil, err
			}
			return &serviceports.SendInvoiceEDIRequest{
				InvoiceID:  invoiceID,
				TenantInfo: tenantFrom(*params),
				Force:      optionalBool(params.Params, paramForceResend),
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *serviceports.SendInvoiceEDIRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.InvoiceEDISendPreview, error) {
			return invoices.PreviewSendEDI(ctx, req, params.Actor)
		},
		refused: func(*serviceports.SendInvoiceEDIRequest) string {
			return "Would send the invoice by EDI."
		},
		render: renderEDISend,
		run: func(
			ctx context.Context,
			req *serviceports.SendInvoiceEDIRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := invoices.SendEDI(ctx, req, params.Actor)
			return nil, err
		},
	})
}
