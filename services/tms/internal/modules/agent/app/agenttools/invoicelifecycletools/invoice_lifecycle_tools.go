package invoicelifecycletools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/mailtext"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/shopspring/decimal"
)

const ParamOffCycleNote = "offCycleReason"

const ParamVoidDisposition = "disposition"

const paramMemoQuantity = "quantity"

const paramAccessorialChargeID = "accessorialChargeId"

const paramReferenceInvoiceID = "referenceInvoiceId"

const paramInvoiceMemo = "memo"

const paramRemittance = "remittanceInstructions"

const paramEmailSubject = "emailSubject"

const paramEmailBody = "emailBody"

const paramEmailTo = "emailTo"

const paramEmailCc = "emailCc"

const paramEmailBcc = "emailBcc"

const paramAttachmentDocumentIDs = "attachmentDocumentIds"

const maxInvoiceTextChars = 4000

const maxInvoiceEmailBodyChars = 20000

const maxInvoiceRecipients = 25

const maxInvoiceAttachments = 50

const maxInvoiceShipments = 100

const maxInvoiceReasonChars = 1000

const maxMemoLineCount = 200

const maxMemoLineTextChars = 500

var ErrNothingToUpdate = errors.New(
	"name at least one thing to change on the draft: memo, remittanceInstructions, " +
		"emailSubject, emailBody, emailTo, emailCc, emailBcc or attachmentDocumentIds",
)

var ErrPDFWouldSend = errortypes.NewBusinessError(
	"rendering this invoice's PDF would email it to the customer at once, as their " +
		"billing profile sends invoices when the PDF is made; use invoice_send, which a " +
		"person approves, instead",
)

var ErrOrderOrShipments = errors.New(
	"name the freight to invoice: an orderId, shipmentIds, or an orderId with the " +
		"shipmentIds of it to bill",
)

var memoBillTypes = agenttoolschema.Source("invoice.memoBillType", []billingqueue.BillType{
	billingqueue.BillTypeCreditMemo,
	billingqueue.BillTypeDebitMemo,
})

var voidDispositions = agenttoolschema.Source("invoice.voidDisposition", []invoice.VoidDisposition{
	invoice.VoidDispositionRebill,
	invoice.VoidDispositionDoNotRebill,
})

var recipientParams = []string{paramEmailTo, paramEmailCc, paramEmailBcc}

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
		"invoice_list or invoice_get")
}

func recipientsProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: description,
		toolschema.KeyItems:       map[string]any{toolschema.KeyType: toolschema.TypeString},
		toolschema.KeyMaxItems:    maxInvoiceRecipients,
	}
}

