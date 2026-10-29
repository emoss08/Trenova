package factoringtools

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/factoringkit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/outboundpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	paramNoaID          = "noaId"
	paramCustomerID     = "customerId"
	paramNote           = "note"
	paramRecipient      = "recipient"
	paramAcknowledgedAt = "acknowledgedAt"
	paramSignedDocument = "signedDocumentId"
	paramExpiresAt      = "expiresAt"

	toolListNoas = "factoring_list_noas"
)

func noaRecord(noa *factoring.Noa) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceFactoringNoa,
		ID:       noa.ID,
		Label:    "Notice of assignment " + noa.ID.String(),
		Version:  changepreview.PreviewVersion(noa.Version),
	}
}

func noaPreviewOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(
			"status", "sentAt", "sentTo", "acknowledgedAt", "signedDocumentId",
			"expiresAt", "releasedAt", "releaseReason", "notes",
		),
		toolpreview.Volatile("sentAt", "releasedAt"),
		toolpreview.WithRefs(map[string]permission.Resource{
			"signedDocumentId": permission.ResourceDocument,
		}),
		toolpreview.Labels(map[string]string{
			"sentTo":           "Sent to",
			"acknowledgedAt":   "Customer signed",
			"signedDocumentId": "Signed copy",
			"expiresAt":        "Expires",
			"releaseReason":    "Why it was released",
		}),
	}
}

func noaIDParam(description string) map[string]any {
	return agenttoolschema.RecordIDText(permission.ResourceFactoringNoa, description)
}

func versionParam() map[string]any {
	return jsonschemautils.Integer(
		"The notice's version from factoring_list_noas, so a notice someone changed " +
			"since is not overwritten.",
	)
}

type noaLoader interface {
	Get(ctx context.Context, req factoringports.GetNoaRequest) (*factoring.Noa, error)
}

func loadNoa(
	ctx context.Context,
	noas noaLoader,
	params *agenttoolports.ToolExecuteParams,
) (*factoring.Noa, int64, error) {
	id, err := toolparams.RequirePulid(params.Params, paramNoaID)
	if err != nil {
		return nil, 0, err
	}
	version, err := toolparams.RequireIntInRange(
		params.Params,
		factoringkit.ParamVersion,
		0,
		factoringkit.MaxVersion,
	)
	if err != nil {
		return nil, 0, err
	}
	noa, err := noas.Get(ctx, factoringports.GetNoaRequest{
		ID:         id,
		TenantInfo: toolparams.TenantFrom(*params),
	})
	if err != nil {
		return nil, 0, err
	}
	return noa, int64(version), nil
}

type DraftFactoringNoaTool struct {
	noas factoringports.NoaService
}

func NewDraftFactoringNoaTool(noas factoringports.NoaService) agenttoolports.AgentTool {
	return &DraftFactoringNoaTool{noas: noas}
}

func (t *DraftFactoringNoaTool) Name() string { return "factoring_create_noa" }

func (t *DraftFactoringNoaTool) Description() string {
	return "Draft a notice of assignment telling a factored customer to pay the factor. " +
		"It sends nothing: factoring_send_noa emails the letter once it is drafted. A " +
		"customer can have only one open notice per factor, so check factoring_list_noas " +
		"first; an invoice is not ready to factor until the customer acknowledges one."
}

func (t *DraftFactoringNoaTool) SearchTerms() []string {
	return []string{"notice of assignment", "noa", "factoring letter"}
}

func (t *DraftFactoringNoaTool) Prerequisites() []string { return []string{toolListNoas} }

func (t *DraftFactoringNoaTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramCustomerID: agenttoolschema.RecordIDText(
			permission.ResourceCustomer,
			"The factored customer, from the invoice or customer_list.",
		),
		factoringkit.ParamFactorID: agenttoolschema.RecordIDText(
			permission.ResourceFactoringCompany,
			"The factor the customer's invoices are assigned to, from invoice_get or factoring_list_readiness.",
		),
		paramNote: jsonschemautils.Text("A note printed on the letter, when there is one."),
	}, paramCustomerID, factoringkit.ParamFactorID)
}

