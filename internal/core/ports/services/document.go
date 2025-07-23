// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
)

// UploadDocumentRequest contains the details needed to upload a document
type UploadDocumentRequest struct {
	OrganizationID  pulid.ID            `json:"organizationId"`
	BusinessUnitID  pulid.ID            `json:"businessUnitId"`
	UploadedByID    pulid.ID            `json:"uploadedById"`
	ResourceID      pulid.ID            `json:"resourceId"`
	ResourceType    permission.Resource `json:"resourceType"`
	DocumentTypeID  pulid.ID            `json:"documentTypeId"`
	File            []byte              `json:"file"`
	FileName        string              `json:"fileName"`
	OriginalName    string              `json:"originalName"`
	Description     string              `json:"description"`
	Tags            []string            `json:"tags"`
	ExpirationDate  *int64              `json:"expirationDate"`
	Status          document.Status     `json:"status"`
	RequireApproval bool                `json:"requireApproval"`
}

func (r *UploadDocumentRequest) Validate(ctx context.Context) error {
	me := errors.NewMultiError()

	err := validation.ValidateStructWithContext(
		ctx,
		r,
		validation.Field(
			&r.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&r.BusinessUnitID,
			validation.Required.Error("Business Unit ID is required"),
		),
		validation.Field(&r.UploadedByID, validation.Required.Error("Uploaded By ID is required")),
		validation.Field(&r.ResourceID, validation.Required.Error("Resource ID is required")),
		validation.Field(&r.ResourceType, validation.Required.Error("Resource Type is required")),
		validation.Field(
			&r.DocumentTypeID,
			validation.Required.Error("Document Type ID is required"),
		),
		validation.Field(&r.File, validation.Required.Error("File is required")),
		validation.Field(&r.FileName, validation.Required.Error("File Name is required")),
		validation.Field(&r.OriginalName, validation.Required.Error("Original Name is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, me)
		}
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

// UploadDocumentResponse contains the result of a document upload
type UploadDocumentResponse struct {
	Document  *document.Document `json:"document"`
	Location  string             `json:"location"`
	Checksum  string             `json:"checksum"`
	Size      int64              `json:"size"`
	VersionID string             `json:"versionId"`
}

type DeleteDocumentRequest struct {
	OrgID        pulid.ID `json:"orgId"`
	BuID         pulid.ID `json:"buId"`
	DocID        pulid.ID `json:"docId"`
	UploadedByID pulid.ID `json:"uploadedById"`
}

// FailedUpload contains information about a failed document upload
type FailedUpload struct {
	FileName string `json:"fileName"`
	Error    error  `json:"error"`
}

// DocumentService defines the interface for document management operations
type DocumentService interface {
	UploadDocument(ctx context.Context, req *UploadDocumentRequest) (*UploadDocumentResponse, error)
	GetDocumentCountByResource(
		ctx context.Context,
		req ports.TenantOptions,
	) ([]*repositories.GetDocumentCountByResourceResponse, error)
	GetResourceSubFolders(
		ctx context.Context,
		req repositories.GetResourceSubFoldersRequest,
	) ([]*repositories.GetResourceSubFoldersResponse, error)
	GetDocumentsByResourceID(
		ctx context.Context,
		req *repositories.GetDocumentsByResourceIDRequest,
	) (*ports.ListResult[*document.Document], error)
	DeleteDocument(ctx context.Context, req *DeleteDocumentRequest) error
}
