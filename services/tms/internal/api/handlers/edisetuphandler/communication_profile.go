package edisetuphandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundpoll"
	"github.com/emoss08/trenova/internal/modules/edi/app/edisetup"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerCommunicationProfileRoutes(profiles *gin.RouterGroup) {
	profiles.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createCommunicationProfile,
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

func (h *Handler) createCommunicationProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(edisetup.UpsertEDICommunicationProfileRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	profile, err := h.setup.CreateCommunicationProfile(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusCreated, profile)
}

func (h *Handler) updateCommunicationProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(edisetup.UpsertEDICommunicationProfileRequest)
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

	profile, err := h.setup.UpdateCommunicationProfile(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, profile)
}

func (h *Handler) pollCommunicationProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	result, err := h.pollService.PollAndProcessMailbox(
		c.Request.Context(),
		&ediinboundpoll.PollMailboxRequest{
			ProfileID:  profileID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) testCommunicationProfileConnection(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	result, err := h.setup.TestCommunicationProfileConnection(
		c.Request.Context(),
		&edisetup.TestCommunicationProfileConnectionRequest{
			ProfileID:  profileID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, result)
}
