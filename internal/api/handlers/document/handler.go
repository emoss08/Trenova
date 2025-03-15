package document

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/file"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/fileutils"
	"github.com/emoss08/trenova/internal/pkg/utils/intutils"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	DocumentService services.DocumentService
	ErrorHandler    *validator.ErrorHandler
}

type Handler struct {
	ds services.DocumentService
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ds: p.DocumentService, eh: p.ErrorHandler}
}

func (h Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/documents")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	// Get single document
	api.Get("/:id", rl.WithRateLimit(
		[]fiber.Handler{h.getByID},
		middleware.PerSecond(5),
	)...)

	// Get document content
	api.Get("/:id/content", rl.WithRateLimit(
		[]fiber.Handler{h.getContent},
		middleware.PerSecond(3),
	)...)

	// Get document download URL
	api.Get("/:id/download", rl.WithRateLimit(
		[]fiber.Handler{h.getDownloadURL},
		middleware.PerSecond(3),
	)...)

	// Upload document
	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.upload},
		middleware.PerSecond(2), // More restrictive for uploads
	)...)

	// Bulk upload documents
	api.Post("/bulk", rl.WithRateLimit(
		[]fiber.Handler{h.bulkUpload},
		middleware.PerSecond(1), // Even more restrictive for bulk operations
	)...)

	// Get document versions
	api.Get("/:id/versions", rl.WithRateLimit(
		[]fiber.Handler{h.getVersions},
		middleware.PerSecond(3),
	)...)

	// Restore document version
	api.Post("/:id/versions/:versionId/restore", rl.WithRateLimit(
		[]fiber.Handler{h.restoreVersion},
		middleware.PerSecond(2),
	)...)

	// Workflow endpoints
	api.Put("/:id/approve", rl.WithRateLimit(
		[]fiber.Handler{h.approve},
		middleware.PerSecond(3),
	)...)

	api.Put("/:id/reject", rl.WithRateLimit(
		[]fiber.Handler{h.reject},
		middleware.PerSecond(3),
	)...)

	api.Put("/:id/archive", rl.WithRateLimit(
		[]fiber.Handler{h.archive},
		middleware.PerSecond(3),
	)...)

	// Delete document
	api.Delete("/:id", rl.WithRateLimit(
		[]fiber.Handler{h.delete},
		middleware.PerSecond(2),
	)...)
}

