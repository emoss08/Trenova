// Package agentbudgetservice bounds what an agent may spend and do on its
// own. A cap is read against a window in the organization's own time: the
// month for cost, the day for runs and for each tool's executions.
package agentbudgetservice

import (
	"context"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Usage         aiusageports.AIUsageRepository
	Runs          agentports.AgentRunRepository
	Proposals     agentports.AgentProposalRepository
	Organizations tenantports.OrganizationRepository
}

type Service struct {
	l         *zap.Logger
	usage     costReader
	runs      runCounter
	proposals toolCounter
	zones     timezoneReader
	now       func() int64
}

type costReader interface {
	CostByDefinition(
		ctx context.Context,
		req aiusageports.AIUsageCostRequest,
	) (*aiusageports.AIUsageCost, error)
}

type runCounter interface {
	CountSince(ctx context.Context, req agentports.CountAgentRunsSinceRequest) (int, error)
}

type toolCounter interface {
	CountExecutedTool(ctx context.Context, req agentports.CountExecutedToolRequest) (int, error)
}

// timezoneReader names the organization's zone, which is what "this month"
// and "today" mean for a cap.
type timezoneReader interface {
	Timezone(ctx context.Context, tenant pagination.TenantInfo) string
}

func New(p Params) agentports.AgentBudgetService {
	return &Service{
		l:         p.Logger.Named("service.agentbudget"),
		usage:     p.Usage,
		runs:      p.Runs,
		proposals: p.Proposals,
		zones:     organizationZones{organizations: p.Organizations, logger: p.Logger},
		now:       timeutils.NowUnix,
	}
}

func (s *Service) CheckRun(
	ctx context.Context,
	definition *agentdefinition.Definition,
) (agentports.BudgetRefusal, error) {
	if definition == nil {
		return agentports.BudgetRefusal{}, nil
	}
	tenant := tenantOf(definition)
	windows := s.windows(ctx, tenant)

	if definition.MonthlyBudgetUSD != nil && definition.MonthlyBudgetUSD.IsPositive() {
		cost, err := s.usage.CostByDefinition(ctx, aiusageports.AIUsageCostRequest{
			TenantInfo:   tenant,
			DefinitionID: definition.ID,
			Since:        windows.monthStart,
		})
		if err != nil {
			return agentports.BudgetRefusal{}, err
		}
		if cost.CostUSD.GreaterThanOrEqual(*definition.MonthlyBudgetUSD) {
			return agentports.BudgetRefusal{
				Cap:      agentports.BudgetCapMonthly,
				Spent:    cost.CostUSD.StringFixed(2),
				Limit:    definition.MonthlyBudgetUSD.StringFixed(2),
				ResetsAt: windows.nextMonth,
			}, nil
		}
	}

	if definition.DailyRunLimit > 0 {
		started, err := s.runs.CountSince(ctx, agentports.CountAgentRunsSinceRequest{
			TenantInfo:   tenant,
			DefinitionID: definition.ID,
			Since:        windows.dayStart,
		})
		if err != nil {
			return agentports.BudgetRefusal{}, err
		}
		if started >= definition.DailyRunLimit {
			return agentports.BudgetRefusal{
				Cap:      agentports.BudgetCapDailyRuns,
				Spent:    strconv.Itoa(started),
				Limit:    strconv.Itoa(definition.DailyRunLimit),
				ResetsAt: windows.nextDay,
			}, nil
		}
	}

	return agentports.BudgetRefusal{}, nil
}

func (s *Service) CheckTool(
	ctx context.Context,
	req agentports.CheckToolBudgetRequest,
) (agentports.BudgetRefusal, error) {
	definition, toolName := req.Definition, req.ToolName
	if definition == nil {
		return agentports.BudgetRefusal{}, nil
	}
	limit := definition.ToolDailyLimit(toolName)
	if limit <= 0 {
		return agentports.BudgetRefusal{}, nil
	}

	tenant := tenantOf(definition)
	windows := s.windows(ctx, tenant)
	used, err := s.proposals.CountExecutedTool(ctx, agentports.CountExecutedToolRequest{
		TenantInfo:   tenant,
		DefinitionID: definition.ID,
		ToolName:     toolName,
		Since:        windows.dayStart,
	})
	if err != nil {
		return agentports.BudgetRefusal{}, err
	}
	used += max(req.Unrecorded, 0)
	if used >= limit {
		return agentports.BudgetRefusal{
			Cap:      agentports.BudgetCapTool,
			Tool:     toolName,
			Spent:    strconv.Itoa(used),
			Limit:    strconv.Itoa(limit),
			ResetsAt: windows.nextDay,
		}, nil
	}

	return agentports.BudgetRefusal{}, nil
}

