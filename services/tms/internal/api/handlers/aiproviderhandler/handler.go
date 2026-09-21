package aiproviderhandler

import (
	"github.com/shopspring/decimal"
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              serviceports.AIProviderService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service serviceports.AIProviderService
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
	api := rg.Group("/ai-providers")
	resource := permission.ResourceAIProvider.String()

	api.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.list)
	api.GET("/catalog/", h.pm.RequirePermission(resource, permission.OpRead), h.catalog)
	api.GET("/:providerID/", h.pm.RequirePermission(resource, permission.OpRead), h.get)
	api.POST("/", h.pm.RequirePermission(resource, permission.OpCreate), h.create)
	api.PUT("/:providerID/", h.pm.RequirePermission(resource, permission.OpUpdate), h.update)
	api.DELETE("/:providerID/", h.pm.RequirePermission(resource, permission.OpDelete), h.remove)
	api.POST("/:providerID/test/", h.pm.RequirePermission(resource, permission.OpManage), h.test)
}

func requestActorFromAuthContext(authCtx *authctx.AuthContext) serviceports.RequestActor {
	return serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalType(authCtx.PrincipalType),
		PrincipalID:    authCtx.PrincipalID,
		UserID:         authCtx.UserID,
		APIKeyID:       authCtx.APIKeyID,
		BusinessUnitID: authCtx.BusinessUnitID,
		OrganizationID: authCtx.OrganizationID,
	}
}

func tenantFromAuthContext(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}
}

func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*aiprovider.Provider], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListAIProviderRequest{Filter: req},
			)
		},
	)
}

func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	providerID, err := pulid.Parse(c.Param("providerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	provider, err := h.service.GetByID(c.Request.Context(), repositories.GetAIProviderByIDRequest{
		ID:         providerID,
		TenantInfo: tenantFromAuthContext(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, provider)
}

// saveProviderRequest mirrors the service request. APIKey is a pointer so an
// omitted field leaves the stored credential untouched, which is what lets the
// edit form load without ever receiving the secret.
type saveProviderRequest struct {
	Name                 string                          `json:"name"`
	Description          string                          `json:"description"`
	Kind                 aiprovider.Kind                 `json:"kind"`
	BaseURL              string                          `json:"baseUrl"`
	Model                string                          `json:"model"`
	APIKey               *string                         `json:"apiKey"`
	AllowPrivateNetwork  bool                            `json:"allowPrivateNetwork"`
	StructuredOutputMode aiprovider.StructuredOutputMode `json:"structuredOutputMode"`
	ReasoningEffort      aiprovider.ReasoningEffort      `json:"reasoningEffort"`
	ExtraBody            map[string]any                  `json:"extraBody"`
	InputCostPerMillion  *decimal.Decimal                `json:"inputCostPerMillion"`
	OutputCostPerMillion *decimal.Decimal                `json:"outputCostPerMillion"`
	MaxTokens            int                             `json:"maxTokens"`
	Tasks                []aiprovider.Task               `json:"tasks"`
	Priority             int                             `json:"priority"`
	Trusted              bool                            `json:"trusted"`
	Enabled              bool                            `json:"enabled"`
	Version              int64                           `json:"version"`
}

func (r *saveProviderRequest) toServiceRequest(
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *serviceports.SaveAIProviderRequest {
	return &serviceports.SaveAIProviderRequest{
		ID:                   id,
		Name:                 r.Name,
		Description:          r.Description,
		Kind:                 r.Kind,
		BaseURL:              r.BaseURL,
		Model:                r.Model,
		APIKey:               r.APIKey,
		AllowPrivateNetwork:  r.AllowPrivateNetwork,
		StructuredOutputMode: r.StructuredOutputMode,
		ReasoningEffort:      r.ReasoningEffort,
		ExtraBody:            r.ExtraBody,
		InputCostPerMillion:  r.InputCostPerMillion,
		OutputCostPerMillion: r.OutputCostPerMillion,
		MaxTokens:            r.MaxTokens,
		Tasks:                r.Tasks,
		Priority:             r.Priority,
		Trusted:              r.Trusted,
		Enabled:              r.Enabled,
		Version:              r.Version,
		TenantInfo:           tenantInfo,
	}
}

func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var body saveProviderRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	created, err := h.service.Create(
		c.Request.Context(),
		body.toServiceRequest(pulid.Nil, tenantFromAuthContext(authCtx)),
		&actor,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	providerID, err := pulid.Parse(c.Param("providerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body saveProviderRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	updated, err := h.service.Update(
		c.Request.Context(),
		body.toServiceRequest(providerID, tenantFromAuthContext(authCtx)),
		&actor,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (h *Handler) remove(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	providerID, err := pulid.Parse(c.Param("providerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	if err = h.service.Delete(c.Request.Context(), repositories.DeleteAIProviderRequest{
		ID:         providerID,
		TenantInfo: tenantFromAuthContext(authCtx),
	}, &actor); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) test(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	providerID, err := pulid.Parse(c.Param("providerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.Test(c.Request.Context(), repositories.GetAIProviderByIDRequest{
		ID:         providerID,
		TenantInfo: tenantFromAuthContext(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// catalog describes the protocols, presets, and routable tasks the UI offers, so
// the form does not hardcode a list that drifts from the domain.
func (h *Handler) catalog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"kinds":   kindDescriptors(),
		"presets": Presets(),
		"tasks":   taskDescriptors(),
	})
}