func (h Handler) list(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*document.Document], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		statuses := []document.DocumentStatus{}
		if status := fc.Query("status"); status != "" {
			statuses = append(statuses, document.DocumentStatus(status))
		}

		tags := []string{}
		if tag := fc.Query("tag"); tag != "" {
			tags = append(tags, tag)
		}

		return h.ds.List(fc.UserContext(), &repositories.ListDocumentsRequest{
			Filter:              filter,
			ResourceType:        permission.Resource(fc.Query("resourceType")),
			DocumentType:        document.DocumentType(fc.Params("documentType")),
			ResourceID:          pulid.Must(fc.Query("resourceID")),
			Statuses:            statuses,
			Tags:                tags,
			SortBy:              fc.Query("sortBy"),
			SortDir:             fc.Query("sortDir"),
			ExpirationDateStart: intutils.SafeInt64PtrOrNil(fc.QueryInt("expirationDateStart")),
			ExpirationDateEnd:   intutils.SafeInt64PtrOrNil(fc.QueryInt("expirationDateEnd")),
			CreatedAtStart:      intutils.SafeInt64PtrOrNil(fc.QueryInt("createdAtStart")),
			CreatedAtEnd:        intutils.SafeInt64PtrOrNil(fc.QueryInt("createdAtEnd")),
			DocumentRequest: repositories.DocumentRequest{
				ExpandDocumentDetails: fc.QueryBool("expandDocumentDetails"),
			},
		})
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h Handler) getByID(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	docID, err := pulid.MustParse(c.Params("id"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	doc, err := h.ds.GetDocumentByID(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, docID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(doc)
}

func (h Handler) getContent(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	docID, err := pulid.MustParse(c.Params("id"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	doc, err := h.ds.GetDocumentByID(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, docID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	content, err := h.ds.GetDocumentContent(c.UserContext(), doc)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// * Set appropriate content type header
	c.Set("Content-Type", fileutils.GetContentType(doc.FileType))
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%q\"", doc.OriginalName))
	c.Set("Content-Length", strconv.FormatInt(doc.FileSize, 10))

	return c.Send(content)
}

func (h Handler) getDownloadURL(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	docID, err := pulid.MustParse(c.Params("id"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// * Parse expiry from query param or use default
	var expiry time.Duration
	if expiryStr := c.Query("expiry"); expiryStr != "" {
		expiryInt, err := strconv.Atoi(expiryStr)
		if err != nil {
			expiry = file.DefaultExpiry
		} else {
			expiry = time.Duration(expiryInt) * time.Minute
		}
	} else {
		expiry = file.DefaultExpiry
	}

	doc, err := h.ds.GetDocumentByID(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, docID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	url, err := h.ds.GetDocumentDownloadURL(c.UserContext(), doc, expiry)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(fiber.Map{
		"url":           url,
		"expiryMinutes": int(expiry.Minutes()),
		"fileName":      doc.OriginalName,
	})
}

func (h Handler) upload(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// * Ensure the content type is multipart/form-data
	if !strings.Contains(c.Get("Content-Type"), "multipart/form-data") {
		return h.eh.HandleError(c, errors.NewValidationError(
			"content-type",
			errors.ErrInvalid,
			"Content type must be multipart/form-data",
		))
	}

	// * Get the file from the form
	fh, err := c.FormFile("file")
	if err != nil {
		return h.eh.HandleError(c, errors.NewValidationError(
			"file",
			errors.ErrRequired,
			"File is required",
		))
	}

	// * Check file size
	if fh.Size > file.MaxFileSize {
		return h.eh.HandleError(c, errors.NewValidationError(
			"file",
			errors.ErrInvalid,
			fmt.Sprintf("File size exceeds the maximum limit of %d MB", file.MaxFileSize/(1024*1024)),
		))
	}

	// * Parse other form fields
	resourceID, err := pulid.MustParse(c.FormValue("resourceId"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// * Parse tags
	var tags []string
	if tagsParam := c.FormValue("tags"); tagsParam != "" {
		tags = strings.Split(tagsParam, ",")
	}

	// * Parse expiration date if provided
	var expirationDate *int64
	if expDateStr := c.FormValue("expirationDate"); expDateStr != "" {
		expDateInt, err := strconv.ParseInt(expDateStr, 10, 64)
		if err == nil {
			expirationDate = &expDateInt
		}
	}

	// * Open the file
	fileHandle, err := fh.Open()
	if err != nil {
		return h.eh.HandleError(c, errors.NewBusinessError("Failed to open file"))
	}
	defer fileHandle.Close()

	// * Read the file content
	fileContent, err := io.ReadAll(fileHandle)
	if err != nil {
		return h.eh.HandleError(c, errors.NewBusinessError("Failed to read file content"))
	}

	// * Create the upload request
	uploadReq := &services.UploadDocumentRequest{
		OrganizationID:  reqCtx.OrgID,
		BusinessUnitID:  reqCtx.BuID,
		UploadedByID:    reqCtx.UserID,
		ResourceID:      resourceID,
		ResourceType:    permission.Resource(c.FormValue("resourceType")),
		DocumentType:    document.DocumentType(c.FormValue("documentType")),
		File:            fileContent,
		FileName:        fh.Filename,
		OriginalName:    fh.Filename,
		Description:     c.FormValue("description"),
		Tags:            tags,
		ExpirationDate:  expirationDate,
		IsPublic:        c.FormValue("isPublic") == "true",
		RequireApproval: c.FormValue("requireApproval") == "true",
	}

	// * Upload the document
	resp, err := h.ds.UploadDocument(c.UserContext(), uploadReq)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h Handler) bulkUpload(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// * Get common fields
	resourceID, err := pulid.MustParse(c.FormValue("resourceId"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// * Parse the files
	form, err := c.MultipartForm()
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// * Get all of the files from "files" key
	files := form.File["files"]
	if len(files) == 0 {
		return h.eh.HandleError(c, errors.NewValidationError(
			"files",
			errors.ErrRequired,
			"No files provided",
		))
	}

	// * Parse metadata for each file
	fileMetadata := make(map[string]services.BulkDocumentInfo)
	if metadataJSON := c.FormValue("metadata"); metadataJSON != "" {
		var metadata []map[string]any
		if err := sonic.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			return h.eh.HandleError(c, errors.NewValidationError(
				"metadata",
				errors.ErrInvalid,
				"Invalid metadata JSON",
			))
		}

		for _, meta := range metadata {
			fileName, ok := meta["fileName"].(string)
			if !ok {
				continue
			}

			docType, _ := meta["documentType"].(string)
			description, _ := meta["description"].(string)
			isPublicVal, _ := meta["isPublic"].(bool)

			var tags []string
			if tagsArr, ok := meta["tags"].([]interface{}); ok {
				for _, t := range tagsArr {
					if tagStr, ok := t.(string); ok {
						tags = append(tags, tagStr)
					}
				}
			}

			var expirationDate *int64
			if expDateVal, ok := meta["expirationDate"].(float64); ok {
				expDateInt := int64(expDateVal)
				expirationDate = &expDateInt
			}

			fileMetadata[fileName] = services.BulkDocumentInfo{
				DocumentType:   document.DocumentType(docType),
				Description:    description,
				Tags:           tags,
				ExpirationDate: expirationDate,
				IsPublic:       isPublicVal,
			}
		}
	}

	// * Process each file
	bulkReq := &services.BulkUploadDocumentRequest{
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
		UploadedByID:   reqCtx.UserID,
		ResourceID:     resourceID,
		ResourceType:   permission.Resource(c.FormValue("resourceType")),
		Documents:      make([]services.BulkDocumentInfo, 0, len(files)),
	}

	for _, fh := range files {
		if fh.Size > file.MaxFileSize {
			return h.eh.HandleError(c, errors.NewValidationError(
				"file",
				errors.ErrInvalid,
				fmt.Sprintf("File %s exceeds the maximum size limit", fh.Filename),
			))
		}

		fileHandle, err := fh.Open()
		if err != nil {
			return h.eh.HandleError(c, errors.NewBusinessError(fmt.Sprintf("Failed to open file %s", fh.Filename)))
		}

		fileContent, err := io.ReadAll(fileHandle)
		fileHandle.Close()
		if err != nil {
			return h.eh.HandleError(c, errors.NewBusinessError(fmt.Sprintf("Failed to read file %s", fh.Filename)))
		}

		// * Get metadata for this file or use defaults
		meta, exists := fileMetadata[fh.Filename]
		if !exists {
			meta = services.BulkDocumentInfo{
				DocumentType: document.DocumentTypeOther,
			}
		}

		meta.File = fileContent
		meta.FileName = fh.Filename
		meta.OriginalName = fh.Filename

		bulkReq.Documents = append(bulkReq.Documents, meta)
	}

	// * Upload the documents
	resp, err := h.ds.BulkUploadDocuments(c.UserContext(), bulkReq)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h Handler) getVersions(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	docID, err := pulid.MustParse(c.Params("id"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	doc, err := h.ds.GetDocumentByID(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, docID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	versions, err := h.ds.GetDocumentVersions(c.UserContext(), doc)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(versions)
}

func (h Handler) restoreVersion(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	docID, err := pulid.MustParse(c.Params("id"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	versionID := c.Params("versionId")

	doc, err := h.ds.GetDocumentByID(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, docID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	restoredDoc, err := h.ds.RestoreDocumentVersion(c.UserContext(), doc, versionID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(restoredDoc)
}

func (h Handler) approve(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	docID, err := pulid.MustParse(c.Params("id"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	doc, err := h.ds.ApproveDocument(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, docID, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(doc)
}

func (h Handler) reject(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	docID, err := pulid.MustParse(c.Params("id"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	// * Parse reason from body
	var payload struct {
		Reason string `json:"reason"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return h.eh.HandleError(c, errors.NewValidationError(
			"reason",
			errors.ErrInvalid,
			"Invalid payload",
		))
	}

	if payload.Reason == "" {
		return h.eh.HandleError(c, errors.NewValidationError(
			"reason",
			errors.ErrRequired,
			"Rejection reason is required",
		))
	}

	doc, err := h.ds.RejectDocument(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, docID, reqCtx.UserID, payload.Reason)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(doc)
}

func (h Handler) archive(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	docID, err := pulid.MustParse(c.Params("id"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	doc, err := h.ds.ArchiveDocument(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, docID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(doc)
}

func (h Handler) delete(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	docID, err := pulid.MustParse(c.Params("id"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	err = h.ds.DeleteDocument(c.UserContext(), reqCtx.OrgID, reqCtx.BuID, docID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}
