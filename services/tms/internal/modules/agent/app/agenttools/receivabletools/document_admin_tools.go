package receivabletools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramDocumentID = "documentId"

const paramDocumentIDs = "documentIds"

const paramVersionNumber = "versionNumber"

const maxDocumentsPerDelete = 25

const maxDocumentVersion = 10_000

const documentSupplier = "from document_list or document_get_summary"

const documentKind = "document"

type DocumentKeeper interface {
	PlanDelete(
		ctx context.Context,
		req *documentservice.BulkDeleteRequest,
	) (*documentservice.DeletePlan, error)
	Delete(ctx context.Context, req documentports.DeleteDocumentRequest, userID pulid.ID) error
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
		Version:  common.PinnedVersion(doc.Version),
	}
}

func deleteDocumentsRequest(
	params *agentports.ToolExecuteParams,
) (*documentservice.BulkDeleteRequest, error) {
	ids, err := common.RequirePulidSlice(params.Params, paramDocumentIDs, maxDocumentsPerDelete)
	if err != nil {
		return nil, err
	}

	return &documentservice.BulkDeleteRequest{
		IDs:        ids,
		TenantInfo: common.TenantFrom(*params),
		UserID:     params.Actor.UserID,
	}, nil
}

func NewDeleteDocumentsTool(documents DocumentKeeper) agentports.AgentTool {
	return common.NewReceivableTool(&common.ReceivableSpec{
		Name: "document_delete",
		Description: "Delete documents that should not be on file, such as a duplicate " +
			"upload or a scan of the wrong paperwork. Every stored version of each goes with " +
			"it and nothing brings them back, so it is always proposed.",
		Resource:    permission.ResourceDocument,
		Operation:   permission.OpDelete,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		Rationale: "Removes documents and their stored files from Trenova for good; nothing " +
			"is sent, but nothing restores them either.",
		Properties: map[string]any{
			paramDocumentIDs: agenttoolschema.RecordIDs(
				permission.ResourceDocument,
				"The documents to delete, "+documentSupplier+
					". One id is the normal case.",
				maxDocumentsPerDelete,
			),
		},
		Required:    []string{paramDocumentIDs},
		SearchTerms: []string{"delete document", "remove duplicate upload", "wrong file"},
	}, common.ReceivablePlan[*documentservice.BulkDeleteRequest, *documentservice.DeletePlan]{
		Request: deleteDocumentsRequest,
		Plan: func(
			ctx context.Context,
			req *documentservice.BulkDeleteRequest,
			_ *agentports.ToolExecuteParams,
		) (*documentservice.DeletePlan, error) {
			return documents.PlanDelete(ctx, req)
		},
		Refused: func(req *documentservice.BulkDeleteRequest) string {
			return fmt.Sprintf("Would delete %s.", common.CountOf(len(req.IDs), documentKind))
		},
		Render: func(
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
				common.CountOf(
					len(plan.Documents),
					documentKind,
				),
				common.CountOf(plan.FileCount(), "file"),
			), changes...), nil
		},
		Run: func(
			ctx context.Context,
			req *documentservice.BulkDeleteRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if len(req.IDs) == 1 {
				if err := documents.Delete(ctx, documentports.DeleteDocumentRequest{
					ID: req.IDs[0],
					TenantInfo: pagination.TenantInfo{
						OrgID: req.TenantInfo.OrgID,
						BuID:  req.TenantInfo.BuID,
					},
				}, params.Actor.UserID); err != nil {
					return nil, err
				}

				return &agent.ToolExecutionResult{
					Action: common.ActionDeleted,
					Kind:   documentKind,
					Name:   common.CountOf(1, documentKind),
				}, nil
			}
			result, err := documents.BulkDelete(ctx, req)
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: common.ActionDeleted,
				Kind:   documentKind,
				Name:   common.CountOf(result.DeletedCount, "file"),
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

func restoreRequest(params *agentports.ToolExecuteParams) (*documentRestore, error) {
	id, err := common.RequirePulid(params.Params, paramDocumentID)
	if err != nil {
		return nil, err
	}
	version, err := common.RequireIntInRange(
		params.Params,
		paramVersionNumber,
		1,
		maxDocumentVersion,
	)
	if err != nil {
		return nil, err
	}

	return &documentRestore{documentID: id, version: int64(version)}, nil
}

func restoreTarget(
	ctx context.Context,
	documents DocumentKeeper,
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

func NewRestoreDocumentVersionTool(documents DocumentKeeper) agentports.AgentTool {
	return common.NewReceivableTool(&common.ReceivableSpec{
		Name: "document_restore_version",
		Description: "Make an earlier version of a document the current one again, when " +
			"the newest upload was the wrong file. Every version is kept, so a later restore " +
			"can bring the newer one back.",
		Resource:    permission.ResourceDocument,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Changes which stored version of a document Trenova shows; nothing is sent " +
			"or removed.",
		Properties: map[string]any{
			paramDocumentID: agenttoolschema.RecordIDText(permission.ResourceDocument,
				"Any version of the document, "+documentSupplier+". Never guess one."),
			paramVersionNumber: common.IntegerProperty("The version number to make current.", 1,
				maxDocumentVersion),
		},
		Required:    []string{paramDocumentID, paramVersionNumber},
		SearchTerms: []string{"restore document", "previous version", "undo upload"},
		Target: func(params map[string]any) (agentports.ToolTarget, bool) {
			return common.TargetOf(params, paramDocumentID, permission.ResourceDocument)
		},
	}, common.ReceivablePlan[*documentRestore, *restoreDocumentPlan]{
		Request: restoreRequest,
		Plan: func(
			ctx context.Context,
			req *documentRestore,
			params *agentports.ToolExecuteParams,
		) (*restoreDocumentPlan, error) {
			tenant := common.TenantFrom(*params)
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
		Refused: func(req *documentRestore) string {
			return fmt.Sprintf("Would make version %d of the document current.", req.version)
		},
		Render: func(req *documentRestore, planned *restoreDocumentPlan) (*agent.ToolPreview, error) {
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
		Run: func(
			ctx context.Context,
			req *documentRestore,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			tenant := common.TenantFrom(*params)
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
