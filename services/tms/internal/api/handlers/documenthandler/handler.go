package documenthandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentuploadservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *documentservice.Service
	UploadService        *documentuploadservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	StorageConfig        *storageconfig.Config `optional:"true"`
}

type Handler struct {
	service       *documentservice.Service
	uploadService *documentuploadservice.Service
	eh            *helpers.ErrorHandler
	pm            *middleware.PermissionMiddleware
	storageConfig storageconfig.Config
}

func requestActorFromAuthContext(authCtx *authctx.AuthContext) permissionports.RequestActor {
	return permissionports.RequestActor{
		PrincipalType:  kernelports.PrincipalType(authCtx.PrincipalType),
		PrincipalID:    authCtx.PrincipalID,
		UserID:         authCtx.UserID,
		APIKeyID:       authCtx.APIKeyID,
		BusinessUnitID: authCtx.BusinessUnitID,
		OrganizationID: authCtx.OrganizationID,
	}
}

const multipartFormOverheadBytes int64 = 1 << 20

func New(p Params) *Handler {
	storageConfig := storageconfig.Config{}
	if p.StorageConfig != nil {
		storageConfig = *p.StorageConfig
	}

	return &Handler{
		service:       p.Service,
		uploadService: p.UploadService,
		eh:            p.ErrorHandler,
		pm:            p.PermissionMiddleware,
		storageConfig: storageConfig,
	}
}