func (t *DraftFactoringNoaTool) Policy() agenttoolports.ToolPolicy {
	return factoringkit.Policy(t.Name(), &factoringkit.PolicySpec{
		Resource:    permission.ResourceFactoringNoa,
		Operation:   permission.OpCreate,
		DefaultTier: agent.TierAutoExecute,
		MaxTier:     agent.TierAutoExecute,
		Reversible:  true,
		Rationale:   "Drafts a letter nobody outside sees until it is sent.",
	})
}

func (t *DraftFactoringNoaTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*factoringports.DraftNoaRequest, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}
	customerID, err := toolparams.RequirePulid(params.Params, paramCustomerID)
	if err != nil {
		return nil, err
	}
	factorID, err := toolparams.RequirePulid(params.Params, factoringkit.ParamFactorID)
	if err != nil {
		return nil, err
	}
	return &factoringports.DraftNoaRequest{
		TenantInfo:         toolparams.TenantFrom(*params),
		CustomerID:         customerID,
		FactoringCompanyID: factorID,
		Notes:              toolparams.OptionalString(params.Params, paramNote),
	}, nil
}

func (t *DraftFactoringNoaTool) Validate(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, err := t.request(&params)
	return err
}

func (t *DraftFactoringNoaTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.noas.Draft(ctx, req, params.Actor)
	return err
}

func (t *DraftFactoringNoaTool) Preview(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	const summary = "Would draft a notice of assignment for the customer. Nothing is sent."
	req, err := t.request(&params)
	if err != nil {
		return factoringkit.Refused(summary, err)
	}
	draft := &factoring.Noa{
		OrganizationID:     req.TenantInfo.OrgID,
		BusinessUnitID:     req.TenantInfo.BuID,
		CustomerID:         req.CustomerID,
		FactoringCompanyID: req.FactoringCompanyID,
		Status:             factoring.NoaStatusDraft,
		Notes:              req.Notes,
	}
	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceFactoringNoa,
		Label:    "New notice of assignment",
	}, draft, append(noaPreviewOptions(), toolpreview.WithRefs(map[string]permission.Resource{
		"customerId":         permission.ResourceCustomer,
		"factoringCompanyId": permission.ResourceFactoringCompany,
	}))...)
	if err != nil {
		return nil, err
	}
	return toolpreview.Build(summary, change), nil
}

type SendFactoringNoaTool struct {
	noas    factoringports.NoaService
	senders emailports.EmailSenderResolver
}

func NewSendFactoringNoaTool(
	noas factoringports.NoaService,
	senders emailports.EmailSenderResolver,
) agenttoolports.AgentTool {
	return &SendFactoringNoaTool{noas: noas, senders: senders}
}

func (t *SendFactoringNoaTool) Name() string { return "factoring_send_noa" }

func (t *SendFactoringNoaTool) Description() string {
	return "Email the customer the notice of assignment letter as a PDF, from the " +
		"organization's own template, asking them to pay the factor and return a signed copy. " +
		"It goes to the customer's invoice email recipients unless recipient names others. " +
		"Sending the same version twice does not send twice. Once the signed copy is back, " +
		"upload it to the notice and call factoring_record_noa_acknowledgement."
}

func (t *SendFactoringNoaTool) SearchTerms() []string {
	return []string{"send notice of assignment", "email noa", "factoring letter"}
}

func (t *SendFactoringNoaTool) Prerequisites() []string { return []string{toolListNoas} }

func (t *SendFactoringNoaTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramNoaID: noaIDParam(
			"The drafted notice, from factoring_list_noas or factoring_create_noa.",
		),
		factoringkit.ParamVersion: versionParam(),
		paramRecipient: jsonschemautils.Text(
			"Email addresses to send to, comma separated. Leave it out to use the " +
				"customer's invoice email recipients.",
		),
	}, paramNoaID, factoringkit.ParamVersion)
}

