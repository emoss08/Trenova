package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	paramAdjustmentID           = "adjustmentId"
	paramDraftAdjustmentID      = "draftAdjustmentId"
	paramAdjustments            = "adjustments"
	paramAdjustmentKind         = "kind"
	paramRebillStrategy         = "rebillStrategy"
	paramAdjustmentLines        = "lines"
	paramInvoiceLineID          = "invoiceLineId"
	paramCreditAmount           = "creditAmount"
	paramCreditQuantity         = "creditQuantity"
	paramRebillAmount           = "rebillAmount"
	paramRebillQuantity         = "rebillQuantity"
	paramLineDescription        = "description"
	paramSupportingDocumentIDs  = "supportingDocumentIds"
	maxAdjustmentsPerSubmission = 25
	maxAdjustmentLines          = 200
	maxAdjustmentDocuments      = 20
	maxAdjustmentReasonChars    = 1000
	maxAdjustmentLineTextChars  = 500
	adjustmentKeyPrefix         = "agent-proposal:"
)

var (
	adjustmentKinds = []invoiceadjustment.Kind{
		invoiceadjustment.KindCreditOnly,
		invoiceadjustment.KindCreditRebill,
		invoiceadjustment.KindFullReversal,
		invoiceadjustment.KindWriteOff,
	}
	rebillStrategies = []invoiceadjustment.RebillStrategy{
		invoiceadjustment.RebillStrategyCloneExact,
		invoiceadjustment.RebillStrategyRerate,
		invoiceadjustment.RebillStrategyManual,
	}
	errDraftOrAdjustments = errors.New(
		"give either draftAdjustmentId, to submit a saved draft, or adjustments, not both",
	)
)

