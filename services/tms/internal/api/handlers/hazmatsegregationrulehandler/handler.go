package hazmatsegregationrulehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/hazmatsegregationruleservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *hazmatsegregationruleservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *hazmatsegregationruleservice.Service
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

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/hazmat-segregation-rules")
	api.GET(
		"/",
		h.pm.RequirePermission(
			permission.ResourceHazmatSegregationRule.String(),
			permission.OpRead,
		),
		h.list,
	)
	api.GET(
		"/:hazmatSegregationRuleID",
		h.pm.RequirePermission(
			permission.ResourceHazmatSegregationRule.String(),
			permission.OpRead,
		),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(
			permission.ResourceHazmatSegregationRule.String(),
			permission.OpCreate,
		),
		h.create,
	)
	api.PUT(
		"/:hazmatSegregationRuleID/",
		h.pm.RequirePermission(
			permission.ResourceHazmatSegregationRule.String(),
			permission.OpUpdate,
		),
		h.update,
	)
	api.PATCH(
		"/:hazmatSegregationRuleID/",
		h.pm.RequirePermission(
			permission.ResourceHazmatSegregationRule.String(),
			permission.OpUpdate,
		),
		h.patch,
	)
}

// @Summary List hazmat segregation rules
// @ID listHazmatSegregationRules
// @Tags Hazmat Segregation Rules
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]hazmatsegregationrule.HazmatSegregationRule]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /hazmat-segregation-rules/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*hazmatsegregationrule.HazmatSegregationRule], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListHazmatSegregationRuleRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Get a hazmat segregation rule
// @ID getHazmatSegregationRule
// @Tags Hazmat Segregation Rules
// @Produce json
// @Param hazmatSegregationRuleID path string true "Hazmat segregation rule ID"
// @Success 200 {object} hazmatsegregationrule.HazmatSegregationRule
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /hazmat-segregation-rules/{hazmatSegregationRuleID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	entityID, err := pulid.MustParse(c.Param("hazmatSegregationRuleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetHazmatSegregationRuleByIDRequest{
			ID: entityID,
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

// @Summary Create a hazmat segregation rule
// @ID createHazmatSegregationRule
// @Tags Hazmat Segregation Rules
// @Accept json
// @Produce json
// @Param request body hazmatsegregationrule.HazmatSegregationRule true "Hazmat segregation rule payload"
// @Success 201 {object} hazmatsegregationrule.HazmatSegregationRule
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /hazmat-segregation-rules/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(hazmatsegregationrule.HazmatSegregationRule)
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

// @Summary Update a hazmat segregation rule
// @ID updateHazmatSegregationRule
// @Tags Hazmat Segregation Rules
// @Accept json
// @Produce json
// @Param hazmatSegregationRuleID path string true "Hazmat segregation rule ID"
// @Param request body hazmatsegregationrule.HazmatSegregationRule true "Hazmat segregation rule payload"
// @Success 200 {object} hazmatsegregationrule.HazmatSegregationRule
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /hazmat-segregation-rules/{hazmatSegregationRuleID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	entityID, err := pulid.MustParse(c.Param("hazmatSegregationRuleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(hazmatsegregationrule.HazmatSegregationRule)
	entity.ID = entityID
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

// @Summary Patch a hazmat segregation rule
// @ID patchHazmatSegregationRule
// @Tags Hazmat Segregation Rules
// @Accept json
// @Produce json
// @Param hazmatSegregationRuleID path string true "Hazmat segregation rule ID"
// @Param request body hazmatsegregationrule.HazmatSegregationRule true "Hazmat segregation rule payload"
// @Success 200 {object} hazmatsegregationrule.HazmatSegregationRule
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /hazmat-segregation-rules/{hazmatSegregationRuleID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	entityID, err := pulid.MustParse(c.Param("hazmatSegregationRuleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetHazmatSegregationRuleByIDRequest{
			ID: entityID,
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

	if err = c.ShouldBindJSON(existing); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updatedEntity, err := h.service.Update(c.Request.Context(), existing, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}
