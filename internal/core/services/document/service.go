package document

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/billing"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/infrastructure/storage/minio"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
)

// ServiceParams defines dependencies required for initializing the DocumentService.
// This includes database connection, logger, permission service, audit service,
// config manager, document repository, and file service.
type ServiceParams struct {
	fx.In

	DB             db.Connection
	Client         *minio.Client
	Logger         *logger.Logger
	PermService    services.PermissionService
	AuditService   services.AuditService
	ConfigM        *config.Manager
	DocRepo        repositories.DocumentRepository
	FileService    services.FileService
	OrgRepo        repositories.OrganizationRepository
	DocTypeRepo    repositories.DocumentTypeRepository
	PreviewService services.PreviewService
}

// service implements the DocumentService interface
// and provides methods to manage documents, including CRUD operations,
// status updates, duplication, and cancellation.
type service struct {
	l              *zerolog.Logger
	db             db.Connection
	client         *minio.Client
	endpoint       string
	docRepo        repositories.DocumentRepository
	fileService    services.FileService
	orgRepo        repositories.OrganizationRepository
	docTypeRepo    repositories.DocumentTypeRepository
	ps             services.PermissionService
	as             services.AuditService
	previewService services.PreviewService
}

// NewService initializes a new DocumentService instance with the provided dependencies.
//
// Parameters:
//   - p: ServiceParams containing dependencies.
//
// Returns:
//   - services.DocumentService: A ready-to-use DocumentService instance.
func NewService(p ServiceParams) services.DocumentService {
	log := p.Logger.With().
		Str("service", "document").
		Logger()

	return &service{
		l:              &log,
		db:             p.DB,
		client:         p.Client,
		endpoint:       p.ConfigM.Minio().Endpoint,
		docRepo:        p.DocRepo,
		fileService:    p.FileService,
		orgRepo:        p.OrgRepo,
		docTypeRepo:    p.DocTypeRepo,
		ps:             p.PermService,
		as:             p.AuditService,
		previewService: p.PreviewService,
	}
}

// GetDocumentCountByResource gets the total number of documents per resource
func (s *service) GetDocumentCountByResource(ctx context.Context, req ports.TenantOptions) ([]*repositories.GetDocumentCountByResourceResponse, error) {
	log := s.l.With().
		Str("operation", "GetDocumentCountByResource").
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.UserID,
			Resource:       permission.ResourceDocument,
			Action:         permission.ActionRead,
			BusinessUnitID: req.BuID,
			OrganizationID: req.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read documents")
	}

	return s.docRepo.GetDocumentCountByResource(ctx, &req)
}

func (s *service) GetResourceSubFolders(ctx context.Context, req repositories.GetResourceSubFoldersRequest) ([]*repositories.GetResourceSubFoldersResponse, error) {
	log := s.l.With().
		Str("operation", "GetResourceSubFolders").
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.UserID,
			Resource:       permission.ResourceDocument,
			Action:         permission.ActionRead,
			BusinessUnitID: req.BuID,
			OrganizationID: req.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read documents")
	}

	return s.docRepo.GetResourceSubFolders(ctx, req)
}

