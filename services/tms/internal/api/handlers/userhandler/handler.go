package userhandler

import (
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/roleservice"
	"github.com/emoss08/trenova/internal/core/services/userservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *userservice.Service
	RoleService          *roleservice.Service
	Config               *config.Config
	PermissionEngine     services.PermissionEngine
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service     *userservice.Service
	roleService *roleservice.Service
	cfg         *config.Config
	permEngine  services.PermissionEngine
	eh          *helpers.ErrorHandler
	pm          *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service:     p.Service,
		roleService: p.RoleService,
		cfg:         p.Config,
		permEngine:  p.PermissionEngine,
		eh:          p.ErrorHandler,
		pm:          p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/users")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceUser.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:userID/",
		h.pm.RequirePermission(permission.ResourceUser.String(), permission.OpRead),
		h.get,
	)
	api.PUT(
		"/:userID/",
		h.pm.RequirePermission(permission.ResourceUser.String(), permission.OpUpdate),
		h.update,
	)
	api.GET(
		"/:userID/role-assignments/",
		h.pm.RequirePermission(permission.ResourceUser.String(), permission.OpRead),
		h.getRoleAssignments,
	)
	api.GET(
		"/:userID/effective-permissions/",
		h.pm.RequirePermission(permission.ResourceUser.String(), permission.OpRead),
		h.getUserEffectivePermissions,
	)
	api.PATCH(
		"/:userID/",
		h.pm.RequirePermission(permission.ResourceUser.String(), permission.OpUpdate),
		h.patch,
	)
	api.POST(
		"/bulk-update-status/",
		h.pm.RequirePermission(permission.ResourceUser.String(), permission.OpUpdate),
		h.bulkUpdateStatus,
	)
	api.GET(
		"/:userID/organization-memberships/",
		h.pm.RequirePermission(permission.ResourceUser.String(), permission.OpRead),
		h.listOrganizationMemberships,
	)
	api.PUT(
		"/:userID/organization-memberships/",
		h.pm.RequirePermission(permission.ResourceUser.String(), permission.OpUpdate),
		h.replaceOrganizationMemberships,
	)
	api.POST("/:userID/permissions/simulate/", h.simulatePermissions)

	meAPI := api.Group("/me")
	meAPI.GET("/", h.me)
	meAPI.POST("/switch-organization/", h.switchOrganization)
	meAPI.GET("/organizations/", h.getOrganizations)
	meAPI.PATCH("/settings/", h.updateMySettings)
	meAPI.POST("/change-password/", h.changeMyPassword)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:userID", h.getOption)
}

// @Summary List users
// @ID listUsers
// @Tags Users
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param includeMemberships query bool false "Include organization memberships"
// @Success 200 {object} pagination.Response[[]tenant.User]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*tenant.User], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListUsersRequest{
					Filter:             req,
					IncludeMemberships: helpers.QueryBool(c, "includeMemberships", false),
				},
			)
		},
	)
}

// @Summary Bulk update user statuses
// @ID bulkUpdateUserStatus
// @Tags Users
// @Accept json
// @Produce json
// @Param request body repositories.BulkUpdateUserStatusRequest true "Bulk status update request"
// @Success 200 {array} tenant.User
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/bulk-update-status/ [post]
func (h *Handler) bulkUpdateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(repositories.BulkUpdateUserStatusRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	results, err := h.service.BulkUpdateStatus(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, results)
}

