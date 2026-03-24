package documenttypehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/documenttypeservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *documenttypeservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *documenttypeservice.Service
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
	api := rg.Group("/document-types")
	api.GET(
		"/",
		h.pm.RequirePermission(
			permission.ResourceDocumentType.String(),
			permission.OpRead,
		),
		h.list,
	)
	api.GET(
		"/:docTypeID/",
		h.pm.RequirePermission(
			permission.ResourceDocumentType.String(),
			permission.OpRead,
		),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(
			permission.ResourceDocumentType.String(),
			permission.OpCreate,
		),
		h.create,
	)
	api.PUT(
		"/:docTypeID/",
		h.pm.RequirePermission(
			permission.ResourceDocumentType.String(),
			permission.OpUpdate,
		),
		h.update,
	)
	api.PATCH(
		"/:docTypeID/",
		h.pm.RequirePermission(
			permission.ResourceDocumentType.String(),
			permission.OpUpdate,
		),
		h.patch,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:docTypeID", h.getOption)
}

// @Summary List document types
// @ID listDocumentTypes
// @Tags Document Types
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]documenttype.DocumentType]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /document-types/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*documenttype.DocumentType], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListDocumentTypesRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Get a document type option
// @ID getDocumentTypeOption
// @Tags Document Types
// @Produce json
// @Param docTypeID path string true "Document type ID"
// @Success 200 {object} documenttype.DocumentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /document-types/select-options/{docTypeID} [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	docTypeID, err := pulid.MustParse(c.Param("docTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetDocumentTypeByIDRequest{
			ID: docTypeID,
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

// @Summary List document type options
// @ID listDocumentTypeOptions
// @Tags Document Types
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]documenttype.DocumentType]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /document-types/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*documenttype.DocumentType], error) {
			return h.service.SelectOptions(c.Request.Context(), req)
		},
	)
}

// @Summary Get a document type
// @ID getDocumentType
// @Tags Document Types
// @Produce json
// @Param docTypeID path string true "Document type ID"
// @Success 200 {object} documenttype.DocumentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /document-types/{docTypeID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	docTypeID, err := pulid.MustParse(c.Param("docTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetDocumentTypeByIDRequest{
			ID: docTypeID,
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

// @Summary Create a document type
// @ID createDocumentType
// @Tags Document Types
// @Accept json
// @Produce json
// @Param request body documenttype.DocumentType true "Document type payload"
// @Success 201 {object} documenttype.DocumentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /document-types/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(documenttype.DocumentType)
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

// @Summary Update a document type
// @ID updateDocumentType
// @Tags Document Types
// @Accept json
// @Produce json
// @Param docTypeID path string true "Document type ID"
// @Param request body documenttype.DocumentType true "Document type payload"
// @Success 200 {object} documenttype.DocumentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /document-types/{docTypeID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	docTypeID, err := pulid.MustParse(c.Param("docTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(documenttype.DocumentType)
	entity.ID = docTypeID
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

// @Summary Patch a document type
// @ID patchDocumentType
// @Tags Document Types
// @Accept json
// @Produce json
// @Param docTypeID path string true "Document type ID"
// @Param request body documenttype.DocumentType true "Partial document type payload"
// @Success 200 {object} documenttype.DocumentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /document-types/{docTypeID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	docTypeID, err := pulid.MustParse(c.Param("docTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetDocumentTypeByIDRequest{
			ID: docTypeID,
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
