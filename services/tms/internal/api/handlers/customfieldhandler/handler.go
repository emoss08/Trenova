package customfieldhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/customfieldservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *customfieldservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *customfieldservice.Service
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
	api := rg.Group("/custom-fields")

	definitions := api.Group("/definitions")
	definitions.GET(
		"/",
		h.pm.RequirePermission(
			permission.ResourceCustomFieldDefinition.String(),
			permission.OpRead,
		),
		h.list,
	)
	definitions.GET(
		"/:definitionID/",
		h.pm.RequirePermission(
			permission.ResourceCustomFieldDefinition.String(),
			permission.OpRead,
		),
		h.get,
	)
	definitions.POST(
		"/",
		h.pm.RequirePermission(
			permission.ResourceCustomFieldDefinition.String(),
			permission.OpCreate,
		),
		h.create,
	)
	definitions.PUT(
		"/:definitionID/",
		h.pm.RequirePermission(
			permission.ResourceCustomFieldDefinition.String(),
			permission.OpUpdate,
		),
		h.update,
	)
	definitions.PATCH(
		"/:definitionID/",
		h.pm.RequirePermission(
			permission.ResourceCustomFieldDefinition.String(),
			permission.OpUpdate,
		),
		h.patch,
	)
	definitions.DELETE(
		"/:definitionID/",
		h.pm.RequirePermission(
			permission.ResourceCustomFieldDefinition.String(),
			permission.OpUpdate,
		),
		h.delete,
	)

	api.GET("/resource-types/", h.getResourceTypes)
	api.GET(
		"/resources/:resourceType/",
		h.pm.RequirePermission(
			permission.ResourceCustomFieldDefinition.String(),
			permission.OpRead,
		),
		h.getByResourceType,
	)
}

// @Summary List custom field definitions
// @ID listCustomFieldDefinitions
// @Tags Custom Fields
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param resourceType query string false "Filter by resource type"
// @Success 200 {object} pagination.Response[[]customfield.CustomFieldDefinition]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /custom-fields/definitions/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*customfield.CustomFieldDefinition], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListCustomFieldDefinitionsRequest{
					Filter:       req,
					ResourceType: helpers.QueryString(c, "resourceType"),
				},
			)
		},
	)
}

// @Summary Get a custom field definition
// @ID getCustomFieldDefinition
// @Tags Custom Fields
// @Produce json
// @Param definitionID path string true "Custom field definition ID"
// @Success 200 {object} customfield.CustomFieldDefinition
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /custom-fields/definitions/{definitionID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	definitionID, err := pulid.MustParse(c.Param("definitionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetCustomFieldDefinitionByIDRequest{
			ID: definitionID,
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

// @Summary Create a custom field definition
// @ID createCustomFieldDefinition
// @Tags Custom Fields
// @Accept json
// @Produce json
// @Param request body customfield.CustomFieldDefinition true "Custom field definition payload"
// @Success 201 {object} customfield.CustomFieldDefinition
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /custom-fields/definitions/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(customfield.CustomFieldDefinition)
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

// @Summary Update a custom field definition
// @ID updateCustomFieldDefinition
// @Tags Custom Fields
// @Accept json
// @Produce json
// @Param definitionID path string true "Custom field definition ID"
// @Param request body customfield.CustomFieldDefinition true "Custom field definition payload"
// @Success 200 {object} customfield.CustomFieldDefinition
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /custom-fields/definitions/{definitionID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	definitionID, err := pulid.MustParse(c.Param("definitionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(customfield.CustomFieldDefinition)
	entity.ID = definitionID
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

// @Summary Patch a custom field definition
// @ID patchCustomFieldDefinition
// @Tags Custom Fields
// @Accept json
// @Produce json
// @Param definitionID path string true "Custom field definition ID"
// @Param request body customfield.CustomFieldDefinition true "Custom field definition payload"
// @Success 200 {object} customfield.CustomFieldDefinition
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /custom-fields/definitions/{definitionID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	definitionID, err := pulid.MustParse(c.Param("definitionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetCustomFieldDefinitionByIDRequest{
			ID: definitionID,
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

// @Summary Delete a custom field definition
// @ID deleteCustomFieldDefinition
// @Tags Custom Fields
// @Param definitionID path string true "Custom field definition ID"
// @Success 204 "No Content"
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /custom-fields/definitions/{definitionID}/ [delete]
func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	definitionID, err := pulid.MustParse(c.Param("definitionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.service.Delete(c.Request.Context(), repositories.GetCustomFieldDefinitionByIDRequest{
		ID: definitionID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	}, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary List supported custom field resource types
// @ID listCustomFieldResourceTypes
// @Tags Custom Fields
// @Produce json
// @Success 200 {object} gin.H
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Router /custom-fields/resource-types/ [get]
func (h *Handler) getResourceTypes(c *gin.Context) {
	resourceTypes := h.service.GetSupportedResourceTypes()
	c.JSON(http.StatusOK, gin.H{"resourceTypes": resourceTypes})
}

// @Summary List active custom field definitions by resource type
// @ID getCustomFieldsByResourceType
// @Tags Custom Fields
// @Produce json
// @Param resourceType path string true "Custom field resource type"
// @Success 200 {array} customfield.CustomFieldDefinition
// @Failure 400 {object} gin.H
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /custom-fields/resources/{resourceType}/ [get]
func (h *Handler) getByResourceType(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	resourceType := c.Param("resourceType")

	if !customfield.IsResourceTypeSupported(resourceType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported resource type"})
		return
	}

	definitions, err := h.service.GetActiveByResourceType(
		c.Request.Context(),
		repositories.GetActiveByResourceTypeRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			ResourceType: resourceType,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, definitions)
}
