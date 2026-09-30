package edihandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerDocumentProfileRoutes(documentProfiles *gin.RouterGroup) {
	selectOptions := documentProfiles.Group("/select-options")
	selectOptions.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectPartnerDocumentProfileOptions,
	)
	selectOptions.GET(
		"/:profileID",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getPartnerDocumentProfileOption,
	)
	documentProfiles.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listPartnerDocumentProfiles,
	)
	documentProfiles.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createPartnerDocumentProfile,
	)
	documentProfiles.GET(
		"/:profileID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getPartnerDocumentProfile,
	)
	documentProfiles.PUT(
		"/:profileID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updatePartnerDocumentProfile,
	)
}

func (h *Handler) listPartnerDocumentProfiles(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
			return h.service.ListPartnerDocumentProfiles(
				c.Request.Context(),
				&repositories.ListEDIPartnerDocumentProfilesRequest{
					Filter: req,
					TransactionSet: edi.TransactionSet(
						helpers.QueryString(c, "transactionSet", ""),
					),
					Direction: edi.DocumentDirection(
						helpers.QueryString(c, "direction", ""),
					),
					Status:    edi.DocumentStatus(helpers.QueryString(c, "status", "")),
					PartnerID: helpers.QueryPulid(c, "partnerId"),
				},
			)
		},
	)
}

func (h *Handler) selectPartnerDocumentProfileOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
			return h.service.SelectPartnerDocumentProfileOptions(
				c.Request.Context(),
				&repositories.EDIPartnerDocumentProfileSelectOptionsRequest{
					SelectQueryRequest: req,
					TransactionSet: edi.TransactionSet(
						helpers.QueryString(c, "transactionSet", ""),
					),
					Direction: edi.DocumentDirection(
						helpers.QueryString(c, "direction", ""),
					),
					Status:    edi.DocumentStatus(helpers.QueryString(c, "status", "")),
					PartnerID: helpers.QueryPulid(c, "partnerId"),
				},
			)
		},
	)
}

func (h *Handler) getPartnerDocumentProfileOption(c *gin.Context) {
	h.getPartnerDocumentProfile(c)
}

func (h *Handler) getPartnerDocumentProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity, err := h.service.GetPartnerDocumentProfile(
		c.Request.Context(),
		repositories.GetEDIPartnerDocumentProfileByIDRequest{
			ID:         profileID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entity)
}

func (h *Handler) createPartnerDocumentProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.UpsertEDIPartnerDocumentProfileRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	created, err := h.service.UpsertPartnerDocumentProfile(
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

func (h *Handler) updatePartnerDocumentProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(ediservice.UpsertEDIPartnerDocumentProfileRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.ProfileID = profileID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	updated, err := h.service.UpsertPartnerDocumentProfile(
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
