package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramCaptureItems     = "items"
	paramCaptureItemID    = "itemId"
	paramCaptureItemIDOne = "captureItemId"
	paramCaptureBatchID   = "captureBatchId"
	paramCaptureTarget    = "targetType"
	paramCaptureTargetID  = "targetId"
	paramCaptureDocType   = "documentTypeId"
	maxCaptureFilings     = 25
	captureSources        = "list_capture_batches"
)

var captureTargetTypes = agenttoolschema.Source(
	"capture.targetType",
	capture.FileableResources(),
)

type captureKeeper interface {
	GetItem(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		itemID pulid.ID,
	) (*capture.CaptureItem, error)
	GetBatch(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		batchID pulid.ID,
	) (*capture.CaptureBatch, error)
	PlanFileItem(
		ctx context.Context,
		in *captureservice.FileItemInput,
	) (*captureservice.FilePlan, error)
	FileItem(ctx context.Context, in *captureservice.FileItemInput) (*capture.CaptureItem, error)
	FileItems(
		ctx context.Context,
		in *captureservice.FileItemsInput,
	) (*captureservice.FileItemsResult, error)
	PlanDiscardItem(
		ctx context.Context,
		in *captureservice.DiscardItemInput,
	) (*captureservice.DiscardItemPlan, error)
	DiscardItem(
		ctx context.Context,
		in *captureservice.DiscardItemInput,
	) (*capture.CaptureBatch, error)
	PlanDiscardBatch(
		ctx context.Context,
		in *captureservice.DiscardBatchInput,
	) (*captureservice.DiscardBatchPlan, error)
	DiscardBatch(
		ctx context.Context,
		in *captureservice.DiscardBatchInput,
	) (*capture.CaptureBatch, error)
}

type captureItemView struct {
	Status         string `json:"status"`
	Pages          int    `json:"pages"`
	TargetType     string `json:"targetType,omitempty"`
	TargetID       string `json:"targetId,omitempty"`
	DocumentTypeID string `json:"documentTypeId,omitempty"`
}

func captureItemRecord(item *capture.CaptureItem) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceCaptureBatch,
		ID:       item.ID,
		Label:    fmt.Sprintf("Scanned document %d", item.Position),
		Version:  pinnedVersion(item.Version),
	}
}

func captureItemBefore(item *capture.CaptureItem) *captureItemView {
	return &captureItemView{Status: string(item.Status), Pages: len(item.PageIDs)}
}

func pulidPointerText(id *pulid.ID) string {
	if id == nil || id.IsNil() {
		return ""
	}

	return id.String()
}

type captureFiling struct {
	entries []captureservice.FileItemInput
}

func captureFilingFrom(params *serviceports.ToolExecuteParams) (*captureFiling, error) {
	raw, ok := params.Params[paramCaptureItems].([]any)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("parameter %q must list at least one document", paramCaptureItems)
	}
	if len(raw) > maxCaptureFilings {
		return nil, fmt.Errorf("parameter %q holds %d documents; file at most %d at once",
			paramCaptureItems, len(raw), maxCaptureFilings)
	}

	filing := &captureFiling{entries: make([]captureservice.FileItemInput, 0, len(raw))}
	seen := make(map[pulid.ID]bool, len(raw))
	for idx, entry := range raw {
		values, isObject := entry.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("%s[%d] must be an object", paramCaptureItems, idx)
		}
		input, err := captureFilingEntry(values, idx)
		if err != nil {
			return nil, err
		}
		if seen[input.ItemID] {
			return nil, fmt.Errorf("%s[%d] names a document already listed", paramCaptureItems,
				idx)
		}
		seen[input.ItemID] = true
		input.TenantInfo = tenantFrom(*params)
		filing.entries = append(filing.entries, *input)
	}

	return filing, nil
}

func captureFilingEntry(values map[string]any, idx int) (*captureservice.FileItemInput, error) {
	prefix := fmt.Sprintf("%s[%d].", paramCaptureItems, idx)
	itemID, err := requirePulid(values, paramCaptureItemID)
	if err != nil {
		return nil, fmt.Errorf("%s%w", prefix, err)
	}
	targetType, err := requireEnum(values, paramCaptureTarget, captureTargetTypes.Values)
	if err != nil {
		return nil, fmt.Errorf("%s%w", prefix, err)
	}
	targetID, err := requirePulid(values, paramCaptureTargetID)
	if err != nil {
		return nil, fmt.Errorf("%s%w", prefix, err)
	}
	docType, err := optionalPulidParam(values, paramCaptureDocType)
	if err != nil {
		return nil, fmt.Errorf("%s%w", prefix, err)
	}

	return &captureservice.FileItemInput{
		ItemID:         itemID,
		TargetType:     string(targetType),
		TargetID:       targetID,
		DocumentTypeID: docType,
	}, nil
}

