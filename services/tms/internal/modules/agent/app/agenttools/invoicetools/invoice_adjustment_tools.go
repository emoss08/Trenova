package invoicetools

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const ParamAdjustmentID = "adjustmentId"

const ParamDraftAdjustmentID = "draftAdjustmentId"

const paramAdjustments = "adjustments"

const paramAdjustmentKind = "kind"

const paramRebillStrategy = "rebillStrategy"

const paramAdjustmentLines = "lines"

const paramInvoiceLineID = "invoiceLineId"

const paramCreditAmount = "creditAmount"

const paramCreditQuantity = "creditQuantity"

const paramRebillAmount = "rebillAmount"

const paramRebillQuantity = "rebillQuantity"

const paramLineDescription = "description"

const paramSupportingDocumentIDs = "supportingDocumentIds"

const maxAdjustmentsPerSubmission = 25

const maxAdjustmentLines = 200

const maxAdjustmentDocuments = 20

const maxAdjustmentReasonChars = 1000

const maxAdjustmentLineTextChars = 500

const adjustmentKeyPrefix = "agent-proposal:"

var adjustmentKinds = agenttoolschema.Source("invoiceAdjustment.kind", []invoiceadjustment.Kind{
	invoiceadjustment.KindCreditOnly,
	invoiceadjustment.KindCreditRebill,
	invoiceadjustment.KindFullReversal,
	invoiceadjustment.KindWriteOff,
})

var rebillStrategies = agenttoolschema.Source(
	"invoiceAdjustment.rebillStrategy",
	[]invoiceadjustment.RebillStrategy{
		invoiceadjustment.RebillStrategyCloneExact,
		invoiceadjustment.RebillStrategyRerate,
		invoiceadjustment.RebillStrategyManual,
	},
)

var errDraftOrAdjustments = errors.New(
	"name what to submit: draftAdjustmentId for a saved draft, or adjustments for new ones",
)

// errDraftAndAdjustments answers a submission that names a saved draft and
// also lists adjustments. A draft is submitted as it stands, so the list
// beside it is either a copy or a second credit, and the tool cannot tell
// which; the refusal spells out both calls so the next one is the one meant.
var errDraftAndAdjustments = errors.New(
	"give draftAdjustmentId or adjustments, not both. To submit the saved draft as it " +
		"stands, send draftAdjustmentId without adjustments, for example " +
		`{"draftAdjustmentId": "iadj_…"}. To submit new adjustments, send adjustments ` +
		"without draftAdjustmentId, for example " +
		`{"adjustments": [{"invoiceId": "inv_…", "kind": "CreditOnly", "reason": "…"}]}`,
)

var errNoDraftTarget = errors.New(
	"give invoiceId to start a new draft, or adjustmentId to rewrite a saved one",
)