func NewUpdateInvoiceDraftTool(invoices invoiceDraftEditor) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "invoice_update_draft",
		Description: "Change what a draft invoice says and where it goes: its memo, remittance " +
			"instructions, email subject and body, recipients and attached documents. Only " +
			"the fields you pass change; an empty string or list clears one. A posted " +
			"invoice is changed with an invoice adjustment instead.",
		Resource:    permission.ResourceInvoice,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		TaintHold: "Outside content could put its own wording or recipients on an invoice " +
			"the customer receives.",
		Condition: &agenttoolports.TierCondition{
			Description: "Changing who the invoice is emailed to is a proposal a person " +
				"decides; any other change runs once a person approves it.",
			Limit: draftUpdateTierLimit,
		},
		Rationale: "Edits a draft nobody outside has seen; the customer sees it only when a " +
			"person sends it, but the recipients decide where it goes then.",
		Properties: map[string]any{
			toolparams.ParamInvoiceID: invoiceIDProperty(),
			paramInvoiceMemo: toolschema.KeepEmpty(toolparams.StringProperty(
				"The note printed on the invoice.", maxInvoiceTextChars)),
			paramRemittance: toolschema.KeepEmpty(
				toolparams.StringProperty("How the customer should "+
					"pay, printed on the invoice.", maxInvoiceTextChars),
			),
			paramEmailSubject: toolschema.KeepEmpty(
				toolparams.StringProperty("The subject of the email "+
					"the invoice goes in.", invoice.MaxEmailSubjectLength),
			),
			paramEmailBody: toolschema.KeepEmpty(
				toolparams.StringProperty("The body of the email the "+
					"invoice goes in.", maxInvoiceEmailBodyChars),
			),
			paramEmailTo: toolschema.KeepEmpty(
				recipientsProperty("Who the invoice is emailed to; replaces the list.")),
			paramEmailCc: toolschema.KeepEmpty(
				recipientsProperty("Who is copied; replaces the list."),
			),
			paramEmailBcc: toolschema.KeepEmpty(
				recipientsProperty("Who is blind-copied; replaces the list."),
			),
			paramAttachmentDocumentIDs: toolschema.KeepEmpty(agenttoolschema.OfResource(
				map[string]any{
					toolschema.KeyType: toolschema.TypeArray,
					toolschema.KeyDescription: "The documents attached to the email, from " +
						"document_list; replaces the list.",
					toolschema.KeyItems: map[string]any{
						toolschema.KeyType: toolschema.TypeString,
					},
					toolschema.KeyMaxItems: maxInvoiceAttachments,
				},
				permission.ResourceDocument,
			)),
		},
		Required: []string{toolparams.ParamInvoiceID},
		Target:   toolparams.TargetInvoice,
	}, toolspec.ReceivablePlan[*invoiceports.UpdateInvoiceDraftRequest, *invoiceports.InvoiceDraftUpdatePreview]{
		Request: draftUpdateRequest,
		Plan: func(
			ctx context.Context,
			req *invoiceports.UpdateInvoiceDraftRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*invoiceports.InvoiceDraftUpdatePreview, error) {
			return invoices.PreviewUpdateDraft(ctx, req)
		},
		Refused: func(*invoiceports.UpdateInvoiceDraftRequest) string {
			return "Would update the draft invoice."
		},
		Render: renderDraftUpdate,
		Run: func(
			ctx context.Context,
			req *invoiceports.UpdateInvoiceDraftRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := invoices.UpdateDraft(ctx, req, params.Actor)
			return nil, err
		},
	})
}

func draftUpdateTierLimit(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // a TierCondition passes params by value
) agent.AutonomyTier {
	for _, key := range recipientParams {
		if _, ok := params.Params[key]; ok {
			return agent.TierPropose
		}
	}

	return agent.TierActWithApproval
}

