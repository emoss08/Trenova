// Package agentbudgetservice bounds what an agent may spend and do on its
// own. A cap is read against a window in the organization's own time: the
// month for cost, the day for runs and for each tool's executions.
package agentbudgetservice

import (
	"context"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Usage         repositories.AIUsageRepository
	Runs          repositories.AgentRunRepository
	Proposals     repositories.AgentProposalRepository
	Organizations repositories.OrganizationRepository
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
	CostByDefinition(ctx context.Context, req repositories.AIUsageCostRequest) (*repositories.AIUsageCost, error)
}

type runCounter interface {
	CountSince(ctx context.Context, req repositories.CountAgentRunsSinceRequest) (int, error)
}

type toolCounter interface {
	CountExecutedTool(ctx context.Context, req repositories.CountExecutedToolRequest) (int, error)
}

// timezoneReader names the organization's zone, which is what "this month"
// and "today" mean for a cap.
type timezoneReader interface {
	Timezone(ctx context.Context, tenant pagination.TenantInfo) string
}

func New(p Params) services.AgentBudgetService {
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
) (services.BudgetRefusal, error) {
	if definition == nil {
		return services.BudgetRefusal{}, nil
	}
	tenant := tenantOf(definition)
	windows := s.windows(ctx, tenant)

	if definition.MonthlyBudgetUSD != nil && definition.MonthlyBudgetUSD.IsPositive() {
		cost, err := s.usage.CostByDefinition(ctx, repositories.AIUsageCostRequest{
			TenantInfo:   tenant,
			DefinitionID: definition.ID,
			Since:        windows.monthStart,
		})
		if err != nil {
			return services.BudgetRefusal{}, err
		}
		if cost.CostUSD.GreaterThanOrEqual(*definition.MonthlyBudgetUSD) {
			return services.BudgetRefusal{
				Cap:      services.BudgetCapMonthly,
				Spent:    cost.CostUSD.StringFixed(2),
				Limit:    definition.MonthlyBudgetUSD.StringFixed(2),
				ResetsAt: windows.nextMonth,
			}, nil
		}
	}

	if definition.DailyRunLimit > 0 {
		started, err := s.runs.CountSince(ctx, repositories.CountAgentRunsSinceRequest{
			TenantInfo:   tenant,
			DefinitionID: definition.ID,
			Since:        windows.dayStart,
		})
		if err != nil {
			return services.BudgetRefusal{}, err
		}
		if started >= definition.DailyRunLimit {
			return services.BudgetRefusal{
				Cap:      services.BudgetCapDailyRuns,
				Spent:    strconv.Itoa(started),
				Limit:    strconv.Itoa(definition.DailyRunLimit),
				ResetsAt: windows.nextDay,
			}, nil
		}
	}

	return services.BudgetRefusal{}, nil
}

func (s *Service) CheckTool(
	ctx context.Context,
	definition *agentdefinition.Definition,
	toolName string,
) (services.BudgetRefusal, error) {
	if definition == nil {
		return services.BudgetRefusal{}, nil
	}
	limit := definition.ToolDailyLimit(toolName)
	if limit <= 0 {
		return services.BudgetRefusal{}, nil
	}

	tenant := tenantOf(definition)
	windows := s.windows(ctx, tenant)
	used, err := s.proposals.CountExecutedTool(ctx, repositories.CountExecutedToolRequest{
		TenantInfo:   tenant,
		DefinitionID: definition.ID,
		ToolName:     toolName,
		Since:        windows.dayStart,
	})
	if err != nil {
		return services.BudgetRefusal{}, err
	}
	if used >= limit {
		return services.BudgetRefusal{
			Cap:      services.BudgetCapTool,
			Tool:     toolName,
			Spent:    strconv.Itoa(used),
			Limit:    strconv.Itoa(limit),
			ResetsAt: windows.nextDay,
		}, nil
	}

	return services.BudgetRefusal{}, nil
}

func (s *Service) Status(
	ctx context.Context,
	definition *agentdefinition.Definition,
) (*services.AgentBudgetStatus, error) {
	tenant := tenantOf(definition)
	windows := s.windows(ctx, tenant)

	cost, err := s.usage.CostByDefinition(ctx, repositories.AIUsageCostRequest{
		TenantInfo:   tenant,
		DefinitionID: definition.ID,
		Since:        windows.monthStart,
	})
	if err != nil {
		return nil, err
	}
	started, err := s.runs.CountSince(ctx, repositories.CountAgentRunsSinceRequest{
		TenantInfo:   tenant,
		DefinitionID: definition.ID,
		Since:        windows.dayStart,
	})
	if err != nil {
		return nil, err
	}

	status := &services.AgentBudgetStatus{
		MonthStart:     windows.monthStart,
		DayStart:       windows.dayStart,
		SpentUSD:       cost.CostUSD.StringFixed(2),
		MonthCalls:     cost.Calls,
		UnpricedCalls:  cost.UnpricedCalls,
		RunsToday:      started,
		DailyRunLimit:  definition.DailyRunLimit,
		SimulationMode: definition.SimulationMode,
		Tools:          make([]services.ToolBudgetUse, 0, len(definition.ToolDailyLimits)),
	}
	if definition.MonthlyBudgetUSD != nil {
		budget := definition.MonthlyBudgetUSD.StringFixed(2)
		status.MonthlyBudget = &budget
	}

	for _, tool := range definition.ToolNames {
		limit := definition.ToolDailyLimit(tool)
		if limit <= 0 {
			continue
		}
		used, cErr := s.proposals.CountExecutedTool(ctx, repositories.CountExecutedToolRequest{
			TenantInfo:   tenant,
			DefinitionID: definition.ID,
			ToolName:     tool,
			Since:        windows.dayStart,
		})
		if cErr != nil {
			return nil, cErr
		}
		status.Tools = append(status.Tools, services.ToolBudgetUse{Tool: tool, Used: used, Limit: limit})
	}

	return status, nil
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
	organizations repositories.OrganizationRepository
	logger        *zap.Logger
}

func (z organizationZones) Timezone(ctx context.Context, tenant pagination.TenantInfo) string {
	org, err := z.organizations.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenant,
	})
	if err != nil {
		z.logger.Warn("agent budget: organization lookup failed",
			zap.String("organization", tenant.OrgID.String()), zap.Error(err))

		return ""
	}

	return org.Timezone
}