type invoiceAdjuster interface {
	SaveDraft(
		ctx context.Context,
		req *invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest,
		actor *permissionports.RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	PreviewSaveDraft(
		ctx context.Context,
		req *invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest,
		actor *permissionports.RequestActor,
	) (*invoiceadjustmentports.InvoiceAdjustmentDraftPreview, error)
	GetDetail(
		ctx context.Context,
		req *invoiceadjustmentports.GetInvoiceAdjustmentDetailRequest,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	PreviewDraft(
		ctx context.Context,
		req *invoiceadjustmentports.GetInvoiceAdjustmentDetailRequest,
		actor *permissionports.RequestActor,
	) (*invoiceadjustmentports.InvoiceAdjustmentPreview, error)
	SubmitDraft(
		ctx context.Context,
		req *invoiceadjustmentports.GetInvoiceAdjustmentDetailRequest,
		actor *permissionports.RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	Preview(
		ctx context.Context,
		req *invoiceadjustmentports.InvoiceAdjustmentRequest,
		actor *permissionports.RequestActor,
	) (*invoiceadjustmentports.InvoiceAdjustmentPreview, error)
	Submit(
		ctx context.Context,
		req *invoiceadjustmentports.InvoiceAdjustmentRequest,
		actor *permissionports.RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	BulkPreview(
		ctx context.Context,
		req *invoiceadjustmentports.InvoiceAdjustmentBulkRequest,
		actor *permissionports.RequestActor,
	) ([]*invoiceadjustmentports.InvoiceAdjustmentPreview, error)
	BulkSubmit(
		ctx context.Context,
		req *invoiceadjustmentports.InvoiceAdjustmentBulkRequest,
		actor *permissionports.RequestActor,
	) (*invoiceadjustment.InvoiceAdjustmentBatch, error)
	Approve(
		ctx context.Context,
		req *invoiceadjustmentports.ApproveInvoiceAdjustmentRequest,
		actor *permissionports.RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	Reject(
		ctx context.Context,
		req *invoiceadjustmentports.RejectInvoiceAdjustmentRequest,
		actor *permissionports.RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	PreviewDecision(
		ctx context.Context,
		req *invoiceadjustmentports.InvoiceAdjustmentDecisionRequest,
	) (*invoiceadjustmentports.InvoiceAdjustmentDecisionPreview, error)
}

func adjustmentKindProperty() map[string]any {
	return agenttoolschema.Enum("CreditOnly credits lines; CreditAndRebill credits them and "+
		"reissues a corrected invoice; FullReversal credits the whole invoice; WriteOff writes "+
		"an uncollectable balance off.", adjustmentKinds)
}

func rebillStrategyProperty() map[string]any {
	return agenttoolschema.Enum("How a CreditAndRebill reissues: CloneExact copies the lines, "+
		"Rerate prices them again, Manual takes the rebill amounts you give. Defaults to "+
		"CloneExact.", rebillStrategies)
}

func adjustmentLinesProperty() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeArray,
		toolschema.KeyDescription: "The invoice lines to adjust. Leave out a line to leave it " +
			"alone; leave out an amount to credit the whole of what is left on the line.",
		toolschema.KeyMaxItems: maxAdjustmentLines,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				paramInvoiceLineID: agenttoolschema.KindID(
					"The line's id, from invoice_get's lines.", permission.KindInvoiceLine),
				paramCreditAmount: toolparams.StringProperty(
					"How much of the line to credit, as a decimal.", 0),
				paramCreditQuantity: toolparams.StringProperty(
					"The quantity credited, as a decimal.",
					0,
				),
				paramRebillAmount: toolparams.StringProperty(
					"The line's corrected amount on the reissued invoice, for a Manual rebill.", 0),
				paramRebillQuantity: toolparams.StringProperty(
					"The line's corrected quantity on the reissued invoice.", 0),
				paramLineDescription: toolparams.StringProperty(
					"The line's wording on the credit memo, when it should differ.",
					maxAdjustmentLineTextChars),
			},
			toolschema.KeyRequired:             []string{paramInvoiceLineID},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func supportingDocumentsProperty() map[string]any {
	return agenttoolschema.RecordIDs(permission.ResourceDocument,
		"Documents on the invoice's shipments that support it, from document_list or "+
			"shipment_get.", maxAdjustmentDocuments)
}

func adjustmentIDProperty(status string) map[string]any {
	return agenttoolschema.KindID(agenttoolschema.IDDescription("The "+status+" adjustment",
		"invoice_list_adjustments or invoice_get_adjustment"), permission.KindInvoiceAdjustment)
}

func optionalPositiveDecimal(fields map[string]any, key string) (decimal.Decimal, error) {
	minor, present, err := toolparams.OptionalMoney(fields, key)
	if err != nil || !present {
		return decimal.Zero, err
	}

	return decimal.New(minor, -2), nil
}

func readAdjustmentLines(
	fields map[string]any,
) ([]*invoiceadjustmentports.InvoiceAdjustmentLineInput, error) {
	raw, ok := fields[paramAdjustmentLines]
	if !ok || raw == nil {
		return nil, nil
	}
	entries, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a list of {invoiceLineId, ...}", paramAdjustmentLines)
	}
	if len(entries) > maxAdjustmentLines {
		return nil, fmt.Errorf("%s lists %d lines; at most %d", paramAdjustmentLines,
			len(entries), maxAdjustmentLines)
	}

	lines := make([]*invoiceadjustmentports.InvoiceAdjustmentLineInput, 0, len(entries))
	seen := make(map[pulid.ID]struct{}, len(entries))
	for idx, entry := range entries {
		line, lineErr := readAdjustmentLine(entry, seen)
		if lineErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramAdjustmentLines, idx, lineErr)
		}
		lines = append(lines, line)
	}

	return lines, nil
}

func readAdjustmentLine(
	entry any,
	seen map[pulid.ID]struct{},
) (*invoiceadjustmentports.InvoiceAdjustmentLineInput, error) {
	fields, ok := entry.(map[string]any)
	if !ok {
		return nil, errors.New("must be an object with invoiceLineId")
	}
	lineID, err := toolparams.RequirePulid(fields, paramInvoiceLineID)
	if err != nil {
		return nil, err
	}
	if _, duplicate := seen[lineID]; duplicate {
		return nil, fmt.Errorf("repeats line %s", lineID)
	}
	seen[lineID] = struct{}{}

	line := &invoiceadjustmentports.InvoiceAdjustmentLineInput{OriginalLineID: lineID}
	values := []struct {
		key  string
		into *decimal.Decimal
	}{
		{paramCreditAmount, &line.CreditAmount},
		{paramCreditQuantity, &line.CreditQuantity},
		{paramRebillAmount, &line.RebillAmount},
		{paramRebillQuantity, &line.RebillQuantity},
	}
	for _, value := range values {
		if *value.into, err = optionalPositiveDecimal(fields, value.key); err != nil {
			return nil, err
		}
	}
	if line.Description, err = toolparams.BoundedText(fields, paramLineDescription,
		maxAdjustmentLineTextChars); err != nil {
		return nil, err
	}

	return line, nil
}

func readAdjustmentDocuments(fields map[string]any) ([]pulid.ID, error) {
	if _, ok := fields[paramSupportingDocumentIDs]; !ok {
		return nil, nil
	}

	return toolparams.RequirePulidSlice(fields, paramSupportingDocumentIDs, maxAdjustmentDocuments)
}

func readRebillStrategy(fields map[string]any) (invoiceadjustment.RebillStrategy, error) {
	strategy, _, err := toolparams.OptionalEnum(
		fields,
		paramRebillStrategy,
		rebillStrategies.Values,
	)
	if err != nil {
		return "", err
	}
	if strategy == "" {
		strategy = invoiceadjustment.RebillStrategyCloneExact
	}

	return strategy, nil
}

func adjustmentKey(params *agenttoolports.ToolExecuteParams) string {
	key := strings.TrimSpace(params.IdempotencyKey)
	if key == "" {
		key = "preview"
	}

	return adjustmentKeyPrefix + key
}

func figuresRefusal(figures []*invoiceadjustmentports.InvoiceAdjustmentPreview) error {
	multiErr := errortypes.NewMultiError()
	for idx, preview := range figures {
		if preview == nil || len(preview.Errors) == 0 {
			continue
		}
		fields := make([]string, 0, len(preview.Errors))
		for field := range preview.Errors {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		for _, field := range fields {
			for _, message := range preview.Errors[field] {
				path := field
				if len(figures) > 1 {
					path = fmt.Sprintf("%s[%d].%s", paramAdjustments, idx, field)
				}
				multiErr.Add(path, errortypes.ErrInvalid, message)
			}
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type adjustmentSubmission struct {
	draftID pulid.ID
	items   []*invoiceadjustmentports.InvoiceAdjustmentRequest
	key     string
}

type submissionPlan struct {
	draft   *invoiceadjustment.InvoiceAdjustment
	figures []*invoiceadjustmentports.InvoiceAdjustmentPreview
}

func NewSubmitInvoiceAdjustmentTool(adjustments invoiceAdjuster) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:        "invoice_submit_adjustment",
		Recipe:      []string{toolInvoiceGet, "invoice_submit_adjustment"},
		SearchTerms: []string{"credit and rebill", "rebill"},
		Description: "Propose crediting, rebilling, reversing or writing off posted invoices. " +
			"Give adjustments, one per invoice and up to 25 at once, or draftAdjustmentId to " +
			"submit a saved draft as it stands. Each executes at once or waits for approval as " +
			"the organization's adjustment policy says. It creates credit memos and replacement " +
			"invoices, so a person always decides.",
		Resource:    permission.ResourceInvoice,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Idempotent:  true,
		Artifact:    changepreview.InvoiceRecordEntity,
		Rationale: "Credits, rebills or writes off receivables and books the entries that say " +
			"so; only a person submits an adjustment, and the adjustment policy may still hold " +
			"it for an approver.",
		Properties: map[string]any{
			ParamDraftAdjustmentID: adjustmentIDProperty("draft"),
			paramAdjustments: map[string]any{
				toolschema.KeyType: toolschema.TypeArray,
				toolschema.KeyDescription: "The adjustments to submit, one per invoice. Leave " +
					"out when submitting a draft.",
				toolschema.KeyMinItems: 1,
				toolschema.KeyMaxItems: maxAdjustmentsPerSubmission,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeObject,
					toolschema.KeyProperties: map[string]any{
						toolparams.ParamInvoiceID: agenttoolschema.RecordIDText(
							permission.ResourceInvoice,
							"The posted invoice, from invoice_list or invoice_get.",
						),
						paramAdjustmentKind: adjustmentKindProperty(),
						paramRebillStrategy: rebillStrategyProperty(),
						toolparams.ParamReason: toolparams.StringProperty(
							"Why, as the customer or the biller would say it.",
							maxAdjustmentReasonChars),
						paramAdjustmentLines:       adjustmentLinesProperty(),
						paramSupportingDocumentIDs: supportingDocumentsProperty(),
					},
					toolschema.KeyRequired: []string{
						toolparams.ParamInvoiceID, paramAdjustmentKind, toolparams.ParamReason,
					},
					toolschema.KeyAdditionalProperties: false,
				},
			},
		},
		Required: []string{},
		Target:   targetSubmittedInvoice,
	}, toolspec.ReceivablePlan[*adjustmentSubmission, *submissionPlan]{
		Request: submissionRequest,
		Plan: func(
			ctx context.Context,
			req *adjustmentSubmission,
			params *agenttoolports.ToolExecuteParams,
		) (*submissionPlan, error) {
			return planSubmission(ctx, adjustments, req, params)
		},
		Refused: func(req *adjustmentSubmission) string {
			if req.draftID.IsNotNil() {
				return "Would submit the draft adjustment."
			}
			return fmt.Sprintf(
				"Would submit %s.",
				changepreview.CountOf(len(req.items), "invoice adjustment"),
			)
		},
		Render: renderSubmission,
		Run: func(
			ctx context.Context,
			req *adjustmentSubmission,
			params *agenttoolports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			return runSubmission(ctx, adjustments, req, params)
		},
	})
}

func targetAdjustment(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, ParamAdjustmentID, agenttoolports.RecordInvoiceAdjustment)
}

func targetSubmittedInvoice(params map[string]any) (agenttoolports.ToolTarget, bool) {
	if target, ok := toolparams.TargetOf(
		params, ParamDraftAdjustmentID, agenttoolports.RecordInvoiceAdjustment,
	); ok {
		return target, true
	}
	items, _ := params[paramAdjustments].([]any)
	if len(items) != 1 {
		return agenttoolports.ToolTarget{}, false
	}
	fields, ok := items[0].(map[string]any)
	if !ok {
		return agenttoolports.ToolTarget{}, false
	}

	return toolparams.TargetOf(fields, toolparams.ParamInvoiceID, permission.ResourceInvoice)
}

func submissionRequest(params *agenttoolports.ToolExecuteParams) (*adjustmentSubmission, error) {
	draftID, hasDraft, err := toolparams.OptionalPulid(params.Params, ParamDraftAdjustmentID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", ParamDraftAdjustmentID, err)
	}
	raw, hasItems := params.Params[paramAdjustments]
	switch {
	case hasDraft && hasItems:
		return nil, errDraftAndAdjustments
	case !hasDraft && !hasItems:
		return nil, errDraftOrAdjustments
	}

	submission := &adjustmentSubmission{draftID: draftID, key: adjustmentKey(params)}
	if hasDraft {
		return submission, nil
	}

	entries, ok := raw.([]any)
	if !ok || len(entries) == 0 {
		return nil, fmt.Errorf("%s must be a non-empty list", paramAdjustments)
	}
	if len(entries) > maxAdjustmentsPerSubmission {
		return nil, fmt.Errorf("%s lists %d adjustments; at most %d go at once, so split it",
			paramAdjustments, len(entries), maxAdjustmentsPerSubmission)
	}

	tenant := toolparams.TenantFrom(*params)
	submission.items = make([]*invoiceadjustmentports.InvoiceAdjustmentRequest, 0, len(entries))
	for idx, entry := range entries {
		item, itemErr := readAdjustmentRequest(entry)
		if itemErr != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramAdjustments, idx, itemErr)
		}
		item.TenantInfo = tenant
		item.IdempotencyKey = submission.key
		if len(entries) > 1 {
			item.IdempotencyKey = fmt.Sprintf("%s-%d", submission.key, idx)
		}
		submission.items = append(submission.items, item)
	}

	return submission, nil
}

