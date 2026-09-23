package agentdefinitionhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              serviceports.AgentDefinitionService
	Trust                serviceports.AgentTrustService
	Budgets              serviceports.AgentBudgetService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service serviceports.AgentDefinitionService
	trust   serviceports.AgentTrustService
	budgets serviceports.AgentBudgetService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		trust:   p.Trust,
		budgets: p.Budgets,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/agent-definitions")
	resource := permission.ResourceAgentDefinition.String()

	api.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.list)
	api.GET("/templates/", h.pm.RequirePermission(resource, permission.OpRead), h.templates)
	api.GET("/tools/", h.pm.RequirePermission(resource, permission.OpRead), h.tools)
	api.GET("/event-kinds/", h.pm.RequirePermission(resource, permission.OpRead), h.eventKinds)
	api.POST(
		"/preview-prompt/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.previewPrompt,
	)
	api.GET(
		"/system/:systemKey/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.getBySystemKey,
	)
	api.GET("/:agentID/", h.pm.RequirePermission(resource, permission.OpRead), h.get)
	api.GET("/:agentID/trust/", h.pm.RequirePermission(resource, permission.OpRead), h.trustLedger)
	api.GET("/:agentID/budget/", h.pm.RequirePermission(resource, permission.OpRead), h.budget)
	api.POST("/", h.pm.RequirePermission(resource, permission.OpCreate), h.create)
	api.PUT("/:agentID/", h.pm.RequirePermission(resource, permission.OpUpdate), h.update)
	api.DELETE("/:agentID/", h.pm.RequirePermission(resource, permission.OpDelete), h.remove)
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
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	enabledOnly := c.Query("enabledOnly") == "true"
	chatOnly := c.Query("chatOnly") == "true"

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*agentdefinition.Definition], error) {
			return h.service.List(c.Request.Context(), &repositories.ListAgentDefinitionRequest{
				Filter:      req,
				EnabledOnly: enabledOnly,
				ChatOnly:    chatOnly,
			})
		},
	)
}

func (h *Handler) templates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"templates": h.service.Templates()})
}

func (h *Handler) tools(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tools": h.service.ToolCatalog()})
}

func (h *Handler) eventKinds(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"events": h.service.EventKinds()})
}

