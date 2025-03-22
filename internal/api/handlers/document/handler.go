package document

import (
	"fmt"
	"io"
	"strconv"
	"strings"

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

	api.Get("/count-by-resource/", rl.WithRateLimit(
		[]fiber.Handler{h.getDocumentCountByResource},
		middleware.PerSecond(5),
	)...)

	api.Get("/:resourceType/sub-folders/", rl.WithRateLimit(
		[]fiber.Handler{h.getResourceSubFolders},
		middleware.PerSecond(5),
	)...)

	api.Get("/:resourceType/:resourceID/", rl.WithRateLimit(
		[]fiber.Handler{h.getDocumentsByResourceID},
		middleware.PerSecond(5),
	)...)

	// Upload document
	api.Post("/upload/", rl.WithRateLimit(
		[]fiber.Handler{h.upload},
		middleware.PerSecond(2), // More restrictive for uploads
	)...)

	// Bulk upload documents
	api.Post("/bulk-upload/", rl.WithRateLimit(
		[]fiber.Handler{h.bulkUpload},
		middleware.PerSecond(1), // Even more restrictive for bulk operations
	)...)
}

func (h Handler) getDocumentCountByResource(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	resp, err := h.ds.GetDocumentCountByResource(c.UserContext(), ports.TenantOptions{
		UserID: reqCtx.UserID,
		BuID:   reqCtx.BuID,
		OrgID:  reqCtx.OrgID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(resp)
}

func (h Handler) getResourceSubFolders(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	resp, err := h.ds.GetResourceSubFolders(c.UserContext(), repositories.GetResourceSubFoldersRequest{
		ResourceType: permission.Resource(c.Params("resourceType")),
		TenantOptions: ports.TenantOptions{
			UserID: reqCtx.UserID,
			BuID:   reqCtx.BuID,
			OrgID:  reqCtx.OrgID,
		},
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(resp)
}

func (h Handler) getDocumentsByResourceID(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*document.Document], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		return h.ds.GetDocumentsByResourceID(fc.UserContext(), &repositories.GetDocumentsByResourceIDRequest{
			Filter:       filter,
			ResourceType: permission.Resource(c.Params("resourceType")),
			ResourceID:   c.Params("resourceID"),
			TenantOptions: ports.TenantOptions{
				UserID: reqCtx.UserID,
				BuID:   reqCtx.BuID,
				OrgID:  reqCtx.OrgID,
			},
		})
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
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