func readAdjustmentRequest(entry any) (*invoiceadjustmentports.InvoiceAdjustmentRequest, error) {
	fields, ok := entry.(map[string]any)
	if !ok {
		return nil, errors.New("must be an object with invoiceId, kind and reason")
	}
	invoiceID, err := toolparams.RequirePulid(fields, toolparams.ParamInvoiceID)
	if err != nil {
		return nil, err
	}
	kind, err := toolparams.RequireEnum(fields, paramAdjustmentKind, adjustmentKinds.Values)
	if err != nil {
		return nil, err
	}
	strategy, err := readRebillStrategy(fields)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.RequireBoundedText(
		fields,
		toolparams.ParamReason,
		maxAdjustmentReasonChars,
	)
	if err != nil {
		return nil, err
	}
	lines, err := readAdjustmentLines(fields)
	if err != nil {
		return nil, err
	}
	documents, err := readAdjustmentDocuments(fields)
	if err != nil {
		return nil, err
	}

	return &invoiceadjustmentports.InvoiceAdjustmentRequest{
		InvoiceID:      invoiceID,
		Kind:           kind,
		RebillStrategy: strategy,
		Reason:         reason,
		AttachmentIDs:  documents,
		Lines:          lines,
	}, nil
}

func planSubmission(
	ctx context.Context,
	adjustments invoiceAdjuster,
	req *adjustmentSubmission,
	params *agenttoolports.ToolExecuteParams,
) (*submissionPlan, error) {
	var (
		plan *submissionPlan
		err  error
	)
	if req.draftID.IsNotNil() {
		plan, err = planDraftSubmission(ctx, adjustments, req, params)
	} else {
		plan, err = planBulkSubmission(ctx, adjustments, req, params)
	}
	if err != nil {
		return nil, err
	}

	if err = figuresRefusal(plan.figures); err != nil {
		return nil, err
	}

	return plan, nil
}

