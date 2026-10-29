package factoringtools

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/shopspring/decimal"
)

const (
	paramSubmissionID = "submissionId"
	paramInvoiceIDs   = "invoiceIds"

	toolListSubmissions = "factoring_list_submissions"
	toolListReady       = "factoring_list_readiness"

	maxCancelReason = 500
)

func submissionRecord(sub *factoringsubmission.Submission) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceFactoringSubmission,
		ID:       sub.ID,
		Label:    "Factoring schedule " + sub.Number,
		Version:  changepreview.PreviewVersion(sub.Version),
	}
}

func submissionPreviewOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(
			"status", fieldFactoringCompany, "transportMethod", "transportTarget",
			"invoiceCount", "sendAttemptGroup", "lastError", "cancelReason",
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldFactoringCompany: permission.ResourceFactoringCompany,
		}),
		toolpreview.Labels(map[string]string{
			fieldFactoringCompany: "Factor",
			"transportMethod":     "Sent by",
			"transportTarget":     "Sent to",
			"invoiceCount":        "Invoices",
			"sendAttemptGroup":    "Send",
			"lastError":           "Last error",
			"cancelReason":        "Why it was cancelled",
		}),
	}
}

func submissionIDParam(description string) map[string]any {
	return agenttoolschema.RecordIDText(permission.ResourceFactoringSubmission, description)
}

func submissionVersionParam() map[string]any {
	return jsonschemautils.Integer(
		"The schedule's version from factoring_get_submission, so a schedule someone " +
			"changed since is not overwritten.",
	)
}

func submissionActionRequest(
	params *agenttoolports.ToolExecuteParams,
) (*factoringsubmissionports.SubmissionActionRequest, error) {
	id, err := toolparams.RequirePulid(params.Params, paramSubmissionID)
	if err != nil {
		return nil, err
	}
	version, err := toolparams.RequireIntInRange(params.Params, paramVersion, 0, maxVersion)
	if err != nil {
		return nil, err
	}
	return &factoringsubmissionports.SubmissionActionRequest{
		TenantInfo:   toolparams.TenantFrom(*params),
		SubmissionID: id,
		Version:      int64(version),
	}, nil
}

type SubmitInvoicesToFactorTool struct {
	submissions factoringsubmissionports.SubmissionService
}

func NewSubmitInvoicesToFactorTool(
	submissions factoringsubmissionports.SubmissionService,
) agenttoolports.AgentTool {
	return &SubmitInvoicesToFactorTool{submissions: submissions}
}

func (t *SubmitInvoicesToFactorTool) Name() string { return "factoring_create_submission" }

func (t *SubmitInvoicesToFactorTool) Description() string {
	return "Sell Ready invoices to their factor on one schedule of accounts. It builds the " +
		"schedule PDF and CSV and each invoice's packet (the invoice and its supporting " +
		"documents), then sends them to the factor by email or SFTP, as the factor's " +
		"delivery settings say. Every invoice must be Ready in factoring_list_readiness " +
		"and assigned to this factor; batch one factor's invoices per call, up to 200. " +
		"Submitting the same invoices again returns the first schedule instead of sending twice."
}

func (t *SubmitInvoicesToFactorTool) SearchTerms() []string {
	return []string{"factor invoices", "schedule of accounts", "submit to factor", "sell invoices"}
}

func (t *SubmitInvoicesToFactorTool) Prerequisites() []string { return []string{toolListReady} }

func (t *SubmitInvoicesToFactorTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramFactorID: agenttoolschema.RecordIDText(
			permission.ResourceFactoringCompany,
			"The factor every invoice is assigned to, from factoring_list_readiness.",
		),
		paramInvoiceIDs: agenttoolschema.RecordIDs(
			permission.ResourceInvoice,
			"The Ready invoices to sell, from factoring_list_readiness.",
			factoringsubmission.MaxSubmissionInvoices,
		),
	}, paramFactorID, paramInvoiceIDs)
}