func (t *SendFactoringNoaTool) Policy() agenttoolports.ToolPolicy {
	return factoringkit.Policy(t.Name(), &factoringkit.PolicySpec{
		Resource:    permission.ResourceFactoringNoa,
		Operation:   permission.OpUpdate,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		Egress:      agent.EgressExternalRecipient,
		Idempotent:  true,
		Rationale: "Tells a customer, in writing, to pay someone else; a person decides " +
			"every one.",
	})
}

func (t *SendFactoringNoaTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramNoaID, permission.ResourceFactoringNoa)
}

func (t *SendFactoringNoaTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*factoringports.SendNoaRequest, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}
	id, err := toolparams.RequirePulid(params.Params, paramNoaID)
	if err != nil {
		return nil, err
	}
	version, err := toolparams.RequireIntInRange(
		params.Params,
		factoringkit.ParamVersion,
		0,
		factoringkit.MaxVersion,
	)
	if err != nil {
		return nil, err
	}
	return &factoringports.SendNoaRequest{
		TenantInfo: toolparams.TenantFrom(*params),
		ID:         id,
		Version:    int64(version),
		Recipient:  toolparams.OptionalString(params.Params, paramRecipient),
	}, nil
}

func (t *SendFactoringNoaTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.noas.PlanSend(ctx, req, params.Actor.UserID)
	return err
}

func (t *SendFactoringNoaTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.noas.Send(ctx, req, params.Actor)
	return err
}

func (t *SendFactoringNoaTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	const summary = "Would email the customer the notice of assignment."
	req, err := t.request(&params)
	if err != nil {
		return factoringkit.Refused(summary, err)
	}
	plan, err := t.noas.PlanSend(ctx, req, params.Actor.UserID)
	if err != nil {
		return factoringkit.Refused(summary, err)
	}

	send := messageRequest(&plan.Message)
	sender, err := outboundpreview.ResolveEmailSender(ctx, t.senders, send)
	if err != nil {
		return nil, err
	}
	message := toolpreview.Send(
		toolpreview.Record{
			Resource: permission.ResourceFactoringNoa,
			ID:       plan.Noa.ID,
			Label:    "Notice of assignment",
		},
		outboundpreview.EmailMessagePreview(
			sender,
			outboundpreview.RenderedEmailPreview(send, plan.Message.VersionID),
		),
	)

	recipients := plan.Message.To
	update, err := changepreview.PlanUpdate(
		noaRecord(plan.Noa),
		plan.Noa,
		func(after *factoring.Noa) error {
			return after.MarkSent(timeutils.NowUnix(), recipients)
		},
		noaPreviewOptions()...)
	if err != nil {
		return nil, err
	}
	preview := update.PlanPreview(
		"Would email the notice of assignment to "+strings.Join(recipients, ", ")+
			", telling the customer to pay the factor. It cannot be recalled once sent.",
		message,
	)
	outboundpreview.WarnSuppressedRecipients(preview, sender)
	return preview, nil
}

type RecordFactoringNoaAcknowledgementTool struct {
	noas factoringports.NoaService
}

func NewRecordFactoringNoaAcknowledgementTool(
	noas factoringports.NoaService,
) agenttoolports.AgentTool {
	return &RecordFactoringNoaAcknowledgementTool{noas: noas}
}

func (t *RecordFactoringNoaAcknowledgementTool) Name() string {
	return "factoring_record_noa_acknowledgement"
}

func (t *RecordFactoringNoaAcknowledgementTool) Description() string {
	return "Record that the customer signed and returned the notice of assignment. Name " +
		"the signed copy, which must already be uploaded as a document on the notice, and " +
		"the date they signed. From then on the customer's invoices can be ready to factor " +
		"until the notice expires or is released."
}