// @Summary Get a user
// @ID getUser
// @Tags Users
// @Produce json
// @Param userID path string true "User ID"
// @Param includeMemberships query bool false "Include organization memberships"
// @Success 200 {object} tenant.User
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/{userID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity, err := h.service.GetByID(c.Request.Context(), repositories.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		IncludeMemberships: helpers.QueryBool(c, "includeMemberships", false),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
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

	entity, err := h.service.GetByID(c.Request.Context(), repositories.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		IncludeMemberships: true,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Get a user option
// @ID getUserOption
// @Tags Users
// @Produce json
// @Param userID path string true "User ID"
// @Success 200 {object} tenant.User
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/select-options/{userID} [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity, err := h.service.GetByID(c.Request.Context(), repositories.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary List user options
// @ID listUserOptions
// @Tags Users
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]tenant.User]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(c, req, h.eh, func() (*pagination.ListResult[*tenant.User], error) {
		return h.service.SelectOptions(c.Request.Context(), req)
	})
}

// @Summary Get user role assignments
// @ID getUserRoleAssignments
// @Tags Users
// @Produce json
// @Param userID path string true "User ID"
// @Success 200 {array} permission.UserRoleAssignment
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/{userID}/role-assignments/ [get]
// todo: this needs to move into the me handler because we're automatically passing in the user ID
func (h *Handler) getRoleAssignments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	assignments, err := h.roleService.GetUserRoleAssignments(
		c.Request.Context(),
		authCtx.UserID,
		authCtx.OrganizationID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, assignments)
}

// @Summary Get user effective permissions
// @ID getUserEffectivePermissions
// @Tags Users
// @Produce json
// @Param userID path string true "User ID"
// @Success 200 {object} services.EffectivePermissions
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/{userID}/effective-permissions/ [get]
func (h *Handler) getUserEffectivePermissions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	userID, err := pulid.MustParse(c.Param("userID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	perms, err := h.permEngine.GetEffectivePermissions(
		c.Request.Context(),
		userID,
		authCtx.OrganizationID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, perms)
}

// @Summary Simulate user permissions
// @ID simulateUserPermissions
// @Tags Users
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param request body SimulatePermissionsRequest true "Permission simulation request"
// @Success 200 {object} services.EffectivePermissions
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/{userID}/permissions/simulate/ [post]
func (h *Handler) simulatePermissions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	userID, err := pulid.MustParse(c.Param("userID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req SimulatePermissionsRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	perms, err := h.permEngine.SimulatePermissions(
		c.Request.Context(),
		&services.SimulatePermissionsRequest{
			UserID:         userID,
			OrganizationID: authCtx.OrganizationID,
			AddRoleIDs:     req.AddRoleIDs,
			RemoveRoleIDs:  req.RemoveRoleIDs,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, perms)
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
		authCtx.BusinessUnitID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, orgs)
}

// @Summary Update current user settings
// @ID updateCurrentUserSettings
// @Tags Users
// @Accept json
// @Produce json
// @Param request body UpdateMySettingsRequest true "Current user settings payload"
// @Success 200 {object} tenant.User
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/me/settings/ [patch]
func (h *Handler) updateMySettings(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var raw map[string]any
	if err := c.ShouldBindJSON(&raw); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err := validateMySettingsFields(raw); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	body, err := sonic.Marshal(raw)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req UpdateMySettingsRequest
	if err = sonic.Unmarshal(body, &req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.UpdateMySettings(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		userservice.UpdateMySettingsRequest{
			Timezone:      req.Timezone,
			TimeFormat:    req.TimeFormat,
			ProfilePicURL: req.ProfilePicURL,
			ThumbnailURL:  req.ThumbnailURL,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
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

	c.JSON(http.StatusOK, updated)
}

func validateMySettingsFields(raw map[string]any) error {
	me := errortypes.NewMultiError()

	for _, field := range []string{"name", "username", "emailAddress"} {
		if _, ok := raw[field]; ok {
			me.Add(field, errortypes.ErrInvalidOperation, "This field can only be changed by an administrator")
		}
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

// @Summary List user organization memberships
// @ID listUserOrganizationMemberships
// @Tags Users
// @Produce json
// @Param userID path string true "User ID"
// @Success 200 {array} tenant.OrganizationMembership
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/{userID}/organization-memberships/ [get]
func (h *Handler) listOrganizationMemberships(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	userID, err := pulid.MustParse(c.Param("userID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	memberships, err := h.service.ListOrganizationMemberships(
		c.Request.Context(),
		userID,
		authCtx.BusinessUnitID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, memberships)
}

// @Summary Replace user organization memberships
// @ID replaceUserOrganizationMemberships
// @Tags Users
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param request body ReplaceOrganizationMembershipsRequest true "Organization membership replacement request"
// @Success 200 {array} tenant.OrganizationMembership
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/{userID}/organization-memberships/ [put]
func (h *Handler) replaceOrganizationMemberships(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	userID, err := pulid.MustParse(c.Param("userID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req ReplaceOrganizationMembershipsRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	organizationIDs := make([]pulid.ID, 0, len(req.OrganizationIDs))
	for _, id := range req.OrganizationIDs {
		orgID, parseErr := pulid.MustParse(id)
		if parseErr != nil {
			h.eh.HandleError(c, parseErr)
			return
		}
		organizationIDs = append(organizationIDs, orgID)
	}

	updatedMemberships, err := h.service.ReplaceOrganizationMemberships(
		c.Request.Context(),
		authCtx.UserID,
		userID,
		authCtx.OrganizationID,
		authCtx.BusinessUnitID,
		organizationIDs,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedMemberships)
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
	sessionIDStr, err := c.Cookie(h.cfg.Security.Session.Name)
	if err != nil || sessionIDStr == "" {
		h.eh.HandleError(c, errortypes.NewAuthenticationError("Session not found"))
		return
	}

	sessionID, err := pulid.MustParse(sessionIDStr)
	if err != nil {
		h.eh.HandleError(c, errortypes.NewAuthenticationError("Invalid session ID"))
		return
	}

	var req SwitchOrganizationRequest
	if err = c.ShouldBindJSON(&req); err != nil {
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
		repositories.SwitchOrganizationRequest{
			SessionID:      sessionID,
			OrganizationID: orgID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary Patch a user
// @ID patchUser
// @Tags Users
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param request body tenant.User true "User payload"
// @Success 200 {object} tenant.User
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/{userID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	userID, err := pulid.MustParse(c.Param("userID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.GetByID(c.Request.Context(), repositories.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			UserID: userID,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
		},
		IncludeMemberships: true,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = c.ShouldBindJSON(existing); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.Update(c.Request.Context(), existing, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Update a user
// @ID updateUser
// @Tags Users
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param request body tenant.User true "User payload"
// @Success 200 {object} tenant.User
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /users/{userID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	userID, err := pulid.MustParse(c.Param("userID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(tenant.User)
	entity.ID = userID
	entity.CurrentOrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}