func (t *SubmitInvoicesToFactorTool) Policy() agenttoolports.ToolPolicy {
	return policy(t.Name(), &policySpec{
		resource:    permission.ResourceFactoringSubmission,
		operation:   permission.OpSubmit,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		egress:      agent.EgressExternalRecipient,
		idempotent:  true,
		rationale: "Sells receivables to a factor and sends them out of the organization; " +
			"a person approves every schedule.",
	})
}

func (t *SubmitInvoicesToFactorTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramFactorID, permission.ResourceFactoringCompany)
}

func (t *SubmitInvoicesToFactorTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*factoringsubmissionports.PlanSubmissionRequest, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}
	factorID, err := toolparams.RequirePulid(params.Params, paramFactorID)
	if err != nil {
		return nil, err
	}
	invoiceIDs, err := toolparams.RequirePulidSlice(
		params.Params,
		paramInvoiceIDs,
		factoringsubmission.MaxSubmissionInvoices,
	)
	if err != nil {
		return nil, err
	}
	return &factoringsubmissionports.PlanSubmissionRequest{
		TenantInfo:         toolparams.TenantFrom(*params),
		FactoringCompanyID: factorID,
		InvoiceIDs:         invoiceIDs,
		AgentRunID:         params.RunID,
	}, nil
}

func (t *SubmitInvoicesToFactorTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.submissions.Plan(ctx, req)
	return err
}

func (t *SubmitInvoicesToFactorTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.submissions.Submit(ctx, req, params.Actor)
	return err
}

func (t *SubmitInvoicesToFactorTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	const summary = "Would send a schedule of accounts to the factor."
	req, err := t.request(&params)
	if err != nil {
		return refused(summary, err)
	}
	plan, err := t.submissions.Plan(ctx, req)
	if err != nil {
		return refused(summary, err)
	}

	draft := &factoringsubmission.Submission{
		OrganizationID:     req.TenantInfo.OrgID,
		BusinessUnitID:     req.TenantInfo.BuID,
		FactoringCompanyID: plan.FactoringCompanyID,
		Status:             factoringsubmission.SubmissionStatusSending,
		TransportMethod:    plan.TransportMethod,
		TransportTarget:    plan.TransportTarget,
		InvoiceCount:       plan.InvoiceCount,
	}
	created, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceFactoringSubmission,
		Label:    "New schedule to " + plan.FactoringCompanyName,
	}, draft, submissionPreviewOptions()...)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(created, planMoney(plan))

	changes := []*agent.RecordChange{created}
	if plan.TransportMethod == factoringsubmission.TransportMethodEmail {
		changes = append(changes, toolpreview.Send(toolpreview.Record{
			Resource: permission.ResourceFactoringSubmission,
			Label:    "Schedule to " + plan.FactoringCompanyName,
		}, planMessage(plan)))
	}

	return toolpreview.Build(
		"Would sell "+changepreview.CountOf(plan.InvoiceCount, "invoice")+" ("+
			formatMinor(plan.CurrencyCode, plan.FaceMinor)+") to "+plan.FactoringCompanyName+
			" by "+strings.ToLower(plan.TransportMethod.String())+" to "+plan.TransportTarget+
			", expecting an advance of "+formatMinor(plan.CurrencyCode, plan.ExpectedAdvanceMinor)+
			". It cannot be recalled once sent.",
		changes...,
	), nil
}