type invoiceAdjuster interface {
	SaveDraft(
		ctx context.Context,
		req *serviceports.SaveInvoiceAdjustmentDraftRequest,
		actor *serviceports.RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	PreviewSaveDraft(
		ctx context.Context,
		req *serviceports.SaveInvoiceAdjustmentDraftRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.InvoiceAdjustmentDraftPreview, error)
	GetDetail(
		ctx context.Context,
		req *serviceports.GetInvoiceAdjustmentDetailRequest,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	PreviewDraft(
		ctx context.Context,
		req *serviceports.GetInvoiceAdjustmentDetailRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.InvoiceAdjustmentPreview, error)
	SubmitDraft(
		ctx context.Context,
		req *serviceports.GetInvoiceAdjustmentDetailRequest,
		actor *serviceports.RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	Preview(
		ctx context.Context,
		req *serviceports.InvoiceAdjustmentRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.InvoiceAdjustmentPreview, error)
	Submit(
		ctx context.Context,
		req *serviceports.InvoiceAdjustmentRequest,
		actor *serviceports.RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	BulkPreview(
		ctx context.Context,
		req *serviceports.InvoiceAdjustmentBulkRequest,
		actor *serviceports.RequestActor,
	) ([]*serviceports.InvoiceAdjustmentPreview, error)
	BulkSubmit(
		ctx context.Context,
		req *serviceports.InvoiceAdjustmentBulkRequest,
		actor *serviceports.RequestActor,
	) (*invoiceadjustment.InvoiceAdjustmentBatch, error)
	Approve(
		ctx context.Context,
		req *serviceports.ApproveInvoiceAdjustmentRequest,
		actor *serviceports.RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	Reject(
		ctx context.Context,
		req *serviceports.RejectInvoiceAdjustmentRequest,
		actor *serviceports.RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	PreviewDecision(
		ctx context.Context,
		req *serviceports.InvoiceAdjustmentDecisionRequest,
	) (*serviceports.InvoiceAdjustmentDecisionPreview, error)
}

func adjustmentKindProperty() map[string]any {
	return enumProperty("CreditOnly credits lines; CreditAndRebill credits them and reissues "+
		"a corrected invoice; FullReversal credits the whole invoice; WriteOff writes an "+
		"uncollectable balance off.", enumNames(adjustmentKinds))
}

func rebillStrategyProperty() map[string]any {
	return enumProperty("How a CreditAndRebill reissues: CloneExact copies the lines, Rerate "+
		"prices them again, Manual takes the rebill amounts you give. Defaults to CloneExact.",
		enumNames(rebillStrategies))
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
				paramInvoiceLineID: stringProperty(
					"The line's id, from get_invoice's lines.", 0),
				paramCreditAmount: stringProperty(
					"How much of the line to credit, as a decimal.", 0),
				paramCreditQuantity: stringProperty("The quantity credited, as a decimal.", 0),
				paramRebillAmount: stringProperty(
					"The line's corrected amount on the reissued invoice, for a Manual rebill.", 0),
				paramRebillQuantity: stringProperty(
					"The line's corrected quantity on the reissued invoice.", 0),
				paramLineDescription: stringProperty(
					"The line's wording on the credit memo, when it should differ.",
					maxAdjustmentLineTextChars),
			},
			toolschema.KeyRequired:             []string{paramInvoiceLineID},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func supportingDocumentsProperty() map[string]any {
	return idListProperty("Documents on the invoice's shipments that support it, from "+
		"search_documents or get_shipment.", maxAdjustmentDocuments)
}

func adjustmentIDProperty(status string) map[string]any {
	return stringProperty("The "+status+" adjustment, from list_invoice_adjustments or "+
		"get_invoice_adjustment. Never guess one.", 0)
}

func optionalPositiveDecimal(fields map[string]any, key string) (decimal.Decimal, error) {
	minor, present, err := optionalMoney(fields, key)
	if err != nil || !present {
		return decimal.Zero, err
	}

	return decimal.New(minor, -2), nil
}

func readAdjustmentLines(
	fields map[string]any,
) ([]*serviceports.InvoiceAdjustmentLineInput, error) {
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

	lines := make([]*serviceports.InvoiceAdjustmentLineInput, 0, len(entries))
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
) (*serviceports.InvoiceAdjustmentLineInput, error) {
	fields, ok := entry.(map[string]any)
	if !ok {
		return nil, errors.New("must be an object with invoiceLineId")
	}
	lineID, err := requirePulid(fields, paramInvoiceLineID)
	if err != nil {
		return nil, err
	}
	if _, duplicate := seen[lineID]; duplicate {
		return nil, fmt.Errorf("repeats line %s", lineID)
	}
	seen[lineID] = struct{}{}

	line := &serviceports.InvoiceAdjustmentLineInput{OriginalLineID: lineID}
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
	if line.Description, err = boundedText(fields, paramLineDescription,
		maxAdjustmentLineTextChars); err != nil {
		return nil, err
	}

	return line, nil
}

func readAdjustmentDocuments(fields map[string]any) ([]pulid.ID, error) {
	if _, ok := fields[paramSupportingDocumentIDs]; !ok {
		return nil, nil
	}

	return requirePulidSlice(fields, paramSupportingDocumentIDs, maxAdjustmentDocuments)
}

func readRebillStrategy(fields map[string]any) (invoiceadjustment.RebillStrategy, error) {
	strategy, err := optionalEnum(fields, paramRebillStrategy, rebillStrategies)
	if err != nil {
		return "", err
	}
	if strategy == "" {
		strategy = invoiceadjustment.RebillStrategyCloneExact
	}

	return strategy, nil
}

func adjustmentKey(params *serviceports.ToolExecuteParams) string {
	key := strings.TrimSpace(params.IdempotencyKey)
	if key == "" {
		key = "preview"
	}

	return adjustmentKeyPrefix + key
}

func figuresRefusal(figures []*serviceports.InvoiceAdjustmentPreview) error {
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
	items   []*serviceports.InvoiceAdjustmentRequest
	key     string
}

type submissionPlan struct {
	draft   *invoiceadjustment.InvoiceAdjustment
	figures []*serviceports.InvoiceAdjustmentPreview
}

func newSubmitInvoiceAdjustmentTool(adjustments invoiceAdjuster) serviceports.AgentTool {
	return newReportingReceivableTool(receivableSpec{
		name: "submit_invoice_adjustment",
		description: "Propose crediting, rebilling, reversing or writing off posted invoices. " +
			"Give adjustments, one per invoice and up to 25 at once, or draftAdjustmentId to " +
			"submit a saved draft as it stands. Each executes at once or waits for approval as " +
			"the organization's adjustment policy says. It creates credit memos and replacement " +
			"invoices, so a person always decides.",
		resource:    permission.ResourceInvoice,
		operation:   permission.OpUpdate,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		idempotent:  true,
		artifact:    invoiceRecordEntity,
		rationale: "Credits, rebills or writes off receivables and books the entries that say " +
			"so; only a person submits an adjustment, and the adjustment policy may still hold " +
			"it for an approver.",
		properties: map[string]any{
			paramDraftAdjustmentID: adjustmentIDProperty("draft"),
			paramAdjustments: map[string]any{
				toolschema.KeyType: toolschema.TypeArray,
				toolschema.KeyDescription: "The adjustments to submit, one per invoice. Leave " +
					"out when submitting a draft.",
				toolschema.KeyMinItems: 1,
				toolschema.KeyMaxItems: maxAdjustmentsPerSubmission,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeObject,
					toolschema.KeyProperties: map[string]any{
						paramInvoiceID: stringProperty(
							"The posted invoice, from list_invoices or get_invoice.", 0),
						paramAdjustmentKind: adjustmentKindProperty(),
						paramRebillStrategy: rebillStrategyProperty(),
						paramReason: stringProperty(
							"Why, as the customer or the biller would say it.",
							maxAdjustmentReasonChars),
						paramAdjustmentLines:       adjustmentLinesProperty(),
						paramSupportingDocumentIDs: supportingDocumentsProperty(),
					},
					toolschema.KeyRequired: []string{
						paramInvoiceID, paramAdjustmentKind, paramReason,
					},
					toolschema.KeyAdditionalProperties: false,
				},
			},
		},
		required: []string{},
		target:   targetSubmittedInvoice,
	}, receivablePlan[*adjustmentSubmission, *submissionPlan]{
		request: submissionRequest,
		plan: func(
			ctx context.Context,
			req *adjustmentSubmission,
			params *serviceports.ToolExecuteParams,
		) (*submissionPlan, error) {
			return planSubmission(ctx, adjustments, req, params)
		},
		refused: func(req *adjustmentSubmission) string {
			if req.draftID.IsNotNil() {
				return "Would submit the draft adjustment."
			}
			return fmt.Sprintf("Would submit %s.", countOf(len(req.items), "invoice adjustment"))
		},
		render: renderSubmission,
		run: func(
			ctx context.Context,
			req *adjustmentSubmission,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			return runSubmission(ctx, adjustments, req, params)
		},
	})
}

func targetAdjustment(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramAdjustmentID, serviceports.RecordInvoiceAdjustment)
}

func targetSubmittedInvoice(params map[string]any) (serviceports.ToolTarget, bool) {
	if target, ok := targetOf(
		params, paramDraftAdjustmentID, serviceports.RecordInvoiceAdjustment,
	); ok {
		return target, true
	}
	items, _ := params[paramAdjustments].([]any)
	if len(items) != 1 {
		return serviceports.ToolTarget{}, false
	}
	fields, ok := items[0].(map[string]any)
	if !ok {
		return serviceports.ToolTarget{}, false
	}

	return targetOf(fields, paramInvoiceID, permission.ResourceInvoice)
}

func submissionRequest(params *serviceports.ToolExecuteParams) (*adjustmentSubmission, error) {
	draftID, hasDraft, err := optionalPulid(params.Params, paramDraftAdjustmentID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramDraftAdjustmentID, err)
	}
	raw, hasItems := params.Params[paramAdjustments]
	if hasDraft == hasItems {
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

	tenant := tenantFrom(*params)
	submission.items = make([]*serviceports.InvoiceAdjustmentRequest, 0, len(entries))
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

func readAdjustmentRequest(entry any) (*serviceports.InvoiceAdjustmentRequest, error) {
	fields, ok := entry.(map[string]any)
	if !ok {
		return nil, errors.New("must be an object with invoiceId, kind and reason")
	}
	invoiceID, err := requirePulid(fields, paramInvoiceID)
	if err != nil {
		return nil, err
	}
	kind, err := requireEnum(fields, paramAdjustmentKind, adjustmentKinds)
	if err != nil {
		return nil, err
	}
	strategy, err := readRebillStrategy(fields)
	if err != nil {
		return nil, err
	}
	reason, err := requireBoundedText(fields, paramReason, maxAdjustmentReasonChars)
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

	return &serviceports.InvoiceAdjustmentRequest{
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
	params *serviceports.ToolExecuteParams,
) (*submissionPlan, error) {
	plan := &submissionPlan{}
	if req.draftID.IsNotNil() {
		detail := &serviceports.GetInvoiceAdjustmentDetailRequest{
			AdjustmentID: req.draftID,
			TenantInfo:   tenantFrom(*params),
		}
		draft, err := adjustments.GetDetail(ctx, detail)
		if err != nil {
			return nil, err
		}
		if draft.Status != invoiceadjustment.StatusDraft {
			return nil, errortypes.NewValidationError(
				paramDraftAdjustmentID,
				errortypes.ErrInvalidOperation,
				"Only draft adjustments may be submitted; this one is {0}",
				string(draft.Status),
			)
		}
		figures, err := adjustments.PreviewDraft(ctx, detail, params.Actor)
		if err != nil {
			return nil, err
		}
		plan.draft = draft
		plan.figures = []*serviceports.InvoiceAdjustmentPreview{figures}
	} else {
		figures, err := adjustments.BulkPreview(ctx, &serviceports.InvoiceAdjustmentBulkRequest{
			IdempotencyKey: req.key,
			Items:          req.items,
			TenantInfo:     tenantFrom(*params),
		}, params.Actor)
		if err != nil {
			return nil, err
		}
		plan.figures = figures
	}

	if err := figuresRefusal(plan.figures); err != nil {
		return nil, err
	}

	return plan, nil
}

func runSubmission(
	ctx context.Context,
	adjustments invoiceAdjuster,
	req *adjustmentSubmission,
	params *serviceports.ToolExecuteParams,
) (*agent.ToolExecutionResult, error) {
	switch {
	case req.draftID.IsNotNil():
		submitted, err := adjustments.SubmitDraft(ctx, &serviceports.GetInvoiceAdjustmentDetailRequest{
			AdjustmentID: req.draftID,
			TenantInfo:   tenantFrom(*params),
		}, params.Actor)
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
		batch, err := adjustments.BulkSubmit(ctx, &serviceports.InvoiceAdjustmentBulkRequest{
			IdempotencyKey: req.key,
			Items:          req.items,
			TenantInfo:     tenantFrom(*params),
		}, params.Actor)
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
		IDs:    map[string]string{paramAdjustmentID: adjustment.ID.String()},
		Record: &agent.RecordRef{
			EntityType: invoiceRecordEntity,
			ID:         adjustment.OriginalInvoiceID.String(),
		},
	}
}

func newSaveInvoiceAdjustmentDraftTool(adjustments invoiceAdjuster) serviceports.AgentTool {
	return newReportingReceivableTool(receivableSpec{
		name: "save_invoice_adjustment_draft",
		description: "Save a draft credit, rebill, reversal or write-off on a posted invoice for " +
			"a biller to review and submit; nothing is credited until it is submitted. Give " +
			"invoiceId for a new draft, or adjustmentId to rewrite a draft already saved. The " +
			"result names the draft, which submit_invoice_adjustment takes as draftAdjustmentId.",
		resource:    permission.ResourceInvoice,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierAutoExecute,
		reversible:  true,
		artifact:    invoiceRecordEntity,
		taintHold: "A draft written from what a customer sent is proposed, since its reason " +
			"and amounts are what a biller submits.",
		rationale: "Saves a draft adjustment inside Trenova; it credits nothing until a " +
			"person submits it, and a later save rewrites it.",
		properties: map[string]any{
			paramAdjustmentID: adjustmentIDProperty("draft"),
			paramInvoiceID: stringProperty(
				"The posted invoice a new draft adjusts, from list_invoices or get_invoice.", 0),
			paramAdjustmentKind: adjustmentKindProperty(),
			paramRebillStrategy: rebillStrategyProperty(),
			paramReason: stringProperty("Why, as the customer or the biller would say it.",
				maxAdjustmentReasonChars),
			paramAdjustmentLines:       adjustmentLinesProperty(),
			paramSupportingDocumentIDs: supportingDocumentsProperty(),
		},
		required: []string{paramAdjustmentKind},
		target:   targetInvoice,
	}, receivablePlan[*serviceports.SaveInvoiceAdjustmentDraftRequest, *serviceports.InvoiceAdjustmentDraftPreview]{
		request: draftRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.SaveInvoiceAdjustmentDraftRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.InvoiceAdjustmentDraftPreview, error) {
			return adjustments.PreviewSaveDraft(ctx, req, params.Actor)
		},
		refused: func(req *serviceports.SaveInvoiceAdjustmentDraftRequest) string {
			return "Would save a draft " + string(req.Kind) + " adjustment."
		},
		render: renderDraft,
		run: func(
			ctx context.Context,
			req *serviceports.SaveInvoiceAdjustmentDraftRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			saved, err := adjustments.SaveDraft(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			return &agent.ToolExecutionResult{
				Action: "saved",
				Kind:   "invoice adjustment draft",
				IDs:    map[string]string{paramAdjustmentID: saved.ID.String()},
				Record: &agent.RecordRef{
					EntityType: invoiceRecordEntity,
					ID:         saved.OriginalInvoiceID.String(),
				},
			}, nil
		},
	})
}

func draftRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.SaveInvoiceAdjustmentDraftRequest, error) {
	adjustmentID, _, err := optionalPulid(params.Params, paramAdjustmentID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramAdjustmentID, err)
	}
	invoiceID, _, err := optionalPulid(params.Params, paramInvoiceID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramInvoiceID, err)
	}
	if adjustmentID.IsNil() == invoiceID.IsNil() {
		return nil, errors.New(
			"give invoiceId for a new draft or adjustmentId for a saved one, not both",
		)
	}
	kind, err := requireEnum(params.Params, paramAdjustmentKind, adjustmentKinds)
	if err != nil {
		return nil, err
	}
	strategy, err := readRebillStrategy(params.Params)
	if err != nil {
		return nil, err
	}
	reason, err := boundedText(params.Params, paramReason, maxAdjustmentReasonChars)
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

	return &serviceports.SaveInvoiceAdjustmentDraftRequest{
		AdjustmentID:          adjustmentID,
		InvoiceID:             invoiceID,
		Kind:                  kind,
		RebillStrategy:        strategy,
		Reason:                reason,
		ReferencedDocumentIDs: documents,
		Lines:                 lines,
		TenantInfo:            tenantFrom(*params),
	}, nil
}

type adjustmentDecision struct {
	adjustmentID pulid.ID
	reason       string
	approve      bool
}

func newApproveInvoiceAdjustmentTool(adjustments invoiceAdjuster) serviceports.AgentTool {
	return newDecisionTool(adjustments, true, receivableSpec{
		name: "approve_invoice_adjustment",
		description: "Propose approving an invoice adjustment that waits for approval, which " +
			"executes it: the credit memo and any replacement invoice are made and posted. It is " +
			"checked again against the invoice as it stands, as execution does. A person always " +
			"decides.",
		resource:    permission.ResourceInvoice,
		operation:   permission.OpApprove,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		rationale: "Executes a credit, rebill or write-off an approver was asked to sign off; " +
			"approving is a person's decision.",
		properties: map[string]any{paramAdjustmentID: adjustmentIDProperty("pending")},
		required:   []string{paramAdjustmentID},
		target:     targetAdjustment,
	})
}

func newRejectInvoiceAdjustmentTool(adjustments invoiceAdjuster) serviceports.AgentTool {
	return newDecisionTool(adjustments, false, receivableSpec{
		name: "reject_invoice_adjustment",
		description: "Propose rejecting an invoice adjustment that waits for approval, with the " +
			"reason the submitter will read. Nothing is credited and the invoice is left as it " +
			"is. A person always decides.",
		resource:    permission.ResourceInvoice,
		operation:   permission.OpApprove,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		rationale: "Turns down an adjustment an approver was asked to sign off; the approval " +
			"decision is a person's.",
		properties: map[string]any{
			paramAdjustmentID: adjustmentIDProperty("pending"),
			paramReason: stringProperty("Why it is rejected, for the person who submitted it.",
				maxAdjustmentReasonChars),
		},
		required: []string{paramAdjustmentID, paramReason},
		target:   targetAdjustment,
	})
}

func newDecisionTool(
	adjustments invoiceAdjuster,
	approve bool,
	spec receivableSpec,
) serviceports.AgentTool {
	return newReceivableTool(spec, receivablePlan[*adjustmentDecision, *serviceports.InvoiceAdjustmentDecisionPreview]{
		request: func(params *serviceports.ToolExecuteParams) (*adjustmentDecision, error) {
			id, err := requirePulid(params.Params, paramAdjustmentID)
			if err != nil {
				return nil, err
			}
			decision := &adjustmentDecision{adjustmentID: id, approve: approve}
			if !approve {
				if decision.reason, err = requireBoundedText(
					params.Params, paramReason, maxAdjustmentReasonChars,
				); err != nil {
					return nil, err
				}
			}
			return decision, nil
		},
		plan: func(
			ctx context.Context,
			req *adjustmentDecision,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.InvoiceAdjustmentDecisionPreview, error) {
			plan, err := adjustments.PreviewDecision(ctx, &serviceports.InvoiceAdjustmentDecisionRequest{
				AdjustmentID: req.adjustmentID,
				Approve:      req.approve,
				TenantInfo:   tenantFrom(*params),
			})
			if err != nil {
				return nil, err
			}
			if req.approve {
				if refusal := figuresRefusal([]*serviceports.InvoiceAdjustmentPreview{
					plan.Figures,
				}); refusal != nil {
					return nil, refusal
				}
			}
			return plan, nil
		},
		refused: func(req *adjustmentDecision) string {
			if req.approve {
				return "Would approve the invoice adjustment."
			}
			return "Would reject the invoice adjustment."
		},
		render: renderDecision,
		run: func(
			ctx context.Context,
			req *adjustmentDecision,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			var err error
			if req.approve {
				_, err = adjustments.Approve(ctx, &serviceports.ApproveInvoiceAdjustmentRequest{
					AdjustmentID: req.adjustmentID,
					TenantInfo:   tenantFrom(*params),
				}, params.Actor)
			} else {
				_, err = adjustments.Reject(ctx, &serviceports.RejectInvoiceAdjustmentRequest{
					AdjustmentID: req.adjustmentID,
					Reason:       req.reason,
					TenantInfo:   tenantFrom(*params),
				}, params.Actor)
			}
			return nil, err
		},
	})
}