func draftUpdateRequest(
	params *agenttoolports.ToolExecuteParams,
) (*invoiceports.UpdateInvoiceDraftRequest, error) {
	invoiceID, err := toolparams.RequirePulid(params.Params, toolparams.ParamInvoiceID)
	if err != nil {
		return nil, err
	}
	req := &invoiceports.UpdateInvoiceDraftRequest{
		InvoiceID:  invoiceID,
		TenantInfo: toolparams.TenantFrom(params),
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
		return nil, ErrNothingToUpdate
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
	if text, err = toolparams.BoundedText(params, key, limit); err != nil {
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

	addresses := mailtext.NormalizeEmailAddresses(values)
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

func NewGenerateInvoicePDFTool(invoices invoicePDFGenerator) agenttoolports.AgentTool {
	plan := func(
		ctx context.Context,
		req *invoiceports.InvoicePreviewRequest,
		params *agenttoolports.ToolExecuteParams,
	) (*invoiceports.InvoicePDFGenerationPlan, error) {
		generation, err := invoices.PlanPDFGeneration(ctx, req, params.Actor)
		if err != nil {
			return nil, err
		}
		if generation.AutoSends {
			return nil, ErrPDFWouldSend
		}
		return generation, nil
	}

	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "invoice_generate_pdf",
		Description: "Render an invoice's PDF from what it says now and file it on the " +
			"invoice, replacing the one there. Refused when the customer's billing profile " +
			"would email the invoice the moment its PDF is made; send it with invoice_send.",
		Resource:    permission.ResourceInvoice,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierAutoExecute,
		Rationale: "Renders the invoice as it stands into its filed PDF; nothing is sent, and " +
			"it is refused whenever rendering would email the customer.",
		Properties: map[string]any{toolparams.ParamInvoiceID: invoiceIDProperty()},
		Required:   []string{toolparams.ParamInvoiceID},
		Target:     toolparams.TargetInvoice,
	}, toolspec.ReceivablePlan[*invoiceports.InvoicePreviewRequest, *invoiceports.InvoicePDFGenerationPlan]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*invoiceports.InvoicePreviewRequest, error) {
			invoiceID, err := toolparams.RequirePulid(params.Params, toolparams.ParamInvoiceID)
			if err != nil {
				return nil, err
			}
			return &invoiceports.InvoicePreviewRequest{
				InvoiceID:  invoiceID,
				TenantInfo: toolparams.TenantFrom(params),
			}, nil
		},
		Plan: plan,
		Refused: func(*invoiceports.InvoicePreviewRequest) string {
			return "Would render the invoice's PDF."
		},
		Render: renderPDFGeneration,
		Run: func(
			ctx context.Context,
			req *invoiceports.InvoicePreviewRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
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

func NewCreateInvoiceTool(invoices invoiceCreator) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:        "invoice_create",
		SearchTerms: []string{"invoice delivered", "bill completed"},
		Description: "Propose drafting invoices for completed freight: an order's billable " +
			"shipments as one grouped invoice, or shipments on their own. A split-billed " +
			"shipment makes one draft per payer. Drafts are posted with invoice_post. A " +
			"person always decides.",
		Resource:    permission.ResourceInvoice,
		Operation:   permission.OpCreate,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Artifact:    changepreview.InvoiceRecordEntity,
		Rationale: "Turns freight into receivables and takes it off the billing queue and any " +
			"statement; only a person decides what is billed and when.",
		Properties: map[string]any{
			toolparams.ParamOrderID: agenttoolschema.RecordIDText(permission.ResourceOrder,
				"The order to bill as one grouped invoice, from order_get or shipment_get."),
			toolparams.ParamShipmentIDs: agenttoolschema.RecordIDs(
				permission.ResourceShipment,
				"The shipments to bill, from shipment_list or "+
					"shipment_get. With orderId, only these of the order's shipments.",
				maxInvoiceShipments,
			),
			ParamOffCycleNote: toolparams.StringProperty(
				"Why this customer's freight is billed now "+
					"rather than on their periodic statement. Required only for statement "+
					"customers.",
				maxInvoiceReasonChars,
			),
		},
	}, toolspec.ReceivablePlan[*createInvoicesRequest, *invoiceports.CreateInvoicesPlan]{
		Request: createInvoicesRequestFrom,
		Plan: func(
			ctx context.Context,
			req *createInvoicesRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*invoiceports.CreateInvoicesPlan, error) {
			return invoices.PlanCreateInvoices(ctx, req.plan)
		},
		Refused: func(req *createInvoicesRequest) string {
			if req.plan.OrderID.IsNotNil() {
				return "Would invoice the order."
			}
			return fmt.Sprintf(
				"Would invoice %s.",
				changepreview.CountOf(len(req.plan.ShipmentIDs), "shipment"),
			)
		},
		Render: renderCreateInvoices,
		Run: func(
			ctx context.Context,
			req *createInvoicesRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
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
	params *agenttoolports.ToolExecuteParams,
) (*createInvoicesRequest, error) {
	orderID, hasOrder, err := toolparams.OptionalPulid(params.Params, toolparams.ParamOrderID)
	if err != nil {
		return nil, err
	}
	req := &invoiceports.PlanCreateInvoicesRequest{
		OrderID:    orderID,
		TenantInfo: toolparams.TenantFrom(params),
	}
	if _, ok := params.Params[toolparams.ParamShipmentIDs]; ok {
		if req.ShipmentIDs, err = toolparams.RequirePulidSlice(
			params.Params, toolparams.ParamShipmentIDs, maxInvoiceShipments,
		); err != nil {
			return nil, err
		}
	}
	if !hasOrder && len(req.ShipmentIDs) == 0 {
		return nil, ErrOrderOrShipments
	}
	if req.OffCycleReason, err = toolparams.BoundedText(
		params.Params, ParamOffCycleNote, maxInvoiceReasonChars,
	); err != nil {
		return nil, err
	}

	return &createInvoicesRequest{plan: req}, nil
}

func invoiceResult(action string, entity *invoice.Invoice) *agentrecord.ToolExecutionResult {
	if entity == nil || entity.ID.IsNil() {
		return &agentrecord.ToolExecutionResult{Action: action, Kind: "invoice"}
	}

	return &agentrecord.ToolExecutionResult{
		Action: action,
		Kind:   strings.ToLower(string(entity.BillType)),
		IDs: map[string]string{
			toolparams.ParamInvoiceID: entity.ID.String(),
			"number":                  entity.Number,
		},
		Record: &agentrecord.RecordRef{
			EntityType: changepreview.InvoiceRecordEntity,
			ID:         entity.ID.String(),
		},
	}
}

func NewCreateInvoiceMemoTool(invoices invoiceMemoRaiser) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name: "invoice_create_memo",
		Description: "Propose a draft credit or debit memo to a customer with no shipment " +
			"behind it, such as a goodwill credit or a returned-check fee. It stays a draft " +
			"until invoice_post posts it; apply a posted credit with receivable_apply_credit_memo. A " +
			"person always decides.",
		Resource:    permission.ResourceInvoice,
		Operation:   permission.OpCreate,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Artifact:    changepreview.InvoiceRecordEntity,
		Rationale: "Raises what a customer owes or is owed outside any shipment; only a " +
			"person decides it, and it is never posted as it is made.",
		Properties: map[string]any{
			toolparams.ParamCustomerID: agenttoolschema.RecordIDText(permission.ResourceCustomer,
				"The customer, from customer_list or invoice_get."),
			toolparams.ParamBillType: agenttoolschema.Enum(
				"CreditMemo lowers what the customer owes; "+
					"DebitMemo raises it.",
				memoBillTypes,
			),
			toolparams.ParamReason: toolparams.StringProperty(
				"Why the memo is raised; the customer sees it.",
				maxInvoiceReasonChars,
			),
			toolparams.ParamAdjustmentLines: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "The memo's lines, each a positive amount.",
				toolschema.KeyMinItems:    1,
				toolschema.KeyMaxItems:    maxMemoLineCount,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeObject,
					toolschema.KeyProperties: map[string]any{
						toolparams.ParamLineDescription: toolparams.StringProperty(
							"What the line is for.",
							maxMemoLineTextChars,
						),
						toolparams.ParamAmount: toolparams.StringProperty(
							"The line's unit amount in major units, "+
								"such as 125.00.",
							0,
						),
						paramMemoQuantity: toolparams.StringProperty("How many; defaults to 1.", 0),
						paramAccessorialChargeID: agenttoolschema.RecordIDText(
							permission.ResourceAccessorialCharge,
							"The accessorial the line corrects, from invoice_get's lines.",
						),
					},
					toolschema.KeyRequired: []string{
						toolparams.ParamLineDescription,
						toolparams.ParamAmount,
					},
					toolschema.KeyAdditionalProperties: false,
				},
			},
			paramReferenceInvoiceID: agenttoolschema.RecordIDText(permission.ResourceInvoice,
				"The posted invoice this memo relates to, from invoice_list or invoice_get."),
			toolparams.ParamInvoiceDate: agenttoolschema.Date(
				"The memo's date; defaults to today.",
			),
			paramInvoiceMemo: toolparams.StringProperty(
				"A note printed on the memo.",
				maxInvoiceTextChars,
			),
		},
		Required: []string{
			toolparams.ParamCustomerID,
			toolparams.ParamBillType,
			toolparams.ParamReason,
			toolparams.ParamAdjustmentLines,
		},
	}, toolspec.ReceivablePlan[*invoiceports.CreateMemoRequest, *invoice.Invoice]{
		Request: memoRequest,
		Plan: func(
			ctx context.Context,
			req *invoiceports.CreateMemoRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*invoice.Invoice, error) {
			return invoices.PreviewMemo(ctx, req)
		},
		Refused: func(req *invoiceports.CreateMemoRequest) string {
			return fmt.Sprintf("Would raise a draft %s.", req.BillType)
		},
		Render: renderMemo,
		Run: func(
			ctx context.Context,
			req *invoiceports.CreateMemoRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			created, err := invoices.CreateMemo(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			return invoiceResult("created", created), nil
		},
	})
}