func planDraftSubmission(
	ctx context.Context,
	adjustments invoiceAdjuster,
	req *adjustmentSubmission,
	params *agenttoolports.ToolExecuteParams,
) (*submissionPlan, error) {
	detail := &invoiceadjustmentports.GetInvoiceAdjustmentDetailRequest{
		AdjustmentID: req.draftID,
		TenantInfo:   toolparams.TenantFrom(*params),
	}
	draft, err := adjustments.GetDetail(ctx, detail)
	if err != nil {
		return nil, err
	}
	if draft.Status != invoiceadjustment.StatusDraft {
		return nil, errortypes.NewValidationError(
			ParamDraftAdjustmentID,
			errortypes.ErrInvalidOperation,
			"Only draft adjustments may be submitted; this one is {0}",
			string(draft.Status),
		)
	}
	figures, err := adjustments.PreviewDraft(ctx, detail, params.Actor)
	if err != nil {
		return nil, err
	}

	return &submissionPlan{
		draft:   draft,
		figures: []*invoiceadjustmentports.InvoiceAdjustmentPreview{figures},
	}, nil
}

func planBulkSubmission(
	ctx context.Context,
	adjustments invoiceAdjuster,
	req *adjustmentSubmission,
	params *agenttoolports.ToolExecuteParams,
) (*submissionPlan, error) {
	figures, err := adjustments.BulkPreview(
		ctx,
		&invoiceadjustmentports.InvoiceAdjustmentBulkRequest{
			IdempotencyKey: req.key,
			Items:          req.items,
			TenantInfo:     toolparams.TenantFrom(*params),
		},
		params.Actor,
	)
	if err != nil {
		return nil, err
	}

	return &submissionPlan{figures: figures}, nil
}