func (f *captureFiling) versioned(
	ctx context.Context,
	captures captureKeeper,
) ([]captureservice.FileItemInput, error) {
	entries := make([]captureservice.FileItemInput, 0, len(f.entries))
	for idx := range f.entries {
		entry := f.entries[idx]
		item, err := captures.GetItem(ctx, entry.TenantInfo, entry.ItemID)
		if err != nil {
			return nil, err
		}
		entry.Version = item.Version
		entries = append(entries, entry)
	}

	return entries, nil
}

func newFileCaptureItemsTool(captures captureKeeper) serviceports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name:   "file_capture_items",
		recipe: []string{"list_capture_batches", "list_document_types", "file_capture_items"},
		description: "File scanned documents from capture intake onto the shipment, worker, " +
			"tractor, trailer, customer or carrier they belong to, as the document type they " +
			"are. Take the documents and their suggested records from list_capture_batches.",
		resource:    permission.ResourceCaptureBatch,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		rationale: "Files scanned pages as documents on records inside Trenova, as the person " +
			"it acts for; nothing is sent, and a misfiled document can be deleted.",
		properties:  captureFilingProperties(),
		required:    []string{paramCaptureItems},
		searchTerms: []string{"file scan", "scanned paperwork", "capture intake", "file pod"},
	}, receivablePlan[*captureFiling, []*captureservice.FilePlan]{
		request: captureFilingFrom,
		plan: func(
			ctx context.Context,
			filing *captureFiling,
			_ *serviceports.ToolExecuteParams,
		) ([]*captureservice.FilePlan, error) {
			entries, err := filing.versioned(ctx, captures)
			if err != nil {
				return nil, err
			}
			plans := make([]*captureservice.FilePlan, 0, len(entries))
			for idx := range entries {
				plan, planErr := captures.PlanFileItem(ctx, &entries[idx])
				if planErr != nil {
					return nil, planErr
				}
				if plan.Unchanged {
					return nil, errortypes.NewValidationError(
						fmt.Sprintf("%s[%d].%s", paramCaptureItems, idx, paramCaptureItemID),
						errortypes.ErrInvalid, "This document is already filed or being filed")
				}
				plans = append(plans, plan)
			}

			return plans, nil
		},
		refused: func(filing *captureFiling) string {
			return fmt.Sprintf("Would file %s.", countOf(len(filing.entries), scannedDocument))
		},
		render: renderCaptureFiling,
		run: func(
			ctx context.Context,
			filing *captureFiling,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entries, err := filing.versioned(ctx, captures)
			if err != nil {
				return nil, err
			}
			if len(entries) == 1 {
				if _, err = captures.FileItem(ctx, &entries[0]); err != nil {
					return nil, err
				}

				return &agent.ToolExecutionResult{Action: "filed", Kind: scannedDocument,
					Name: countOf(1, scannedDocument)}, nil
			}
			result, err := captures.FileItems(ctx, &captureservice.FileItemsInput{
				TenantInfo: tenantFrom(*params),
				Items:      entries,
			})
			if err != nil {
				return nil, err
			}

			if len(result.Failures) > 0 {
				return nil, errors.New(capturedOutcome(result))
			}

			return &agent.ToolExecutionResult{
				Action: "filed",
				Kind:   scannedDocument,
				Name:   capturedOutcome(result),
			}, nil
		},
	})
}

func captureFilingProperties() map[string]any {
	return map[string]any{
		paramCaptureItems: map[string]any{
			toolschema.KeyType: toolschema.TypeArray,
			toolschema.KeyDescription: "The documents to file, each with the record it " +
				"goes on.",
			toolschema.KeyMinItems: 1,
			toolschema.KeyMaxItems: maxCaptureFilings,
			toolschema.KeyItems: map[string]any{
				toolschema.KeyType: toolschema.TypeObject,
				toolschema.KeyProperties: map[string]any{
					paramCaptureItemID: agenttoolschema.KindID(
						agenttoolschema.IDDescription("The "+scannedDocument, captureSources),
						permission.KindScannedDocument,
					),
					paramCaptureTarget: agenttoolschema.Enum("The kind of record it "+
						"goes on.", captureTargetTypes),
					paramCaptureTargetID: agenttoolschema.KindID("The record it goes on, by id "+
						"from that record's read tool.",
						permission.RecordKind(permission.ResourceShipment),
						permission.RecordKind(permission.ResourceWorker),
						permission.RecordKind(permission.ResourceTractor),
						permission.RecordKind(permission.ResourceTrailer),
						permission.RecordKind(permission.ResourceCustomer),
						permission.RecordKind(permission.ResourceCarrier),
					),
					paramCaptureDocType: agenttoolschema.RecordIDText(
						permission.ResourceDocumentType,
						"What kind of document it is, from list_document_types.",
					),
				},
				toolschema.KeyRequired: []string{
					paramCaptureItemID, paramCaptureTarget, paramCaptureTargetID,
				},
				toolschema.KeyAdditionalProperties: false,
			},
		},
	}
}