func memoRequest(
	params *agenttoolports.ToolExecuteParams,
) (*invoiceports.CreateMemoRequest, error) {
	customerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamCustomerID)
	if err != nil {
		return nil, err
	}
	billType, err := toolparams.RequireEnum(
		params.Params,
		toolparams.ParamBillType,
		memoBillTypes.Values,
	)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.RequireBoundedText(
		params.Params,
		toolparams.ParamReason,
		maxInvoiceReasonChars,
	)
	if err != nil {
		return nil, err
	}
	note, err := toolparams.BoundedText(params.Params, paramInvoiceMemo, maxInvoiceTextChars)
	if err != nil {
		return nil, err
	}
	referenceID, _, err := toolparams.OptionalPulid(params.Params, paramReferenceInvoiceID)
	if err != nil {
		return nil, err
	}
	lines, err := memoLines(params.Params)
	if err != nil {
		return nil, err
	}

	req := &invoiceports.CreateMemoRequest{
		TenantInfo:         toolparams.TenantFrom(params),
		CustomerID:         customerID,
		BillType:           billType,
		Lines:              lines,
		ReferenceInvoiceID: referenceID,
		Reason:             reason,
		Memo:               note,
		MemoKind:           invoice.MemoKindManual,
	}
	if strings.TrimSpace(
		toolparams.OptionalString(params.Params, toolparams.ParamInvoiceDate),
	) != "" {
		day, dayErr := toolparams.RequireUTCDay(params.Params, toolparams.ParamInvoiceDate)
		if dayErr != nil {
			return nil, dayErr
		}
		req.InvoiceDate = day.Unix()
	}

	return req, nil
}

