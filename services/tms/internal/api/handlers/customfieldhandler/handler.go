package customfieldhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/customfieldservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/customfieldports"
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
			permission.OpDelete,
		),
		h.delete,
	)
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
	if err := authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	created, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusCreated, created)
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
	if err = authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, updated)
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
		customfieldports.GetCustomFieldDefinitionByIDRequest{
			ID:         definitionID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = authctx.BindJSON(c, authCtx, existing); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updatedEntity, err := h.service.Update(c.Request.Context(), existing, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, updatedEntity)
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

	err = h.service.Delete(
		c.Request.Context(),
		customfieldports.GetCustomFieldDefinitionByIDRequest{
			ID:         definitionID,
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
