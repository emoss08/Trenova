package edihandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerDocumentRoutes(documents *gin.RouterGroup) {
	documents.POST(
		"/preview/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.previewDocument,
	)
	documents.POST(
		"/generate/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.generateDocument,
	)
}

func (h *Handler) previewDocument(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.PreviewEDIDocumentRequest)
	if err := authctx.BindJSON(c, authCtx, req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	preview, err := h.service.PreviewDocument(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, preview)
}

func (h *Handler) generateDocument(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.GenerateEDIDocumentRequest)
	if err := authctx.BindJSON(c, authCtx, req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	req.GeneratedByID = authCtx.UserID
	message, err := h.service.GenerateDocument(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, message)
}