func (h *Handler) limitUploadBody(maxFiles int) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := h.maxUploadBodyBytes(maxFiles)
		if c.Request.ContentLength > limit {
			c.Abort()
			helpers.JSON(c, http.StatusRequestEntityTooLarge, gin.H{
				"error": "request body too large",
			})
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

func (h *Handler) maxUploadBodyBytes(maxFiles int) int64 {
	if maxFiles < 1 {
		maxFiles = 1
	}

	return int64(maxFiles) * (h.storageConfig.GetMaxFileSize() + multipartFormOverheadBytes)
}

func NewTestHandlerWithConfig(
	service *documentservice.Service,
	eh *helpers.ErrorHandler,
	pm *middleware.PermissionMiddleware,
	cfg *storageconfig.Config,
) *Handler {
	storageConfig := storageconfig.Config{}
	if cfg != nil {
		storageConfig = *cfg
	}

	return &Handler{
		service:       service,
		eh:            eh,
		pm:            pm,
		storageConfig: storageConfig,
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
	api.GET(
		"/:documentID/pages/:pageNumber/image/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.pageImage,
	)
	api.POST(
		"/upload/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpCreate),
		h.limitUploadBody(1),
		h.upload,
	)
	api.POST(
		"/upload-bulk/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpCreate),
		h.limitUploadBody(h.storageConfig.GetMaxFilesPerUpload()),
		h.uploadBulk,
	)
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

	content, err := h.service.GetDownloadContent(
		c.Request.Context(),
		documentports.GetDocumentByIDRequest{
			ID:         id,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	defer content.Body.Close()

	c.Header("Content-Disposition", content.ContentDisposition)
	c.DataFromReader(http.StatusOK, -1, content.ContentType, content.Body, nil)
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

	content, err := h.service.GetViewContent(
		c.Request.Context(),
		documentports.GetDocumentByIDRequest{
			ID:         id,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	defer content.Body.Close()

	c.Header("Content-Disposition", content.ContentDisposition)
	c.DataFromReader(http.StatusOK, -1, content.ContentType, content.Body, nil)
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

	content, err := h.service.GetPreviewContent(
		c.Request.Context(),
		documentports.GetDocumentByIDRequest{
			ID:         id,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	defer content.Body.Close()

	c.Header("Content-Disposition", content.ContentDisposition)
	c.DataFromReader(http.StatusOK, -1, content.ContentType, content.Body, nil)
}

// @Summary Get one page of a document as an image
// @ID getDocumentPageImage
// @Tags Documents
// @Produce image/webp
// @Param documentID path string true "Document ID"
// @Param pageNumber path int true "Page number, from 1"
// @Success 200 {file} binary
// @Success 304 "Not modified"
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/{documentID}/pages/{pageNumber}/image/ [get]
func (h *Handler) pageImage(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	page, err := strconv.Atoi(c.Param("pageNumber"))
	if err != nil || page < 1 {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError(
				"pageNumber",
				errortypes.ErrInvalid,
				"Page number must be a whole number from 1",
			),
		)
		return
	}

	image, err := h.service.GetPageImage(
		c.Request.Context(),
		&documentports.GetDocumentPageImageRequest{
			GetDocumentByIDRequest: documentports.GetDocumentByIDRequest{
				ID:         id,
				TenantInfo: pagination.FromAuth(authCtx),
			},
			PageNumber:  page,
			IfNoneMatch: c.GetHeader("If-None-Match"),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("ETag", image.ETag)
	if image.NotModified {
		c.Status(http.StatusNotModified)
		return
	}
	defer image.Body.Close()

	c.Header("Content-Disposition", image.ContentDisposition)
	c.Header("X-Content-Type-Options", "nosniff")
	c.DataFromReader(http.StatusOK, image.ContentLength, image.ContentType, image.Body, nil)
}

type uploadRequest struct {
	ResourceID        string   `form:"resourceId"        binding:"required"`
	ResourceType      string   `form:"resourceType"      binding:"required"`
	ProcessingProfile string   `form:"processingProfile"`
	Description       string   `form:"description"`
	Tags              []string `form:"tags"`
	DocumentTypeID    string   `form:"documentTypeId"`
	LineageID         string   `form:"lineageId"`
}

type createUploadSessionRequest struct {
	ResourceID        string   `json:"resourceId"        binding:"required"`
	ResourceType      string   `json:"resourceType"      binding:"required"`
	ProcessingProfile string   `json:"processingProfile"`
	FileName          string   `json:"fileName"          binding:"required"`
	FileSize          int64    `json:"fileSize"          binding:"required,min=1"`
	ContentType       string   `json:"contentType"       binding:"required"`
	Description       string   `json:"description"`
	Tags              []string `json:"tags"`
	DocumentTypeID    string   `json:"documentTypeId"`
	LineageID         string   `json:"lineageId"`
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
			TenantInfo:        pagination.FromAuthAsUser(authCtx),
			Actor:             requestActorFromAuthContext(authCtx),
			File:              file,
			ResourceID:        req.ResourceID,
			ResourceType:      req.ResourceType,
			ProcessingProfile: req.ProcessingProfile,
			Description:       req.Description,
			Tags:              req.Tags,
			DocumentTypeID:    req.DocumentTypeID,
			LineageID:         req.LineageID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusCreated, result.Document)
}

func (h *Handler) createUploadSession(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req createUploadSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	session, err := h.uploadService.CreateSession(
		c.Request.Context(),
		&documentuploadports.CreateSessionRequest{
			TenantInfo:        pagination.FromAuthAsUser(authCtx),
			Actor:             requestActorFromAuthContext(authCtx),
			ResourceID:        req.ResourceID,
			ResourceType:      req.ResourceType,
			ProcessingProfile: req.ProcessingProfile,
			FileName:          req.FileName,
			FileSize:          req.FileSize,
			ContentType:       req.ContentType,
			Description:       req.Description,
			Tags:              req.Tags,
			DocumentTypeID:    req.DocumentTypeID,
			LineageID:         req.LineageID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusCreated, session)
}

func (h *Handler) listActiveUploadSessions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	sessions, err := h.uploadService.ListActive(
		c.Request.Context(),
		&documentuploadports.ListActiveDocumentUploadSessionsRequest{
			TenantInfo:   pagination.FromAuth(authCtx),
			ResourceID:   helpers.QueryString(c, "resourceId", ""),
			ResourceType: helpers.QueryString(c, "resourceType", ""),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, sessions)
}

func (h *Handler) getUploadSession(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("uploadSessionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	state, err := h.uploadService.GetSessionState(
		c.Request.Context(),
		documentuploadports.GetDocumentUploadSessionByIDRequest{
			ID:         id,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, state)
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

	targets, err := h.uploadService.GetPartUploadTargets(
		c.Request.Context(),
		&documentuploadports.PartRequest{
			TenantInfo:  pagination.FromAuthAsUser(authCtx),
			SessionID:   id,
			PartNumbers: req.PartNumbers,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, gin.H{"parts": targets})
}

func (h *Handler) completeUploadSession(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("uploadSessionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	session, err := h.uploadService.Complete(
		c.Request.Context(),
		&documentuploadports.CompletionRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			Actor:      requestActorFromAuthContext(authCtx),
			SessionID:  id,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusAccepted, session)
}

func (h *Handler) cancelUploadSession(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("uploadSessionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.uploadService.Cancel(c.Request.Context(), &documentuploadports.CancelRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		SessionID:  id,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, gin.H{"status": documentupload.StatusCanceled})
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
		h.eh.HandleError(
			c,
			errortypes.NewValidationError(
				"files",
				errortypes.ErrRequired,
				"At least one file is required",
			),
		)
		return
	}

	if len(files) > h.storageConfig.GetMaxFilesPerUpload() {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError(
				"files",
				errortypes.ErrInvalidLength,
				"Too many files in upload request",
			),
		)
		return
	}

	result, err := h.service.BulkUpload(
		c.Request.Context(),
		&documentservice.BulkUploadRequest{
			TenantInfo:   pagination.FromAuthAsUser(authCtx),
			Actor:        requestActorFromAuthContext(authCtx),
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

	helpers.JSON(c, http.StatusCreated, gin.H{
		"documents":    result.Documents,
		"errorCount":   len(result.Errors),
		"successCount": len(result.Documents),
	})
}