func runSubmission(
	ctx context.Context,
	adjustments invoiceAdjuster,
	req *adjustmentSubmission,
	params *agenttoolports.ToolExecuteParams,
) (*agent.ToolExecutionResult, error) {
	switch {
	case req.draftID.IsNotNil():
		submitted, err := adjustments.SubmitDraft(
			ctx,
			&invoiceadjustmentports.GetInvoiceAdjustmentDetailRequest{
				AdjustmentID: req.draftID,
				TenantInfo:   toolparams.TenantFrom(*params),
			},
			params.Actor,
		)
		if err != nil {
			return nil, err
		}
		return adjustmentResult(submitted), nil
	case len(req.items) == 1:
		submitted, err := adjustments.Submit(ctx, req.items[0], params.Actor)
		if err != nil {
			return nil, err
		}
		return adjustmentResult(submitted), nil
	default:
		batch, err := adjustments.BulkSubmit(
			ctx,
			&invoiceadjustmentports.InvoiceAdjustmentBulkRequest{
				IdempotencyKey: req.key,
				Items:          req.items,
				TenantInfo:     toolparams.TenantFrom(*params),
			},
			params.Actor,
		)
		if err != nil {
			return nil, err
		}
		return &agent.ToolExecutionResult{
			Action: "submitted",
			Kind:   "invoice adjustment batch",
			IDs:    map[string]string{"batchId": batch.ID.String()},
		}, nil
	}
}