func (t *RecordFactoringNoaAcknowledgementTool) SearchTerms() []string {
	return []string{"signed noa", "acknowledged notice", "customer signed"}
}

func (t *RecordFactoringNoaAcknowledgementTool) Prerequisites() []string {
	return []string{toolListNoas}
}

func (t *RecordFactoringNoaAcknowledgementTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramNoaID:                noaIDParam("The sent notice, from factoring_list_noas."),
		factoringkit.ParamVersion: versionParam(),
		paramAcknowledgedAt: agenttoolschema.DateTime(
			"When the customer signed the notice. It cannot be in the future.",
		),
		paramSignedDocument: agenttoolschema.RecordIDText(
			permission.ResourceDocument,
			"The signed copy, a document uploaded to this notice, from document_list.",
		),
		paramExpiresAt: agenttoolschema.DateTime(
			"When the notice stops applying, if the factor's agreement says so. Leave it " +
				"out for a notice that runs until released.",
		),
	}, paramNoaID, factoringkit.ParamVersion, paramAcknowledgedAt, paramSignedDocument)
}

func (t *RecordFactoringNoaAcknowledgementTool) Policy() agenttoolports.ToolPolicy {
	return factoringkit.Policy(t.Name(), &factoringkit.PolicySpec{
		Resource:    permission.ResourceFactoringNoa,
		Operation:   permission.OpUpdate,
		DefaultTier: agent.TierActWithApproval,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Lets the customer's invoices be sold to the factor; a person confirms " +
			"the signed copy is the customer's.",
	})
}

func (t *RecordFactoringNoaAcknowledgementTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramNoaID, permission.ResourceFactoringNoa)
}

type acknowledgement struct {
	noa *factoring.Noa
	req *factoringports.AcknowledgeNoaRequest
}

func (t *RecordFactoringNoaAcknowledgementTool) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*acknowledgement, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}
	noa, version, err := loadNoa(ctx, t.noas, params)
	if err != nil {
		return nil, err
	}
	acknowledgedAt, err := toolparams.RequireDateTime(params.Params, paramAcknowledgedAt)
	if err != nil {
		return nil, err
	}
	documentID, err := toolparams.RequirePulid(params.Params, paramSignedDocument)
	if err != nil {
		return nil, err
	}
	expiresAt, err := toolparams.OptionalDateTime(params.Params, paramExpiresAt)
	if err != nil {
		return nil, err
	}
	return &acknowledgement{noa: noa, req: &factoringports.AcknowledgeNoaRequest{
		TenantInfo:       toolparams.TenantFrom(*params),
		ID:               noa.ID,
		Version:          version,
		AcknowledgedAt:   acknowledgedAt,
		SignedDocumentID: documentID,
		ExpiresAt:        expiresAt,
	}}, nil
}

func (a *acknowledgement) apply(noa *factoring.Noa) error {
	return noa.Acknowledge(
		timeutils.NowUnix(),
		a.req.AcknowledgedAt,
		a.req.SignedDocumentID,
		a.req.ExpiresAt,
	)
}

func (t *RecordFactoringNoaAcknowledgementTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	planned, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}
	return planned.apply(planned.noa)
}

func (t *RecordFactoringNoaAcknowledgementTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	planned, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.noas.Acknowledge(ctx, planned.req, params.Actor)
	return err
}

func (t *RecordFactoringNoaAcknowledgementTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	const summary = "Would record that the customer signed the notice of assignment."
	planned, err := t.plan(ctx, &params)
	if err != nil {
		return factoringkit.Refused(summary, err)
	}
	update, err := changepreview.PlanUpdate(
		noaRecord(planned.noa),
		planned.noa,
		planned.apply,
		noaPreviewOptions()...)
	if err != nil {
		return nil, err
	}
	return update.PlanPreview(
		summary + " The customer's invoices can then be sold to the factor.",
	), nil
}

