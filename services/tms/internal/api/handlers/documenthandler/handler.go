package documenthandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentuploadservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentimportassistantports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func contentServiceUnavailable() gin.H {
	return gin.H{"message": "Document content service unavailable"}
}

type Params struct {
	fx.In

	Service                *documentservice.Service
	UploadService          *documentuploadservice.Service
	DocumentContentService documentports.DocumentContentService
	ExtractionStatus       documentports.ExtractionStatusReader
	Reader                 documentports.DocumentReader
	ImportAssistant        shipmentimportassistantports.ShipmentImportAssistantService
	PageAssistant          assistantports.PageAssistant
	ErrorHandler           *helpers.ErrorHandler
	PermissionMiddleware   *middleware.PermissionMiddleware
	StorageConfig          *storageconfig.Config
	Logger                 *zap.Logger
}

type Handler struct {
	service         *documentservice.Service
	uploadService   *documentuploadservice.Service
	contentService  documentports.DocumentContentService
	extraction      documentports.ExtractionStatusReader
	reader          documentports.DocumentReader
	importAssistant shipmentimportassistantports.ShipmentImportAssistantService
	pageAssistant   assistantports.PageAssistant
	eh              *helpers.ErrorHandler
	pm              *middleware.PermissionMiddleware
	storageConfig   storageconfig.Config
	logger          *zap.Logger
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
	storageConfig = *p.StorageConfig

	return &Handler{
		service:         p.Service,
		uploadService:   p.UploadService,
		contentService:  p.DocumentContentService,
		extraction:      p.ExtractionStatus,
		reader:          p.Reader,
		importAssistant: p.ImportAssistant,
		pageAssistant:   p.PageAssistant,
		eh:              p.ErrorHandler,
		pm:              p.PermissionMiddleware,
		storageConfig:   storageConfig,
		logger:          p.Logger,
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

//nolint:funlen // existing workflow or route registration is intentionally kept together
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/documents")
	api.POST(
		"/uploads/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpCreate),
		h.createUploadSession,
	)
	api.GET(
		"/extraction-status/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.extractionStatus,
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
	api.PUT(
		"/uploads/:uploadSessionID/parts/:partNumber/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpCreate),
		h.uploadSessionPart,
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
		"/:documentID/reading/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.getReading,
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
	api.POST(
		"/:documentID/approve/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpApprove),
		h.approve,
	)
	api.POST(
		"/:documentID/reject/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpReject),
		h.reject,
	)
	api.POST(
		"/:documentID/signature/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpUpdate),
		h.setSignature,
	)
	api.POST(
		"/:documentID/delivery-facts/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpUpdate),
		h.setDeliveryFacts,
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
	api.POST(
		"/:documentID/attach-to-shipment/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpUpdate),
		h.attachToShipment,
	)
	api.POST(
		"/:documentID/import-assistant/thread/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.pm.RequirePermission(permission.ResourceAssistant.String(), permission.OpCreate),
		h.openImportAssistantThread,
	)
	api.GET(
		"/:documentID/import-assistant/history/",
		h.pm.RequirePermission(permission.ResourceDocument.String(), permission.OpRead),
		h.getImportAssistantHistory,
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
				&documentports.ListDocumentsRequest{
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
		documentports.GetDocumentByIDRequest{
			ID:         id,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
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

	helpers.JSON(c, http.StatusOK, versions)
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

	helpers.JSON(c, http.StatusOK, doc)
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

// @Summary Get a document's first page as a thumbnail
// @Description The 240 × 320 WebP thumbnail drawn when the document was stored. Cached for a day and revalidated by ETag.
// @ID previewDocument
// @Tags Documents
// @Produce image/webp
// @Param documentID path string true "Document ID"
// @Success 200 {file} binary
// @Success 304 "The thumbnail the browser holds is current"
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
		&documentports.GetDocumentPreviewRequest{
			GetDocumentByIDRequest: documentports.GetDocumentByIDRequest{
				ID:         id,
				TenantInfo: pagination.FromAuth(authCtx),
			},
			IfNoneMatch: c.GetHeader("If-None-Match"),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("ETag", content.ETag)
	if content.NotModified {
		c.Status(http.StatusNotModified)
		return
	}
	defer content.Body.Close()

	c.Header("Content-Disposition", content.ContentDisposition)
	c.Header("X-Content-Type-Options", "nosniff")
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

// uploadSessionPart receives the bytes of one part at the URL getUploadPartURLs issued. The
// client never names this path; it PUTs to the target's url, so a search of client code for
// callers does not find it.
func (h *Handler) uploadSessionPart(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("uploadSessionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	partNumber, err := strconv.Atoi(c.Param("partNumber"))
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"partNumber",
			errortypes.ErrInvalid,
			"Invalid part number",
		))
		return
	}

	session, err := h.uploadService.UploadPart(
		c.Request.Context(),
		&documentuploadports.UploadPartRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			SessionID:  id,
			PartNumber: partNumber,
			Body:       c.Request.Body,
			Size:       c.Request.ContentLength,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, session)
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
		documentports.DeleteDocumentRequest{
			ID:         id,
			TenantInfo: pagination.FromAuth(authCtx),
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
			IDs:        ids,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			UserID:     authCtx.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, gin.H{
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

	if searchQuery != "" {
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

		helpers.JSON(c, http.StatusOK, documents)
		return
	}

	documents, err := h.service.GetByResource(
		c.Request.Context(),
		&documentports.GetDocumentsByResourceRequest{
			TenantInfo:          pagination.FromAuth(authCtx),
			ResourceID:          resourceID,
			ResourceType:        resourceType,
			IncludeDocumentType: helpers.QueryBool(c, "includeDocumentType", false),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, documents)
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

	helpers.JSON(c, http.StatusOK, summary)
}

// @Summary Read a document as it was read
// @Description Each value read from the document with the page and the box on that page it was found in, the same reading the Desk's extraction card draws. A document nothing was read from returns no fields.
// @ID getDocumentReading
// @Tags Documents
// @Produce json
// @Param documentID path string true "Document ID"
// @Success 200 {object} map[string]any
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/{documentID}/reading/ [get]
func (h *Handler) getReading(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	documentID, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	reading, err := h.reader.ReadDocument(
		c.Request.Context(),
		documentID,
		pagination.TenantInfo{OrgID: authCtx.OrganizationID, BuID: authCtx.BusinessUnitID},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, reading)
}

func (h *Handler) getContent(c *gin.Context) {
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

	helpers.JSON(c, http.StatusOK, content)
}

func (h *Handler) getShipmentDraft(c *gin.Context) {
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

	helpers.JSON(c, http.StatusOK, draft)
}

func (h *Handler) reextractDocumentContent(c *gin.Context) {
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

	if archiveErr := h.importAssistant.ArchiveHistory(
		c.Request.Context(),
		documentID.String(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	); archiveErr != nil {
		h.eh.HandleError(c, archiveErr)
		return
	}

	c.Status(http.StatusAccepted)
}

func (h *Handler) approve(c *gin.Context) {
	h.review(c, document.ReviewDecisionApprove, "")
}

type rejectDocumentRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) reject(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(rejectDocumentRequest)
	if err := authctx.BindJSON(c, authCtx, req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.review(c, document.ReviewDecisionReject, req.Reason)
}

func (h *Handler) review(c *gin.Context, decision document.ReviewDecision, reason string) {
	authCtx := authctx.GetAuthContext(c)
	documentID, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	reviewed, err := h.service.Review(c.Request.Context(), &documentservice.ReviewRequest{
		DocumentID: documentID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		Decision: decision,
		Reason:   reason,
		Actor:    requestActorFromAuthContext(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, reviewed)
}

type setSignatureRequest struct {
	Status   document.SignatureStatus `json:"status"`
	SignedAt *int64                   `json:"signedAt"`
	Version  int64                    `json:"version"`
}

// @Summary Record whether a document is signed
// @Description Records a person's reading of a document's signature: Signed with an optional signing date, Unsigned, or Unknown. A person's answer is final; extraction no longer changes it. Requires document update.
// @ID setDocumentSignature
// @Tags Documents
// @Accept json
// @Produce json
// @Param documentID path string true "Document ID"
// @Param request body setSignatureRequest true "Signature status, signing date and the document version being changed"
// @Success 200 {object} document.Document
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 409 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/{documentID}/signature/ [post]
func (h *Handler) setSignature(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	documentID, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(setSignatureRequest)
	if err = authctx.BindJSON(c, authCtx, req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.SetSignature(
		c.Request.Context(),
		&documentservice.SetSignatureRequest{
			DocumentID: documentID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			Status:   req.Status,
			SignedAt: req.SignedAt,
			Version:  req.Version,
			Actor:    requestActorFromAuthContext(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, updated)
}

type attachToShipmentRequest struct {
	ShipmentID string `json:"shipmentId"`
}

func (h *Handler) attachToShipment(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	documentID, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(attachToShipmentRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if req.ShipmentID == "" {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError(
				"shipmentId",
				errortypes.ErrInvalid,
				"Shipment ID is required",
			),
		)
		return
	}
	if _, err = pulid.MustParse(req.ShipmentID); err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError(
				"shipmentId",
				errortypes.ErrInvalid,
				"Invalid shipment ID",
			),
		)
		return
	}

	updated, err := h.service.AttachLineageToResource(
		c.Request.Context(),
		documentID,
		"shipment",
		req.ShipmentID,
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if completeErr := h.importAssistant.CompleteHistory(
		c.Request.Context(),
		documentID.String(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	); completeErr != nil {
		h.eh.HandleError(c, completeErr)
		return
	}

	helpers.JSON(c, http.StatusOK, updated)
}

// @Summary Open the import assistant's conversation about a document
// @Description Opens, or returns, the caller's conversation with the shipment import assistant about one document. Turns are then asked on it through the assistant's turn routes, carrying the page's draft. Requires document read, assistant create, and access to the import assistant.
// @ID openImportAssistantThread
// @Tags Documents
// @Produce json
// @Param documentID path string true "Document ID"
// @Success 200 {object} serviceports.PageThread
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/{documentID}/import-assistant/thread/ [post]
func (h *Handler) openImportAssistantThread(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	documentID, err := pulid.Parse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError(
				"documentId",
				errortypes.ErrInvalid,
				"Invalid document ID",
			),
		)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	opened, err := h.pageAssistant.OpenPageThread(
		c.Request.Context(),
		&assistantports.OpenPageThreadRequest{
			TenantInfo:  pagination.FromAuthAsUser(authCtx),
			Origin:      conversation.ThreadOriginImport,
			SubjectType: agent.SubjectDocument,
			SubjectID:   documentID,
		},
		&actor,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, opened)
}

func (h *Handler) getImportAssistantHistory(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	documentID, err := pulid.Parse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError(
				"documentId",
				errortypes.ErrInvalid,
				"Invalid document ID",
			),
		)
		return
	}

	resp, err := h.importAssistant.GetHistory(
		c.Request.Context(),
		documentID.String(),
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

	helpers.JSON(c, http.StatusOK, resp)
}

func (h *Handler) extractionStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	status, err := h.extraction.ExtractionStatus(
		c.Request.Context(),
		pagination.FromAuthAsUser(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, status)
}

type setDeliveryFactsRequest struct {
	Facts   document.DeliveryFacts `json:"facts"`
	Version int64                  `json:"version"`
}

// @Summary Record what a delivery document says
// @Description Who signed, pieces, over/short/damaged, seal, in and out times, and for a receipt its amount and the charge it bills as. A receipt that names a charge the shipment does not bill raises a check on its billing queue item.
// @ID setDocumentDeliveryFacts
// @Tags Documents
// @Accept json
// @Produce json
// @Param documentID path string true "Document ID"
// @Param request body setDeliveryFactsRequest true "The facts and the document version being changed"
// @Success 200 {object} document.Document
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 409 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /documents/{documentID}/delivery-facts/ [post]
func (h *Handler) setDeliveryFacts(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	documentID, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(setDeliveryFactsRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.SetDeliveryFacts(
		c.Request.Context(),
		&documentservice.SetDeliveryFactsRequest{
			DocumentID: documentID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			Facts:   req.Facts,
			Version: req.Version,
			Actor:   requestActorFromAuthContext(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, updated)
}
