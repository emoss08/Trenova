package documenthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentupload"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/internal/core/services/documentuploadservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service                *documentservice.Service
	UploadService          *documentuploadservice.Service
	DocumentContentService serviceports.DocumentContentService
	ErrorHandler           *helpers.ErrorHandler
	PermissionMiddleware   *middleware.PermissionMiddleware
}

type Handler struct {
	service        *documentservice.Service
	uploadService  *documentuploadservice.Service
	contentService serviceports.DocumentContentService
	eh             *helpers.ErrorHandler
	pm             *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service:        p.Service,
		uploadService:  p.UploadService,
		contentService: p.DocumentContentService,
		eh:             p.ErrorHandler,
		pm:             p.PermissionMiddleware,
	}
}

func NewTestHandler(
	service *documentservice.Service,
	eh *helpers.ErrorHandler,
	pm *middleware.PermissionMiddleware,
) *Handler {
	return &Handler{
		service: service,
		eh:      eh,
		pm:      pm,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/documents")
	api.POST(
		"/uploads/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpCreate),
		h.createUploadSession,
	)
	api.GET(
		"/uploads/active/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.listActiveUploadSessions,
	)
	api.GET(
		"/uploads/:uploadSessionID/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.getUploadSession,
	)
	api.POST(
		"/uploads/:uploadSessionID/parts/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpCreate),
		h.getUploadPartURLs,
	)
	api.POST(
		"/uploads/:uploadSessionID/complete/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpCreate),
		h.completeUploadSession,
	)
	api.POST(
		"/uploads/:uploadSessionID/cancel/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpCreate),
		h.cancelUploadSession,
	)
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:documentID/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.get,
	)
	api.GET(
		"/:documentID/content/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.getContent,
	)
	api.GET(
		"/:documentID/versions/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.listVersions,
	)
	api.POST(
		"/:documentID/restore/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpUpdate),
		h.restoreVersion,
	)
	api.GET(
		"/:documentID/shipment-draft/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.getShipmentDraft,
	)
	api.POST(
		"/:documentID/shipment-draft/reextract/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpUpdate),
		h.reextractDocumentContent,
	)
	api.GET(
		"/:documentID/download/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.download,
	)
	api.GET(
		"/:documentID/view/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.view,
	)
	api.GET(
		"/:documentID/preview/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.preview,
	)
	api.POST(
		"/upload/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpCreate),
		h.upload,
	)
	api.POST(
		"/upload-bulk/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpCreate),
		h.uploadBulk,
	)
	api.DELETE(
		"/:documentID/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpDelete),
		h.delete,
	)
	api.POST(
		"/bulk-delete/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpDelete),
		h.bulkDelete,
	)
	api.GET(
		"/resource/:resourceType/:resourceID/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.getByResource,
	)
	api.GET(
		"/resource/:resourceType/:resourceID/packet-summary/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.getPacketSummary,
	)
}

// @Summary List documents
// @Description Returns paginated documents. Protected routes accept a Bearer token or an authenticated session cookie.
// @ID listDocuments
// @Tags Documents
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param resourceId query string false "Filter by resource ID"
// @Param resourceType query string false "Filter by resource type"
// @Param status query string false "Filter by document status"
// @Success 200 {object} pagination.Response[[]document.Document]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*document.Document], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListDocumentsRequest{
					Filter:       req,
					ResourceID:   helpers.QueryString(c, "resourceId", ""),
					ResourceType: helpers.QueryString(c, "resourceType", ""),
					Status:       helpers.QueryString(c, "status", ""),
				},
			)
		},
	)
}

// @Summary Get a document
// @ID getDocument
// @Tags Documents
// @Produce json
// @Param documentID path string true "Document ID"
// @Success 200 {object} document.Document
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/{documentID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetDocumentByIDRequest{
			ID: id,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) listVersions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	versions, err := h.service.ListVersions(c.Request.Context(), id, pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, versions)
}

func (h *Handler) restoreVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	doc, err := h.service.RestoreVersion(c.Request.Context(), id, pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, doc)
}

