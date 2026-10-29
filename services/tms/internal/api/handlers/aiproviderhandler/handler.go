package aiproviderhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              aiproviderports.AIProviderService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	Config               *config.Config `optional:"true"`
}

type Handler struct {
	service aiproviderports.AIProviderService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
	ai      *config.AIConfig
}

func New(p Params) *Handler {
	var ai *config.AIConfig
	if p.Config != nil {
		ai = p.Config.GetAIConfig()
	}

	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
		ai:      ai,
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

func requestActorFromAuthContext(authCtx *authctx.AuthContext) permissionports.RequestActor {
	return permissionports.RequestActor{
		PrincipalType:  kernelports.PrincipalType(authCtx.PrincipalType),
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
				&aiproviderports.ListAIProviderRequest{Filter: req},
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

	provider, err := h.service.GetByID(
		c.Request.Context(),
		aiproviderports.GetAIProviderByIDRequest{
			ID:         providerID,
			TenantInfo: tenantFromAuthContext(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, provider)
}

// saveProviderRequest mirrors the service request. APIKey is a pointer so an
// omitted field leaves the stored credential untouched, which is what lets the
// edit form load without ever receiving the secret.
type saveProviderRequest struct {
	Name                     string                          `json:"name"`
	Description              string                          `json:"description"`
	Kind                     aiprovider.Kind                 `json:"kind"`
	BaseURL                  string                          `json:"baseUrl"`
	Model                    string                          `json:"model"`
	APIKey                   *string                         `json:"apiKey"`
	AllowPrivateNetwork      bool                            `json:"allowPrivateNetwork"`
	StructuredOutputMode     aiprovider.StructuredOutputMode `json:"structuredOutputMode"`
	ReasoningEffort          aireasoning.ReasoningEffort     `json:"reasoningEffort"`
	ThinkingStyle            aireasoning.ThinkingStyle       `json:"thinkingStyle"`
	AdvisorEnabled           bool                            `json:"advisorEnabled"`
	AdvisorModel             string                          `json:"advisorModel"`
	Scripts                  bool                            `json:"scripts"`
	ScriptWrites             bool                            `json:"scriptWrites"`
	Platform                 aiplatform.Platform             `json:"platform"`
	PlatformSettings         *aiplatform.PlatformSettings    `json:"platformSettings"`
	ExtraBody                map[string]any                  `json:"extraBody"`
	InputCostPerMillion      *decimal.Decimal                `json:"inputCostPerMillion"`
	OutputCostPerMillion     *decimal.Decimal                `json:"outputCostPerMillion"`
	CacheReadCostPerMillion  *decimal.Decimal                `json:"cacheReadCostPerMillion"`
	CacheWriteCostPerMillion *decimal.Decimal                `json:"cacheWriteCostPerMillion"`
	MaxTokens                int                             `json:"maxTokens"`
	ContextWindow            *int                            `json:"contextWindow"`
	Tasks                    []aitask.Task                   `json:"tasks"`
	Priority                 int                             `json:"priority"`
	EmbeddingDimensions      *int                            `json:"embeddingDimensions"`
	EmbeddingInputStyle      aiprovider.EmbeddingInputStyle  `json:"embeddingInputStyle"`
	Trusted                  bool                            `json:"trusted"`
	Enabled                  bool                            `json:"enabled"`
	TimeoutSeconds           int                             `json:"timeoutSeconds"`
	MaxConcurrent            int                             `json:"maxConcurrent"`
	MonthlyCapUSD            *decimal.Decimal                `json:"monthlyCapUsd"`
	OnCap                    aiprovider.CapAction            `json:"onCap"`
	KeepPreviousKey          bool                            `json:"keepPreviousKey"`
	Version                  int64                           `json:"version"`
}

func (r *saveProviderRequest) toServiceRequest(
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *aiproviderports.SaveAIProviderRequest {
	return &aiproviderports.SaveAIProviderRequest{
		ID:                       id,
		Name:                     r.Name,
		Description:              r.Description,
		Kind:                     r.Kind,
		BaseURL:                  r.BaseURL,
		Model:                    r.Model,
		APIKey:                   r.APIKey,
		AllowPrivateNetwork:      r.AllowPrivateNetwork,
		StructuredOutputMode:     r.StructuredOutputMode,
		ReasoningEffort:          r.ReasoningEffort,
		ThinkingStyle:            r.ThinkingStyle,
		AdvisorEnabled:           r.AdvisorEnabled,
		AdvisorModel:             r.AdvisorModel,
		Scripts:                  r.Scripts,
		ScriptWrites:             r.ScriptWrites,
		Platform:                 r.Platform,
		PlatformSettings:         r.PlatformSettings,
		ExtraBody:                r.ExtraBody,
		InputCostPerMillion:      r.InputCostPerMillion,
		OutputCostPerMillion:     r.OutputCostPerMillion,
		CacheReadCostPerMillion:  r.CacheReadCostPerMillion,
		CacheWriteCostPerMillion: r.CacheWriteCostPerMillion,
		MaxTokens:                r.MaxTokens,
		ContextWindow:            r.ContextWindow,
		Tasks:                    r.Tasks,
		Priority:                 r.Priority,
		EmbeddingDimensions:      r.EmbeddingDimensions,
		EmbeddingInputStyle:      r.EmbeddingInputStyle,
		Trusted:                  r.Trusted,
		Enabled:                  r.Enabled,
		TimeoutSeconds:           r.TimeoutSeconds,
		MaxConcurrent:            r.MaxConcurrent,
		MonthlyCapUSD:            r.MonthlyCapUSD,
		OnCap:                    r.OnCap,
		KeepPreviousKey:          r.KeepPreviousKey,
		Version:                  r.Version,
		TenantInfo:               tenantInfo,
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

	helpers.JSON(c, http.StatusCreated, created)
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

	helpers.JSON(c, http.StatusOK, updated)
}

func (h *Handler) remove(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	providerID, err := pulid.Parse(c.Param("providerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	if err = h.service.Delete(c.Request.Context(), aiproviderports.DeleteAIProviderRequest{
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

	result, err := h.service.Test(c.Request.Context(), aiproviderports.GetAIProviderByIDRequest{
		ID:         providerID,
		TenantInfo: tenantFromAuthContext(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}

// catalog describes the protocols, presets, and routable tasks the UI offers, so
// the form does not hardcode a list that drifts from the domain.
func (h *Handler) catalog(c *gin.Context) {
	helpers.JSON(c, http.StatusOK, gin.H{
		"kinds":                kindDescriptors(),
		"presets":              Presets(),
		"tasks":                taskDescriptors(),
		"embeddingDimensions":  aiprovider.AllowedEmbeddingDimensions(),
		"embeddingInputStyles": embeddingInputStyleDescriptors(),
		"platforms":            platformDescriptors(),
		// serverCloudCredentials says whether a platform provider may sign in
		// with this server's own cloud identity, so the editor offers it only
		// where saving it would be accepted.
		"serverCloudCredentials": h.ai.ServerCloudCredentialsAllowed(),
	})
}