func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	agentID, err := pulid.Parse(c.Param("agentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	definition, err := h.service.GetByID(
		c.Request.Context(),
		repositories.GetAgentDefinitionByIDRequest{
			ID:         agentID,
			TenantInfo: tenantFromAuthContext(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, definition)
}

// trustLedger answers how each of the agent's tools has been decided on: the
// streak, the totals and any tier the ledger granted. Rows exist only for
// tools that have had a decision, so a tool with none is simply absent.
func (h *Handler) trustLedger(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	agentID, err := pulid.Parse(c.Param("agentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	rows, err := h.trust.ListForDefinition(c.Request.Context(), repositories.ListToolTrustRequest{
		TenantInfo:        tenantFromAuthContext(authCtx),
		AgentDefinitionID: agentID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": rows})
}

func (h *Handler) getBySystemKey(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	definition, err := h.service.GetBySystemKey(
		c.Request.Context(),
		repositories.GetAgentDefinitionBySystemKeyRequest{
			SystemKey:  c.Param("systemKey"),
			TenantInfo: tenantFromAuthContext(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, definition)
}

type saveAgentRequest struct {
	Name                   string                            `json:"name"`
	Description            string                            `json:"description"`
	Template               agentdefinition.Template          `json:"template"`
	Instructions           string                            `json:"instructions"`
	Guardrails             []string                          `json:"guardrails"`
	ToolNames              []string                          `json:"toolNames"`
	ToolTiers              map[string]agent.AutonomyTier     `json:"toolTiers"`
	AutonomyCeiling        agent.AutonomyTier                `json:"autonomyCeiling"`
	Enabled                bool                              `json:"enabled"`
	ShadowMode             bool                              `json:"shadowMode"`
	DecisionTimeoutSeconds int                               `json:"decisionTimeoutSeconds"`
	Icon                   string                            `json:"icon"`
	Accent                 string                            `json:"accent"`
	TriggerMode            agentdefinition.TriggerMode       `json:"triggerMode"`
	CronExpression         string                            `json:"cronExpression"`
	CronTimezone           string                            `json:"cronTimezone"`
	EventKinds             []agent.EventKind                 `json:"eventKinds"`
	IntervalSeconds        int                               `json:"intervalSeconds"`
	EndsAt                 *int64                            `json:"endsAt"`
	MaxConcurrentRuns      int                               `json:"maxConcurrentRuns"`
	RunTimeoutSeconds      int                               `json:"runTimeoutSeconds"`
	MaxToolCalls           int                               `json:"maxToolCalls"`
	MonthlyBudgetUSD       *decimal.Decimal                  `json:"monthlyBudgetUsd"`
	DailyRunLimit          int                               `json:"dailyRunLimit"`
	ToolDailyLimits        map[string]int                    `json:"toolDailyLimits"`
	SimulationMode         bool                              `json:"simulationMode"`
	ContextProviders       []agentdefinition.ContextProvider `json:"contextProviders"`
	OutputMode             agentdefinition.OutputMode        `json:"outputMode"`
	PreferredProviderID    pulid.ID                          `json:"preferredProviderId"`
	// DelegateIDs is absent to keep the agents this one may hand work to,
	// and a list, empty or not, to replace them.
	DelegateIDs *[]pulid.ID `json:"delegateIds"`
	Version     int64       `json:"version"`
}

func (r *saveAgentRequest) toServiceRequest(
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *serviceports.SaveAgentDefinitionRequest {
	return &serviceports.SaveAgentDefinitionRequest{
		ID:                     id,
		Name:                   r.Name,
		Description:            r.Description,
		Template:               r.Template,
		Instructions:           r.Instructions,
		Guardrails:             r.Guardrails,
		ToolNames:              r.ToolNames,
		ToolTiers:              r.ToolTiers,
		AutonomyCeiling:        r.AutonomyCeiling,
		Enabled:                r.Enabled,
		ShadowMode:             r.ShadowMode,
		DecisionTimeoutSeconds: r.DecisionTimeoutSeconds,
		Icon:                   r.Icon,
		Accent:                 r.Accent,
		TriggerMode:            r.TriggerMode,
		CronExpression:         r.CronExpression,
		CronTimezone:           r.CronTimezone,
		EventKinds:             r.EventKinds,
		IntervalSeconds:        r.IntervalSeconds,
		EndsAt:                 r.EndsAt,
		MaxConcurrentRuns:      r.MaxConcurrentRuns,
		RunTimeoutSeconds:      r.RunTimeoutSeconds,
		MaxToolCalls:           r.MaxToolCalls,
		MonthlyBudgetUSD:       r.MonthlyBudgetUSD,
		DailyRunLimit:          r.DailyRunLimit,
		ToolDailyLimits:        r.ToolDailyLimits,
		SimulationMode:         r.SimulationMode,
		ContextProviders:       r.ContextProviders,
		OutputMode:             r.OutputMode,
		PreferredProviderID:    r.PreferredProviderID,
		DelegateIDs:            r.DelegateIDs,
		Version:                r.Version,
		TenantInfo:             tenantInfo,
	}
}

func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var body saveAgentRequest
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

	agentID, err := pulid.Parse(c.Param("agentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body saveAgentRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	updated, err := h.service.Update(
		c.Request.Context(),
		body.toServiceRequest(agentID, tenantFromAuthContext(authCtx)),
		&actor,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (h *Handler) previewPrompt(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var body saveAgentRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	prompt, err := h.service.PreviewPrompt(c.Request.Context(), &serviceports.PreviewPromptRequest{
		Definition: body.toServiceRequest(pulid.Nil, tenantFromAuthContext(authCtx)),
		Actor:      &actor,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"prompt": prompt})
}

func (h *Handler) remove(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	agentID, err := pulid.Parse(c.Param("agentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	if err = h.service.Delete(c.Request.Context(), repositories.DeleteAgentDefinitionRequest{
		ID:         agentID,
		TenantInfo: tenantFromAuthContext(authCtx),
	}, &actor); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// budget answers where the agent stands against its caps this month and
// today, for the form that sets them.
func (h *Handler) budget(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	agentID, err := pulid.Parse(c.Param("agentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	definition, err := h.service.GetByID(
		c.Request.Context(),
		repositories.GetAgentDefinitionByIDRequest{
			ID:         agentID,
			TenantInfo: tenantFromAuthContext(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	status, err := h.budgets.Status(c.Request.Context(), definition)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}