func (s *service) GetDocumentsByResourceID(ctx context.Context, req *repositories.GetDocumentsByResourceIDRequest) (*ports.ListResult[*document.Document], error) {
	log := s.l.With().
		Str("operation", "GetDocumentsByResourceID").
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.UserID,
			Resource:       permission.ResourceDocument,
			Action:         permission.ActionRead,
			BusinessUnitID: req.BuID,
			OrganizationID: req.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read documents")
	}

	// * Get the organization bucket name
	bucketName, err := s.orgRepo.GetOrganizationBucketName(ctx, req.OrgID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get organization bucket name")
		return nil, eris.Wrap(err, "get organization bucket name")
	}

	docs, err := s.docRepo.GetDocumentsByResourceID(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get documents by resource ID")
		return nil, err
	}

	// * Use goroutines and errgroup to parallelize the presigned URL generation
	g, ctx := errgroup.WithContext(ctx)

	var mu sync.Mutex

	for i := range docs.Items {
		idx := i // Use different name to avoid shadowing
		doc := docs.Items[idx]

		g.Go(func() error {
			// Generate presigned URL for the document
			presignedURL, iErr := s.fileService.GetFileURL(ctx, bucketName, doc.StoragePath, time.Hour*24)
			if iErr != nil {
				return iErr
			}

			mu.Lock()
			doc.PresignedURL = presignedURL

			// Generate presigned URL for the preview if we have one
			if doc.PreviewStoragePath != "" {
				previewURL, pErr := s.previewService.GetPreviewURL(ctx, &services.GetPreviewURLRequest{
					PreviewPath: doc.PreviewStoragePath,
					BucketName:  bucketName,
					OrgID:       req.OrgID,
					ExpiryTime:  time.Hour * 24,
				})
				if pErr == nil {
					doc.PreviewURL = previewURL
				}
			}
			mu.Unlock()

			return nil
		})
	}

	if wErr := g.Wait(); wErr != nil {
		log.Error().Err(wErr).Msg("failed to generate presigned URLs")
		return nil, wErr
	}

	return docs, nil
}

// UploadDocument uploads a single document and stores its metadata
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: The request containing document details.
//
// Returns:
//   - *services.UploadDocumentResponse: The response containing the uploaded document.
//   - error: An error if the operation fails.
func (s *service) UploadDocument(ctx context.Context, req *services.UploadDocumentRequest) (*services.UploadDocumentResponse, error) {
	log := s.l.With().
		Str("operation", "UploadDocument").
		Logger()

	// Check permissions
	if err := s.checkUploadPermissions(ctx, req); err != nil {
		return nil, err
	}

	// // * Validate the request and file size
	// if err := req.Validate(ctx); err != nil {
	// 	return nil, err
	// }

	// Get bucket name
	bucketName, err := s.orgRepo.GetOrganizationBucketName(ctx, req.OrganizationID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get organization bucket name")
		return nil, eris.Wrap(err, "get organization bucket name")
	}

	// Validate request
	if err = req.Validate(ctx); err != nil {
		return nil, err
	}

	// Fetch document type and prepare file storage
	docType, objectKey, err := s.prepareDocumentStorage(ctx, req)
	if err != nil {
		return nil, err
	}

	// Upload file to storage
	fileUploadResp, previewPath, err := s.uploadDocumentFile(ctx, req, bucketName, objectKey, docType)
	if err != nil {
		return nil, err
	}

	// Create and save document record
	savedDoc, err := s.createDocumentRecord(ctx, req, objectKey, bucketName, previewPath)
	if err != nil {
		return nil, err
	}

	// Log the action
	if err = s.logDocumentCreation(savedDoc, req); err != nil {
		log.Error().Err(err).Msg("failed to log document creation")
	}

	return &services.UploadDocumentResponse{
		Document:  savedDoc,
		Location:  fileUploadResp.Location,
		Checksum:  fileUploadResp.Checksum,
		Size:      fileUploadResp.Size,
		VersionID: fileUploadResp.Etag, // * MinIO uses ETag for version ID
	}, nil
}

// checkUploadPermissions checks if the user has permission to upload documents
func (s *service) checkUploadPermissions(ctx context.Context, req *services.UploadDocumentRequest) error {
	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.UploadedByID,
			Resource:       permission.ResourceDocument,
			Action:         permission.ActionCreate,
			BusinessUnitID: req.BusinessUnitID,
			OrganizationID: req.OrganizationID,
		},
	})
	if err != nil {
		s.l.Error().Err(err).Msg("failed to check permissions")
		return err
	}

	if !result.Allowed {
		return errors.NewAuthorizationError("You do not have permission to create documents")
	}

	return nil
}

// prepareDocumentStorage fetches document type and generates a storage path
func (s *service) prepareDocumentStorage(ctx context.Context, req *services.UploadDocumentRequest) (*billing.DocumentType, string, error) {
	docType, err := s.docTypeRepo.GetByID(ctx, repositories.GetDocumentTypeByIDRequest{
		ID:    req.DocumentTypeID,
		OrgID: req.OrganizationID,
		BuID:  req.BusinessUnitID,
	})
	if err != nil {
		s.l.Error().Err(err).Msg("failed to get document type by ID")
		return nil, "", eris.Wrap(err, "get document type by ID")
	}

	objectKey := s.generateObjectPath(req, docType.Name)
	return docType, objectKey, nil
}

