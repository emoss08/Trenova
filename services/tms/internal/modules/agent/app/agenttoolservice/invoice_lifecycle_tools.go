package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
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
	memoBillTypes = agenttoolschema.Source("invoice.memoBillType", []billingqueue.BillType{
		billingqueue.BillTypeCreditMemo,
		billingqueue.BillTypeDebitMemo,
	})
	voidDispositions = agenttoolschema.Source("invoice.voidDisposition", []invoice.VoidDisposition{
		invoice.VoidDispositionRebill,
		invoice.VoidDispositionDoNotRebill,
	})
	recipientParams = []string{paramEmailTo, paramEmailCc, paramEmailBcc}
)

type invoiceDraftEditor interface {
	UpdateDraft(
		ctx context.Context,
		req *invoiceports.UpdateInvoiceDraftRequest,
		actor *permissionports.RequestActor,
	) (*invoice.Invoice, error)
	PreviewUpdateDraft(
		ctx context.Context,
		req *invoiceports.UpdateInvoiceDraftRequest,
	) (*invoiceports.InvoiceDraftUpdatePreview, error)
}

type invoicePDFGenerator interface {
	GeneratePDF(
		ctx context.Context,
		req *invoiceports.InvoicePreviewRequest,
		actor *permissionports.RequestActor,
	) (*invoiceports.GenerateInvoicePDFResult, error)
	PlanPDFGeneration(
		ctx context.Context,
		req *invoiceports.InvoicePreviewRequest,
		actor *permissionports.RequestActor,
	) (*invoiceports.InvoicePDFGenerationPlan, error)
}

type invoiceCreator interface {
	PlanCreateInvoices(
		ctx context.Context,
		req *invoiceports.PlanCreateInvoicesRequest,
	) (*invoiceports.CreateInvoicesPlan, error)
	CreateInvoicesFromOrder(
		ctx context.Context,
		req *invoiceports.CreateInvoiceFromOrderRequest,
		actor *permissionports.RequestActor,
	) (*invoiceports.CreateInvoicesResult, error)
	CreateInvoicesFromShipments(
		ctx context.Context,
		req *invoiceports.CreateInvoiceFromShipmentsRequest,
		actor *permissionports.RequestActor,
	) (*invoiceports.CreateInvoicesResult, error)
}

type invoiceMemoRaiser interface {
	CreateMemo(
		ctx context.Context,
		req *invoiceports.CreateMemoRequest,
		actor *permissionports.RequestActor,
	) (*invoice.Invoice, error)
	PreviewMemo(
		ctx context.Context,
		req *invoiceports.CreateMemoRequest,
	) (*invoice.Invoice, error)
}

type invoiceVoider interface {
	VoidInvoice(
		ctx context.Context,
		req *invoiceports.VoidInvoiceRequest,
		actor *permissionports.RequestActor,
	) (*invoiceports.VoidInvoiceResult, error)
	PreviewVoid(
		ctx context.Context,
		req *invoiceports.VoidInvoiceRequest,
		actor *permissionports.RequestActor,
	) (*invoiceports.InvoiceVoidPreview, error)
}

type invoiceEDISender interface {
	SendEDI(
		ctx context.Context,
		req *invoiceports.SendInvoiceEDIRequest,
		actor *permissionports.RequestActor,
	) (*invoiceports.InvoiceEDISendResult, error)
	PreviewSendEDI(
		ctx context.Context,
		req *invoiceports.SendInvoiceEDIRequest,
		actor *permissionports.RequestActor,
	) (*invoiceports.InvoiceEDISendPreview, error)
}

func invoiceIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceInvoice, "The invoice",
		"list_invoices or get_invoice")
}

func recipientsProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: description,
		toolschema.KeyItems:       map[string]any{toolschema.KeyType: toolschema.TypeString},
		toolschema.KeyMaxItems:    maxInvoiceRecipients,
	}
}

