package documentparsingrulehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/documentparsingrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/documentparsingruleservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *documentparsingruleservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *documentparsingruleservice.Service
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
	api := rg.Group("/document-parsing-rules")
	api.GET("/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpRead), h.listRuleSets)
	api.POST("/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpCreate), h.createRuleSet)
	api.GET("/:ruleSetID/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpRead), h.getRuleSet)
	api.PUT("/:ruleSetID/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpUpdate), h.updateRuleSet)
	api.DELETE("/:ruleSetID/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpDelete), h.deleteRuleSet)

	api.GET("/:ruleSetID/versions/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpRead), h.listVersions)
	api.POST("/:ruleSetID/versions/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpCreate), h.createVersion)
	api.GET("/versions/:versionID/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpRead), h.getVersion)
	api.PUT("/versions/:versionID/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpUpdate), h.updateVersion)
	api.POST("/versions/:versionID/publish/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpActivate), h.publishVersion)
	api.POST("/versions/:versionID/simulate/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpRead), h.simulateVersion)

	api.GET("/:ruleSetID/fixtures/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpRead), h.listFixtures)
	api.POST("/:ruleSetID/fixtures/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpCreate), h.saveFixture)
	api.GET("/fixtures/:fixtureID/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpRead), h.getFixture)
	api.PUT("/fixtures/:fixtureID/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpUpdate), h.saveFixture)
	api.DELETE("/fixtures/:fixtureID/", h.pm.RequirePermission(permission.ResourceDocumentParsingRule.String(), permission.OpDelete), h.deleteFixture)
}

func (h *Handler) listRuleSets(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	items, err := h.service.ListRuleSets(c.Request.Context(), pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}, c.Query("documentKind"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) getRuleSet(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	ruleSetID, err := pulid.MustParse(c.Param("ruleSetID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	item, err := h.service.GetRuleSet(c.Request.Context(), ruleSetID, pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) createRuleSet(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	entity := new(documentparsingrule.RuleSet)
	authctx.AddContextToRequest(authCtx, entity)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	item, err := h.service.CreateRuleSet(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) updateRuleSet(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	ruleSetID, err := pulid.MustParse(c.Param("ruleSetID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity := new(documentparsingrule.RuleSet)
	authctx.AddContextToRequest(authCtx, entity)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity.ID = ruleSetID
	item, err := h.service.UpdateRuleSet(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) deleteRuleSet(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	ruleSetID, err := pulid.MustParse(c.Param("ruleSetID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if err = h.service.DeleteRuleSet(c.Request.Context(), ruleSetID, pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}, authCtx.UserID); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listVersions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	ruleSetID, err := pulid.MustParse(c.Param("ruleSetID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	items, err := h.service.ListVersions(c.Request.Context(), ruleSetID, pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) getVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	item, err := h.service.GetVersion(c.Request.Context(), versionID, pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) createVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	ruleSetID, err := pulid.MustParse(c.Param("ruleSetID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity := new(documentparsingrule.RuleVersion)
	authctx.AddContextToRequest(authCtx, entity)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity.RuleSetID = ruleSetID
	item, err := h.service.CreateVersion(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) updateVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity := new(documentparsingrule.RuleVersion)
	authctx.AddContextToRequest(authCtx, entity)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity.ID = versionID
	item, err := h.service.UpdateVersion(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) publishVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	item, err := h.service.PublishVersion(c.Request.Context(), versionID, pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) simulateVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	versionID, err := pulid.MustParse(c.Param("versionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(struct {
		FileName            string                                `json:"fileName"`
		Text                string                                `json:"text"`
		Pages               []serviceports.DocumentParsingPage    `json:"pages"`
		ProviderFingerprint string                                `json:"providerFingerprint"`
		Baseline            *serviceports.DocumentParsingAnalysis `json:"baseline"`
	})
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	result, err := h.service.SimulateVersion(c.Request.Context(), &serviceports.DocumentParsingSimulationRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
		VersionID:           versionID,
		FileName:            req.FileName,
		Text:                req.Text,
		Pages:               req.Pages,
		ProviderFingerprint: req.ProviderFingerprint,
		Baseline:            req.Baseline,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) listFixtures(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	ruleSetID, err := pulid.MustParse(c.Param("ruleSetID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	items, err := h.service.ListFixtures(c.Request.Context(), ruleSetID, pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) getFixture(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fixtureID, err := pulid.MustParse(c.Param("fixtureID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	item, err := h.service.GetFixture(c.Request.Context(), fixtureID, pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) saveFixture(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	entity := new(documentparsingrule.Fixture)
	authctx.AddContextToRequest(authCtx, entity)
	isCreate := c.Param("fixtureID") == ""
	var fixtureID pulid.ID
	if rawFixtureID := c.Param("fixtureID"); rawFixtureID != "" {
		id, err := pulid.MustParse(rawFixtureID)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		fixtureID = id
	}
	var ruleSetID pulid.ID
	if rawRuleSetID := c.Param("ruleSetID"); rawRuleSetID != "" {
		id, err := pulid.MustParse(rawRuleSetID)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		ruleSetID = id
	}
	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if !fixtureID.IsNil() {
		entity.ID = fixtureID
	}
	if !ruleSetID.IsNil() {
		entity.RuleSetID = ruleSetID
	}
	item, err := h.service.SaveFixture(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if isCreate {
		c.JSON(http.StatusCreated, item)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) deleteFixture(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fixtureID, err := pulid.MustParse(c.Param("fixtureID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if err = h.service.DeleteFixture(c.Request.Context(), fixtureID, pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}, authCtx.UserID); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