// uploadDocumentFile handles uploading the file to storage
func (s *service) uploadDocumentFile(ctx context.Context, req *services.UploadDocumentRequest, bucketName, objectKey string, docType *billing.DocumentType) (*services.SaveFileResponse, string, error) {
	log := s.l.With().
		Str("operation", "uploadDocumentFile").
		Str("fileName", req.FileName).
		Logger()

	fileReq := &services.SaveFileRequest{
		OrgID:          req.OrganizationID.String(),
		BucketName:     bucketName,
		UserID:         req.UploadedByID.String(),
		FileName:       objectKey,
		File:           req.File,
		FileType:       s.determineFileType(req.FileName),
		Classification: s.mapDocTypeToClassification(req.DocumentTypeID),
		Category:       s.mapResourceTypeToCategory(req.ResourceType),
		Tags: map[string]string{
			"resource_id":   req.ResourceID.String(),
			"resource_type": string(req.ResourceType),
			"doc_type":      docType.Name,
		},
		Metadata: map[string][]string{
			"description": {req.Description},
		},
	}

	// Add tags to metadata
	if len(req.Tags) > 0 {
		fileReq.Metadata["tags"] = req.Tags
	}

	// Upload file
	fileUploadResp, err := s.fileService.SaveFileVersion(ctx, fileReq)
	if err != nil {
		log.Error().Err(err).Msg("failed to save file version")
		return nil, "", err
	}

	// * Generate a preview image if the file is a PDF
	previewResp, err := s.previewService.GeneratePreview(ctx, &services.GeneratePreviewRequest{
		File:         req.File,
		FileName:     req.FileName,
		OrgID:        req.OrganizationID,
		UserID:       req.UploadedByID.String(),
		ResourceID:   req.ResourceID,
		ResourceType: req.ResourceType,
		BucketName:   bucketName,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to generate preview")
		return nil, "", err
	}

	return fileUploadResp, previewResp.PreviewPath, nil
}

// createDocumentRecord creates and saves the document record in the database
func (s *service) createDocumentRecord(ctx context.Context, req *services.UploadDocumentRequest, objectKey, bucketName, previewPath string) (*document.Document, error) {
	// Determine document status
	docStatus := req.Status
	if docStatus == "" {
		if req.RequireApproval {
			docStatus = document.StatusPendingApproval
		} else {
			docStatus = document.StatusActive
		}
	}

	// Create document record
	doc := &document.Document{
		OrganizationID:     req.OrganizationID,
		BusinessUnitID:     req.BusinessUnitID,
		FileName:           filepath.Base(objectKey),
		OriginalName:       req.OriginalName,
		FileSize:           int64(len(req.File)),
		FileType:           filepath.Ext(req.FileName),
		StoragePath:        objectKey,
		DocumentTypeID:     req.DocumentTypeID,
		Status:             docStatus,
		ResourceID:         req.ResourceID,
		ResourceType:       req.ResourceType,
		ExpirationDate:     req.ExpirationDate,
		Tags:               req.Tags,
		UploadedByID:       req.UploadedByID,
		PreviewStoragePath: previewPath,
	}

	// Save document to database
	savedDoc, err := s.docRepo.Create(ctx, doc)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to create document")
		// Try to clean up the file if we can't save the metadata
		_ = s.fileService.DeleteFile(ctx, bucketName, objectKey)
		return nil, eris.Wrap(err, "create document")
	}

	return savedDoc, nil
}

// logDocumentCreation logs the document creation action
func (s *service) logDocumentCreation(doc *document.Document, req *services.UploadDocumentRequest) error {
	return s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDocument,
			ResourceID:     doc.ID.String(),
			Action:         permission.ActionCreate,
			UserID:         req.UploadedByID,
			CurrentState:   jsonutils.MustToJSON(doc),
			OrganizationID: doc.OrganizationID,
			BusinessUnitID: doc.BusinessUnitID,
		},
		audit.WithComment("Document created"),
	)
}

