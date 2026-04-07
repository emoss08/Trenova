package rolehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	_ "github.com/emoss08/trenova/internal/core/domain/tenant" // import for documentation generation
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/roleservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service          *roleservice.Service
	PermissionEngine services.PermissionEngine
	ErrorHandler     *helpers.ErrorHandler
}

type Handler struct {
	service    *roleservice.Service
	permEngine services.PermissionEngine
	eh         *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service:    p.Service,
		permEngine: p.PermissionEngine,
		eh:         p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/roles")
	api.GET("/", h.list)
	api.GET("/:roleID", h.get)
	api.POST("/", h.create)
	api.PUT("/:roleID", h.update)
	api.GET("/:roleID/impact", h.getImpact)

	api.POST("/:roleID/permissions", h.addPermission)
	api.PUT("/:roleID/permissions/:permID", h.updatePermission)
	api.DELETE("/:roleID/permissions/:permID", h.deletePermission)

	api.POST("/:roleID/assignments", h.assignRole)
	api.DELETE("/assignments/:assignmentID", h.unassignRole)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:roleID", h.getOption)
}

// @Summary List roles
// @ID listRoles
// @Tags Roles
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]permission.Role]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /roles/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*permission.Role], error) {
			return h.service.ListRoles(
				c.Request.Context(),
				&repositories.ListRolesRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Get a role
// @ID getRole
// @Tags Roles
// @Produce json
// @Param roleID path string true "Role ID"
// @Success 200 {object} permission.Role
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /roles/{roleID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("roleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	role, err := h.service.GetRoleByID(
		c.Request.Context(),
		repositories.GetRoleByIDRequest{
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

	c.JSON(http.StatusOK, role)
}

// @Summary Get a role option
// @ID getRoleOption
// @Tags Roles
// @Produce json
// @Param roleID path string true "Role ID"
// @Success 200 {object} permission.Role
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /roles/select-options/{roleID} [get]
func (h *Handler) getOption(c *gin.Context) {
	h.get(c)
}

// @Summary List role options
// @ID listRoleOptions
// @Tags Roles
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]permission.Role]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /roles/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*permission.Role], error) {
			return h.service.SelectRoleOptions(c.Request.Context(), req)
		},
	)
}

// @Summary Create a role
// @ID createRole
// @Tags Roles
// @Accept json
// @Produce json
// @Param request body permission.Role true "Role payload"
// @Success 201 {object} permission.Role
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /roles/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	role := new(permission.Role)
	if err := c.ShouldBindJSON(role); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err := h.service.CreateRole(c.Request.Context(), roleservice.CreateRoleRequest{
		ActorID:        authCtx.UserID,
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		Role:           role,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, role)
}

// @Summary Update a role
// @ID updateRole
// @Tags Roles
// @Accept json
// @Produce json
// @Param roleID path string true "Role ID"
// @Param request body permission.Role true "Role payload"
// @Success 200 {object} permission.Role
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /roles/{roleID} [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	roleID, err := pulid.MustParse(c.Param("roleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	role := new(permission.Role)
	if err = c.ShouldBindJSON(role); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	role.ID = roleID

	err = h.service.UpdateRole(c.Request.Context(), roleservice.UpdateRoleRequest{
		ActorID:        authCtx.UserID,
		OrganizationID: authCtx.OrganizationID,
		Role:           role,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, role)
}

// @Summary Get role impact
// @ID getRoleImpact
// @Tags Roles
// @Produce json
// @Param roleID path string true "Role ID"
// @Success 200 {array} tenant.User
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /roles/{roleID}/impact [get]
func (h *Handler) getImpact(c *gin.Context) {
	id, err := pulid.MustParse(c.Param("roleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	users, err := h.service.GetImpactedUsers(c.Request.Context(), id)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, users)
}

// @Summary Add a role permission
// @ID addRolePermission
// @Tags Roles
// @Accept json
// @Produce json
// @Param roleID path string true "Role ID"
// @Param request body AddPermissionRequest true "Permission payload"
// @Success 201 {object} permission.ResourcePermission
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /roles/{roleID}/permissions [post]
func (h *Handler) addPermission(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	roleID, err := pulid.MustParse(c.Param("roleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req AddPermissionRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	rp := &permission.ResourcePermission{
		RoleID:     roleID,
		Resource:   req.Resource,
		Operations: req.Operations,
		DataScope:  req.DataScope,
	}

	err = h.service.CreateResourcePermission(
		c.Request.Context(),
		authCtx.UserID,
		authCtx.OrganizationID,
		rp,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, rp)
}

// @Summary Update a role permission
// @ID updateRolePermission
// @Tags Roles
// @Accept json
// @Produce json
// @Param roleID path string true "Role ID"
// @Param permID path string true "Permission ID"
// @Param request body AddPermissionRequest true "Permission payload"
// @Success 200 {object} permission.ResourcePermission
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /roles/{roleID}/permissions/{permID} [put]
func (h *Handler) updatePermission(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	roleID, err := pulid.MustParse(c.Param("roleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	permID, err := pulid.MustParse(c.Param("permID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req AddPermissionRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	rp := &permission.ResourcePermission{
		ID:         permID,
		RoleID:     roleID,
		Resource:   req.Resource,
		Operations: req.Operations,
		DataScope:  req.DataScope,
	}

	err = h.service.UpdateResourcePermission(
		c.Request.Context(),
		authCtx.UserID,
		authCtx.OrganizationID,
		rp,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, rp)
}

// @Summary Delete a role permission
// @ID deleteRolePermission
// @Tags Roles
// @Param roleID path string true "Role ID"
// @Param permID path string true "Permission ID"
// @Success 204 "No Content"
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /roles/{roleID}/permissions/{permID} [delete]
func (h *Handler) deletePermission(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	roleID, err := pulid.MustParse(c.Param("roleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	permID, err := pulid.MustParse(c.Param("permID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.service.DeleteResourcePermission(
		c.Request.Context(),
		authCtx.OrganizationID,
		permID,
		roleID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Assign a role to a user
// @ID assignRole
// @Tags Roles
// @Accept json
// @Produce json
// @Param roleID path string true "Role ID"
// @Param request body AssignRoleRequest true "Role assignment payload"
// @Success 201 {object} permission.UserRoleAssignment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /roles/{roleID}/assignments [post]
func (h *Handler) assignRole(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	roleID, err := pulid.MustParse(c.Param("roleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req AssignRoleRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	assignment := &permission.UserRoleAssignment{
		UserID:    req.UserID,
		RoleID:    roleID,
		ExpiresAt: req.ExpiresAt,
	}

	err = h.service.AssignRole(c.Request.Context(), roleservice.AssignRoleRequest{
		ActorID:        authCtx.UserID,
		OrganizationID: authCtx.OrganizationID,
		Assignment:     assignment,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, assignment)
}

// @Summary Unassign a role
// @ID unassignRole
// @Tags Roles
// @Param assignmentID path string true "Role assignment ID"
// @Success 204 "No Content"
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /roles/assignments/{assignmentID} [delete]
func (h *Handler) unassignRole(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	assignmentID, err := pulid.MustParse(c.Param("assignmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.service.UnassignRole(c.Request.Context(), roleservice.UnassignRoleRequest{
		ActorID:        authCtx.UserID,
		OrganizationID: authCtx.OrganizationID,
		AssignmentID:   assignmentID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