func adjustmentResult(adjustment *invoiceadjustment.InvoiceAdjustment) *agent.ToolExecutionResult {
	if adjustment == nil {
		return &agent.ToolExecutionResult{Action: "submitted", Kind: "invoice adjustment"}
	}

	return &agent.ToolExecutionResult{
		Action: strings.ToLower(string(adjustment.Status)),
		Kind:   "invoice adjustment",
		IDs:    map[string]string{ParamAdjustmentID: adjustment.ID.String()},
		Record: &agent.RecordRef{
			EntityType: changepreview.InvoiceRecordEntity,
			ID:         adjustment.OriginalInvoiceID.String(),
		},
	}
}

func NewSaveInvoiceAdjustmentDraftTool(adjustments invoiceAdjuster) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name: "invoice_save_adjustment_draft",
		Description: "Save a draft credit, rebill, reversal or write-off on a posted invoice for " +
			"a biller to review and submit; nothing is credited until it is submitted. Give " +
			"invoiceId for a new draft, or adjustmentId to rewrite a draft already saved. The " +
			"result names the draft, which invoice_submit_adjustment takes as draftAdjustmentId.",
		Resource:    permission.ResourceInvoice,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierAutoExecute,
		Reversible:  true,
		Artifact:    changepreview.InvoiceRecordEntity,
		TaintHold: "A draft written from what a customer sent is proposed, since its reason " +
			"and amounts are what a biller submits.",
		Rationale: "Saves a draft adjustment inside Trenova; it credits nothing until a " +
			"person submits it, and a later save rewrites it.",
		Properties: map[string]any{
			ParamAdjustmentID: adjustmentIDProperty("draft"),
			toolparams.ParamInvoiceID: agenttoolschema.RecordIDText(permission.ResourceInvoice,
				"The posted invoice a new draft adjusts, from invoice_list or invoice_get. "+
					"Beside adjustmentId it must be that draft's invoice."),
			paramAdjustmentKind: adjustmentKindProperty(),
			paramRebillStrategy: rebillStrategyProperty(),
			toolparams.ParamReason: toolparams.StringProperty(
				"Why, as the customer or the biller would say it.",
				maxAdjustmentReasonChars,
			),
			paramAdjustmentLines:       adjustmentLinesProperty(),
			paramSupportingDocumentIDs: supportingDocumentsProperty(),
		},
		Required: []string{paramAdjustmentKind},
		Target:   toolparams.TargetInvoice,
	}, toolspec.ReceivablePlan[*invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest, *invoiceadjustmentports.InvoiceAdjustmentDraftPreview]{
		Request: draftRequest,
		Settle: func(
			ctx context.Context,
			req *invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest,
			_ *agenttoolports.ToolExecuteParams,
		) error {
			return draftOnInvoice(ctx, adjustments, req)
		},
		Plan: func(
			ctx context.Context,
			req *invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*invoiceadjustmentports.InvoiceAdjustmentDraftPreview, error) {
			return adjustments.PreviewSaveDraft(ctx, req, params.Actor)
		},
		Refused: func(req *invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest) string {
			return "Would save a draft " + string(req.Kind) + " adjustment."
		},
		Render: renderDraft,
		Run: func(
			ctx context.Context,
			req *invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			saved, err := adjustments.SaveDraft(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			return &agent.ToolExecutionResult{
				Action: "saved",
				Kind:   "invoice adjustment draft",
				IDs:    map[string]string{ParamAdjustmentID: saved.ID.String()},
				Record: &agent.RecordRef{
					EntityType: changepreview.InvoiceRecordEntity,
					ID:         saved.OriginalInvoiceID.String(),
				},
			}, nil
		},
	})
}

