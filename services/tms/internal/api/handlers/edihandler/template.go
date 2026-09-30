package edihandler

import (
	"context"
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerTemplateRoutes(templates *gin.RouterGroup) {
	selectOptions := templates.Group("/select-options")
	selectOptions.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectTemplateOptions,
	)
	selectOptions.GET(
		"/:templateID",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTemplateOption,
	)
	templates.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listTemplates,
	)
	templates.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createTemplate,
	)
	templates.GET(
		"/:templateID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTemplate,
	)
	templates.PUT(
		"/:templateID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updateTemplate,
	)
	templates.POST(
		"/:templateID/draft/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createDraftVersion,
	)
	templates.GET(
		"/:templateID/versions/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listTemplateVersions,
	)
	templates.GET(
		"/:templateID/versions/:versionID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTemplateVersion,
	)
	templates.PUT(
		"/:templateID/versions/:versionID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updateTemplateVersion,
	)
	templates.PUT(
		"/:templateID/versions/:versionID/segments/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.replaceTemplateSegments,
	)
	templates.GET(
		"/:templateID/versions/:versionID/script-libraries/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listTemplateScriptLibraries,
	)
	templates.PUT(
		"/:templateID/versions/:versionID/script-libraries/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.replaceTemplateScriptLibraries,
	)
	templates.POST(
		"/:templateID/versions/:versionID/validate/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.validateTemplateVersion,
	)
	templates.POST(
		"/:templateID/versions/:versionID/certify/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.certifyTemplateVersion,
	)
	templates.POST(
		"/:templateID/versions/:versionID/activate/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.activateTemplateVersion,
	)
	templates.POST(
		"/:templateID/versions/:versionID/archive/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.archiveTemplateVersion,
	)
	templates.POST(
		"/:templateID/versions/:versionID/rollback/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.rollbackTemplateVersion,
	)
}

func (h *Handler) listTemplates(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDITemplate], error) {
		return h.service.ListTemplates(
			c.Request.Context(),
			&repositories.ListEDITemplatesRequest{
				Filter:         req,
				TransactionSet: edi.TransactionSet(helpers.QueryString(c, "transactionSet", "")),
				Direction: edi.DocumentDirection(
					helpers.QueryString(c, "direction", ""),
				),
				Status: edi.TemplateStatus(helpers.QueryString(c, "status", "")),
			},
		)
	})
}

func (h *Handler) selectTemplateOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDITemplate], error) {
			return h.service.SelectTemplateOptions(
				c.Request.Context(),
				&repositories.EDITemplateSelectOptionsRequest{
					SelectQueryRequest: req,
					TransactionSet: edi.TransactionSet(
						helpers.QueryString(c, "transactionSet", ""),
					),
					Direction: edi.DocumentDirection(
						helpers.QueryString(c, "direction", ""),
					),
					Status: edi.TemplateStatus(helpers.QueryString(c, "status", "")),
				},
			)
		},
	)
}

func (h *Handler) getTemplateOption(c *gin.Context) {
	h.getTemplate(c)
}

func (h *Handler) createTemplate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.CreateEDITemplateRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	created, err := h.service.CreateTemplate(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) getTemplate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	template, err := h.service.GetTemplate(
		c.Request.Context(),
		repositories.GetEDITemplateByIDRequest{
			ID:         templateID,
			TenantInfo: tenantInfoFromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, template)
}

func (h *Handler) updateTemplate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(ediservice.UpdateEDITemplateRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TemplateID = templateID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	updated, err := h.service.UpdateTemplate(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) createDraftVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(ediservice.CreateEDITemplateDraftRequest)
	if c.Request.ContentLength > 0 {
		if err = c.ShouldBindJSON(req); err != nil {
			h.eh.HandleError(c, err)
			return
		}
	}
	req.TemplateID = templateID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	version, err := h.service.CreateDraftVersion(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, version)
}