func memoLines(params map[string]any) ([]*invoiceports.CreateMemoLineInput, error) {
	entries, err := toolparams.ReadObjects(
		params,
		toolparams.ParamAdjustmentLines,
		maxMemoLineCount,
	)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("%s needs at least one line", toolparams.ParamAdjustmentLines)
	}

	lines := make([]*invoiceports.CreateMemoLineInput, 0, len(entries))
	for idx, fields := range entries {
		line, lineErr := memoLine(fields)
		if lineErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", toolparams.ParamAdjustmentLines, idx, lineErr)
		}
		lines = append(lines, line)
	}

	return lines, nil
}

func memoLine(fields map[string]any) (*invoiceports.CreateMemoLineInput, error) {
	description, err := toolparams.RequireBoundedText(
		fields,
		toolparams.ParamLineDescription,
		maxMemoLineTextChars,
	)
	if err != nil {
		return nil, err
	}
	amount, err := toolparams.RequireAmount(fields, toolparams.ParamAmount)
	if err != nil {
		return nil, err
	}
	line := &invoiceports.CreateMemoLineInput{Description: description, Amount: amount}
	if raw := strings.TrimSpace(toolparams.OptionalString(fields, paramMemoQuantity)); raw != "" {
		quantity, parseErr := decimal.NewFromString(raw)
		if parseErr != nil || !quantity.IsPositive() {
			return nil, fmt.Errorf("%s must be a positive number", paramMemoQuantity)
		}
		line.Quantity = quantity
	}
	if line.AccessorialChargeID, _, err = toolparams.OptionalPulid(
		fields,
		paramAccessorialChargeID,
	); err != nil {
		return nil, err
	}

	return line, nil
}