func draftRequest(
	params *agenttoolports.ToolExecuteParams,
) (*invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest, error) {
	adjustmentID, _, err := toolparams.OptionalPulid(params.Params, ParamAdjustmentID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", ParamAdjustmentID, err)
	}
	invoiceID, _, err := toolparams.OptionalPulid(params.Params, toolparams.ParamInvoiceID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", toolparams.ParamInvoiceID, err)
	}
	if adjustmentID.IsNil() && invoiceID.IsNil() {
		return nil, errNoDraftTarget
	}
	kind, err := toolparams.RequireEnum(params.Params, paramAdjustmentKind, adjustmentKinds.Values)
	if err != nil {
		return nil, err
	}
	strategy, err := readRebillStrategy(params.Params)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.BoundedText(
		params.Params,
		toolparams.ParamReason,
		maxAdjustmentReasonChars,
	)
	if err != nil {
		return nil, err
	}
	lines, err := readAdjustmentLines(params.Params)
	if err != nil {
		return nil, err
	}
	documents, err := readAdjustmentDocuments(params.Params)
	if err != nil {
		return nil, err
	}

	return &invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest{
		AdjustmentID:          adjustmentID,
		InvoiceID:             invoiceID,
		Kind:                  kind,
		RebillStrategy:        strategy,
		Reason:                reason,
		ReferencedDocumentIDs: documents,
		Lines:                 lines,
		TenantInfo:            toolparams.TenantFrom(*params),
	}, nil
}

// draftOnInvoice takes a rewrite that names the draft and also the invoice it
// adjusts. A model that read the draft off the invoice sent both, and was
// refused for naming both though they named the same thing; the draft's own
// invoice is what a rewrite adjusts, so an invoiceId that is that invoice is
// taken. One naming another invoice is refused, since a rewrite cannot move a
// draft to it and a new draft there is a different request.
func draftOnInvoice(
	ctx context.Context,
	adjustments invoiceAdjuster,
	req *invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest,
) error {
	if req.AdjustmentID.IsNil() || req.InvoiceID.IsNil() {
		return nil
	}
	draft, err := adjustments.GetDetail(
		ctx,
		&invoiceadjustmentports.GetInvoiceAdjustmentDetailRequest{
			AdjustmentID: req.AdjustmentID,
			TenantInfo:   req.TenantInfo,
		},
	)
	if err != nil {
		return err
	}
	if draft.OriginalInvoiceID == req.InvoiceID {
		return nil
	}

	return fmt.Errorf(
		"adjustmentId %s is a draft on invoice %s, not on invoiceId %s. To rewrite that "+
			"draft, send adjustmentId without invoiceId, for example "+
			`{"adjustmentId": "%s", "kind": "CreditOnly", "reason": "…"}. To start a new `+
			"draft on invoice %s, send invoiceId without adjustmentId, for example "+
			`{"invoiceId": "%s", "kind": "CreditOnly", "reason": "…"}`,
		req.AdjustmentID, draft.OriginalInvoiceID, req.InvoiceID,
		req.AdjustmentID, req.InvoiceID, req.InvoiceID,
	)
}

type adjustmentDecision struct {
	adjustmentID pulid.ID
	reason       string
	approve      bool
}