func renderCaptureFiling(
	_ *captureFiling,
	plans []*captureservice.FilePlan,
) (*agent.ToolPreview, error) {
	changes := make([]*agent.RecordChange, 0, len(plans))
	for _, plan := range plans {
		change, err := toolpreview.Changed(captureItemRecord(plan.Item),
			captureItemBefore(plan.Item), &captureItemView{
				Status:         string(capture.ItemFiled),
				Pages:          len(plan.Item.PageIDs),
				TargetType:     plan.Target.ResourceType,
				TargetID:       pulidPointerText(plan.Target.ResourceID),
				DocumentTypeID: pulidPointerText(plan.Target.DocumentTypeID),
			}, toolpreview.WithRefs(map[string]permission.Resource{
				paramCaptureTargetID: permission.Resource(plan.Target.ResourceType),
				paramCaptureDocType:  permission.ResourceDocumentType,
			}))
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	return toolpreview.Build(fmt.Sprintf("Would file %s onto their records.",
		countOf(len(plans), scannedDocument)), changes...), nil
}

func capturedOutcome(result *captureservice.FileItemsResult) string {
	summary := countOf(len(result.Filed), scannedDocument) + " filed"
	if len(result.Failures) == 0 {
		return summary
	}
	failures := make([]string, 0, len(result.Failures))
	for _, failure := range result.Failures {
		failures = append(failures, failure.ItemID.String()+": "+failure.Message)
	}

	return summary + "; not filed: " + strings.Join(failures, "; ")
}

func newDiscardCaptureItemTool(captures captureKeeper) serviceports.AgentTool {
	build := func(
		ctx context.Context,
		params *serviceports.ToolExecuteParams,
		id pulid.ID,
	) (*captureservice.DiscardItemInput, error) {
		item, err := captures.GetItem(ctx, tenantFrom(*params), id)
		if err != nil {
			return nil, err
		}

		return &captureservice.DiscardItemInput{
			TenantInfo: tenantFrom(*params),
			ItemID:     id,
			Version:    item.Version,
		}, nil
	}

	return newReceivableTool(&receivableSpec{
		name: "discard_capture_item",
		description: "Throw away one scanned document from capture intake without filing it, " +
			"such as a blank sheet, a duplicate or a stray page. The rest of its stack stays.",
		resource:    permission.ResourceCaptureBatch,
		operation:   permission.OpDelete,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		rationale: "Marks one scanned document not worth filing inside Trenova; nothing is " +
			"sent, and its pages are removed with the stack's other unfiled pages.",
		properties: map[string]any{
			paramCaptureItemIDOne: agenttoolschema.KindID(
				agenttoolschema.IDDescription("The "+scannedDocument, captureSources),
				permission.KindScannedDocument,
			),
		},
		required:    []string{paramCaptureItemIDOne},
		searchTerms: []string{"discard scan", "blank page", "throw away scan"},
	}, receivablePlan[pulid.ID, *captureservice.DiscardItemPlan]{
		request: func(params *serviceports.ToolExecuteParams) (pulid.ID, error) {
			return requirePulid(params.Params, paramCaptureItemIDOne)
		},
		plan: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*captureservice.DiscardItemPlan, error) {
			in, err := build(ctx, params, id)
			if err != nil {
				return nil, err
			}

			return captures.PlanDiscardItem(ctx, in)
		},
		refused: func(pulid.ID) string { return "Would discard a scanned document." },
		render: func(_ pulid.ID, plan *captureservice.DiscardItemPlan) (*agent.ToolPreview, error) {
			after := captureItemBefore(plan.Item)
			after.Status = string(capture.ItemDiscarded)
			change, err := toolpreview.Changed(captureItemRecord(plan.Item),
				captureItemBefore(plan.Item), after)
			if err != nil {
				return nil, err
			}
			change.Operation = agent.PreviewOperationArchive

			return toolpreview.Build(fmt.Sprintf(
				"Would discard %s of %s without filing it.",
				countOf(len(plan.Item.PageIDs), "page"), captureItemRecord(plan.Item).Label,
			), change), nil
		},
		run: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			in, err := build(ctx, params, id)
			if err != nil {
				return nil, err
			}
			if _, err = captures.DiscardItem(ctx, in); err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{Action: "discarded", Kind: scannedDocument,
				IDs: map[string]string{paramCaptureItemIDOne: id.String()}}, nil
		},
	})
}

