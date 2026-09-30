package edihandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ediinboundservice"
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
			&repositories.ListEDIInboundFilesRequest{
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
		repositories.GetEDIInboundFileByIDRequest{
			ID:              fileID,
			TenantInfo:      pagination.FromAuth(authCtx),
			IncludeMessages: true,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, file)
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
	c.JSON(http.StatusOK, file)
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
	c.JSON(http.StatusOK, result)
}
