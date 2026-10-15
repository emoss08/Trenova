package iamhandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tenantboundary"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              iamports.IAMService
	AccessSettings       iamports.AccessSettingsService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service iamports.IAMService
	access  iamports.AccessSettingsService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		access:  p.AccessSettings,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/organizations/:id/iam/")
	h.registerIdentityProviderRoutes(api)
	h.registerSCIMRoutes(api)
	h.registerAccessPolicyRoutes(api)
	api.GET(
		"access-settings",
		h.pm.RequirePermission(permission.ResourceAccessPolicy.String(), permission.OpRead),
		h.getAccessSettings,
	)
	api.PUT(
		"access-settings",
		h.pm.RequirePermission(permission.ResourceAccessPolicy.String(), permission.OpUpdate),
		h.updateAccessSettings,
	)

	api.GET(
		"provisioning-audit",
		h.pm.RequirePermission(permission.ResourceProvisioningAudit.String(), permission.OpRead),
		h.listProvisioningAuditRecords,
	)
	api.GET(
		"auth-events",
		h.pm.RequirePermission(permission.ResourceAuthEvent.String(), permission.OpRead),
		h.listAuthEvents,
	)
	api.GET(
		"risk-decisions",
		h.pm.RequirePermission(permission.ResourceRiskDecision.String(), permission.OpRead),
		h.listRiskDecisions,
	)
	api.GET(
		"external-identities",
		h.pm.RequirePermission(permission.ResourceExternalIdentity.String(), permission.OpRead),
		h.listExternalIdentities,
	)
	api.GET(
		"mfa-authenticators",
		h.pm.RequirePermission(permission.ResourceMFAAuthenticator.String(), permission.OpRead),
		h.listMFAAuthenticators,
	)
}

func (h *Handler) registerIdentityProviderRoutes(api *gin.RouterGroup) {
	resource := permission.ResourceIdentityProvider.String()
	api.GET(
		"identity-providers",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.listIdentityProviders,
	)
	api.POST(
		"identity-providers",
		h.pm.RequirePermission(resource, permission.OpCreate),
		h.createIdentityProvider,
	)
	api.PUT(
		"identity-providers/:providerId",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.updateIdentityProvider,
	)
	api.DELETE(
		"identity-providers/:providerId",
		h.pm.RequirePermission(resource, permission.OpDelete),
		h.deleteIdentityProvider,
	)
}

func (h *Handler) registerSCIMRoutes(api *gin.RouterGroup) {
	resource := permission.ResourceSCIMDirectory.String()
	read := h.pm.RequirePermission(resource, permission.OpRead)
	create := h.pm.RequirePermission(resource, permission.OpCreate)
	update := h.pm.RequirePermission(resource, permission.OpUpdate)
	remove := h.pm.RequirePermission(resource, permission.OpDelete)

	api.GET("scim/directories", read, h.listSCIMDirectories)
	api.POST("scim/directories", create, h.createSCIMDirectory)
	api.PUT("scim/directories/:directoryId", update, h.updateSCIMDirectory)
	api.DELETE("scim/directories/:directoryId", remove, h.deleteSCIMDirectory)
	api.GET("scim/directories/:directoryId/tokens", read, h.listSCIMTokens)
	api.POST("scim/directories/:directoryId/tokens", update, h.createSCIMToken)
	api.POST("scim/tokens/:tokenId/revoke", update, h.revokeSCIMToken)
	api.GET(
		"scim/directories/:directoryId/group-role-mappings",
		read,
		h.listSCIMGroupRoleMappings,
	)
	api.POST(
		"scim/directories/:directoryId/group-role-mappings",
		update,
		h.createSCIMGroupRoleMapping,
	)
	api.PUT(
		"scim/directories/:directoryId/group-role-mappings/:mappingId",
		update,
		h.updateSCIMGroupRoleMapping,
	)
	api.DELETE(
		"scim/directories/:directoryId/group-role-mappings/:mappingId",
		update,
		h.deleteSCIMGroupRoleMapping,
	)
}

func (h *Handler) registerAccessPolicyRoutes(api *gin.RouterGroup) {
	resource := permission.ResourceAccessPolicy.String()
	api.GET(
		"access-policies",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.listAccessPolicies,
	)
	api.POST(
		"access-policies",
		h.pm.RequirePermission(resource, permission.OpCreate),
		h.createAccessPolicy,
	)
	api.PUT(
		"access-policies/:policyId",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.updateAccessPolicy,
	)
	api.DELETE(
		"access-policies/:policyId",
		h.pm.RequirePermission(resource, permission.OpDelete),
		h.deleteAccessPolicy,
	)
}

func (h *Handler) tenantInfo(c *gin.Context) (pagination.TenantInfo, bool) {
	authCtx := authctx.GetAuthContext(c)
	orgID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return pagination.TenantInfo{}, false
	}
	if orgID != authCtx.OrganizationID {
		tenantboundary.Report(c, tenantboundary.Violation{
			Source:         tenantboundary.SourcePath,
			Field:          "id",
			OrganizationID: orgID,
		})
		h.eh.HandleError(c, errortypes.NewNotFoundError("Organization not found"))
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
	req := new(iamports.IdentityProviderRequest)
	if err := authctx.BindJSON(c, authctx.GetAuthContext(c), req); err != nil {
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
	req := new(iamports.IdentityProviderRequest)
	if err := authctx.BindJSON(c, authctx.GetAuthContext(c), req); err != nil {
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
				&iamports.ListSCIMDirectoryRequest{
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
	if err := authctx.BindJSON(c, authctx.GetAuthContext(c), req); err != nil {
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
	if err := authctx.BindJSON(c, authctx.GetAuthContext(c), req); err != nil {
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
	req := new(iamports.SCIMTokenCreateRequest)
	if err := authctx.BindJSON(c, authctx.GetAuthContext(c), req); err != nil {
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
	directoryID, ok := h.pathID(c, "directoryId")
	if !ok {
		return
	}
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*iam.SCIMGroupRoleMapping], error) {
			return h.service.ListSCIMGroupRoleMappings(
				c.Request.Context(),
				&iamports.ListSCIMGroupRoleMappingsRequest{
					Filter:      req,
					DirectoryID: directoryID,
				},
			)
		},
	)
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
	if err := authctx.BindJSON(c, authctx.GetAuthContext(c), req); err != nil {
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
	if err := authctx.BindJSON(c, authctx.GetAuthContext(c), req); err != nil {
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
	if err := authctx.BindJSON(c, authctx.GetAuthContext(c), req); err != nil {
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
	if err := authctx.BindJSON(c, authctx.GetAuthContext(c), req); err != nil {
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

type accessSettingsInput struct {
	RequireRoleActivation bool  `json:"requireRoleActivation"`
	Version               int64 `json:"version"`
}

func (h *Handler) getAccessSettings(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	resp, err := h.access.Get(c.Request.Context(), tenantInfo)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) updateAccessSettings(c *gin.Context) {
	tenantInfo, ok := h.tenantInfo(c)
	if !ok {
		return
	}
	authCtx := authctx.GetAuthContext(c)
	req := new(accessSettingsInput)
	if err := authctx.BindJSON(c, authCtx, req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	resp, err := h.access.Update(c.Request.Context(), &iamports.UpdateAccessSettingsRequest{
		TenantInfo:            tenantInfo,
		Actor:                 actorutil.FromAuthContext(authCtx),
		RequireRoleActivation: req.RequireRoleActivation,
		Version:               req.Version,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
