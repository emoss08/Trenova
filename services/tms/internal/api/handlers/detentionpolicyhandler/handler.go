package detentionpolicyhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/detentionpolicyservice"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *detentionpolicyservice.Service
	Engine               *detentionservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *detentionpolicyservice.Service
	engine  *detentionservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		engine:  p.Engine,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	resource := permission.ResourceDetentionPolicy.String()

	api := rg.Group("/detention-policies")
	api.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.list)
	api.GET("/:detentionPolicyID/", h.pm.RequirePermission(resource, permission.OpRead), h.get)
	api.POST("/", h.pm.RequirePermission(resource, permission.OpCreate), h.create)
	api.PUT(
		"/:detentionPolicyID/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.update,
	)
	api.DELETE(
		"/:detentionPolicyID/",
		h.pm.RequirePermission(resource, permission.OpDelete),
		h.delete,
	)

	api.POST("/preview/", h.pm.RequirePermission(resource, permission.OpRead), h.preview)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:detentionPolicyID/", h.getOption)
}

// @Summary List detention policies
// @ID listDetentionPolicies
// @Tags Detention Policies
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param status query string false "Filter by status"
// @Success 200 {object} pagination.Response[[]detention.DetentionPolicy]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention-policies/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	var customerID *pulid.ID
	if raw := helpers.QueryString(c, "customerId"); raw != "" {
		if parsed, err := pulid.MustParse(raw); err == nil {
			customerID = &parsed
		}
	}

	var locationID *pulid.ID
	if raw := helpers.QueryString(c, "locationId"); raw != "" {
		if parsed, err := pulid.MustParse(raw); err == nil {
			locationID = &parsed
		}
	}

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*detention.DetentionPolicy], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListDetentionPoliciesRequest{
					Filter:     req,
					Status:     helpers.QueryString(c, "status"),
					CustomerID: customerID,
					LocationID: locationID,
				},
			)
		},
	)
}

// @Summary Get a detention policy
// @ID getDetentionPolicy
// @Tags Detention Policies
// @Produce json
// @Param detentionPolicyID path string true "Detention policy ID"
// @Success 200 {object} detention.DetentionPolicy
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention-policies/{detentionPolicyID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	policyID, err := pulid.MustParse(c.Param("detentionPolicyID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(
		c.Request.Context(),
		&repositories.GetDetentionPolicyByIDRequest{
			DetentionPolicyID: policyID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			IncludeTiers: true,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Create a detention policy
// @ID createDetentionPolicy
// @Tags Detention Policies
// @Accept json
// @Produce json
// @Param request body detention.DetentionPolicy true "Detention policy payload"
// @Success 201 {object} detention.DetentionPolicy
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention-policies/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(detention.DetentionPolicy)
	authctx.AddContextToRequest(authCtx, entity)

	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	created, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// @Summary Update a detention policy
// @ID updateDetentionPolicy
// @Tags Detention Policies
// @Accept json
// @Produce json
// @Param detentionPolicyID path string true "Detention policy ID"
// @Param request body detention.DetentionPolicy true "Detention policy payload"
// @Success 200 {object} detention.DetentionPolicy
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention-policies/{detentionPolicyID} [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	policyID, err := pulid.MustParse(c.Param("detentionPolicyID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(detention.DetentionPolicy)
	entity.ID = policyID
	authctx.AddContextToRequest(authCtx, entity)

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

// @Summary Delete a detention policy
// @ID deleteDetentionPolicy
// @Tags Detention Policies
// @Produce json
// @Param detentionPolicyID path string true "Detention policy ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention-policies/{detentionPolicyID} [delete]
func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	policyID, err := pulid.MustParse(c.Param("detentionPolicyID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.Delete(c.Request.Context(), &repositories.GetDetentionPolicyByIDRequest{
		DetentionPolicyID: policyID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	}, authCtx.UserID); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary List detention policy options
// @ID listDetentionPolicyOptions
// @Tags Detention Policies
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]detention.DetentionPolicy]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention-policies/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*detention.DetentionPolicy], error) {
			return h.service.SelectOptions(
				c.Request.Context(),
				&repositories.DetentionPolicySelectOptionsRequest{
					SelectQueryRequest: req,
				},
			)
		},
	)
}

// @Summary Get a detention policy option
// @ID getDetentionPolicyOption
// @Tags Detention Policies
// @Produce json
// @Param detentionPolicyID path string true "Detention policy ID"
// @Success 200 {object} detention.DetentionPolicy
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention-policies/select-options/{detentionPolicyID} [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	policyID, err := pulid.MustParse(c.Param("detentionPolicyID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(
		c.Request.Context(),
		&repositories.GetDetentionPolicyByIDRequest{
			DetentionPolicyID: policyID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

type previewRequest struct {
	Policy   *detention.DetentionPolicy       `json:"policy"   binding:"required"`
	Scenario detentionservice.PreviewScenario `json:"scenario" binding:"required"`
}

// @Summary Preview what a detention policy would charge for a scenario
// @ID previewDetentionPolicy
// @Tags Detention Policies
// @Accept json
// @Produce json
// @Param request body previewRequest true "Policy and scenario"
// @Success 200 {object} detentionservice.PreviewResult
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention-policies/preview [post]
func (h *Handler) preview(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(previewRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	authctx.AddContextToRequest(authCtx, req.Policy)

	result, err := h.engine.PreviewPolicy(
		c.Request.Context(),
		req.Policy,
		req.Scenario,
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

	c.JSON(http.StatusOK, result)
}
