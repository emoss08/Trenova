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
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

type partnerRequest struct {
	Kind                    edi.PartnerKind `json:"kind"`
	Status                  string          `json:"status"`
	Code                    string          `json:"code"`
	Name                    string          `json:"name"`
	Description             string          `json:"description"`
	InternalOrganizationID  pulid.ID        `json:"internalOrganizationId"`
	EDIConnectionID         pulid.ID        `json:"ediConnectionId"`
	CustomerID              pulid.ID        `json:"customerId"`
	DefaultTransportID      pulid.ID        `json:"defaultTransportId"`
	DefaultMappingProfileID pulid.ID        `json:"defaultMappingProfileId"`
	Timezone                string          `json:"timezone"`
	Country                 string          `json:"country"`
	ContactName             string          `json:"contactName"`
	ContactEmail            string          `json:"contactEmail"`
	ContactPhone            string          `json:"contactPhone"`
	EnabledForInbound       *bool           `json:"enabledForInbound"`
	EnabledForOutbound      *bool           `json:"enabledForOutbound"`
	Settings                map[string]any  `json:"settings"`
	Version                 int64           `json:"version"`
}

func (h *Handler) registerPartnerRoutes(partners *gin.RouterGroup) {
	partners.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listPartners,
	)
	partners.GET(
		"/:partnerID/readiness/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getPartnerReadiness,
	)
	partners.GET(
		"/select-options/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectPartnerOptions,
	)
	partners.POST(
		"/internal-pairs/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createInternalPartnerPair,
	)
	partners.GET(
		"/:partnerID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getPartner,
	)
	partners.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createPartner,
	)
	partners.PUT(
		"/:partnerID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updatePartner,
	)
	partners.GET(
		"/:partnerID/mapping-profile/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getMappingProfile,
	)
	partners.PUT(
		"/:partnerID/mapping-profile/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updateMappingProfile,
	)
	partners.DELETE(
		"/:partnerID/mapping-profile/items/:mappingItemID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.deleteMappingItem,
	)
}

func (h *Handler) listPartners(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDIPartner], error) {
		return h.service.ListPartners(
			c.Request.Context(),
			&repositories.ListEDIPartnersRequest{Filter: req},
		)
	})
}

func (h *Handler) selectPartnerOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(c, req, h.eh, func() (*pagination.ListResult[*edi.EDIPartner], error) {
		return h.service.SelectPartnerOptions(
			c.Request.Context(),
			&repositories.EDIPartnerSelectOptionsRequest{
				SelectQueryRequest: req,
				Kind:               edi.PartnerKind(helpers.QueryString(c, "kind", "")),
				EnabledForOutbound: helpers.QueryBool(c, "enabledForOutbound", false),
			},
		)
	})
}

func (h *Handler) getPartner(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	partnerID, err := pulid.MustParse(c.Param("partnerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetPartner(c.Request.Context(), repositories.GetEDIPartnerByIDRequest{
		ID:         partnerID,
		TenantInfo: pagination.FromAuth(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) createPartner(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(partnerRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := req.toEntity(true)
	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	created, err := h.service.CreatePartner(
		c.Request.Context(),
		entity,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) createInternalPartnerPair(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.CreateInternalPartnerPairRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	pair, err := h.service.CreateInternalPartnerPair(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, pair)
}

func (h *Handler) updatePartner(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	partnerID, err := pulid.MustParse(c.Param("partnerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(partnerRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := req.toEntity(false)
	entity.ID = partnerID
	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	updated, err := h.service.UpdatePartner(
		c.Request.Context(),
		entity,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (r *partnerRequest) toEntity(defaultEnabled bool) *edi.EDIPartner {
	enabledForInbound := defaultEnabled
	if r.EnabledForInbound != nil {
		enabledForInbound = *r.EnabledForInbound
	}
	enabledForOutbound := defaultEnabled
	if r.EnabledForOutbound != nil {
		enabledForOutbound = *r.EnabledForOutbound
	}

	return &edi.EDIPartner{
		Kind:                    r.Kind,
		Status:                  domaintypes.Status(r.Status),
		Code:                    r.Code,
		Name:                    r.Name,
		Description:             r.Description,
		InternalOrganizationID:  r.InternalOrganizationID,
		EDIConnectionID:         r.EDIConnectionID,
		CustomerID:              r.CustomerID,
		DefaultTransportID:      r.DefaultTransportID,
		DefaultMappingProfileID: r.DefaultMappingProfileID,
		Timezone:                r.Timezone,
		Country:                 r.Country,
		ContactName:             r.ContactName,
		ContactEmail:            r.ContactEmail,
		ContactPhone:            r.ContactPhone,
		EnabledForInbound:       enabledForInbound,
		EnabledForOutbound:      enabledForOutbound,
		Settings:                r.Settings,
		Version:                 r.Version,
	}
}

type inspectCertificateRequest struct {
	Certificate string `json:"certificate"`
}

func (h *Handler) inspectCertificate(c *gin.Context) {
	req := new(inspectCertificateRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	summary, err := h.service.InspectAS2Certificate(req.Certificate)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) getPartnerReadiness(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	partnerID, err := pulid.MustParse(c.Param("partnerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	states, err := h.service.GetPartnerReadiness(
		c.Request.Context(),
		&ediservice.GetEDIPartnerReadinessRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			PartnerIDs: []pulid.ID{partnerID},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if len(states) == 0 {
		h.eh.HandleError(c, errortypes.NewNotFoundError("EDIPartner not found"))
		return
	}
	c.JSON(http.StatusOK, states[0])
}
