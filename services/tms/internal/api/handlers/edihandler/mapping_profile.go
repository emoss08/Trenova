package edihandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerMappingProfileRoutes(mappingProfiles *gin.RouterGroup) {
	selectOptions := mappingProfiles.Group("/select-options")
	selectOptions.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectMappingProfileOptions,
	)
	selectOptions.GET(
		"/:profileID",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getMappingProfileOption,
	)
	mappingProfiles.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listMappingProfiles,
	)
	mappingProfiles.GET(
		"/:profileID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getMappingProfileByID,
	)
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

	profile, err := h.service.GetMappingProfile(
		c.Request.Context(),
		repositories.GetMappingProfileRequest{
			PartnerID:  partnerID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
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

	items, err := h.service.SaveMappingProfile(
		c.Request.Context(),
		&repositories.SaveMappingItemsRequest{
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

	c.JSON(http.StatusOK, items)
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

	err = h.service.DeleteMappingItem(
		c.Request.Context(),
		repositories.DeleteMappingItemRequest{
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

func (h *Handler) listMappingProfiles(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDIMappingProfile], error) {
		return h.service.ListMappingProfiles(
			c.Request.Context(),
			&repositories.ListEDIMappingProfilesRequest{Filter: req},
		)
	})
}

func (h *Handler) selectMappingProfileOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIMappingProfile], error) {
			return h.service.SelectMappingProfileOptions(
				c.Request.Context(),
				&repositories.EDIMappingProfileSelectOptionsRequest{
					SelectQueryRequest: req,
					PartnerID:          helpers.QueryPulid(c, "partnerId"),
				},
			)
		},
	)
}

func (h *Handler) getMappingProfileOption(c *gin.Context) {
	h.getMappingProfileByID(c)
}

func (h *Handler) getMappingProfileByID(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	profile, err := h.service.GetMappingProfileByID(
		c.Request.Context(),
		repositories.GetMappingProfileByIDRequest{
			ProfileID:  profileID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
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

	items, err := h.service.SaveMappingProfileItems(
		c.Request.Context(),
		&repositories.SaveMappingProfileItemsRequest{
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

	c.JSON(http.StatusOK, items)
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

	err = h.service.DeleteMappingProfileItem(
		c.Request.Context(),
		repositories.DeleteMappingProfileItemRequest{
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
