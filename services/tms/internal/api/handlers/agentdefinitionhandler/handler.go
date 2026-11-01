package agentdefinitionhandler

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Service              agentdefinitionports.AgentDefinitionService
	Workflows            kernelports.WorkflowStarter
	Streams              assistantstreamports.TurnStreamReader
	Logger               *zap.Logger
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service   agentdefinitionports.AgentDefinitionService
	workflows kernelports.WorkflowStarter
	streams   assistantstreamports.TurnStreamReader
	logger    *zap.Logger
	eh        *helpers.ErrorHandler
	pm        *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service:   p.Service,
		workflows: p.Workflows,
		streams:   p.Streams,
		logger:    p.Logger.Named("handler.agentdefinition"),
		eh:        p.ErrorHandler,
		pm:        p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/agent-definitions")
	resource := permission.ResourceAgentDefinition.String()

	api.POST("/dry-run/", h.pm.RequirePermission(resource, permission.OpUpdate), h.dryRun)
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
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

type saveAgentRequest struct {
	Name                   string                            `json:"name"`
	Description            string                            `json:"description"`
	Template               agenttemplate.Template            `json:"template"`
	Instructions           string                            `json:"instructions"`
	Guardrails             []string                          `json:"guardrails"`
	ToolNames              []string                          `json:"toolNames"`
	ToolTiers              map[string]agent.AutonomyTier     `json:"toolTiers"`
	AutonomyCeiling        agent.AutonomyTier                `json:"autonomyCeiling"`
	DataAccessCeiling      agentdefinition.DataAccessCeiling `json:"dataAccessCeiling"`
	Enabled                bool                              `json:"enabled"`
	ShadowMode             bool                              `json:"shadowMode"`
	DecisionTimeoutSeconds int                               `json:"decisionTimeoutSeconds"`
	Icon                   string                            `json:"icon"`
	Accent                 string                            `json:"accent"`
	TriggerMode            agentdefinition.TriggerMode       `json:"triggerMode"`
	CronExpression         string                            `json:"cronExpression"`
	CronTimezone           string                            `json:"cronTimezone"`
	EventKinds             []agenttrigger.EventKind          `json:"eventKinds"`
	IntervalSeconds        int                               `json:"intervalSeconds"`
	EndsAt                 *int64                            `json:"endsAt"`
	MaxConcurrentRuns      int                               `json:"maxConcurrentRuns"`
	RunTimeoutSeconds      int                               `json:"runTimeoutSeconds"`
	MaxToolCalls           int                               `json:"maxToolCalls"`
	MonthlyBudgetUSD       *decimal.Decimal                  `json:"monthlyBudgetUsd"`
	DailyRunLimit          int                               `json:"dailyRunLimit"`
	ToolDailyLimits        map[string]int                    `json:"toolDailyLimits"`
	SimulationMode         bool                              `json:"simulationMode"`
	MemoryTokenBudget      *int                              `json:"memoryTokenBudget"`
	LearningOff            *bool                             `json:"learningOff"`
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
) *agentdefinitionports.SaveAgentDefinitionRequest {
	return &agentdefinitionports.SaveAgentDefinitionRequest{
		ID:                     id,
		Name:                   r.Name,
		Description:            r.Description,
		Template:               r.Template,
		Instructions:           r.Instructions,
		Guardrails:             r.Guardrails,
		ToolNames:              r.ToolNames,
		ToolTiers:              r.ToolTiers,
		AutonomyCeiling:        r.AutonomyCeiling,
		DataAccessCeiling:      r.DataAccessCeiling,
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
		MemoryTokenBudget:      r.MemoryTokenBudget,
		LearningOff:            r.LearningOff,
		ContextProviders:       r.ContextProviders,
		OutputMode:             r.OutputMode,
		PreferredProviderID:    r.PreferredProviderID,
		DelegateIDs:            r.DelegateIDs,
		Version:                r.Version,
		TenantInfo:             tenantInfo,
	}
}
