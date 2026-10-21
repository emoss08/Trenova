package edihandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerInboundFileRoutes(inboundFiles *gin.RouterGroup) {
	inboundFiles.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listInboundFiles,
	)
	inboundFiles.GET(
		"/:fileID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getInboundFile,
	)
	inboundFiles.GET(
		"/:fileID/inspect/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.inspectInboundFile,
	)
	inboundFiles.POST(
		"/:fileID/reprocess/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.reprocessInboundFile,
	)
	inboundFiles.POST(
		"/bulk-reprocess/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.bulkReprocessInboundFiles,
	)
}

func (h *Handler) listInboundFiles(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	partnerID, _ := pulid.MustParse(helpers.QueryString(c, "partnerId", ""))
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDIInboundFile], error) {
		return h.inboundService.ListInboundFiles(
			c.Request.Context(),
			&ediports.ListEDIInboundFilesRequest{
				Filter:    req,
				Status:    edi.InboundFileStatus(helpers.QueryString(c, "status", "")),
				PartnerID: partnerID,
			},
		)
	})
}

func (h *Handler) getInboundFile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fileID, err := pulid.MustParse(c.Param("fileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	file, err := h.inboundService.GetInboundFile(
		c.Request.Context(),
		ediports.GetEDIInboundFileByIDRequest{
			ID:              fileID,
			TenantInfo:      pagination.FromAuth(authCtx),
			IncludeMessages: true,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, file)
}

func (h *Handler) inspectInboundFile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fileID, err := pulid.MustParse(c.Param("fileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	inspection, err := h.inboundService.InspectInboundFile(
		c.Request.Context(),
		ediports.GetEDIInboundFileByIDRequest{
			ID:         fileID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, inspection)
}

func (h *Handler) reprocessInboundFile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fileID, err := pulid.MustParse(c.Param("fileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	file, err := h.inboundService.ProcessInboundFile(
		c.Request.Context(),
		&ediinboundservice.ProcessInboundFileRequest{
			FileID:     fileID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			Reprocess:  true,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, file)
}

func (h *Handler) bulkReprocessInboundFiles(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediinboundservice.BulkReprocessInboundFilesRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	result, err := h.inboundService.BulkReprocessInboundFiles(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, result)
}