func NewApproveInvoiceAdjustmentTool(adjustments invoiceAdjuster) agenttoolports.AgentTool {
	return newDecisionTool(adjustments, true, &toolspec.ReceivableSpec{
		Name: "invoice_approve_adjustment",
		Description: "Propose approving an invoice adjustment that waits for approval, which " +
			"executes it: the credit memo and any replacement invoice are made and posted. It is " +
			"checked again against the invoice as it stands, as execution does. A person always " +
			"decides.",
		Resource:    permission.ResourceInvoice,
		Operation:   permission.OpApprove,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Rationale: "Executes a credit, rebill or write-off an approver was asked to sign off; " +
			"approving is a person's decision.",
		Properties: map[string]any{ParamAdjustmentID: adjustmentIDProperty("pending")},
		Required:   []string{ParamAdjustmentID},
		Target:     targetAdjustment,
	})
}

func NewRejectInvoiceAdjustmentTool(adjustments invoiceAdjuster) agenttoolports.AgentTool {
	return newDecisionTool(adjustments, false, &toolspec.ReceivableSpec{
		Name:        "invoice_reject_adjustment",
		SearchTerms: []string{"deny adjustment", "decline adjustment"},
		Description: "Propose rejecting an invoice adjustment that waits for approval, with the " +
			"reason the submitter will read. Nothing is credited and the invoice is left as it " +
			"is. A person always decides.",
		Resource:    permission.ResourceInvoice,
		Operation:   permission.OpApprove,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Rationale: "Turns down an adjustment an approver was asked to sign off; the approval " +
			"decision is a person's.",
		Properties: map[string]any{
			ParamAdjustmentID: adjustmentIDProperty("pending"),
			toolparams.ParamReason: toolparams.StringProperty(
				"Why it is rejected, for the person who submitted it.",
				maxAdjustmentReasonChars,
			),
		},
		Required: []string{ParamAdjustmentID, toolparams.ParamReason},
		Target:   targetAdjustment,
	})
}

func newDecisionTool(
	adjustments invoiceAdjuster,
	approve bool,
	spec *toolspec.ReceivableSpec,
) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(
		spec,
		toolspec.ReceivablePlan[*adjustmentDecision, *invoiceadjustmentports.InvoiceAdjustmentDecisionPreview]{
			Request: func(params *agenttoolports.ToolExecuteParams) (*adjustmentDecision, error) {
				id, err := toolparams.RequirePulid(params.Params, ParamAdjustmentID)
				if err != nil {
					return nil, err
				}
				decision := &adjustmentDecision{adjustmentID: id, approve: approve}
				if !approve {
					if decision.reason, err = toolparams.RequireBoundedText(
						params.Params, toolparams.ParamReason, maxAdjustmentReasonChars,
					); err != nil {
						return nil, err
					}
				}
				return decision, nil
			},
			Plan: func(
				ctx context.Context,
				req *adjustmentDecision,
				params *agenttoolports.ToolExecuteParams,
			) (*invoiceadjustmentports.InvoiceAdjustmentDecisionPreview, error) {
				plan, err := adjustments.PreviewDecision(
					ctx,
					&invoiceadjustmentports.InvoiceAdjustmentDecisionRequest{
						AdjustmentID: req.adjustmentID,
						Approve:      req.approve,
						TenantInfo:   toolparams.TenantFrom(*params),
					},
				)
				if err != nil {
					return nil, err
				}
				if req.approve {
					if refusal := figuresRefusal([]*invoiceadjustmentports.InvoiceAdjustmentPreview{
						plan.Figures,
					}); refusal != nil {
						return nil, refusal
					}
				}
				return plan, nil
			},
			Refused: func(req *adjustmentDecision) string {
				if req.approve {
					return "Would approve the invoice adjustment."
				}
				return "Would reject the invoice adjustment."
			},
			Render: renderDecision,
			Run: func(
				ctx context.Context,
				req *adjustmentDecision,
				params *agenttoolports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				var err error
				if req.approve {
					_, err = adjustments.Approve(
						ctx,
						&invoiceadjustmentports.ApproveInvoiceAdjustmentRequest{
							AdjustmentID: req.adjustmentID,
							TenantInfo:   toolparams.TenantFrom(*params),
						},
						params.Actor,
					)
				} else {
					_, err = adjustments.Reject(
						ctx,
						&invoiceadjustmentports.RejectInvoiceAdjustmentRequest{
							AdjustmentID: req.adjustmentID,
							Reason:       req.reason,
							TenantInfo:   toolparams.TenantFrom(*params),
						},
						params.Actor,
					)
				}
				return nil, err
			},
		},
	)
}