func (s *Service) Status(
	ctx context.Context,
	definition *agentdefinition.Definition,
) (*agentports.AgentBudgetStatus, error) {
	tenant := tenantOf(definition)
	windows := s.windows(ctx, tenant)

	cost, err := s.usage.CostByDefinition(ctx, aiusageports.AIUsageCostRequest{
		TenantInfo:   tenant,
		DefinitionID: definition.ID,
		Since:        windows.monthStart,
	})
	if err != nil {
		return nil, err
	}
	started, err := s.runs.CountSince(ctx, agentports.CountAgentRunsSinceRequest{
		TenantInfo:   tenant,
		DefinitionID: definition.ID,
		Since:        windows.dayStart,
	})
	if err != nil {
		return nil, err
	}

	status := &agentports.AgentBudgetStatus{
		MonthStart:     windows.monthStart,
		DayStart:       windows.dayStart,
		SpentUSD:       cost.CostUSD.StringFixed(2),
		MonthCalls:     cost.Calls,
		UnpricedCalls:  cost.UnpricedCalls,
		RunsToday:      started,
		DailyRunLimit:  definition.DailyRunLimit,
		SimulationMode: definition.SimulationMode,
		Tools:          make([]agentports.ToolBudgetUse, 0, len(definition.ToolDailyLimits)),
	}
	if definition.MonthlyBudgetUSD != nil {
		budget := definition.MonthlyBudgetUSD.StringFixed(2)
		status.MonthlyBudget = &budget
	}

	for _, tool := range definition.EffectiveToolNames() {
		limit := definition.ToolDailyLimit(tool)
		if limit <= 0 {
			continue
		}
		used, cErr := s.proposals.CountExecutedTool(ctx, agentports.CountExecutedToolRequest{
			TenantInfo:   tenant,
			DefinitionID: definition.ID,
			ToolName:     tool,
			Since:        windows.dayStart,
		})
		if cErr != nil {
			return nil, cErr
		}
		status.Tools = append(
			status.Tools,
			agentports.ToolBudgetUse{Tool: tool, Used: used, Limit: limit},
		)
	}

	return status, nil
}

func (s *Service) WithinBusinessHours(
	ctx context.Context,
	definition *agentdefinition.Definition,
) bool {
	if !definition.BusinessHoursOnly {
		return true
	}

	return definition.WithinBusinessHours(time.Unix(s.now(), 0), s.Timezone(ctx, definition))
}

func (s *Service) Timezone(ctx context.Context, definition *agentdefinition.Definition) string {
	return s.zones.Timezone(ctx, tenantOf(definition))
}

type budgetWindows struct {
	monthStart int64
	nextMonth  int64
	dayStart   int64
	nextDay    int64
}

// windows draws the month and the day in the organization's zone. A cap
// that rolled over at UTC midnight would reset in the middle of a Pacific
// afternoon.
func (s *Service) windows(ctx context.Context, tenant pagination.TenantInfo) budgetWindows {
	loc := timeutils.LoadLocation(s.zones.Timezone(ctx, tenant))
	now := time.Unix(s.now(), 0).In(loc)

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	return budgetWindows{
		monthStart: monthStart.Unix(),
		nextMonth:  monthStart.AddDate(0, 1, 0).Unix(),
		dayStart:   dayStart.Unix(),
		nextDay:    dayStart.AddDate(0, 0, 1).Unix(),
	}
}

func tenantOf(definition *agentdefinition.Definition) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: definition.OrganizationID, BuID: definition.BusinessUnitID}
}

// organizationZones reads the zone from the organization, falling back to
// UTC when it cannot be read: a cap checked in the wrong zone is still a
// cap, and a run refused because the organization could not be read is not.
type organizationZones struct {
	organizations tenantports.OrganizationRepository
	logger        *zap.Logger
}

func (z organizationZones) Timezone(ctx context.Context, tenant pagination.TenantInfo) string {
	org, err := z.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenant,
	})
	if err != nil {
		z.logger.Warn("agent budget: organization lookup failed",
			zap.String("organization", tenant.OrgID.String()), zap.Error(err))

		return ""
	}

	return org.Timezone
}
