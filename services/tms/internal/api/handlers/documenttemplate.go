package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	documenttemplateservice "github.com/emoss08/trenova/internal/core/services/documenttemplate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type DocumentTemplateHandlerParams struct {
	fx.In

	Service      *documenttemplateservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type DocumentTemplateHandler struct {
	service      *documenttemplateservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewDocumentTemplateHandler(p DocumentTemplateHandlerParams) *DocumentTemplateHandler {
	return &DocumentTemplateHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *DocumentTemplateHandler) RegisterRoutes(rg *gin.RouterGroup) {
	templates := rg.Group("/document-templates/")
	templates.GET(
		"",
		h.pm.RequirePermission(permission.ResourceDocumentTemplate, "read"),
		h.listTemplates,
	)
	templates.GET(
		":id/",
		h.pm.RequirePermission(permission.ResourceDocumentTemplate, "read"),
		h.getTemplate,
	)
	templates.POST(
		"",
		h.pm.RequirePermission(permission.ResourceDocumentTemplate, "create"),
		h.createTemplate,
	)
	templates.PUT(
		":id/",
		h.pm.RequirePermission(permission.ResourceDocumentTemplate, "update"),
		h.updateTemplate,
	)
	templates.DELETE(
		":id/",
		h.pm.RequirePermission(permission.ResourceDocumentTemplate, "delete"),
		h.deleteTemplate,
	)
	templates.POST(
		":id/preview/",
		h.pm.RequirePermission(permission.ResourceDocumentTemplate, "read"),
		h.previewTemplate,
	)

	generated := rg.Group("/generated-documents/")
	generated.GET(
		"",
		h.pm.RequirePermission(permission.ResourceGeneratedDocument, "read"),
		h.listGeneratedDocuments,
	)
	generated.GET(
		":id/",
		h.pm.RequirePermission(permission.ResourceGeneratedDocument, "read"),
		h.getGeneratedDocument,
	)
	generated.DELETE(
		":id/",
		h.pm.RequirePermission(permission.ResourceGeneratedDocument, "delete"),
		h.deleteGeneratedDocument,
	)
	generated.GET(
		"by-reference/",
		h.pm.RequirePermission(permission.ResourceGeneratedDocument, "read"),
		h.getByReference,
	)
}

func (h *DocumentTemplateHandler) listTemplates(c *gin.Context) {
	pagination.Handle[*documenttemplate.DocumentTemplate](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*documenttemplate.DocumentTemplate], error) {
			var documentTypeID *pulid.ID
			if dtID := helpers.QueryString(c, "documentTypeId"); dtID != "" {
				id, err := pulid.MustParse(dtID)
				if err == nil {
					documentTypeID = &id
				}
			}

			var status *documenttemplate.TemplateStatus
			if s := helpers.QueryString(c, "status"); s != "" {
				st := documenttemplate.TemplateStatus(s)
				status = &st
			}

			var isDefault *bool
			if d := helpers.QueryString(c, "isDefault"); d != "" {
				val := d == "true"
				isDefault = &val
			}

			return h.service.ListTemplates(
				c.Request.Context(),
				&repositories.ListDocumentTemplateRequest{
					Filter:         opts,
					DocumentTypeID: documentTypeID,
					Status:         status,
					IsDefault:      isDefault,
					IncludeType:    helpers.QueryBool(c, "includeType"),
				},
			)
		})
}

func (h *DocumentTemplateHandler) getTemplate(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.GetTemplate(
		c.Request.Context(),
		repositories.GetDocumentTemplateByIDRequest{
			ID:          id,
			OrgID:       authCtx.OrganizationID,
			BuID:        authCtx.BusinessUnitID,
			UserID:      authCtx.UserID,
			IncludeType: helpers.QueryBool(c, "includeType"),
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *DocumentTemplateHandler) createTemplate(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(documenttemplate.DocumentTemplate)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, entity)
	entity, err := h.service.CreateTemplate(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

func (h *DocumentTemplateHandler) updateTemplate(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(documenttemplate.DocumentTemplate)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity.ID = id
	context.AddContextToRequest(authCtx, entity)

	entity, err = h.service.UpdateTemplate(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *DocumentTemplateHandler) deleteTemplate(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	err = h.service.DeleteTemplate(c.Request.Context(), repositories.GetDocumentTemplateByIDRequest{
		ID:     id,
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

type previewRequest struct {
	Data any `json:"data"`
}

func (h *DocumentTemplateHandler) previewTemplate(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	var req previewRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	tmpl, err := h.service.GetTemplate(
		c.Request.Context(),
		repositories.GetDocumentTemplateByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	pdfData, err := h.service.PreviewTemplate(c.Request.Context(), tmpl, req.Data)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "inline; filename=preview.pdf")
	c.Data(http.StatusOK, "application/pdf", pdfData)
}

func (h *DocumentTemplateHandler) listGeneratedDocuments(c *gin.Context) {
	pagination.Handle[*documenttemplate.GeneratedDocument](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*documenttemplate.GeneratedDocument], error) {
			var referenceType *string
			if rt := helpers.QueryString(c, "referenceType"); rt != "" {
				referenceType = &rt
			}

			var referenceID *pulid.ID
			if refID := helpers.QueryString(c, "referenceId"); refID != "" {
				id, err := pulid.MustParse(refID)
				if err == nil {
					referenceID = &id
				}
			}

			var status *documenttemplate.GenerationStatus
			if s := helpers.QueryString(c, "status"); s != "" {
				st := documenttemplate.GenerationStatus(s)
				status = &st
			}

			return h.service.ListGeneratedDocuments(
				c.Request.Context(),
				&repositories.ListGeneratedDocumentRequest{
					Filter:        opts,
					ReferenceType: referenceType,
					ReferenceID:   referenceID,
					Status:        status,
					IncludeType:   helpers.QueryBool(c, "includeType"),
				},
			)
		})
}

func (h *DocumentTemplateHandler) getGeneratedDocument(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.GetGeneratedDocument(
		c.Request.Context(),
		repositories.GetGeneratedDocumentByIDRequest{
			ID:          id,
			OrgID:       authCtx.OrganizationID,
			BuID:        authCtx.BusinessUnitID,
			UserID:      authCtx.UserID,
			IncludeType: helpers.QueryBool(c, "includeType"),
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *DocumentTemplateHandler) deleteGeneratedDocument(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	err = h.service.DeleteGeneratedDocument(
		c.Request.Context(),
		repositories.GetGeneratedDocumentByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *DocumentTemplateHandler) getByReference(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	refType := helpers.QueryString(c, "referenceType")
	refIDStr := helpers.QueryString(c, "referenceId")

	if refType == "" || refIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "referenceType and referenceId are required"})
		return
	}

	refID, err := pulid.MustParse(refIDStr)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	documents, err := h.service.GetGeneratedDocumentsByReference(
		c.Request.Context(),
		&repositories.GetByReferenceRequest{
			OrgID:   authCtx.OrganizationID,
			BuID:    authCtx.BusinessUnitID,
			RefType: refType,
			RefID:   refID,
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": documents})
}