func newUpdateInvoiceDraftTool(invoices invoiceDraftEditor) agentports.AgentTool {
	return newReceivableTool(&receivableSpec{
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
		condition: &agentports.TierCondition{
			Description: "Changing who the invoice is emailed to is a proposal a person " +
				"decides; any other change runs once a person approves it.",
			Limit: draftUpdateTierLimit,
		},
		rationale: "Edits a draft nobody outside has seen; the customer sees it only when a " +
			"person sends it, but the recipients decide where it goes then.",
		properties: map[string]any{
			paramInvoiceID: invoiceIDProperty(),
			paramInvoiceMemo: toolschema.KeepEmpty(stringProperty(
				"The note printed on the invoice.", maxInvoiceTextChars)),
			paramRemittance: toolschema.KeepEmpty(stringProperty("How the customer should "+
				"pay, printed on the invoice.", maxInvoiceTextChars)),
			paramEmailSubject: toolschema.KeepEmpty(stringProperty("The subject of the email "+
				"the invoice goes in.", invoice.MaxEmailSubjectLength)),
			paramEmailBody: toolschema.KeepEmpty(stringProperty("The body of the email the "+
				"invoice goes in.", maxInvoiceEmailBodyChars)),
			paramEmailTo: toolschema.KeepEmpty(
				recipientsProperty("Who the invoice is emailed to; replaces the list.")),
			paramEmailCc:  toolschema.KeepEmpty(recipientsProperty("Who is copied; replaces the list.")),
			paramEmailBcc: toolschema.KeepEmpty(recipientsProperty("Who is blind-copied; replaces the list.")),
			paramAttachmentDocumentIDs: toolschema.KeepEmpty(agenttoolschema.OfResource(
				map[string]any{
					toolschema.KeyType: toolschema.TypeArray,
					toolschema.KeyDescription: "The documents attached to the email, from " +
						"search_documents; replaces the list.",
					toolschema.KeyItems: map[string]any{
						toolschema.KeyType: toolschema.TypeString,
					},
					toolschema.KeyMaxItems: maxInvoiceAttachments,
				},
				permission.ResourceDocument,
			)),
		},
		required: []string{paramInvoiceID},
		target:   targetInvoice,
	}, receivablePlan[*invoiceports.UpdateInvoiceDraftRequest, *invoiceports.InvoiceDraftUpdatePreview]{
		request: draftUpdateRequest,
		plan: func(
			ctx context.Context,
			req *invoiceports.UpdateInvoiceDraftRequest,
			_ *agentports.ToolExecuteParams,
		) (*invoiceports.InvoiceDraftUpdatePreview, error) {
			return invoices.PreviewUpdateDraft(ctx, req)
		},
		refused: func(*invoiceports.UpdateInvoiceDraftRequest) string {
			return "Would update the draft invoice."
		},
		render: renderDraftUpdate,
		run: func(
			ctx context.Context,
			req *invoiceports.UpdateInvoiceDraftRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := invoices.UpdateDraft(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func draftUpdateTierLimit(
	_ context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // a TierCondition passes params by value
) agent.AutonomyTier {
	for _, key := range recipientParams {
		if _, ok := params.Params[key]; ok {
			return agent.TierPropose
		}
	}

	return agent.TierActWithApproval
}

func draftUpdateRequest(
	params *agentports.ToolExecuteParams,
) (*invoiceports.UpdateInvoiceDraftRequest, error) {
	invoiceID, err := requirePulid(params.Params, paramInvoiceID)
	if err != nil {
		return nil, err
	}
	req := &invoiceports.UpdateInvoiceDraftRequest{
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
		value, present, textErr := presentText(params.Params, text.key, text.limit)
		if textErr != nil {
			return nil, textErr
		}
		if present {
			*text.into = &value
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

func presentText(
	params map[string]any,
	key string,
	limit int,
) (text string, present bool, err error) {
	raw, ok := params[key]
	if !ok {
		return "", false, nil
	}
	if _, isString := raw.(string); !isString {
		return "", false, fmt.Errorf("parameter %q must be a string", key)
	}
	if text, err = boundedText(params, key, limit); err != nil {
		return "", false, err
	}

	return text, true, nil
}

func presentList(
	params map[string]any,
	key string,
	limit int,
) (values []string, present bool, err error) {
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

	values = make([]string, 0, len(entries))
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

func newGenerateInvoicePDFTool(invoices invoicePDFGenerator) agentports.AgentTool {
	plan := func(
		ctx context.Context,
		req *invoiceports.InvoicePreviewRequest,
		params *agentports.ToolExecuteParams,
	) (*invoiceports.InvoicePDFGenerationPlan, error) {
		generation, err := invoices.PlanPDFGeneration(ctx, req, params.Actor)
		if err != nil {
			return nil, err
		}
		if generation.AutoSends {
			return nil, errPDFWouldSend
		}
		return generation, nil
	}

	return newReceivableTool(&receivableSpec{
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
	}, receivablePlan[*invoiceports.InvoicePreviewRequest, *invoiceports.InvoicePDFGenerationPlan]{
		request: func(params *agentports.ToolExecuteParams) (*invoiceports.InvoicePreviewRequest, error) {
			invoiceID, err := requirePulid(params.Params, paramInvoiceID)
			if err != nil {
				return nil, err
			}
			return &invoiceports.InvoicePreviewRequest{
				InvoiceID:  invoiceID,
				TenantInfo: tenantFrom(*params),
			}, nil
		},
		plan: plan,
		refused: func(*invoiceports.InvoicePreviewRequest) string {
			return "Would render the invoice's PDF."
		},
		render: renderPDFGeneration,
		run: func(
			ctx context.Context,
			req *invoiceports.InvoicePreviewRequest,
			params *agentports.ToolExecuteParams,
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
	plan *invoiceports.PlanCreateInvoicesRequest
}

func newCreateInvoiceTool(invoices invoiceCreator) agentports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:        "create_invoice",
		searchTerms: []string{"invoice delivered", "bill completed"},
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
			paramOrderID: agenttoolschema.RecordIDText(permission.ResourceOrder,
				"The order to bill as one grouped invoice, from get_order or get_shipment."),
			paramShipmentIDs: agenttoolschema.RecordIDs(
				permission.ResourceShipment,
				"The shipments to bill, from search_shipments or "+
					"get_shipment. With orderId, only these of the order's shipments.",
				maxInvoiceShipments,
			),
			paramOffCycleNote: stringProperty("Why this customer's freight is billed now "+
				"rather than on their periodic statement. Required only for statement "+
				"customers.", maxInvoiceReasonChars),
		},
	}, receivablePlan[*createInvoicesRequest, *invoiceports.CreateInvoicesPlan]{
		request: createInvoicesRequestFrom,
		plan: func(
			ctx context.Context,
			req *createInvoicesRequest,
			_ *agentports.ToolExecuteParams,
		) (*invoiceports.CreateInvoicesPlan, error) {
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
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			var (
				made *invoiceports.CreateInvoicesResult
				err  error
			)
			if req.plan.OrderID.IsNotNil() {
				made, err = invoices.CreateInvoicesFromOrder(ctx,
					&invoiceports.CreateInvoiceFromOrderRequest{
						OrderID:        req.plan.OrderID,
						ShipmentIDs:    req.plan.ShipmentIDs,
						TenantInfo:     req.plan.TenantInfo,
						OffCycleReason: req.plan.OffCycleReason,
					}, params.Actor)
			} else {
				made, err = invoices.CreateInvoicesFromShipments(ctx,
					&invoiceports.CreateInvoiceFromShipmentsRequest{
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

func createInvoicesRequestFrom(
	params *agentports.ToolExecuteParams,
) (*createInvoicesRequest, error) {
	orderID, hasOrder, err := optionalPulid(params.Params, paramOrderID)
	if err != nil {
		return nil, err
	}
	req := &invoiceports.PlanCreateInvoicesRequest{
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

func newCreateInvoiceMemoTool(invoices invoiceMemoRaiser) agentports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
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
			paramCustomerID: agenttoolschema.RecordIDText(permission.ResourceCustomer,
				"The customer, from list_customers or get_invoice."),
			paramBillType: agenttoolschema.Enum("CreditMemo lowers what the customer owes; "+
				"DebitMemo raises it.", memoBillTypes),
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
						paramAccessorialChargeID: agenttoolschema.RecordIDText(
							permission.ResourceAccessorialCharge,
							"The accessorial the line corrects, from get_invoice's lines.",
						),
					},
					toolschema.KeyRequired:             []string{paramLineDescription, paramAmount},
					toolschema.KeyAdditionalProperties: false,
				},
			},
			paramReferenceInvoiceID: agenttoolschema.RecordIDText(permission.ResourceInvoice,
				"The posted invoice this memo relates to, from list_invoices or get_invoice."),
			paramInvoiceDate: agenttoolschema.Date("The memo's date; defaults to today."),
			paramInvoiceMemo: stringProperty("A note printed on the memo.", maxInvoiceTextChars),
		},
		required: []string{paramCustomerID, paramBillType, paramReason, paramAdjustmentLines},
	}, receivablePlan[*invoiceports.CreateMemoRequest, *invoice.Invoice]{
		request: memoRequest,
		plan: func(
			ctx context.Context,
			req *invoiceports.CreateMemoRequest,
			_ *agentports.ToolExecuteParams,
		) (*invoice.Invoice, error) {
			return invoices.PreviewMemo(ctx, req)
		},
		refused: func(req *invoiceports.CreateMemoRequest) string {
			return fmt.Sprintf("Would raise a draft %s.", req.BillType)
		},
		render: renderMemo,
		run: func(
			ctx context.Context,
			req *invoiceports.CreateMemoRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := invoices.CreateMemo(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			return invoiceResult("created", created), nil
		},
	})
}

func memoRequest(params *agentports.ToolExecuteParams) (*invoiceports.CreateMemoRequest, error) {
	customerID, err := requirePulid(params.Params, paramCustomerID)
	if err != nil {
		return nil, err
	}
	billType, err := requireEnum(params.Params, paramBillType, memoBillTypes.Values)
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

	req := &invoiceports.CreateMemoRequest{
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

func memoLines(params map[string]any) ([]*invoiceports.CreateMemoLineInput, error) {
	entries, err := readObjects(params, paramAdjustmentLines, maxMemoLineCount)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("%s needs at least one line", paramAdjustmentLines)
	}

	lines := make([]*invoiceports.CreateMemoLineInput, 0, len(entries))
	for idx, fields := range entries {
		line, lineErr := memoLine(fields)
		if lineErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramAdjustmentLines, idx, lineErr)
		}
		lines = append(lines, line)
	}

	return lines, nil
}

func memoLine(fields map[string]any) (*invoiceports.CreateMemoLineInput, error) {
	description, err := requireBoundedText(fields, paramLineDescription, maxMemoLineTextChars)
	if err != nil {
		return nil, err
	}
	amount, err := requireAmount(fields, paramAmount)
	if err != nil {
		return nil, err
	}
	line := &invoiceports.CreateMemoLineInput{Description: description, Amount: amount}
	if raw := strings.TrimSpace(optionalString(fields, paramMemoQuantity)); raw != "" {
		quantity, parseErr := decimal.NewFromString(raw)
		if parseErr != nil || !quantity.IsPositive() {
			return nil, fmt.Errorf("%s must be a positive number", paramMemoQuantity)
		}
		line.Quantity = quantity
	}
	if line.AccessorialChargeID, _, err = optionalPulid(
		fields,
		paramAccessorialChargeID,
	); err != nil {
		return nil, err
	}

	return line, nil
}

func newVoidInvoiceTool(invoices invoiceVoider) agentports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:        "void_invoice",
		searchTerms: []string{"kill invoice", "cancel invoice"},
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
			paramVoidDisposition: agenttoolschema.Enum("Rebill puts its freight back on the "+
				"billing queue to be billed again; DoNotRebill cancels it.", voidDispositions),
		},
		required: []string{paramInvoiceID, paramReason, paramVoidDisposition},
		target:   targetInvoice,
	}, receivablePlan[*invoiceports.VoidInvoiceRequest, *invoiceports.InvoiceVoidPreview]{
		request: voidRequest,
		plan: func(
			ctx context.Context,
			req *invoiceports.VoidInvoiceRequest,
			params *agentports.ToolExecuteParams,
		) (*invoiceports.InvoiceVoidPreview, error) {
			return invoices.PreviewVoid(ctx, req, params.Actor)
		},
		refused: func(*invoiceports.VoidInvoiceRequest) string {
			return "Would void the invoice."
		},
		render: renderVoid,
		run: func(
			ctx context.Context,
			req *invoiceports.VoidInvoiceRequest,
			params *agentports.ToolExecuteParams,
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

func voidRequest(params *agentports.ToolExecuteParams) (*invoiceports.VoidInvoiceRequest, error) {
	invoiceID, err := requirePulid(params.Params, paramInvoiceID)
	if err != nil {
		return nil, err
	}
	reason, err := requireBoundedText(params.Params, paramReason, maxInvoiceReasonChars)
	if err != nil {
		return nil, err
	}
	disposition, err := requireEnum(params.Params, paramVoidDisposition, voidDispositions.Values)
	if err != nil {
		return nil, err
	}

	return &invoiceports.VoidInvoiceRequest{
		InvoiceID:   invoiceID,
		TenantInfo:  tenantFrom(*params),
		Reason:      reason,
		Disposition: disposition,
	}, nil
}

func newSendInvoiceEDITool(invoices invoiceEDISender) agentports.AgentTool {
	return newReceivableTool(&receivableSpec{
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
	}, receivablePlan[*invoiceports.SendInvoiceEDIRequest, *invoiceports.InvoiceEDISendPreview]{
		request: func(params *agentports.ToolExecuteParams) (*invoiceports.SendInvoiceEDIRequest, error) {
			invoiceID, err := requirePulid(params.Params, paramInvoiceID)
			if err != nil {
				return nil, err
			}
			return &invoiceports.SendInvoiceEDIRequest{
				InvoiceID:  invoiceID,
				TenantInfo: tenantFrom(*params),
				Force:      optionalBool(params.Params, paramForceResend),
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *invoiceports.SendInvoiceEDIRequest,
			params *agentports.ToolExecuteParams,
		) (*invoiceports.InvoiceEDISendPreview, error) {
			return invoices.PreviewSendEDI(ctx, req, params.Actor)
		},
		refused: func(*invoiceports.SendInvoiceEDIRequest) string {
			return "Would send the invoice by EDI."
		},
		render: renderEDISend,
		run: func(
			ctx context.Context,
			req *invoiceports.SendInvoiceEDIRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := invoices.SendEDI(ctx, req, params.Actor)
			return nil, err
		},
	})
}
