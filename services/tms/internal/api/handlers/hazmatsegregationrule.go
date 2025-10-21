package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	hazmatsegregationruleservice "github.com/emoss08/trenova/internal/core/services/hazmatsegregationrule"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type HazmatSegregationRuleHandlerParams struct {
	fx.In

	Service      *hazmatsegregationruleservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type HazmatSegregationRuleHandler struct {
	service *hazmatsegregationruleservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func NewHazmatSegregationRuleHandler(
	p HazmatSegregationRuleHandlerParams,
) *HazmatSegregationRuleHandler {
	return &HazmatSegregationRuleHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PM,
	}
}

func (h *HazmatSegregationRuleHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/hazmat-segregation-rules/")
	api.GET("", h.pm.RequirePermission(permission.ResourceHazmatSegregationRule, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceHazmatSegregationRule, "read"), h.get)
	api.POST(
		"",
		h.pm.RequirePermission(permission.ResourceHazmatSegregationRule, "create"),
		h.create,
	)
	api.PUT(
		":id/",
		h.pm.RequirePermission(permission.ResourceHazmatSegregationRule, "update"),
		h.update,
	)
}

func (h *HazmatSegregationRuleHandler) list(c *gin.Context) {
	pagination.Handle[*hazmatsegregationrule.HazmatSegregationRule](
		c,
		context.GetAuthContext(c),
	).
		WithErrorHandler(h.eh).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*hazmatsegregationrule.HazmatSegregationRule], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListHazmatSegregationRuleRequest{
					Filter: opts,
				},
			)
		})
}

func (h *HazmatSegregationRuleHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetHazmatSegregationRuleByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *HazmatSegregationRuleHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(hazmatsegregationrule.HazmatSegregationRule)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, entity)
	entity, err := h.service.Create(
		c.Request.Context(),
		entity,
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

func (h *HazmatSegregationRuleHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(hazmatsegregationrule.HazmatSegregationRule)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity.ID = id
	context.AddContextToRequest(authCtx, entity)
	entity, err = h.service.Update(
		c.Request.Context(),
		entity,
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
