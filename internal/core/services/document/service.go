package document

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/file"
	"github.com/emoss08/trenova/internal/infrastructure/storage/minio"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
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
	}
}

func (s *service) List(ctx context.Context, req *repositories.ListDocumentsRequest) (*ports.ListResult[*document.Document], error) {
	log := s.l.With().Str("operation", "List").Logger()

	entities, err := s.docRepo.List(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to list documents")
		return nil, err
	}

	return entities, nil
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
	bucketName, err := s.orgRepo.GetOrganizationBucketName(ctx, req.OrganizationID)
	if err != nil {
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
		s.l.Error().Str("org", req.OrganizationID.String()).Err(err).Msg("save file version")
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
		Description:    req.Description,
		ResourceID:     req.ResourceID,
		ResourceType:   req.ResourceType,
		ExpirationDate: req.ExpirationDate,
		Tags:           req.Tags,
		IsPublic:       req.IsPublic,
		UploadedByID:   req.UploadedByID,
	}

	// * Save document to database
	savedDoc, err := s.docRepo.Create(ctx, doc)
	if err != nil {
		// * Try to clean up the file if we can't save the metadata
		_ = s.fileService.DeleteFile(ctx, req.OrganizationID.String(), objectKey)
		return nil, eris.Wrap(err, "create document")
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
	if err := s.validateBulkUploadRequest(req); err != nil {
		return nil, err
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
			IsPublic:        docInfo.IsPublic,
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

	return response, nil
}

// GetDocumentByID retrieves a document by its ID
//
// Parameters:
//   - ctx: The context for the operation.
//   - orgID: The organization ID.
//   - buID: The business unit ID.
//   - docID: The document ID.
//
// Returns:
//   - *document.Document: The document.
//   - error: An error if the operation fails.
func (s *service) GetDocumentByID(ctx context.Context, orgID, buID, docID pulid.ID) (*document.Document, error) {
	doc, err := s.docRepo.GetByID(ctx, repositories.GetDocumentByIDOptions{
		ID:    docID,
		OrgID: orgID,
		BuID:  buID,
	})
	if err != nil {
		return nil, eris.Wrap(err, "get document by id")
	}

	return doc, nil
}

// GetDocumentContent retrieves the content of a document
//
// Parameters:
//   - ctx: The context for the operation.
//   - doc: The document.
//
// Returns:
//   - []byte: The document content.
//   - error: An error if the operation fails.
func (s *service) GetDocumentContent(ctx context.Context, doc *document.Document) ([]byte, error) {
	bucketName := doc.OrganizationID.String()
	obj, err := s.fileService.GetFileByBucketName(ctx, bucketName, doc.StoragePath)
	if err != nil {
		return nil, eris.Wrap(err, "get file content")
	}
	defer obj.Close()

	return io.ReadAll(obj)
}

// GetDocumentDownloadURL generates a pre-signed URL for downloading a document
//
// Parameters:
//   - ctx: The context for the operation.
//   - doc: The document.
//   - expiryDuration: The duration for which the URL is valid.
//
// Returns:
//   - string: The pre-signed URL.
//   - error: An error if the operation fails.
func (s *service) GetDocumentDownloadURL(ctx context.Context, doc *document.Document, expiryDuration time.Duration) (string, error) {
	if expiryDuration == 0 {
		expiryDuration = file.DefaultExpiry
	}

	bucketName := doc.OrganizationID.String()
	return s.fileService.GetFileURL(ctx, bucketName, doc.StoragePath, expiryDuration)
}

// ListEntityDocuments retrieves documents associated with a specific entity
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: The request containing document details.
//
// Returns:
//   - *ports.ListResult[*document.Document]: The list of documents.
//   - error: An error if the operation fails.
func (s *service) ListEntityDocuments(ctx context.Context, req *repositories.ListDocumentsRequest) (*ports.ListResult[*document.Document], error) {
	return s.docRepo.List(ctx, req)
}

// ApproveDocument marks a document as approved
//
// Parameters:
//   - ctx: The context for the operation.
//   - orgID: The organization ID.
//   - buID: The business unit ID.
//   - docID: The document ID.
//   - approverID: The approver ID.
//
// Returns:
//   - *document.Document: The document.
//   - error: An error if the operation fails.
func (s *service) ApproveDocument(ctx context.Context, orgID, buID, docID, approverID pulid.ID) (*document.Document, error) {
	doc, err := s.GetDocumentByID(ctx, orgID, buID, docID)
	if err != nil {
		return nil, err
	}

	if doc.Status != document.DocumentStatusPendingApproval {
		return nil, eris.New("document is not pending approval")
	}

	doc.Status = document.DocumentStatusActive
	doc.ApprovedByID = &approverID
	approvedAt := time.Now().Unix()
	doc.ApprovedAt = &approvedAt

	return s.docRepo.Update(ctx, doc)
}

// RejectDocument marks a document as rejected
//
// Parameters:
//   - ctx: The context for the operation.
//   - orgID: The organization ID.
//   - buID: The business unit ID.
//   - docID: The document ID.
//   - rejectorID: The rejector ID.
//   - reason: The reason for rejection.
//
// Returns:
//   - *document.Document: The document.
//   - error: An error if the operation fails.
func (s *service) RejectDocument(ctx context.Context, orgID, buID, docID, rejectorID pulid.ID, reason string) (*document.Document, error) {
	doc, err := s.GetDocumentByID(ctx, orgID, buID, docID)
	if err != nil {
		return nil, err
	}

	if doc.Status != document.DocumentStatusPendingApproval {
		return nil, eris.New("document is not pending approval")
	}

	doc.Status = document.DocumentStatusRejected
	doc.Description = doc.Description + "\nRejection reason: " + reason

	return s.docRepo.Update(ctx, doc)
}

// ArchiveDocument marks a document as archived
//
// Parameters:
//   - ctx: The context for the operation.
//   - orgID: The organization ID.
//   - buID: The business unit ID.
//   - docID: The document ID.
//
// Returns:
//   - *document.Document: The document.
//   - error: An error if the operation fails.
func (s *service) ArchiveDocument(ctx context.Context, orgID, buID, docID pulid.ID) (*document.Document, error) {
	doc, err := s.GetDocumentByID(ctx, orgID, buID, docID)
	if err != nil {
		return nil, err
	}

	doc.Status = document.DocumentStatusArchived

	return s.docRepo.Update(ctx, doc)
}

// DeleteDocument deletes a document
//
// Parameters:
//   - ctx: The context for the operation.
//   - orgID: The organization ID.
//   - buID: The business unit ID.
//   - docID: The document ID.
//
// Returns:
//   - error: An error if the operation fails.
func (s *service) DeleteDocument(ctx context.Context, orgID, buID, docID pulid.ID) error {
	doc, err := s.GetDocumentByID(ctx, orgID, buID, docID)
	if err != nil {
		return err
	}

	bucketName := doc.OrganizationID.String()

	// Delete from storage
	if err := s.fileService.DeleteFile(ctx, bucketName, doc.StoragePath); err != nil {
		s.l.Warn().Err(err).
			Str("docID", docID.String()).
			Str("bucket", bucketName).
			Str("path", doc.StoragePath).
			Msg("failed to delete document file from storage")
		// * Continue with database deletion even if storage deletion fails
	}

	// Delete from database
	return s.docRepo.Delete(ctx, repositories.DeleteDocumentRequest{
		ID:    docID,
		OrgID: orgID,
		BuID:  buID,
	})
}

// GetDocumentVersions retrieves all versions of a document
//
// Parameters:
//   - ctx: The context for the operation.
//   - doc: The document.
//
// Returns:
//   - []services.VersionInfo: The list of versions.
func (s *service) GetDocumentVersions(ctx context.Context, doc *document.Document) ([]services.VersionInfo, error) {
	bucketName := doc.OrganizationID.String()
	return s.fileService.GetFileVersion(ctx, bucketName, doc.StoragePath)
}

// RestoreDocumentVersion restores a document to a specific version
//
// Parameters:
//   - ctx: The context for the operation.
//   - doc: The document.
//   - versionID: The version ID.
//
// Returns:
//   - *document.Document: The document.
//   - error: An error if the operation fails.
func (s *service) RestoreDocumentVersion(ctx context.Context, doc *document.Document, versionID string) (*document.Document, error) {
	bucketName := doc.OrganizationID.String()

	// * Get the document content from the specified version
	data, versionInfo, err := s.fileService.GetSpecificVersion(ctx, bucketName, doc.StoragePath, versionID)
	if err != nil {
		return nil, eris.Wrap(err, "get specific version")
	}

	// * Create a new file save request
	fileReq := &services.SaveFileRequest{
		OrgID:          doc.OrganizationID.String(),
		UserID:         doc.UploadedByID.String(),
		FileName:       doc.StoragePath,
		File:           data,
		FileType:       s.determineFileType(doc.FileName),
		Classification: s.mapDocTypeToClassification(doc.DocumentType),
		Category:       s.mapResourceTypeToCategory(doc.ResourceType),
		VersionComment: fmt.Sprintf("Restored from version %s", versionID),
		Metadata:       versionInfo.Metadata,
	}

	// * Save as a new version
	_, err = s.fileService.SaveFileVersion(ctx, fileReq)
	if err != nil {
		s.l.Error().Str("org", doc.OrganizationID.String()).Err(err).Msg("save restored version")
		return nil, err
	}

	// * Update document metadata
	doc.FileSize = int64(len(data))
	doc.UpdatedAt = time.Now().Unix()
	doc.Version++ // * Increment version

	return s.docRepo.Update(ctx, doc)
}

// CheckExpiringDocuments finds documents that are about to expire
//
// Parameters:
//   - ctx: The context for the operation.
//   - daysToExpiration: The number of days to expiration.
//
// Returns:
//   - []*document.Document: The list of documents.
func (s *service) CheckExpiringDocuments(ctx context.Context, daysToExpiration int) ([]*document.Document, error) {
	now := time.Now().Unix()
	expirationThreshold := now + int64(daysToExpiration*24*60*60) // Convert days to seconds

	return s.docRepo.FindExpiringDocuments(ctx, &repositories.FindExpiringDocumentsRequest{
		ExpirationThreshold: expirationThreshold,
	})
}

func (s *service) validateBulkUploadRequest(req *services.BulkUploadDocumentRequest) error {
	if req.OrganizationID.IsNil() {
		return eris.New("organization ID is required")
	}
	if req.BusinessUnitID.IsNil() {
		return eris.New("business unit ID is required")
	}
	if req.UploadedByID.IsNil() {
		return eris.New("uploader ID is required")
	}
	if req.ResourceID.IsNil() {
		return eris.New("resource ID is required")
	}
	if req.ResourceType == "" {
		return eris.New("resource type is required")
	}
	if len(req.Documents) == 0 {
		return eris.New("at least one document is required")
	}

	return nil
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
