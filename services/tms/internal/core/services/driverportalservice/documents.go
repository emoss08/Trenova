package driverportalservice

import (
	"context"
	"mime/multipart"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	shipmentResourceType = "shipment"
	workerResourceType   = "worker"
)

type PortalDocument struct {
	ID               pulid.ID `json:"id"`
	FileName         string   `json:"fileName"`
	FileSize         int64    `json:"fileSize"`
	Status           string   `json:"status"`
	DocumentTypeName string   `json:"documentTypeName"`
	CreatedAt        int64    `json:"createdAt"`
}

type PortalDocumentType struct {
	ID    pulid.ID `json:"id"`
	Code  string   `json:"code"`
	Name  string   `json:"name"`
	Color string   `json:"color"`
}

func (s *Service) ShipmentDocumentTypes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*PortalDocumentType, error) {
	return s.documentTypesByCategory(ctx, tenantInfo, documenttype.CategoryShipment)
}

func (s *Service) WorkerDocumentTypes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*PortalDocumentType, error) {
	return s.documentTypesByCategory(ctx, tenantInfo, documenttype.CategoryWorker)
}

func (s *Service) documentTypesByCategory(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	category documenttype.DocumentCategory,
) ([]*PortalDocumentType, error) {
	if _, err := s.ResolveWorker(ctx, tenantInfo); err != nil {
		return nil, err
	}

	result, err := s.documentTypeRepo.List(ctx, &repositories.ListDocumentTypesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantInfo,
			Pagination: pagination.Info{Limit: 100},
		},
	})
	if err != nil {
		return nil, err
	}

	views := make([]*PortalDocumentType, 0, len(result.Items))
	for _, item := range result.Items {
		if item.DocumentCategory != category {
			continue
		}
		views = append(views, &PortalDocumentType{
			ID:    item.ID,
			Code:  item.Code,
			Name:  item.Name,
			Color: item.Color,
		})
	}
	return views, nil
}

// MyProfileDocuments lists the driver's own qualification-file documents
// (license, medical card, and other worker-scoped records).
func (s *Service) MyProfileDocuments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*PortalDocument, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	documents, err := s.documentService.GetByResource(
		ctx,
		&repositories.GetDocumentsByResourceRequest{
			TenantInfo:          tenantInfo,
			ResourceID:          wrk.ID.String(),
			ResourceType:        workerResourceType,
			IncludeDocumentType: true,
		},
	)
	if err != nil {
		return nil, err
	}

	views := make([]*PortalDocument, 0, len(documents))
	for _, item := range documents {
		if item.Status == document.StatusArchived {
			continue
		}
		views = append(views, portalDocumentView(item))
	}
	return views, nil
}

// UploadMyProfileDocument accepts a qualification-file upload (renewed CDL,
// medical certificate, ...) into the worker's document set, where the
// carrier's compliance team reviews it alongside back-office uploads.
func (s *Service) UploadMyProfileDocument(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	file *multipart.FileHeader,
	documentTypeID string,
	actor *serviceports.RequestActor,
) (*PortalDocument, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if _, err = s.requireFeature(ctx, tenantInfo,
		func(control *tenant.DashControl) bool { return control.AllowProfileDocumentUpload },
		"Your carrier collects qualification documents outside Dash — see your fleet manager.",
	); err != nil {
		return nil, err
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Document upload requires an authenticated user",
		)
	}

	result, err := s.documentService.Upload(ctx, &documentservice.UploadRequest{
		TenantInfo:     tenantInfo,
		Actor:          *actor,
		File:           file,
		ResourceID:     wrk.ID.String(),
		ResourceType:   workerResourceType,
		DocumentTypeID: documentTypeID,
	})
	if err != nil {
		return nil, err
	}
	return portalDocumentView(result.Document), nil
}

func (s *Service) MyLoadDocuments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*PortalDocument, error) {
	if err := s.requireAssignedShipment(ctx, tenantInfo, shipmentID); err != nil {
		return nil, err
	}

	documents, err := s.documentService.GetByResource(
		ctx,
		&repositories.GetDocumentsByResourceRequest{
			TenantInfo:          tenantInfo,
			ResourceID:          shipmentID.String(),
			ResourceType:        shipmentResourceType,
			IncludeDocumentType: true,
		},
	)
	if err != nil {
		return nil, err
	}

	views := make([]*PortalDocument, 0, len(documents))
	for _, item := range documents {
		if item.Status == document.StatusArchived {
			continue
		}
		views = append(views, portalDocumentView(item))
	}
	return views, nil
}

func (s *Service) UploadMyLoadDocument(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	file *multipart.FileHeader,
	documentTypeID string,
	actor *serviceports.RequestActor,
) (*PortalDocument, error) {
	if err := s.requireAssignedShipment(ctx, tenantInfo, shipmentID); err != nil {
		return nil, err
	}
	if _, err := s.requireFeature(ctx, tenantInfo,
		func(control *tenant.DashControl) bool { return control.AllowLoadDocumentUpload },
		"Your carrier collects paperwork outside Dash — turn in documents the way you do today.",
	); err != nil {
		return nil, err
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Document upload requires an authenticated user",
		)
	}

	result, err := s.documentService.Upload(ctx, &documentservice.UploadRequest{
		TenantInfo:     tenantInfo,
		Actor:          *actor,
		File:           file,
		ResourceID:     shipmentID.String(),
		ResourceType:   shipmentResourceType,
		DocumentTypeID: documentTypeID,
	})
	if err != nil {
		return nil, err
	}
	return portalDocumentView(result.Document), nil
}

func (s *Service) requireAssignedShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) error {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return err
	}
	assigned, err := s.portalRepo.WorkerAssignedToShipment(ctx, tenantInfo, wrk.ID, shipmentID)
	if err != nil {
		return err
	}
	if !assigned {
		return errortypes.NewNotFoundError("Load not found")
	}
	return nil
}

func portalDocumentView(item *document.Document) *PortalDocument {
	if item == nil {
		return nil
	}
	view := &PortalDocument{
		ID:        item.ID,
		FileName:  item.OriginalName,
		FileSize:  item.FileSize,
		Status:    string(item.Status),
		CreatedAt: item.CreatedAt,
	}
	if view.FileName == "" {
		view.FileName = item.FileName
	}
	if item.DocumentType != nil {
		view.DocumentTypeName = item.DocumentType.Name
	}
	return view
}
