package document

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

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
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// ServiceParams defines dependencies required for initializing the DocumentService.
// This includes database connection, logger, permission service, audit service,
// config manager, document repository, and file service.
type ServiceParams struct {
	fx.In

	DB           db.Connection
	Client       *minio.Client
	Logger       *logger.Logger
	PermService  services.PermissionService
	AuditService services.AuditService
	ConfigM      *config.Manager
	DocRepo      repositories.DocumentRepository
	FileService  services.FileService
	OrgRepo      repositories.OrganizationRepository
}

// service implements the DocumentService interface
// and provides methods to manage documents, including CRUD operations,
// status updates, duplication, and cancellation.
type service struct {
	l           *zerolog.Logger
	db          db.Connection
	client      *minio.Client
	endpoint    string
	docRepo     repositories.DocumentRepository
	fileService services.FileService
	orgRepo     repositories.OrganizationRepository
	ps          services.PermissionService
	as          services.AuditService
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
		l:           &log,
		db:          p.DB,
		client:      p.Client,
		endpoint:    p.ConfigM.Minio().Endpoint,
		docRepo:     p.DocRepo,
		fileService: p.FileService,
		orgRepo:     p.OrgRepo,
		ps:          p.PermService,
		as:          p.AuditService,
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

	// * Get a presigned URL for each document
	for _, doc := range docs.Items {
		presignedURL, err := s.fileService.GetFileURL(ctx, bucketName, doc.StoragePath, time.Hour*24)
		if err != nil {
			log.Error().Err(err).Msg("failed to get presigned URL for document")
			return nil, err
		}

		doc.PresignedURL = presignedURL
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
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create documents")
	}

	bucketName, err := s.orgRepo.GetOrganizationBucketName(ctx, req.OrganizationID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get organization bucket name")
		return nil, eris.Wrap(err, "get organization bucket name")
	}

	// * Validate request
	if err := req.Validate(ctx); err != nil {
		return nil, err
	}

	// * Generate file storage path
	objectKey := s.generateObjectPath(req)

	// * Prepare file upload request
	fileReq := &services.SaveFileRequest{
		OrgID:          req.OrganizationID.String(),
		BucketName:     bucketName,
		UserID:         req.UploadedByID.String(),
		FileName:       objectKey,
		File:           req.File,
		FileType:       s.determineFileType(req.FileName),
		Classification: s.mapDocTypeToClassification(req.DocumentType),
		Category:       s.mapResourceTypeToCategory(req.ResourceType),
		Tags: map[string]string{
			"resource_id":   req.ResourceID.String(),
			"resource_type": string(req.ResourceType),
			"doc_type":      string(req.DocumentType),
		},
		Metadata: map[string][]string{
			"description": {req.Description},
		},
	}

	// * Add tags to metadata
	if len(req.Tags) > 0 {
		fileReq.Metadata["tags"] = req.Tags
	}

	// * Determine if versioning should be used
	fileUploadResp, err := s.fileService.SaveFileVersion(ctx, fileReq)
	if err != nil {
		log.Error().Err(err).Msg("failed to save file version")
		return nil, err
	}

	// * Create document record in the database
	docStatus := req.Status
	if docStatus == "" {
		if req.RequireApproval {
			docStatus = document.DocumentStatusPendingApproval
		} else {
			docStatus = document.DocumentStatusActive
		}
	}

	// * Create document record in the database
	doc := &document.Document{
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		FileName:       filepath.Base(objectKey),
		OriginalName:   req.OriginalName,
		FileSize:       int64(len(req.File)),
		FileType:       filepath.Ext(req.FileName),
		StoragePath:    objectKey,
		DocumentType:   req.DocumentType,
		Status:         docStatus,
		ResourceID:     req.ResourceID,
		ResourceType:   req.ResourceType,
		ExpirationDate: req.ExpirationDate,
		Tags:           req.Tags,
		UploadedByID:   req.UploadedByID,
	}

	// * Save document to database
	savedDoc, err := s.docRepo.Create(ctx, doc)
	if err != nil {
		log.Error().Err(err).Msg("failed to create document")
		// * Try to clean up the file if we can't save the metadata
		_ = s.fileService.DeleteFile(ctx, req.OrganizationID.String(), objectKey)
		return nil, eris.Wrap(err, "create document")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDocument,
			ResourceID:     savedDoc.ID.String(),
			Action:         permission.ActionCreate,
			UserID:         req.UploadedByID,
			CurrentState:   jsonutils.MustToJSON(savedDoc),
			OrganizationID: savedDoc.OrganizationID,
			BusinessUnitID: savedDoc.BusinessUnitID,
		},
		audit.WithComment("Document created"),
	)
	if err != nil {
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

// BulkUploadDocuments uploads multiple documents in a single operation
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: The request containing document details.
//
// Returns:
//   - *services.BulkUploadDocumentResponse: The response containing the uploaded documents.
//   - error: An error if the operation fails.
func (s *service) BulkUploadDocuments(ctx context.Context, req *services.BulkUploadDocumentRequest) (*services.BulkUploadDocumentResponse, error) {
	log := s.l.With().
		Str("operation", "BulkUploadDocuments").
		Logger()

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
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create documents")
	}

	response := &services.BulkUploadDocumentResponse{
		Successful: make([]services.UploadDocumentResponse, 0, len(req.Documents)),
		Failed:     make([]services.FailedUpload, 0),
	}

	// * Process each document in the bulk request
	for i := range req.Documents {
		docInfo := &req.Documents[i] // Use pointer to avoid copying the whole struct
		uploadReq := &services.UploadDocumentRequest{
			OrganizationID:  req.OrganizationID,
			BusinessUnitID:  req.BusinessUnitID,
			UploadedByID:    req.UploadedByID,
			ResourceID:      req.ResourceID,
			ResourceType:    req.ResourceType,
			DocumentType:    docInfo.DocumentType,
			File:            docInfo.File,
			FileName:        docInfo.FileName,
			RequireApproval: docInfo.RequireApproval,
			OriginalName:    docInfo.OriginalName,
			Description:     docInfo.Description,
			Tags:            docInfo.Tags,
			ExpirationDate:  docInfo.ExpirationDate,
			Status:          document.DocumentStatusActive, // * Default for bulk uploads
		}

		uploadResp, err := s.UploadDocument(ctx, uploadReq)
		if err != nil {
			response.Failed = append(response.Failed, services.FailedUpload{
				FileName: docInfo.FileName,
				Error:    err,
			})
			continue
		}

		response.Successful = append(response.Successful, *uploadResp)
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDocument,
			ResourceID:     req.ResourceID.String(),
			Action:         permission.ActionCreate,
			UserID:         req.UploadedByID,
			CurrentState:   jsonutils.MustToJSON(response),
			OrganizationID: req.OrganizationID,
			BusinessUnitID: req.BusinessUnitID,
		},
		audit.WithComment("Documents uploaded"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log document creation")
	}

	return response, nil
}

