package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramDocumentID       = "documentId"
	paramDocumentIDs      = "documentIds"
	paramVersionNumber    = "versionNumber"
	maxDocumentsPerDelete = 25
	maxDocumentVersion    = 10_000
	documentSupplier      = "from search_documents or get_document_summary"
	documentKind          = "document"
)

type documentKeeper interface {
	PlanDelete(
		ctx context.Context,
		req *documentservice.BulkDeleteRequest,
	) (*documentservice.DeletePlan, error)
	Delete(ctx context.Context, req repositories.DeleteDocumentRequest, userID pulid.ID) error
	BulkDelete(
		ctx context.Context,
		req *documentservice.BulkDeleteRequest,
	) (*documentservice.BulkDeleteResult, error)
	ListVersions(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) ([]*document.Document, error)
	PlanRestoreVersion(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*documentservice.RestorePlan, error)
	RestoreVersion(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
		userID pulid.ID,
	) (*document.Document, error)
}

type documentView struct {
	Name          string `json:"name"`
	FileType      string `json:"fileType"`
	ResourceType  string `json:"resourceType"`
	VersionNumber int64  `json:"versionNumber"`
	Current       bool   `json:"current"`
}

func documentViewOf(doc *document.Document) *documentView {
	return &documentView{
		Name:          doc.OriginalName,
		FileType:      doc.FileType,
		ResourceType:  doc.ResourceType,
		VersionNumber: doc.VersionNumber,
		Current:       doc.IsCurrentVersion,
	}
}

func documentRecord(doc *document.Document) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceDocument,
		ID:       doc.ID,
		Label:    doc.OriginalName,
		Version:  pinnedVersion(doc.Version),
	}
}

func deleteDocumentsRequest(
	params *serviceports.ToolExecuteParams,
) (*documentservice.BulkDeleteRequest, error) {
	ids, err := requirePulidSlice(params.Params, paramDocumentIDs, maxDocumentsPerDelete)
	if err != nil {
		return nil, err
	}

	return &documentservice.BulkDeleteRequest{
		IDs:        ids,
		TenantInfo: tenantFrom(*params),
		UserID:     params.Actor.UserID,
	}, nil
}

func newDeleteDocumentsTool(documents documentKeeper) serviceports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name: "delete_documents",
		description: "Delete documents that should not be on file, such as a duplicate " +
			"upload or a scan of the wrong paperwork. Every stored version of each goes with " +
			"it and nothing brings them back, so it is always proposed.",
		resource:    permission.ResourceDocument,
		operation:   permission.OpDelete,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		rationale: "Removes documents and their stored files from Trenova for good; nothing " +
			"is sent, but nothing restores them either.",
		properties: map[string]any{
			paramDocumentIDs: agenttoolschema.RecordIDs(
				permission.ResourceDocument,
				"The documents to delete, "+documentSupplier+
					". One id is the normal case.",
				maxDocumentsPerDelete,
			),
		},
		required:    []string{paramDocumentIDs},
		searchTerms: []string{"delete document", "remove duplicate upload", "wrong file"},
	}, receivablePlan[*documentservice.BulkDeleteRequest, *documentservice.DeletePlan]{
		request: deleteDocumentsRequest,
		plan: func(
			ctx context.Context,
			req *documentservice.BulkDeleteRequest,
			_ *serviceports.ToolExecuteParams,
		) (*documentservice.DeletePlan, error) {
			return documents.PlanDelete(ctx, req)
		},
		refused: func(req *documentservice.BulkDeleteRequest) string {
			return fmt.Sprintf("Would delete %s.", countOf(len(req.IDs), documentKind))
		},
		render: func(
			_ *documentservice.BulkDeleteRequest,
			plan *documentservice.DeletePlan,
		) (*agent.ToolPreview, error) {
			changes := make([]*agent.RecordChange, 0, len(plan.Documents))
			for _, doc := range plan.Documents {
				change, err := toolpreview.Delete(documentRecord(doc), documentViewOf(doc))
				if err != nil {
					return nil, err
				}
				changes = append(changes, change)
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would delete %s and %s stored with them; nothing brings them back.",
				countOf(len(plan.Documents), documentKind), countOf(plan.FileCount(), "file"),
			), changes...), nil
		},
		run: func(
			ctx context.Context,
			req *documentservice.BulkDeleteRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if len(req.IDs) == 1 {
				if err := documents.Delete(ctx, repositories.DeleteDocumentRequest{
					ID: req.IDs[0],
					TenantInfo: pagination.TenantInfo{
						OrgID: req.TenantInfo.OrgID,
						BuID:  req.TenantInfo.BuID,
					},
				}, params.Actor.UserID); err != nil {
					return nil, err
				}

				return &agent.ToolExecutionResult{
					Action: actionDeleted,
					Kind:   documentKind,
					Name:   countOf(1, documentKind),
				}, nil
			}
			result, err := documents.BulkDelete(ctx, req)
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: actionDeleted,
				Kind:   documentKind,
				Name:   countOf(result.DeletedCount, "file"),
			}, nil
		},
	})
}