// @Summary Get a document download URL
// @ID downloadDocument
// @Tags Documents
// @Produce json
// @Param documentID path string true "Document ID"
// @Success 200 {object} gin.H
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/{documentID}/download/ [get]
func (h *Handler) download(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	url, err := h.service.GetDownloadURL(
		c.Request.Context(),
		repositories.GetDocumentByIDRequest{
			ID: id,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

// @Summary Get a document view URL
// @ID viewDocument
// @Tags Documents
// @Produce json
// @Param documentID path string true "Document ID"
// @Success 200 {object} gin.H
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/{documentID}/view/ [get]
func (h *Handler) view(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	url, err := h.service.GetViewURL(
		c.Request.Context(),
		repositories.GetDocumentByIDRequest{
			ID: id,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

// @Summary Get a document preview URL
// @ID previewDocument
// @Tags Documents
// @Produce json
// @Param documentID path string true "Document ID"
// @Success 200 {object} gin.H
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/{documentID}/preview/ [get]
func (h *Handler) preview(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	url, err := h.service.GetPreviewURL(
		c.Request.Context(),
		repositories.GetDocumentByIDRequest{
			ID: id,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

type uploadRequest struct {
	ResourceID     string   `form:"resourceId"     binding:"required"`
	ResourceType   string   `form:"resourceType"   binding:"required"`
	Description    string   `form:"description"`
	Tags           []string `form:"tags"`
	DocumentTypeID string   `form:"documentTypeId"`
	LineageID      string   `form:"lineageId"`
}

type createUploadSessionRequest struct {
	ResourceID     string   `json:"resourceId"     binding:"required"`
	ResourceType   string   `json:"resourceType"   binding:"required"`
	FileName       string   `json:"fileName"       binding:"required"`
	FileSize       int64    `json:"fileSize"       binding:"required,min=1"`
	ContentType    string   `json:"contentType"    binding:"required"`
	Description    string   `json:"description"`
	Tags           []string `json:"tags"`
	DocumentTypeID string   `json:"documentTypeId"`
	LineageID      string   `json:"lineageId"`
}

type uploadPartURLsRequest struct {
	PartNumbers []int `json:"partNumbers"`
}

// @Summary Upload a document
// @ID uploadDocument
// @Tags Documents
// @Accept mpfd
// @Produce json
// @Param resourceId formData string true "Resource ID"
// @Param resourceType formData string true "Resource type"
// @Param description formData string false "Document description"
// @Param tags formData []string false "Document tags"
// @Param documentTypeId formData string false "Document type ID"
// @Param file formData file true "Document file"
// @Success 201 {object} document.Document
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/upload/ [post]
func (h *Handler) upload(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req uploadRequest
	if err := c.ShouldBind(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.Upload(
		c.Request.Context(),
		&documentservice.UploadRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			File:           file,
			ResourceID:     req.ResourceID,
			ResourceType:   req.ResourceType,
			Description:    req.Description,
			Tags:           req.Tags,
			DocumentTypeID: req.DocumentTypeID,
			LineageID:      req.LineageID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result.Document)
}

func (h *Handler) createUploadSession(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req createUploadSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	session, err := h.uploadService.CreateSession(c.Request.Context(), &documentuploadservice.CreateSessionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		ResourceID:     req.ResourceID,
		ResourceType:   req.ResourceType,
		FileName:       req.FileName,
		FileSize:       req.FileSize,
		ContentType:    req.ContentType,
		Description:    req.Description,
		Tags:           req.Tags,
		DocumentTypeID: req.DocumentTypeID,
		LineageID:      req.LineageID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, session)
}

func (h *Handler) listActiveUploadSessions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	sessions, err := h.uploadService.ListActive(c.Request.Context(), &repositories.ListActiveDocumentUploadSessionsRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
		ResourceID:   helpers.QueryString(c, "resourceId", ""),
		ResourceType: helpers.QueryString(c, "resourceType", ""),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, sessions)
}

func (h *Handler) getUploadSession(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("uploadSessionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	state, err := h.uploadService.GetSessionState(c.Request.Context(), repositories.GetDocumentUploadSessionByIDRequest{
		ID: id,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, state)
}

func (h *Handler) getUploadPartURLs(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("uploadSessionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req uploadPartURLsRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	targets, err := h.uploadService.GetPartUploadTargets(c.Request.Context(), &documentuploadservice.PartRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		SessionID:   id,
		PartNumbers: req.PartNumbers,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"parts": targets})
}

func (h *Handler) completeUploadSession(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("uploadSessionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	session, err := h.uploadService.Complete(c.Request.Context(), &documentuploadservice.CompletionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		SessionID: id,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, session)
}

func (h *Handler) cancelUploadSession(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("uploadSessionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.uploadService.Cancel(c.Request.Context(), &documentuploadservice.CancelRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		SessionID: id,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": documentupload.StatusCanceled})
}

type bulkUploadRequest struct {
	ResourceID   string `form:"resourceId"   binding:"required"`
	ResourceType string `form:"resourceType" binding:"required"`
	LineageID    string `form:"lineageId"`
}

// @Summary Bulk upload documents
// @ID uploadDocumentsBulk
// @Tags Documents
// @Accept mpfd
// @Produce json
// @Param resourceId formData string true "Resource ID"
// @Param resourceType formData string true "Resource type"
// @Param files formData file true "Document files"
// @Success 201 {object} gin.H
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/upload-bulk/ [post]
func (h *Handler) uploadBulk(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req bulkUploadRequest
	if err := c.ShouldBind(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.BulkUpload(
		c.Request.Context(),
		&documentservice.BulkUploadRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			Files:        files,
			ResourceID:   req.ResourceID,
			ResourceType: req.ResourceType,
			LineageID:    req.LineageID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"documents":    result.Documents,
		"errorCount":   len(result.Errors),
		"successCount": len(result.Documents),
	})
}

// @Summary Delete a document
// @ID deleteDocument
// @Tags Documents
// @Param documentID path string true "Document ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/{documentID}/ [delete]
func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.service.Delete(
		c.Request.Context(),
		repositories.DeleteDocumentRequest{
			ID: id,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

type bulkDeleteRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// @Summary Bulk delete documents
// @ID bulkDeleteDocuments
// @Tags Documents
// @Accept json
// @Produce json
// @Param request body bulkDeleteRequest true "Bulk delete request"
// @Success 200 {object} gin.H
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/bulk-delete/ [post]
func (h *Handler) bulkDelete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req bulkDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	ids := make([]pulid.ID, 0, len(req.IDs))
	for _, idStr := range req.IDs {
		id, err := pulid.MustParse(idStr)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		ids = append(ids, id)
	}

	result, err := h.service.BulkDelete(
		c.Request.Context(),
		&documentservice.BulkDeleteRequest{
			IDs: ids,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deletedCount": result.DeletedCount,
		"errorCount":   len(result.Errors),
	})
}

// @Summary List documents by resource
// @ID listDocumentsByResource
// @Tags Documents
// @Produce json
// @Param resourceType path string true "Resource type"
// @Param resourceID path string true "Resource ID"
// @Success 200 {array} document.Document
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/resource/{resourceType}/{resourceID}/ [get]
func (h *Handler) getByResource(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	resourceType := c.Param("resourceType")
	resourceID := c.Param("resourceID")
	searchQuery := helpers.QueryString(c, "query", "")

	if searchQuery != "" && h.contentService != nil {
		documents, err := h.contentService.SearchDocuments(
			c.Request.Context(),
			pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			resourceType,
			resourceID,
			searchQuery,
		)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}

		c.JSON(http.StatusOK, documents)
		return
	}

	documents, err := h.service.GetByResource(
		c.Request.Context(),
		&repositories.GetDocumentsByResourceRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			ResourceID:   resourceID,
			ResourceType: resourceType,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, documents)
}

func (h *Handler) getPacketSummary(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	summary, err := h.service.GetPacketSummary(
		c.Request.Context(),
		c.Param("resourceType"),
		c.Param("resourceID"),
		pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, summary)
}

func (h *Handler) getContent(c *gin.Context) {
	if h.contentService == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"message": "Document content service unavailable"})
		return
	}

	authCtx := authctx.GetAuthContext(c)
	documentID, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	content, err := h.contentService.GetContent(
		c.Request.Context(),
		documentID,
		pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, content)
}

func (h *Handler) getShipmentDraft(c *gin.Context) {
	if h.contentService == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"message": "Document content service unavailable"})
		return
	}

	authCtx := authctx.GetAuthContext(c)
	documentID, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	draft, err := h.contentService.GetShipmentDraft(
		c.Request.Context(),
		documentID,
		pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, draft)
}

func (h *Handler) reextractDocumentContent(c *gin.Context) {
	if h.contentService == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"message": "Document content service unavailable"})
		return
	}

	authCtx := authctx.GetAuthContext(c)
	documentID, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.contentService.Reextract(
		c.Request.Context(),
		documentID,
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusAccepted)
}
