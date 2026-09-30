package edihandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ediinboundservice"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerCommunicationProfileRoutes(profiles *gin.RouterGroup) {
	selectOptions := profiles.Group("/select-options")
	selectOptions.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectCommunicationProfileOptions,
	)
	selectOptions.GET(
		"/:profileID",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getCommunicationProfileOption,
	)
	profiles.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listCommunicationProfiles,
	)
	profiles.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createCommunicationProfile,
	)
	profiles.GET(
		"/:profileID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getCommunicationProfile,
	)
	profiles.PUT(
		"/:profileID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updateCommunicationProfile,
	)
	profiles.POST(
		"/:profileID/test-connection/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.testCommunicationProfileConnection,
	)
	profiles.POST(
		"/:profileID/poll/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.pollCommunicationProfile,
	)
	profiles.POST(
		"/inspect-certificate/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.inspectCertificate,
	)
}

func (h *Handler) listCommunicationProfiles(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDICommunicationProfile], error) {
			return h.service.ListCommunicationProfiles(
				c.Request.Context(),
				&repositories.ListEDICommunicationProfilesRequest{Filter: req},
			)
		},
	)
}

func (h *Handler) selectCommunicationProfileOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDICommunicationProfile], error) {
			return h.service.SelectCommunicationProfileOptions(
				c.Request.Context(),
				&repositories.EDICommunicationProfileSelectOptionsRequest{
					SelectQueryRequest: req,
					Status:             domaintypes.Status(helpers.QueryString(c, "status", "")),
					Method:             edi.ConnectionMethod(helpers.QueryString(c, "method", "")),
					PartnerID:          helpers.QueryPulid(c, "partnerId"),
				},
			)
		},
	)
}

func (h *Handler) createCommunicationProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.UpsertEDICommunicationProfileRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	profile, err := h.service.CreateCommunicationProfile(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, profile)
}

func (h *Handler) getCommunicationProfileOption(c *gin.Context) {
	h.getCommunicationProfile(c)
}

func (h *Handler) getCommunicationProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	profile, err := h.service.GetCommunicationProfile(
		c.Request.Context(),
		repositories.GetEDICommunicationProfileByIDRequest{
			ID:         profileID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *Handler) updateCommunicationProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(ediservice.UpsertEDICommunicationProfileRequest)
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

	profile, err := h.service.UpdateCommunicationProfile(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *Handler) pollCommunicationProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	result, err := h.inboundService.PollAndProcessMailbox(
		c.Request.Context(),
		&ediinboundservice.PollMailboxRequest{
			ProfileID:  profileID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) testCommunicationProfileConnection(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	result, err := h.service.TestCommunicationProfileConnection(
		c.Request.Context(),
		&ediservice.TestCommunicationProfileConnectionRequest{
			ProfileID:  profileID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