type ReleaseFactoringNoaTool struct {
	noas factoringports.NoaService
}

func NewReleaseFactoringNoaTool(noas factoringports.NoaService) agenttoolports.AgentTool {
	return &ReleaseFactoringNoaTool{noas: noas}
}

func (t *ReleaseFactoringNoaTool) Name() string { return "factoring_release_noa" }

func (t *ReleaseFactoringNoaTool) Description() string {
	return "Release a notice of assignment, for example once the factor's release letter " +
		"arrives or the customer leaves factoring, with the reason. The customer's invoices " +
		"stop being ready to factor. It does not change the customer's billing profile; take " +
		"the factor off it separately when the customer pays the organization again."
}

func (t *ReleaseFactoringNoaTool) SearchTerms() []string {
	return []string{"release notice", "noa release", "stop factoring"}
}

func (t *ReleaseFactoringNoaTool) Prerequisites() []string { return []string{toolListNoas} }

func (t *ReleaseFactoringNoaTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramNoaID:                noaIDParam("The open notice, from factoring_list_noas."),
		factoringkit.ParamVersion: versionParam(),
		factoringkit.ParamReason: jsonschemautils.Text(
			"Why the notice is released, such as the factor's release letter, in one sentence.",
		),
	}, paramNoaID, factoringkit.ParamVersion, factoringkit.ParamReason)
}

func (t *ReleaseFactoringNoaTool) Policy() agenttoolports.ToolPolicy {
	return factoringkit.Policy(t.Name(), &factoringkit.PolicySpec{
		Resource:    permission.ResourceFactoringNoa,
		Operation:   permission.OpUpdate,
		DefaultTier: agent.TierActWithApproval,
		MaxTier:     agent.TierActWithApproval,
		Rationale: "Ends the customer's assignment to the factor; only the factor can release " +
			"it, so a person confirms the release.",
	})
}

func (t *ReleaseFactoringNoaTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramNoaID, permission.ResourceFactoringNoa)
}

type release struct {
	noa *factoring.Noa
	req *factoringports.ReleaseNoaRequest
}

func (t *ReleaseFactoringNoaTool) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*release, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}
	noa, version, err := loadNoa(ctx, t.noas, params)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.RequireString(params.Params, factoringkit.ParamReason)
	if err != nil {
		return nil, err
	}
	return &release{noa: noa, req: &factoringports.ReleaseNoaRequest{
		TenantInfo: toolparams.TenantFrom(*params),
		ID:         noa.ID,
		Version:    version,
		Reason:     reason,
	}}, nil
}

func (r *release) apply(noa *factoring.Noa) error {
	return noa.Release(timeutils.NowUnix(), r.req.Reason)
}

func (t *ReleaseFactoringNoaTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	planned, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}
	return planned.apply(planned.noa)
}

func (t *ReleaseFactoringNoaTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	planned, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.noas.Release(ctx, planned.req, params.Actor)
	return err
}

func (t *ReleaseFactoringNoaTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	const summary = "Would release the notice of assignment."
	planned, err := t.plan(ctx, &params)
	if err != nil {
		return factoringkit.Refused(summary, err)
	}
	update, err := changepreview.PlanUpdate(
		noaRecord(planned.noa),
		planned.noa,
		planned.apply,
		noaPreviewOptions()...)
	if err != nil {
		return nil, err
	}
	return update.PlanPreview(summary + " The customer's invoices stop being ready to factor."), nil
}

func messageRequest(message *factoringports.MessagePlan) *emailports.SendEmailRequest {
	send := &emailports.SendEmailRequest{
		To:      message.To,
		CC:      message.CC,
		BCC:     message.BCC,
		Subject: message.Subject,
		HTML:    message.HTML,
		Text:    message.Text,
	}
	if message.AttachmentName != "" {
		send.Attachments = []emailports.EmailAttachment{{FileName: message.AttachmentName}}
	}
	return send
}
