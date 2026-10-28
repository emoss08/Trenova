package factoringtools

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/shared/jsonschemautils"
)

const (
	paramInvoiceID       = "invoiceId"
	paramOverride        = "override"
	paramDocumentTypeIDs = "documentTypeIds"
	paramMessage         = "message"

	toolGetReadiness       = "factoring_get_readiness"
	maxRequestedDocuments  = 20
	fieldFactoringOverride = "factoringOverride"
	fieldOverrideReason    = "factoringOverrideReason"
	fieldFactoringCompany  = "factoringCompanyId"
	fieldFactoringAgree    = "factoringAgreementId"
)

func invoiceRecord(inv *invoice.Invoice) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceInvoice,
		ID:       inv.ID,
		Label:    "Invoice " + inv.Number,
		Version:  common.PreviewVersion(inv.Version),
	}
}

type SetInvoiceFactoringOverrideTool struct {
	readiness factoringports.ReadinessService
}

func NewSetInvoiceFactoringOverrideTool(
	readiness factoringports.ReadinessService,
) agenttoolports.AgentTool {
	return &SetInvoiceFactoringOverrideTool{readiness: readiness}
}

func (t *SetInvoiceFactoringOverrideTool) Name() string { return "factoring_set_override" }

func (t *SetInvoiceFactoringOverrideTool) Description() string {
	return "Keep one invoice of a factored customer out of factoring, or put it back. " +
		"Exclude, with the reason, leaves it on the organization's own receivables; Default " +
		"puts it back on the customer's factoring. Excluding takes the factor off the invoice, so it is collected " +
		"and counts toward credit like any other; Default assigns it to the customer's " +
		"current factor and the agreement in force when it posted. It cannot bring a single " +
		"invoice into factoring for a customer that is not factored."
}

func (t *SetInvoiceFactoringOverrideTool) SearchTerms() []string {
	return []string{"do not factor", "exclude from factoring", "keep invoice"}
}

func (t *SetInvoiceFactoringOverrideTool) Prerequisites() []string {
	return []string{toolGetReadiness}
}

func (t *SetInvoiceFactoringOverrideTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramInvoiceID: agenttoolschema.RecordIDText(
			permission.ResourceInvoice,
			"The invoice, from invoice_list or factoring_list_readiness.",
		),
		paramOverride: agenttoolschema.Enum(
			"Exclude keeps it out of factoring; Default follows the customer's factoring.",
			agenttoolschema.FactoringOverrides,
		),
		paramReason: jsonschemautils.Text(
			"Why the invoice stays out of factoring. Required for Exclude.",
		),
	}, paramInvoiceID, paramOverride)
}

func (t *SetInvoiceFactoringOverrideTool) Policy() agenttoolports.ToolPolicy {
	return policy(t.Name(), &policySpec{
		resource:    permission.ResourceInvoice,
		operation:   permission.OpUpdate,
		defaultTier: agent.TierActWithApproval,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		idempotent:  true,
		rationale: "Moves one invoice between the organization's receivables and the " +
			"factor's; reversible until the invoice is submitted.",
	})
}

func (t *SetInvoiceFactoringOverrideTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, paramInvoiceID, permission.ResourceInvoice)
}

func (t *SetInvoiceFactoringOverrideTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*factoringports.SetOverrideRequest, error) {
	if err := common.GuardExecute(t, *params); err != nil {
		return nil, err
	}
	invoiceID, err := common.RequirePulid(params.Params, paramInvoiceID)
	if err != nil {
		return nil, err
	}
	raw, err := common.RequireString(params.Params, paramOverride)
	if err != nil {
		return nil, err
	}
	return &factoringports.SetOverrideRequest{
		TenantInfo: common.TenantFrom(*params),
		InvoiceID:  invoiceID,
		Override:   invoice.FactoringOverride(raw),
		Reason:     common.OptionalString(params.Params, paramReason),
	}, nil
}

func (t *SetInvoiceFactoringOverrideTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.readiness.PlanOverride(ctx, req)
	return err
}

func (t *SetInvoiceFactoringOverrideTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.readiness.SetOverride(ctx, req, params.Actor)
	return err
}

func (t *SetInvoiceFactoringOverrideTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	const summary = "Would change whether the invoice is factored."
	req, err := t.request(&params)
	if err != nil {
		return refused(summary, err)
	}
	plan, err := t.readiness.PlanOverride(ctx, req)
	if err != nil {
		return refused(summary, err)
	}
	change, err := toolpreview.Changed(invoiceRecord(plan.Before), plan.Before, plan.After,
		toolpreview.Only(
			fieldFactoringOverride,
			fieldOverrideReason,
			fieldFactoringCompany,
			fieldFactoringAgree,
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldFactoringCompany: permission.ResourceFactoringCompany,
			fieldFactoringAgree:   permission.ResourceFactoringAgreement,
		}),
		toolpreview.Labels(map[string]string{
			fieldFactoringOverride: "Factoring",
			fieldOverrideReason:    "Why it stays out",
			fieldFactoringCompany:  "Assigned to",
			fieldFactoringAgree:    "Agreement",
		}),
	)
	if err != nil {
		return nil, err
	}
	if !plan.Changed {
		return toolpreview.Build(
			"The invoice already reads " + string(req.Override) + "; nothing changes.",
		), nil
	}
	if req.Override.Excludes() {
		return toolpreview.Build(
			"Would keep invoice "+plan.Before.Number+" out of factoring, on the organization's "+
				"own receivables: "+strings.TrimSpace(req.Reason)+".",
			change,
		), nil
	}
	return toolpreview.Build(
		"Would put invoice "+plan.Before.Number+" back on the customer's factoring.",
		change,
	), nil
}