func (s *service) generateObjectPath(req *services.UploadDocumentRequest) string {
	// Format: {resourceType}/{resourceID}/{documentType}/{timestamp}_{filename}
	now := time.Now()
	timestamp := now.Format("20060102150405")
	sanitizedFileName := strings.ReplaceAll(req.FileName, " ", "_")

	return fmt.Sprintf("%s/%s/%s_%s",
		req.ResourceID,
		req.DocumentType,
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

func (s *service) mapDocTypeToClassification(docType document.DocumentType) services.FileClassification {
	switch docType {
	case document.DocumentTypeLicense, document.DocumentTypeRegistration,
		document.DocumentTypeInsurance, document.DocumentTypeMedicalCert:
		return services.ClassificationRegulatory
	case document.DocumentTypeDriverLog, document.DocumentTypeAccidentReport:
		return services.ClassificationSensitive
	case document.DocumentTypeProofOfDelivery, document.DocumentTypeInvoice,
		document.DocumentTypeBillOfLading, document.DocumentTypeContract:
		return services.ClassificationPrivate
	default:
		return services.ClassificationPublic
	}
}

func (s *service) mapResourceTypeToCategory(resourceType permission.Resource) services.FileCategory {
	switch resourceType {
	case permission.ResourceShipment:
		return services.CategoryShipment
	case permission.ResourceWorker:
		return services.CategoryWorker
	default:
		return services.CategoryOther
	}
}
