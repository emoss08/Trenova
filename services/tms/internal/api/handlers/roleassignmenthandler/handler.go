package roleassignmenthandler

import (
	"net/http"

	"go.uber.org/fx"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/roleassignmentservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

type Params struct {
	fx.In

	Service      *roleassignmentservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *roleassignmentservice.Service
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/role-assignments")

	api.GET("/", h.list)
	api.GET("/:roleAssignmentID", h.get)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.list)
	selectOptions.GET("/:roleAssignmentID", h.get)
}

// @Summary List role assignments
// @ID listRoleAssignments
// @Tags Role Assignments
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param expandRoles query bool false "Expand role details"
// @Success 200 {object} pagination.Response[[]permission.UserRoleAssignment]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /role-assignments/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*permission.UserRoleAssignment], error) {
			return h.service.List(c.Request.Context(), &repositories.ListRoleAssignmentsRequest{
				Filter:      req,
				ExpandRoles: helpers.QueryBool(c, "expandRoles", false),
			},
			)
		},
	)
}

// @Summary Get a role assignment
// @ID getRoleAssignment
// @Tags Role Assignments
// @Produce json
// @Param roleAssignmentID path string true "Role assignment ID"
// @Param expandRoles query bool false "Expand role details"
// @Success 200 {object} permission.UserRoleAssignment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /role-assignments/{roleAssignmentID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	roleAssignmentID, err := pulid.MustParse(c.Param("roleAssignmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(
		c.Request.Context(),
		repositories.GetRoleAssignmentByIDRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			RoleAssignmentID: roleAssignmentID,
			ExpandRoles:      helpers.QueryBool(c, "expandRoles", false),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
