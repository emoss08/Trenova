package edisetuphandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerMappingProfileRoutes(mappingProfiles *gin.RouterGroup) {
	mappingProfiles.PUT(
		"/:profileID/items/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updateMappingProfileItems,
	)
	mappingProfiles.DELETE(
		"/:profileID/items/:mappingItemID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.deleteMappingProfileItem,
	)
}

func (h *Handler) getMappingProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	partnerID, err := pulid.MustParse(c.Param("partnerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	profile, err := h.setup.GetMappingProfile(
		c.Request.Context(),
		edipartnerports.GetMappingProfileRequest{
			PartnerID:  partnerID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, profile)
}

func (h *Handler) updateMappingProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	partnerID, err := pulid.MustParse(c.Param("partnerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := struct {
		Items []*edi.EDIMappingProfileItem `json:"items"`
	}{}
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	items, err := h.setup.SaveMappingProfile(
		c.Request.Context(),
		&edipartnerports.SaveMappingItemsRequest{
			PartnerID:  partnerID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			ActorID:    authCtx.UserID,
			Items:      req.Items,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, items)
}

func (h *Handler) deleteMappingItem(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	partnerID, err := pulid.MustParse(c.Param("partnerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	mappingItemID, err := pulid.MustParse(c.Param("mappingItemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.setup.DeleteMappingItem(
		c.Request.Context(),
		&edipartnerports.DeleteMappingItemRequest{
			PartnerID:     partnerID,
			MappingItemID: mappingItemID,
			TenantInfo:    pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) updateMappingProfileItems(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := struct {
		Items []*edi.EDIMappingProfileItem `json:"items"`
	}{}
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	items, err := h.setup.SaveMappingProfileItems(
		c.Request.Context(),
		&edipartnerports.SaveMappingProfileItemsRequest{
			ProfileID:  profileID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			ActorID:    authCtx.UserID,
			Items:      req.Items,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, items)
}

func (h *Handler) deleteMappingProfileItem(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	mappingItemID, err := pulid.MustParse(c.Param("mappingItemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.setup.DeleteMappingProfileItem(
		c.Request.Context(),
		&edipartnerports.DeleteMappingProfileItemRequest{
			ProfileID:     profileID,
			MappingItemID: mappingItemID,
			TenantInfo:    pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
