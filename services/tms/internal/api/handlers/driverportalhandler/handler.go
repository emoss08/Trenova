package driverportalhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/services/driverportalservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Service      *driverportalservice.Service
	Logger       *zap.Logger
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *driverportalservice.Service
	l       *zap.Logger
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		l:       p.Logger.With(zap.String("handler", "driver-portal")),
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/portal/")
	api.GET("invitations/preview", h.previewInvitation)
	api.POST("invitations/accept", h.acceptInvitation)
}

func (h *Handler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/portal/")
	api.GET("document-types/", h.listShipmentDocumentTypes)
	api.GET("loads/:shipmentID/documents/", h.listLoadDocuments)
	api.POST("loads/:shipmentID/documents/", h.uploadLoadDocument)
	api.GET("profile/document-types/", h.listWorkerDocumentTypes)
	api.GET("profile/documents/", h.listProfileDocuments)
	api.POST("profile/documents/", h.uploadProfileDocument)
	api.POST("expenses/:expenseID/receipt/", h.uploadExpenseReceipt)
}

// @Summary List worker document types for the driver portal
// @Description Returns the carrier's worker-category document types (CDL, medical card, ...) for tagging qualification-file uploads.
// @ID listPortalWorkerDocumentTypes
// @Tags DriverPortal
// @Produce json
// @Success 200 {array} driverportalservice.PortalDocumentType
// @Failure 401 {object} helpers.ProblemDetail
// @Router /portal/profile/document-types [get]
func (h *Handler) listWorkerDocumentTypes(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	types, err := h.service.WorkerDocumentTypes(c.Request.Context(), tenantInfoFrom(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, types)
}

// @Summary List the signed-in driver's qualification-file documents
// @ID listPortalProfileDocuments
// @Tags DriverPortal
// @Produce json
// @Success 200 {array} driverportalservice.PortalDocument
// @Failure 401 {object} helpers.ProblemDetail
// @Router /portal/profile/documents [get]
func (h *Handler) listProfileDocuments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	documents, err := h.service.MyProfileDocuments(c.Request.Context(), tenantInfoFrom(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, documents)
}

// @Summary Upload a qualification-file document (license, medical card, ...)
// @ID uploadPortalProfileDocument
// @Tags DriverPortal
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Document file"
// @Param documentTypeId formData string false "Document type"
// @Success 201 {object} driverportalservice.PortalDocument
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Router /portal/profile/documents [post]
func (h *Handler) uploadProfileDocument(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	file, err := c.FormFile("file")
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	uploaded, err := h.service.UploadMyProfileDocument(
		c.Request.Context(),
		tenantInfoFrom(authCtx),
		file,
		c.PostForm("documentTypeId"),
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, uploaded)
}

// @Summary List shipment document types for the driver portal
// @Description Returns the carrier's shipment-category document types (POD, BOL, ...) for tagging uploads.
// @ID listPortalShipmentDocumentTypes
// @Tags DriverPortal
// @Produce json
// @Success 200 {array} driverportalservice.PortalDocumentType
// @Failure 401 {object} helpers.ProblemDetail
// @Router /portal/document-types [get]
func (h *Handler) listShipmentDocumentTypes(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	types, err := h.service.ShipmentDocumentTypes(c.Request.Context(), tenantInfoFrom(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, types)
}

// @Summary List the signed-in driver's documents for a load
// @ID listPortalLoadDocuments
// @Tags DriverPortal
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Success 200 {array} driverportalservice.PortalDocument
// @Failure 401 {object} helpers.ProblemDetail
// @Router /portal/loads/{shipmentID}/documents [get]
func (h *Handler) listLoadDocuments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError("shipmentId", errortypes.ErrInvalid, "Invalid load"),
		)
		return
	}

	documents, err := h.service.MyLoadDocuments(
		c.Request.Context(),
		tenantInfoFrom(authCtx),
		shipmentID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, documents)
}

// @Summary Upload a document (POD, BOL photo) to one of the signed-in driver's loads
// @ID uploadPortalLoadDocument
// @Tags DriverPortal
// @Accept multipart/form-data
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param file formData file true "Document file"
// @Param documentTypeId formData string false "Document type"
// @Success 201 {object} driverportalservice.PortalDocument
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Router /portal/loads/{shipmentID}/documents [post]
func (h *Handler) uploadLoadDocument(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError("shipmentId", errortypes.ErrInvalid, "Invalid load"),
		)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	uploaded, err := h.service.UploadMyLoadDocument(
		c.Request.Context(),
		tenantInfoFrom(authCtx),
		shipmentID,
		file,
		c.PostForm("documentTypeId"),
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, uploaded)
}

// @Summary Attach a receipt photo to one of the signed-in driver's expenses
// @ID uploadPortalExpenseReceipt
// @Tags DriverPortal
// @Accept multipart/form-data
// @Produce json
// @Param expenseID path string true "Expense ID"
// @Param file formData file true "Receipt image"
// @Success 200 {object} driverpay.Expense
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Router /portal/expenses/{expenseID}/receipt [post]
func (h *Handler) uploadExpenseReceipt(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	expenseID, err := pulid.MustParse(c.Param("expenseID"))
	if err != nil {
		h.eh.HandleError(
			c,
			errortypes.NewValidationError("expenseId", errortypes.ErrInvalid, "Invalid expense"),
		)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.UploadMyExpenseReceipt(
		c.Request.Context(),
		tenantInfoFrom(authCtx),
		expenseID,
		file,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func tenantInfoFrom(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

// @Summary Preview a driver portal invitation
// @Description Returns the carrier and driver context for a pending invitation token so the accept page can render.
// @ID previewPortalInvitation
// @Tags DriverPortal
// @Produce json
// @Param token query string true "Invitation token"
// @Success 200 {object} driverportalservice.InvitationPreview
// @Failure 422 {object} helpers.ProblemDetail
// @Router /portal/invitations/preview [get]
func (h *Handler) previewInvitation(c *gin.Context) {
	preview, err := h.service.GetInvitationPreview(c.Request.Context(), c.Query("token"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, preview)
}

type acceptInvitationRequest struct {
	Token    string `json:"token"    binding:"required"`
	Password string `json:"password" binding:"required"`
	Timezone string `json:"timezone"`
}

// @Summary Accept a driver portal invitation
// @Description Creates the driver's portal account from a pending invitation token and chosen password.
// @ID acceptPortalInvitation
// @Tags DriverPortal
// @Accept json
// @Produce json
// @Param request body acceptInvitationRequest true "Invitation acceptance"
// @Success 200 {object} driverportalservice.AcceptInvitationResult
// @Failure 422 {object} helpers.ProblemDetail
// @Router /portal/invitations/accept [post]
func (h *Handler) acceptInvitation(c *gin.Context) {
	var req acceptInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.AcceptInvitation(
		c.Request.Context(),
		&driverportalservice.AcceptInvitationRequest{
			Token:    req.Token,
			Password: req.Password,
			Timezone: req.Timezone,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