// DeleteDocument deletes a document from the database and file storage
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: The request containing document details.
//
// Returns:
//   - error: An error if the operation fails.
func (s *service) DeleteDocument(ctx context.Context, req *services.DeleteDocumentRequest) error {
	log := s.l.With().
		Str("operation", "DeleteDocument").
		Str("docID", req.DocID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
			UserID:         req.UploadedByID,
			Resource:       permission.ResourceDocument,
			Action:         permission.ActionDelete,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return err
	}

	if !result.Allowed {
		return errors.NewAuthorizationError("You do not have permission to delete documents")
	}

	// * Get the document from the database
	doc, err := s.docRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:    req.DocID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get document by ID")
		return err
	}

	// * Get the organization bucket name
	bucketName, err := s.orgRepo.GetOrganizationBucketName(ctx, req.OrgID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get organization bucket name")
		return eris.Wrap(err, "get organization bucket name")
	}

	// * Remove the document from file storage
	err = s.fileService.DeleteFile(ctx, bucketName, doc.StoragePath)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete file from storage")
		return err
	}

	// * Delete the preview from file storage
	if doc.PreviewStoragePath != "" {
		if err = s.previewService.DeletePreview(ctx, &services.DeletePreviewRequest{
			PreviewPath: doc.PreviewStoragePath,
			BucketName:  bucketName,
			OrgID:       req.OrgID,
		}); err != nil {
			log.Error().Err(err).Msg("failed to delete preview from storage")
			// ! Continue with the deletion even if preview deletion fails
		}
	}

	// * Delete the document from the database
	err = s.docRepo.Delete(ctx, repositories.DeleteDocumentRequest{
		ID:    req.DocID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to delete document")
		return err
	}

	// Log the deletion
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDocument,
			ResourceID:     doc.ID.String(),
			Action:         permission.ActionDelete,
			UserID:         req.UploadedByID,
			PreviousState:  jsonutils.MustToJSON(doc),
			OrganizationID: doc.OrganizationID,
			BusinessUnitID: doc.BusinessUnitID,
		},
		audit.WithComment("Document deleted"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log document deletion")
	}

	return nil
}

func (s *service) generateObjectPath(req *services.UploadDocumentRequest, docTypeName string) string {
	// Format: {resourceType}/{resourceID}/{documentType}/{timestamp}_{filename}
	now := time.Now()
	timestamp := now.Format("20060102150405")
	sanitizedFileName := strings.ReplaceAll(req.FileName, " ", "_")

	return fmt.Sprintf("%s/%s/%s_%s",
		req.ResourceID,
		docTypeName,
		timestamp,
		sanitizedFileName)
}

func (s *service) determineFileType(filename string) services.FileType {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".tiff":
		return services.ImageFile
	case ".pdf":
		return services.PDFFile
	case ".doc", ".docx", ".xls", ".xlsx", ".csv", ".ppt", ".pptx", ".txt", ".rtf":
		return services.DocFile
	default:
		return services.OtherFile
	}
}

func (s *service) mapDocTypeToClassification(_ pulid.ID) services.FileClassification {
	// switch docTypeID {
	// case document.DocumentTypeLicense, document.DocumentTypeRegistration,
	// 	document.DocumentTypeInsurance, document.DocumentTypeMedicalCert:
	// 	return services.ClassificationRegulatory
	// case document.DocumentTypeDriverLog, document.DocumentTypeAccidentReport:
	// 	return services.ClassificationSensitive
	// case document.DocumentTypeProofOfDelivery, document.DocumentTypeInvoice,
	// 	document.DocumentTypeBillOfLading, document.DocumentTypeContract:
	// 	return services.ClassificationPrivate
	// default:
	// 	return services.ClassificationPublic
	// }

	return services.ClassificationPublic
}

func (s *service) mapResourceTypeToCategory(resourceType permission.Resource) services.FileCategory {
	//nolint:exhaustive // not all cases are implemented
	switch resourceType {
	case permission.ResourceShipment:
		return services.CategoryShipment
	case permission.ResourceWorker:
		return services.CategoryWorker
	default:
		return services.CategoryOther
	}
}
