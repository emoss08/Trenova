package iamhandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              services.IAMService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service services.IAMService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/organizations/:id/iam/")
	readOrg := h.pm.RequirePermission(permission.ResourceOrganization.String(), permission.OpRead)
	updateOrg := h.pm.RequirePermission(
		permission.ResourceOrganization.String(),
		permission.OpUpdate,
	)
	api.GET(
		"identity-providers",
		readOrg,
		h.listIdentityProviders,
	)
	api.POST(
		"identity-providers",
		updateOrg,
		h.createIdentityProvider,
	)
	api.PUT(
		"identity-providers/:providerId",
		updateOrg,
		h.updateIdentityProvider,
	)
	api.DELETE(
		"identity-providers/:providerId",
		updateOrg,
		h.deleteIdentityProvider,
	)

	api.GET("scim/directories", readOrg, h.listSCIMDirectories)
	api.POST("scim/directories", updateOrg, h.createSCIMDirectory)
	api.PUT("scim/directories/:directoryId", updateOrg, h.updateSCIMDirectory)
	api.DELETE("scim/directories/:directoryId", updateOrg, h.deleteSCIMDirectory)
	api.GET("scim/directories/:directoryId/tokens", readOrg, h.listSCIMTokens)
	api.POST("scim/directories/:directoryId/tokens", updateOrg, h.createSCIMToken)
	api.POST("scim/tokens/:tokenId/revoke", updateOrg, h.revokeSCIMToken)
	api.GET(
		"scim/directories/:directoryId/group-role-mappings",
		readOrg,
		h.listSCIMGroupRoleMappings,
	)
	api.POST(
		"scim/directories/:directoryId/group-role-mappings",
		updateOrg,
		h.createSCIMGroupRoleMapping,
	)
	api.PUT(
		"scim/directories/:directoryId/group-role-mappings/:mappingId",
		updateOrg,
		h.updateSCIMGroupRoleMapping,
	)
	api.DELETE(
		"scim/directories/:directoryId/group-role-mappings/:mappingId",
		updateOrg,
		h.deleteSCIMGroupRoleMapping,
	)
	api.GET("provisioning-audit", readOrg, h.listProvisioningAuditRecords)

	api.GET("access-policies", readOrg, h.listAccessPolicies)
	api.POST("access-policies", updateOrg, h.createAccessPolicy)
	api.PUT("access-policies/:policyId", updateOrg, h.updateAccessPolicy)
	api.DELETE("access-policies/:policyId", updateOrg, h.deleteAccessPolicy)

	api.GET("auth-events", readOrg, h.listAuthEvents)
	api.GET("risk-decisions", readOrg, h.listRiskDecisions)
	api.GET("external-identities", readOrg, h.listExternalIdentities)
	api.GET("mfa-authenticators", readOrg, h.listMFAAuthenticators)
}

func (h *Handler) tenantInfo(c *gin.Context) (pagination.TenantInfo, bool) {
	authCtx := authctx.GetAuthContext(c)
	orgID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return pagination.TenantInfo{}, false
	}
	return pagination.TenantInfo{
		OrgID:  orgID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}, true
}

func (h *Handler) pathID(c *gin.Context, name string) (pulid.ID, bool) {
	id, err := pulid.MustParse(c.Param(name))
	if err != nil {
		h.eh.HandleError(c, err)
		return "", false
	}
	return id, true
}

func queryLimit(c *gin.Context) int {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil {
		return 100
	}
	return limit
}

