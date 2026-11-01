package userhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/responseguard"
	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service      *userservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *userservice.Service
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/users")

	meAPI := api.Group("/me", responseguard.SelfScoped())
	meAPI.GET("/", h.me)
	meAPI.POST("/switch-organization/", h.switchOrganization)
	meAPI.GET("/organizations/", h.getOrganizations)
	meAPI.POST("/profile-picture/", h.uploadProfilePicture)
	meAPI.DELETE("/profile-picture/", h.deleteProfilePicture)
	meAPI.POST("/change-password/", h.changeMyPassword)
	api.GET("/:userID/profile-picture/", h.getProfilePictureURL)
}

// @Summary Get current user
// @ID getCurrentUser
// @Tags Users
// @Produce json
// @Success 200 {object} tenant.User
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/ [get]
func (h *Handler) me(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity, err := h.service.GetByID(c.Request.Context(), tenantports.GetUserByIDRequest{
		TenantInfo:         pagination.FromAuthAsUser(authCtx),
		IncludeMemberships: true,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	respondWithSelf(c, entity)
}

// @Summary List current user organizations
// @ID listCurrentUserOrganizations
// @Tags Users
// @Produce json
// @Success 200 {array} services.OrgSummary
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/organizations/ [get]
func (h *Handler) getOrganizations(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	orgs, err := h.service.GetOrganizations(
		c.Request.Context(),
		authCtx.UserID,
		authCtx.OrganizationID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, orgs)
}

// @Summary Upload current user profile picture
// @ID uploadCurrentUserProfilePicture
// @Tags Users
// @Accept mpfd
// @Produce json
// @Param file formData file true "Profile picture file"
// @Success 200 {object} tenant.User
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/profile-picture/ [post]
func (h *Handler) uploadProfilePicture(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	file, err := c.FormFile("file")
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.UploadProfilePicture(c.Request.Context(), pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}, file)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	respondWithSelf(c, updated)
}

// @Summary Delete current user profile picture
// @ID deleteCurrentUserProfilePicture
// @Tags Users
// @Produce json
// @Success 200 {object} tenant.User
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/profile-picture/ [delete]
func (h *Handler) deleteProfilePicture(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	updated, err := h.service.DeleteProfilePicture(c.Request.Context(), pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	respondWithSelf(c, updated)
}

// @Summary Get user profile picture URL
// @ID getUserProfilePictureURL
// @Tags Users
// @Produce json
// @Param userID path string true "User ID"
// @Param variant query string false "Image variant"
// @Success 200 {object} userhandler.ProfilePictureURLResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/{userID}/profile-picture/ [get]
func (h *Handler) getProfilePictureURL(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	userID, err := pulid.MustParse(c.Param("userID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	url, err := h.service.GetProfilePictureURL(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		userID,
		c.Query("variant"),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, ProfilePictureURLResponse{URL: url})
}

// @Summary Change current user password
// @ID changeCurrentUserPassword
// @Tags Users
// @Accept json
// @Produce json
// @Param request body ChangeMyPasswordRequest true "Password change payload"
// @Success 200 {object} tenant.User
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/change-password/ [post]
func (h *Handler) changeMyPassword(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req ChangeMyPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.ChangeMyPassword(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		userservice.ChangeMyPasswordRequest{
			CurrentPassword: req.CurrentPassword,
			NewPassword:     req.NewPassword,
			ConfirmPassword: req.ConfirmPassword,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	respondWithSelf(c, updated)
}

// @Summary Switch current organization
// @ID switchUserOrganization
// @Tags Users
// @Accept json
// @Produce json
// @Param request body SwitchOrganizationRequest true "Organization switch request"
// @Success 200 {object} gin.H
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Router /users/me/switch-organization/ [post]
func (h *Handler) switchOrganization(c *gin.Context) {
	sessionID := authctx.GetAuthContext(c).SessionID
	if sessionID.IsNil() {
		h.eh.HandleError(c, errortypes.NewAuthenticationError("Session not found"))
		return
	}

	var req SwitchOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	orgID, err := pulid.MustParse(req.OrganizationID)
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"organizationId",
			errortypes.ErrInvalid,
			"Invalid organization ID",
		))
		return
	}

	resp, err := h.service.SwitchOrganization(
		c.Request.Context(),
		tenantports.SwitchOrganizationRequest{
			SessionID:      sessionID,
			OrganizationID: orgID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, resp)
}