func NewVoidInvoiceTool(invoices invoiceVoider) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:        "invoice_void",
		SearchTerms: []string{"kill invoice", "cancel invoice"},
		Description: "Propose voiding an invoice. A draft is voided in place; a posted one " +
			"through a full-reversal adjustment that credits it, which may wait for an " +
			"approver. Its freight is rebilled or canceled as you say. Unapply payments and " +
			"credits first. A person always decides.",
		Resource:    permission.ResourceInvoice,
		Operation:   permission.OpCancel,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Artifact:    changepreview.InvoiceRecordEntity,
		Rationale: "Takes an invoice out of circulation and reverses what it booked; it " +
			"cannot be undone, so only a person voids one.",
		Properties: map[string]any{
			toolparams.ParamInvoiceID: invoiceIDProperty(),
			toolparams.ParamReason: toolparams.StringProperty(
				"Why it is voided; kept on the invoice.",
				maxInvoiceReasonChars,
			),
			ParamVoidDisposition: agenttoolschema.Enum("Rebill puts its freight back on the "+
				"billing queue to be billed again; DoNotRebill cancels it.", voidDispositions),
		},
		Required: []string{toolparams.ParamInvoiceID, toolparams.ParamReason, ParamVoidDisposition},
		Target:   toolparams.TargetInvoice,
	}, toolspec.ReceivablePlan[*invoiceports.VoidInvoiceRequest, *invoiceports.InvoiceVoidPreview]{
		Request: voidRequest,
		Plan: func(
			ctx context.Context,
			req *invoiceports.VoidInvoiceRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*invoiceports.InvoiceVoidPreview, error) {
			return invoices.PreviewVoid(ctx, req, params.Actor)
		},
		Refused: func(*invoiceports.VoidInvoiceRequest) string {
			return "Would void the invoice."
		},
		Render: renderVoid,
		Run: func(
			ctx context.Context,
			req *invoiceports.VoidInvoiceRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
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

func voidRequest(
	params *agenttoolports.ToolExecuteParams,
) (*invoiceports.VoidInvoiceRequest, error) {
	invoiceID, err := toolparams.RequirePulid(params.Params, toolparams.ParamInvoiceID)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.RequireBoundedText(
		params.Params,
		toolparams.ParamReason,
		maxInvoiceReasonChars,
	)
	if err != nil {
		return nil, err
	}
	disposition, err := toolparams.RequireEnum(
		params.Params,
		ParamVoidDisposition,
		voidDispositions.Values,
	)
	if err != nil {
		return nil, err
	}

	return &invoiceports.VoidInvoiceRequest{
		InvoiceID:   invoiceID,
		TenantInfo:  toolparams.TenantFrom(params),
		Reason:      reason,
		Disposition: disposition,
	}, nil
}

func NewSendInvoiceEDITool(invoices invoiceEDISender) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "invoice_send_edi",
		Description: "Propose sending a posted invoice to the customer's EDI partner as a 210. " +
			"Set force only to resend one that already went, when the partner asks for it " +
			"again. A person always decides.",
		Resource:    permission.ResourceInvoice,
		Operation:   permission.OpSubmit,
		Egress:      agent.EgressExternalRecipient,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Rationale: "Transmits the invoice to the customer's EDI trading partner, which " +
			"cannot be called back; only a person sends it.",
		Properties: map[string]any{
			toolparams.ParamInvoiceID: invoiceIDProperty(),
			toolparams.ParamForceResend: map[string]any{
				toolschema.KeyType:        toolschema.TypeBoolean,
				toolschema.KeyDescription: "Resend an invoice that was already sent by EDI.",
			},
		},
		Required: []string{toolparams.ParamInvoiceID},
		Target:   toolparams.TargetInvoice,
	}, toolspec.ReceivablePlan[*invoiceports.SendInvoiceEDIRequest, *invoiceports.InvoiceEDISendPreview]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*invoiceports.SendInvoiceEDIRequest, error) {
			invoiceID, err := toolparams.RequirePulid(params.Params, toolparams.ParamInvoiceID)
			if err != nil {
				return nil, err
			}
			return &invoiceports.SendInvoiceEDIRequest{
				InvoiceID:  invoiceID,
				TenantInfo: toolparams.TenantFrom(params),
				Force:      toolparams.OptionalBool(params.Params, toolparams.ParamForceResend),
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *invoiceports.SendInvoiceEDIRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*invoiceports.InvoiceEDISendPreview, error) {
			return invoices.PreviewSendEDI(ctx, req, params.Actor)
		},
		Refused: func(*invoiceports.SendInvoiceEDIRequest) string {
			return "Would send the invoice by EDI."
		},
		Render: renderEDISend,
		Run: func(
			ctx context.Context,
			req *invoiceports.SendInvoiceEDIRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := invoices.SendEDI(ctx, req, params.Actor)
			return nil, err
		},
	})
}
