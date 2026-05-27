package platformcataloghandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Registry             *platformcatalog.Registry
	EntitlementProvider  services.EntitlementProvider
	BillingProvider      services.BillingProvider
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	registry     *platformcatalog.Registry
	entitlements services.EntitlementProvider
	billing      services.BillingProvider
	eh           *helpers.ErrorHandler
	pm           *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		registry:     p.Registry,
		entitlements: p.EntitlementProvider,
		billing:      p.BillingProvider,
		eh:           p.ErrorHandler,
		pm:           p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	me := rg.Group("/me")
	me.GET("/platform-catalog", h.getMePlatformCatalog)
	me.GET("/entitlements", h.getMeEntitlements)
	me.GET("/billing", h.getMeBilling)

	admin := rg.Group("/platform-catalog")
	admin.Use(h.pm.RequirePermission(
		permission.ResourcePlatformCatalog.String(),
		permission.OpRead,
	))
	admin.GET("/products", h.listProducts)
	admin.GET("/features", h.listFeatures)
	admin.GET("/meters", h.listMeters)
	admin.GET("/validate", h.validate)
}

func (h *Handler) getMePlatformCatalog(c *gin.Context) {
	h.writeCatalog(c)
}

func (h *Handler) getMeEntitlements(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	result, err := h.entitlements.ListEntitlements(
		c.Request.Context(),
		&services.EntitlementsRequest{
			OrganizationID: authCtx.OrganizationID,
			BusinessUnitID: authCtx.BusinessUnitID,
			PrincipalType:  services.PrincipalType(authCtx.PrincipalType),
			PrincipalID:    authCtx.PrincipalID,
			UserID:         authCtx.UserID,
			APIKeyID:       authCtx.APIKeyID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) getMeBilling(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	result, err := h.billing.GetBillingSummary(
		c.Request.Context(),
		&services.BillingSummaryRequest{
			OrganizationID: authCtx.OrganizationID,
			BusinessUnitID: authCtx.BusinessUnitID,
			PrincipalType:  services.PrincipalType(authCtx.PrincipalType),
			PrincipalID:    authCtx.PrincipalID,
			UserID:         authCtx.UserID,
			APIKeyID:       authCtx.APIKeyID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) listProducts(c *gin.Context) {
	c.JSON(http.StatusOK, h.registry.ListProducts())
}

func (h *Handler) listFeatures(c *gin.Context) {
	c.JSON(http.StatusOK, h.registry.ListFeatures())
}

func (h *Handler) listMeters(c *gin.Context) {
	c.JSON(http.StatusOK, h.registry.ListMeters())
}

func (h *Handler) validate(c *gin.Context) {
	if err := h.registry.Validate(); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

func (h *Handler) writeCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"products": h.registry.ListProducts(),
		"features": h.registry.ListFeatures(),
		"meters":   h.registry.ListMeters(),
	})
}