type captureBatchView struct {
	Status        string `json:"status"`
	OpenDocuments int    `json:"openDocuments"`
	FiledItems    int    `json:"filedDocuments"`
}

func newDiscardCaptureBatchTool(captures captureKeeper) serviceports.AgentTool {
	build := func(
		ctx context.Context,
		params *serviceports.ToolExecuteParams,
		id pulid.ID,
	) (*captureservice.DiscardBatchInput, error) {
		batch, err := captures.GetBatch(ctx, tenantFrom(*params), id)
		if err != nil {
			return nil, err
		}

		return &captureservice.DiscardBatchInput{
			TenantInfo: tenantFrom(*params),
			BatchID:    id,
			Version:    batch.Version,
		}, nil
	}

	return newReceivableTool(&receivableSpec{
		name: "discard_capture_batch",
		description: "Throw away every unfiled document of a scanned stack in capture intake, " +
			"when nothing left in it is worth filing. Documents already filed stay on their " +
			"records.",
		resource:    permission.ResourceCaptureBatch,
		operation:   permission.OpDelete,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		rationale: "Closes a scanned stack inside Trenova and drops its unfiled pages; nothing " +
			"is sent, but nothing brings the pages back, so it is always proposed.",
		properties: map[string]any{
			paramCaptureBatchID: agenttoolschema.RecordID(permission.ResourceCaptureBatch,
				"The scanned stack", captureSources),
		},
		required:    []string{paramCaptureBatchID},
		searchTerms: []string{"discard stack", "clear capture intake", "throw away scans"},
	}, receivablePlan[pulid.ID, *captureservice.DiscardBatchPlan]{
		request: func(params *serviceports.ToolExecuteParams) (pulid.ID, error) {
			return requirePulid(params.Params, paramCaptureBatchID)
		},
		plan: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*captureservice.DiscardBatchPlan, error) {
			in, err := build(ctx, params, id)
			if err != nil {
				return nil, err
			}
			plan, err := captures.PlanDiscardBatch(ctx, in)
			if err != nil {
				return nil, err
			}
			if plan.Unchanged {
				return nil, errortypes.NewValidationError(paramCaptureBatchID,
					errortypes.ErrInvalid, "This stack is already closed")
			}

			return plan, nil
		},
		refused: func(pulid.ID) string { return "Would discard a scanned stack." },
		render:  renderCaptureDiscard,
		run: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			in, err := build(ctx, params, id)
			if err != nil {
				return nil, err
			}
			if _, err = captures.DiscardBatch(ctx, in); err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{Action: "discarded", Kind: "scanned stack",
				IDs: map[string]string{paramCaptureBatchID: id.String()}}, nil
		},
	})
}

func renderCaptureDiscard(
	_ pulid.ID,
	plan *captureservice.DiscardBatchPlan,
) (*agent.ToolPreview, error) {
	status := capture.BatchDiscarded
	if plan.Batch.FiledItemCount > 0 {
		status = capture.BatchFiled
	}
	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceCaptureBatch,
		ID:       plan.Batch.ID,
		Label:    captureBatchLabel(plan.Batch),
		Version:  pinnedVersion(plan.Batch.Version),
	}, &captureBatchView{
		Status:        string(plan.Batch.Status),
		OpenDocuments: len(plan.Open),
		FiledItems:    plan.Batch.FiledItemCount,
	}, &captureBatchView{
		Status:     string(status),
		FiledItems: plan.Batch.FiledItemCount,
	})
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would discard %s of %s; %s already filed stay on their records.",
		countOf(len(plan.Open), "unfiled document"), captureBatchLabel(plan.Batch),
		countOf(plan.Batch.FiledItemCount, "document"),
	), change), nil
}

func captureBatchLabel(batch *capture.CaptureBatch) string {
	switch {
	case batch.JobName != "":
		return batch.JobName
	case batch.SourceName != "":
		return "the stack scanned on " + batch.SourceName
	default:
		return "the scanned stack"
	}
}