type documentRestore struct {
	documentID pulid.ID
	version    int64
}

type restoreDocumentPlan struct {
	plan *documentservice.RestorePlan
}

func restoreRequest(params *serviceports.ToolExecuteParams) (*documentRestore, error) {
	id, err := requirePulid(params.Params, paramDocumentID)
	if err != nil {
		return nil, err
	}
	version, err := requireIntInRange(params.Params, paramVersionNumber, 1, maxDocumentVersion)
	if err != nil {
		return nil, err
	}

	return &documentRestore{documentID: id, version: int64(version)}, nil
}

func restoreTarget(
	ctx context.Context,
	documents documentKeeper,
	req *documentRestore,
	tenant pagination.TenantInfo,
) (pulid.ID, error) {
	versions, err := documents.ListVersions(ctx, req.documentID, tenant)
	if err != nil {
		return pulid.Nil, err
	}
	for _, version := range versions {
		if version.VersionNumber == req.version {
			return version.ID, nil
		}
	}

	return pulid.Nil, errortypes.NewValidationError(paramVersionNumber, errortypes.ErrInvalid,
		"This document has no version {0}", req.version)
}

func newRestoreDocumentVersionTool(documents documentKeeper) serviceports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name: "restore_document_version",
		description: "Make an earlier version of a document the current one again, when " +
			"the newest upload was the wrong file. Every version is kept, so a later restore " +
			"can bring the newer one back.",
		resource:    permission.ResourceDocument,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Changes which stored version of a document Trenova shows; nothing is sent " +
			"or removed.",
		properties: map[string]any{
			paramDocumentID: agenttoolschema.RecordIDText(permission.ResourceDocument,
				"Any version of the document, "+documentSupplier+". Never guess one."),
			paramVersionNumber: integerProperty("The version number to make current.", 1,
				maxDocumentVersion),
		},
		required:    []string{paramDocumentID, paramVersionNumber},
		searchTerms: []string{"restore document", "previous version", "undo upload"},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramDocumentID, permission.ResourceDocument)
		},
	}, receivablePlan[*documentRestore, *restoreDocumentPlan]{
		request: restoreRequest,
		plan: func(
			ctx context.Context,
			req *documentRestore,
			params *serviceports.ToolExecuteParams,
		) (*restoreDocumentPlan, error) {
			tenant := tenantFrom(*params)
			targetID, err := restoreTarget(ctx, documents, req, tenant)
			if err != nil {
				return nil, err
			}
			plan, err := documents.PlanRestoreVersion(ctx, targetID, tenant)
			if err != nil {
				return nil, err
			}
			if plan.Unchanged {
				return nil, errortypes.NewValidationError(paramVersionNumber,
					errortypes.ErrInvalid, "Version {0} is already the current one", req.version)
			}

			return &restoreDocumentPlan{plan: plan}, nil
		},
		refused: func(req *documentRestore) string {
			return fmt.Sprintf("Would make version %d of the document current.", req.version)
		},
		render: func(req *documentRestore, planned *restoreDocumentPlan) (*agent.ToolPreview, error) {
			plan := planned.plan
			before := &documentView{}
			if plan.Current != nil {
				before = documentViewOf(plan.Current)
			}
			after := documentViewOf(plan.Target)
			after.Current = true
			change, err := toolpreview.Changed(documentRecord(plan.Target), before, after)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would make version %d of %s the current one; the newer version is kept.",
				req.version, plan.Target.OriginalName,
			), change), nil
		},
		run: func(
			ctx context.Context,
			req *documentRestore,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			tenant := tenantFrom(*params)
			targetID, err := restoreTarget(ctx, documents, req, tenant)
			if err != nil {
				return nil, err
			}
			restored, err := documents.RestoreVersion(ctx, targetID,
				pagination.TenantInfo{OrgID: tenant.OrgID, BuID: tenant.BuID}, params.Actor.UserID)
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: "restored",
				Kind:   documentKind,
				Name:   restored.OriginalName,
				IDs:    map[string]string{paramDocumentID: restored.ID.String()},
			}, nil
		},
	})
}
