package documenthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *documentservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *documentservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
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
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result.Document)
}

type bulkUploadRequest struct {
	ResourceID   string `form:"resourceId"   binding:"required"`
	ResourceType string `form:"resourceType" binding:"required"`
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