func planMoney(plan *factoringsubmissionports.SubmissionPlan) *agent.MoneyPreview {
	lines := make([]agent.MoneyLine, 0, len(plan.Items)+3)
	for _, item := range plan.Items {
		lines = append(lines, agent.MoneyLine{
			Label: item.InvoiceNumber + " · " + item.CustomerName,
			After: minorDecimal(item.Funding.FaceMinor),
		})
	}
	block := toolpreview.MoneyBlock(plan.CurrencyCode, lines...)
	block.Lines = append(block.Lines,
		agent.MoneyLine{Label: "Expected advance", After: minorDecimal(plan.ExpectedAdvanceMinor)},
		agent.MoneyLine{Label: "Expected fee", After: minorDecimal(plan.ExpectedFeeMinor)},
		agent.MoneyLine{Label: "Expected reserve", After: minorDecimal(plan.ExpectedReserveMinor)},
	)
	return block
}

func planMessage(plan *factoringsubmissionports.SubmissionPlan) *agent.MessagePreview {
	attachments := []string{"Schedule of accounts (PDF)", "Schedule of accounts (CSV)"}
	if plan.AttachPackets {
		for _, item := range plan.Items {
			attachments = append(attachments, item.InvoiceNumber+".pdf")
		}
	}
	return &agent.MessagePreview{
		Channel:     agent.MessageChannelEmail,
		To:          strings.Split(plan.TransportTarget, ", "),
		Subject:     "Schedule of accounts",
		Attachments: attachments,
		Body: "Schedule of " + changepreview.CountOf(plan.InvoiceCount, "invoice") + " totalling " +
			formatMinor(plan.CurrencyCode, plan.FaceMinor) + ".",
	}
}

func minorDecimal(minor int64) decimal.NullDecimal {
	return decimal.NewNullDecimal(money.DecimalFromMinor(minor))
}

func formatMinor(currency string, minor int64) string {
	return money.FormatDecimal(currency, money.DecimalFromMinor(minor))
}

type RetryFactoringSubmissionTool struct {
	submissions factoringsubmissionports.SubmissionService
}

func NewRetryFactoringSubmissionTool(
	submissions factoringsubmissionports.SubmissionService,
) agenttoolports.AgentTool {
	return &RetryFactoringSubmissionTool{submissions: submissions}
}

func (t *RetryFactoringSubmissionTool) Name() string { return "factoring_retry_submission" }

func (t *RetryFactoringSubmissionTool) Description() string {
	return "Send a schedule that failed to send again, using the factor's delivery settings " +
		"as they are now. Read the schedule's last error with factoring_get_submission first: " +
		"retry only once the cause is fixed (the factor's inbox or SFTP settings corrected and " +
		"tested), or it will fail the same way. A person approves every retry."
}

func (t *RetryFactoringSubmissionTool) SearchTerms() []string {
	return []string{"resend schedule", "retry factoring", "failed schedule"}
}

func (t *RetryFactoringSubmissionTool) Prerequisites() []string {
	return []string{toolListSubmissions}
}

func (t *RetryFactoringSubmissionTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramSubmissionID: submissionIDParam(
			"The failed schedule, from factoring_list_submissions.",
		),
		paramVersion: submissionVersionParam(),
	}, paramSubmissionID, paramVersion)
}

func (t *RetryFactoringSubmissionTool) Policy() agenttoolports.ToolPolicy {
	return policy(t.Name(), &policySpec{
		resource:    permission.ResourceFactoringSubmission,
		operation:   permission.OpUpdate,
		defaultTier: agent.TierActWithApproval,
		maxTier:     agent.TierActWithApproval,
		egress:      agent.EgressExternalRecipient,
		idempotent:  true,
		rationale:   "Sends receivables out of the organization again; a person approves it.",
	})
}

func (t *RetryFactoringSubmissionTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramSubmissionID, permission.ResourceFactoringSubmission)
}

func (t *RetryFactoringSubmissionTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*factoringsubmissionports.SubmissionActionRequest, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}
	return submissionActionRequest(params)
}

func (t *RetryFactoringSubmissionTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.submissions.PlanRetry(ctx, req)
	return err
}

func (t *RetryFactoringSubmissionTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.submissions.Retry(ctx, req, params.Actor)
	return err
}

func (t *RetryFactoringSubmissionTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	const summary = "Would send the failed schedule to the factor again."
	req, err := t.request(&params)
	if err != nil {
		return refused(summary, err)
	}
	plan, err := t.submissions.PlanRetry(ctx, req)
	if err != nil {
		return refused(summary, err)
	}
	change, err := toolpreview.Changed(
		submissionRecord(plan.Before),
		plan.Before,
		plan.After,
		submissionPreviewOptions()...,
	)
	if err != nil {
		return nil, err
	}
	return toolpreview.Build(
		"Would send schedule "+plan.After.Number+" ("+
			changepreview.CountOf(plan.After.InvoiceCount, "invoice")+") again to "+
			plan.After.TransportTarget+". It last failed with: "+plan.Before.LastError,
		change,
	), nil
}

type CancelFactoringSubmissionTool struct {
	submissions factoringsubmissionports.SubmissionService
}

func NewCancelFactoringSubmissionTool(
	submissions factoringsubmissionports.SubmissionService,
) agenttoolports.AgentTool {
	return &CancelFactoringSubmissionTool{submissions: submissions}
}

func (t *CancelFactoringSubmissionTool) Name() string { return "factoring_cancel_submission" }

func (t *CancelFactoringSubmissionTool) Description() string {
	return "Cancel a schedule that failed to send and release its invoices, so they can go " +
		"on another schedule once they are Ready again. Only a failed schedule can be " +
		"cancelled: a sent one is with the factor. Say why."
}

func (t *CancelFactoringSubmissionTool) SearchTerms() []string {
	return []string{"cancel schedule", "release invoices", "failed schedule"}
}

func (t *CancelFactoringSubmissionTool) Prerequisites() []string {
	return []string{toolListSubmissions}
}

func (t *CancelFactoringSubmissionTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramSubmissionID: submissionIDParam(
			"The failed schedule, from factoring_list_submissions.",
		),
		paramVersion: submissionVersionParam(),
		paramReason:  jsonschemautils.Text("Why the schedule is cancelled."),
	}, paramSubmissionID, paramVersion, paramReason)
}

func (t *CancelFactoringSubmissionTool) Policy() agenttoolports.ToolPolicy {
	return policy(t.Name(), &policySpec{
		resource:    permission.ResourceFactoringSubmission,
		operation:   permission.OpCancel,
		defaultTier: agent.TierActWithApproval,
		maxTier:     agent.TierActWithApproval,
		rationale:   "Releases invoices from a schedule; a person approves it.",
	})
}

func (t *CancelFactoringSubmissionTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramSubmissionID, permission.ResourceFactoringSubmission)
}

func (t *CancelFactoringSubmissionTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*factoringsubmissionports.SubmissionActionRequest, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}
	req, err := submissionActionRequest(params)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.RequireBoundedText(params.Params, paramReason, maxCancelReason)
	if err != nil {
		return nil, err
	}
	req.Reason = reason
	return req, nil
}

func (t *CancelFactoringSubmissionTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.submissions.PlanCancel(ctx, req)
	return err
}

func (t *CancelFactoringSubmissionTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.submissions.Cancel(ctx, req, params.Actor)
	return err
}

func (t *CancelFactoringSubmissionTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	const summary = "Would cancel the failed schedule and release its invoices."
	req, err := t.request(&params)
	if err != nil {
		return refused(summary, err)
	}
	plan, err := t.submissions.PlanCancel(ctx, req)
	if err != nil {
		return refused(summary, err)
	}
	change, err := toolpreview.Changed(
		submissionRecord(plan.Before),
		plan.Before,
		plan.After,
		submissionPreviewOptions()...,
	)
	if err != nil {
		return nil, err
	}
	return toolpreview.Build(
		"Would cancel schedule "+plan.After.Number+" and release its "+
			strconv.Itoa(plan.After.InvoiceCount)+" invoices for another schedule.",
		change,
	), nil
}