func (h *Handler) listTemplateVersions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	versions, err := h.service.ListTemplateVersions(
		c.Request.Context(),
		repositories.ListEDITemplateVersionsRequest{
			TemplateID: templateID,
			TenantInfo: tenantInfoFromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, versions)
}

func (h *Handler) getTemplateVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, versionID, err := h.templateVersionIDs(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	version, err := h.service.GetTemplateVersion(
		c.Request.Context(),
		repositories.GetEDITemplateVersionByIDRequest{
			TemplateID: templateID,
			VersionID:  versionID,
			TenantInfo: tenantInfoFromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, version)
}

func (h *Handler) updateTemplateVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, versionID, err := h.templateVersionIDs(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(ediservice.UpdateEDITemplateVersionRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TemplateID = templateID
	req.VersionID = versionID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	version, err := h.service.UpdateDraftVersion(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, version)
}

func (h *Handler) replaceTemplateSegments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, versionID, err := h.templateVersionIDs(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(ediservice.ReplaceEDITemplateSegmentsRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TemplateID = templateID
	req.VersionID = versionID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	version, err := h.service.ReplaceDraftSegments(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, version)
}

func (h *Handler) listTemplateScriptLibraries(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, versionID, err := h.templateVersionIDs(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	libraries, err := h.service.ListTemplateScriptLibraries(
		c.Request.Context(),
		repositories.ListEDITemplateScriptLibrariesRequest{
			TemplateID: templateID,
			VersionID:  versionID,
			TenantInfo: tenantInfoFromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, libraries)
}

func (h *Handler) replaceTemplateScriptLibraries(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, versionID, err := h.templateVersionIDs(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(ediservice.ReplaceEDITemplateScriptLibrariesRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TemplateID = templateID
	req.VersionID = versionID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	version, err := h.service.ReplaceDraftScriptLibraries(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, version)
}

func (h *Handler) validateTemplateVersion(c *gin.Context) {
	req, err := h.templateActionRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	diagnostics, err := h.service.ValidateTemplateVersion(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"diagnostics": diagnostics})
}

func (h *Handler) certifyTemplateVersion(c *gin.Context) {
	h.templateVersionAction(c, h.service.CertifyTemplateVersion)
}

func (h *Handler) activateTemplateVersion(c *gin.Context) {
	h.templateVersionAction(c, h.service.ActivateTemplateVersion)
}

func (h *Handler) archiveTemplateVersion(c *gin.Context) {
	h.templateVersionAction(c, h.service.ArchiveTemplateVersion)
}

func (h *Handler) rollbackTemplateVersion(c *gin.Context) {
	h.templateVersionAction(c, h.service.RollbackTemplateVersion)
}

func (h *Handler) templateVersionAction(
	c *gin.Context,
	fn func(context.Context, *ediservice.EDIActionNotesRequest, *services.RequestActor) (*edi.EDITemplateVersion, error),
) {
	authCtx := authctx.GetAuthContext(c)
	req, err := h.templateActionRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	version, err := fn(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, version)
}

func (h *Handler) templateActionRequest(c *gin.Context) (*ediservice.EDIActionNotesRequest, error) {
	authCtx := authctx.GetAuthContext(c)
	templateID, versionID, err := h.templateVersionIDs(c)
	if err != nil {
		return nil, err
	}
	req := new(ediservice.EDIActionNotesRequest)
	if c.Request.ContentLength > 0 {
		if err = c.ShouldBindJSON(req); err != nil {
			return nil, err
		}
	}
	req.TemplateID = templateID
	req.VersionID = versionID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	return req, nil
}

func (h *Handler) templateVersionIDs(c *gin.Context) (
	templateID pulid.ID,
	versionID pulid.ID,
	err error,
) {
	templateID, err = pulid.MustParse(c.Param("templateID"))
	if err != nil {
		return pulid.Nil, pulid.Nil, err
	}
	versionID, err = pulid.MustParse(c.Param("versionID"))
	if err != nil {
		return pulid.Nil, pulid.Nil, err
	}
	return templateID, versionID, nil
}
