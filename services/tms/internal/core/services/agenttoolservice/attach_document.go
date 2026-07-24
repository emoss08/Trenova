package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

const documentResourceTypeShipment = "shipment"

type attachDocumentTool struct {
	uploads serviceports.DocumentUploadService
}

func newAttachDocumentTool(uploads serviceports.DocumentUploadService) serviceports.AgentTool {
	return &attachDocumentTool{uploads: uploads}
}

func (t *attachDocumentTool) Name() string { return "attach_document_to_bqi" }

func (t *attachDocumentTool) Description() string {
	return "Attach a document to the shipment backing a billing queue item by starting an upload session."
}

func (t *attachDocumentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId":     map[string]any{"type": "string"},
			"documentTypeId": map[string]any{"type": "string"},
			"fileName":       map[string]any{"type": "string"},
			"contentType":    map[string]any{"type": "string"},
			"fileSize":       map[string]any{"type": "integer"},
			"description":    map[string]any{"type": "string"},
		},
		"required":             []string{"shipmentId", "documentTypeId", "fileName", "contentType"},
		"additionalProperties": false,
	}
}

func (t *attachDocumentTool) Reversible() bool { return true }

func (t *attachDocumentTool) PermissionResource() permission.Resource {
	return permission.ResourceDocument
}

func (t *attachDocumentTool) PermissionOperation() permission.Operation {
	return permission.OpCreate
}

func (t *attachDocumentTool) RequiresIdempotencyKey() bool { return false }

func (t *attachDocumentTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierPropose
}

func (t *attachDocumentTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	shipmentID, err := requireString(params.Params, "shipmentId")
	if err != nil {
		return err
	}

	documentTypeID, err := requireString(params.Params, "documentTypeId")
	if err != nil {
		return err
	}

	fileName, err := requireString(params.Params, "fileName")
	if err != nil {
		return err
	}

	contentType, err := requireString(params.Params, "contentType")
	if err != nil {
		return err
	}

	_, err = t.uploads.CreateSession(ctx, &serviceports.CreateSessionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
		Actor:          *params.Actor,
		ResourceID:     shipmentID,
		ResourceType:   documentResourceTypeShipment,
		FileName:       fileName,
		FileSize:       optionalInt64(params.Params, "fileSize"),
		ContentType:    contentType,
		Description:    optionalString(params.Params, "description"),
		DocumentTypeID: documentTypeID,
	})

	return err
}