type RequestMissingFactoringDocumentTool struct {
	readiness factoringports.ReadinessService
	senders   emailports.EmailSenderResolver
}

func NewRequestMissingFactoringDocumentTool(
	readiness factoringports.ReadinessService,
	senders emailports.EmailSenderResolver,
) agenttoolports.AgentTool {
	return &RequestMissingFactoringDocumentTool{readiness: readiness, senders: senders}
}

func (t *RequestMissingFactoringDocumentTool) Name() string {
	return "factoring_request_document"
}

func (t *RequestMissingFactoringDocumentTool) Description() string {
	return "Email a request for the documents an invoice needs before it can be sold to the " +
		"factor. It goes to the driver, then the carrier, then the customer's invoice email " +
		"recipients, unless recipient names someone. It asks for every missing " +
		"document factoring_get_readiness lists, or only documentTypeIds. The same request " +
		"to the same people goes out at most once a day."
}

func (t *RequestMissingFactoringDocumentTool) SearchTerms() []string {
	return []string{"missing pod", "missing rate confirmation", "factoring documents"}
}

func (t *RequestMissingFactoringDocumentTool) Prerequisites() []string {
	return []string{toolGetReadiness}
}

func (t *RequestMissingFactoringDocumentTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramInvoiceID: agenttoolschema.RecordIDText(
			permission.ResourceInvoice,
			"The invoice, from factoring_get_readiness or factoring_list_readiness.",
		),
		paramDocumentTypeIDs: agenttoolschema.RecordIDs(
			permission.ResourceDocumentType,
			"Ask only for these document types, from factoring_get_readiness.",
			maxRequestedDocuments,
		),
		paramRecipient: jsonschemautils.Text(
			"Email addresses to ask instead, comma separated.",
		),
		paramMessage: jsonschemautils.Text(
			"The request in plain prose. Leave it out for the standard wording; the " +
				"template adds the greeting, letterhead and the list of documents.",
		),
	}, paramInvoiceID)
}

func (t *RequestMissingFactoringDocumentTool) Policy() agenttoolports.ToolPolicy {
	return policy(t.Name(), &policySpec{
		resource:    permission.ResourceCustomerCommunication,
		operation:   permission.OpCreate,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		egress:      agent.EgressExternalRecipient,
		idempotent:  true,
		rationale:   "Emails a driver, carrier or customer outside the organization.",
	})
}

func (t *RequestMissingFactoringDocumentTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, paramInvoiceID, permission.ResourceInvoice)
}

func (t *RequestMissingFactoringDocumentTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*factoringports.RequestMissingDocumentsRequest, error) {
	if err := common.GuardExecute(t, *params); err != nil {
		return nil, err
	}
	invoiceID, err := common.RequirePulid(params.Params, paramInvoiceID)
	if err != nil {
		return nil, err
	}
	typeIDs, err := common.RequirePulidSlice(
		params.Params,
		paramDocumentTypeIDs,
		maxRequestedDocuments,
	)
	if err != nil {
		if _, given := params.Params[paramDocumentTypeIDs]; given {
			return nil, err
		}
		typeIDs = nil
	}
	return &factoringports.RequestMissingDocumentsRequest{
		TenantInfo:      common.TenantFrom(*params),
		InvoiceID:       invoiceID,
		DocumentTypeIDs: typeIDs,
		Recipient:       common.OptionalString(params.Params, paramRecipient),
		Message:         common.OptionalString(params.Params, paramMessage),
	}, nil
}

func (t *RequestMissingFactoringDocumentTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.readiness.PlanDocumentRequest(ctx, req, params.Actor.UserID)
	return err
}

func (t *RequestMissingFactoringDocumentTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.readiness.RequestMissingDocuments(ctx, req, params.Actor)
	return err
}

func (t *RequestMissingFactoringDocumentTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	const summary = "Would email a request for the invoice's missing documents."
	req, err := t.request(&params)
	if err != nil {
		return refused(summary, err)
	}
	plan, err := t.readiness.PlanDocumentRequest(ctx, req, params.Actor.UserID)
	if err != nil {
		return refused(summary, err)
	}
	send := messageRequest(&plan.Message)
	sender, err := common.ResolveEmailSender(ctx, t.senders, send)
	if err != nil {
		return nil, err
	}
	change := toolpreview.Send(
		toolpreview.Record{
			Resource: permission.ResourceCustomerCommunication,
			Label:    strings.Join(plan.Message.To, ", "),
		},
		common.EmailMessagePreview(
			sender,
			common.RenderedEmailPreview(send, plan.Message.VersionID),
		),
	)
	preview := toolpreview.Build(
		"Would email "+strings.Join(plan.Message.To, ", ")+" ("+strings.ToLower(plan.RecipientKind)+
			") for "+strings.Join(plan.DocumentTypes, ", ")+" so invoice "+plan.Invoice.Number+
			" can be factored. It cannot be recalled once sent.",
		change,
	)
	common.WarnSuppressedRecipients(preview, sender)
	return preview, nil
}