func (h *Handler) listIdentityProviders(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	resp, err := h.service.ListIdentityProviders(c.Request.Context(), tenantInfo)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) createIdentityProvider(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	req := new(services.IdentityProviderRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.CreateIdentityProvider(c.Request.Context(), tenantInfo, req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) updateIdentityProvider(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	id, ok := h.pathID(c, "providerId")
	if !ok {
		return
	}
	req := new(services.IdentityProviderRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.UpdateIdentityProvider(c.Request.Context(), tenantInfo, id, req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) deleteIdentityProvider(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	id, ok := h.pathID(c, "providerId")
	if !ok {
		return
	}
	if err := h.service.DeleteIdentityProvider(c.Request.Context(), tenantInfo, id); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listSCIMDirectories(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*iam.SCIMDirectory], error) {
			return h.service.ListSCIMDirectories(
				c.Request.Context(),
				&repositories.ListSCIMDirectoryRequest{
					Filter: req,
				},
			)
		},
	)
}

func (h *Handler) createSCIMDirectory(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	req := new(iam.SCIMDirectory)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.CreateSCIMDirectory(c.Request.Context(), tenantInfo, req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) updateSCIMDirectory(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	id, ok := h.pathID(c, "directoryId")
	if !ok {
		return
	}
	req := new(iam.SCIMDirectory)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.UpdateSCIMDirectory(c.Request.Context(), tenantInfo, id, req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) deleteSCIMDirectory(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	id, ok := h.pathID(c, "directoryId")
	if !ok {
		return
	}
	if err := h.service.DeleteSCIMDirectory(c.Request.Context(), tenantInfo, id); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listSCIMTokens(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	directoryID, ok := h.pathID(c, "directoryId")
	if !ok {
		return
	}
	resp, err := h.service.ListSCIMTokens(c.Request.Context(), tenantInfo.OrgID, directoryID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) createSCIMToken(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	directoryID, ok := h.pathID(c, "directoryId")
	if !ok {
		return
	}
	req := new(services.SCIMTokenCreateRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.CreateSCIMToken(c.Request.Context(), tenantInfo.OrgID, directoryID, req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) revokeSCIMToken(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	tokenID, ok := h.pathID(c, "tokenId")
	if !ok {
		return
	}
	resp, err := h.service.RevokeSCIMToken(c.Request.Context(), tenantInfo.OrgID, tokenID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) listSCIMGroupRoleMappings(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	directoryID, ok := h.pathID(c, "directoryId")
	if !ok {
		return
	}
	resp, err := h.service.ListSCIMGroupRoleMappings(c.Request.Context(), tenantInfo, directoryID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) createSCIMGroupRoleMapping(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	directoryID, ok := h.pathID(c, "directoryId")
	if !ok {
		return
	}
	req := new(iam.SCIMGroupRoleMapping)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.CreateSCIMGroupRoleMapping(
		c.Request.Context(),
		tenantInfo,
		directoryID,
		req,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) updateSCIMGroupRoleMapping(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	id, ok := h.pathID(c, "mappingId")
	if !ok {
		return
	}
	req := new(iam.SCIMGroupRoleMapping)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.UpdateSCIMGroupRoleMapping(c.Request.Context(), tenantInfo, id, req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) deleteSCIMGroupRoleMapping(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	id, ok := h.pathID(c, "mappingId")
	if !ok {
		return
	}
	if err := h.service.DeleteSCIMGroupRoleMapping(
		c.Request.Context(),
		tenantInfo,
		id,
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listProvisioningAuditRecords(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	var directoryID pulid.ID
	if rawDirectoryID := c.Query("directoryId"); rawDirectoryID != "" {
		parsed, err := pulid.MustParse(rawDirectoryID)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		directoryID = parsed
	}
	resp, err := h.service.ListProvisioningAuditRecords(
		c.Request.Context(),
		tenantInfo.OrgID,
		directoryID,
		queryLimit(c),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) listAccessPolicies(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	resp, err := h.service.ListAccessPolicies(c.Request.Context(), tenantInfo)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) createAccessPolicy(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	req := new(iam.AccessPolicy)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.CreateAccessPolicy(c.Request.Context(), tenantInfo, req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) updateAccessPolicy(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	id, ok := h.pathID(c, "policyId")
	if !ok {
		return
	}
	req := new(iam.AccessPolicy)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.service.UpdateAccessPolicy(c.Request.Context(), tenantInfo, id, req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) deleteAccessPolicy(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	id, ok := h.pathID(c, "policyId")
	if !ok {
		return
	}
	if err := h.service.DeleteAccessPolicy(c.Request.Context(), tenantInfo, id); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listAuthEvents(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	resp, err := h.service.ListAuthEvents(c.Request.Context(), tenantInfo.OrgID, queryLimit(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) listRiskDecisions(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	resp, err := h.service.ListRiskDecisions(c.Request.Context(), tenantInfo.OrgID, queryLimit(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) listExternalIdentities(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	resp, err := h.service.ListExternalIdentities(c.Request.Context(), tenantInfo)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) listMFAAuthenticators(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	resp, err := h.service.ListMFAAuthenticators(
		c.Request.Context(),
		tenantInfo.OrgID,
		queryLimit(c),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
