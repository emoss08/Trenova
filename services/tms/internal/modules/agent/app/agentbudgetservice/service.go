// Package agentbudgetservice bounds what an agent may spend and do on its
// own. A cap is read against a window in the organization's own time: the
// month for cost, the day for runs and for each tool's executions.
package agentbudgetservice

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	errNoDefinition = errors.New("money limits are read for an agent, and none was given")
	errNoCurrency   = errors.New("the organization's functional currency is not set")
	errNoTimezone   = errors.New("the organization's timezone could not be read")
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Usage         aiusageports.AIUsageRepository
	Runs          agentruntimeports.AgentRunRepository
	Proposals     agentproposalports.AgentProposalRepository
	Organizations tenantports.OrganizationRepository
	Controls      tenantports.AgentControlRepository
	Books         tenantports.AccountingControlRepository
}

type Service struct {
	l         *zap.Logger
	usage     costReader
	runs      runCounter
	proposals toolCounter
	money     moneyCounter
	controls  controlReader
	books     booksReader
	zones     timezoneReader
	now       func() int64
}

type booksReader interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*accountingcontrol.AccountingControl, error)
}

type moneyCounter interface {
	SumAutomaticMoney(
		ctx context.Context,
		req *agentproposalports.SumAutomaticMoneyRequest,
	) (agenttooltrust.MoneyUse, error)
}

type controlReader interface {
	GetOrCreate(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*agentcontrol.AgentControl, error)
}

type costReader interface {
	CostByDefinition(
		ctx context.Context,
		req aiusageports.AIUsageCostRequest,
	) (*aiusageports.AIUsageCost, error)
}

type runCounter interface {
	CountSince(ctx context.Context, req agentruntimeports.CountAgentRunsSinceRequest) (int, error)
}

type toolCounter interface {
	CountExecutedTool(
		ctx context.Context,
		req *agentproposalports.CountExecutedToolRequest,
	) (int, error)
}

// timezoneReader names the organization's zone, which is what "this month"
// and "today" mean for a cap.
type timezoneReader interface {
	Timezone(ctx context.Context, tenant pagination.TenantInfo) (string, error)
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) agentbudgetports.AgentBudgetService {
	return &Service{
		l:         p.Logger.Named("service.agentbudget"),
		usage:     p.Usage,
		runs:      p.Runs,
		proposals: p.Proposals,
		money:     p.Proposals,
		controls:  p.Controls,
		books:     p.Books,
		zones:     organizationZones{organizations: p.Organizations, logger: p.Logger},
		now:       timeutils.NowUnix,
	}
}

func (s *Service) CheckRun(
	ctx context.Context,
	definition *agentdefinition.Definition,
) (agentbudgetports.BudgetRefusal, error) {
	if definition == nil {
		return agentbudgetports.BudgetRefusal{}, nil
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
			return agentbudgetports.BudgetRefusal{}, err
		}
		if cost.CostUSD.GreaterThanOrEqual(*definition.MonthlyBudgetUSD) {
			return agentbudgetports.BudgetRefusal{
				Cap:      agentbudgetports.BudgetCapMonthly,
				Spent:    cost.CostUSD.StringFixed(2),
				Limit:    definition.MonthlyBudgetUSD.StringFixed(2),
				ResetsAt: windows.nextMonth,
			}, nil
		}
	}

	if definition.DailyRunLimit > 0 {
		started, err := s.runs.CountSince(ctx, agentruntimeports.CountAgentRunsSinceRequest{
			TenantInfo:   tenant,
			DefinitionID: definition.ID,
			Since:        windows.dayStart,
		})
		if err != nil {
			return agentbudgetports.BudgetRefusal{}, err
		}
		if started >= definition.DailyRunLimit {
			return agentbudgetports.BudgetRefusal{
				Cap:      agentbudgetports.BudgetCapDailyRuns,
				Spent:    strconv.Itoa(started),
				Limit:    strconv.Itoa(definition.DailyRunLimit),
				ResetsAt: windows.nextDay,
			}, nil
		}
	}

	return agentbudgetports.BudgetRefusal{}, nil
}

func (s *Service) CheckTool(
	ctx context.Context,
	req agentbudgetports.CheckToolBudgetRequest,
) (agentbudgetports.BudgetRefusal, error) {
	definition, toolName := req.Definition, req.ToolName
	if definition == nil {
		return agentbudgetports.BudgetRefusal{}, nil
	}
	limit := definition.ToolDailyLimit(toolName)
	if limit <= 0 {
		return agentbudgetports.BudgetRefusal{}, nil
	}

	tenant := tenantOf(definition)
	windows := s.windows(ctx, tenant)
	used, err := s.proposals.CountExecutedTool(ctx, &agentproposalports.CountExecutedToolRequest{
		TenantInfo:   tenant,
		DefinitionID: definition.ID,
		ToolName:     toolName,
		Since:        windows.dayStart,
	})
	if err != nil {
		return agentbudgetports.BudgetRefusal{}, err
	}
	used += max(req.Unrecorded, 0)
	if used >= limit {
		return agentbudgetports.BudgetRefusal{
			Cap:      agentbudgetports.BudgetCapTool,
			Tool:     toolName,
			Spent:    strconv.Itoa(used),
			Limit:    strconv.Itoa(limit),
			ResetsAt: windows.nextDay,
		}, nil
	}

	return agentbudgetports.BudgetRefusal{}, nil
}

func (s *Service) MoneyAllowance(
	ctx context.Context,
	req agentbudgetports.MoneyAllowanceRequest,
) (agenttooltrust.MoneyAllowance, error) {
	if req.Definition == nil {
		return agenttooltrust.MoneyAllowance{}, errNoDefinition
	}

	tenantInfo := tenantOf(req.Definition)
	control, err := s.controls.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return agenttooltrust.MoneyAllowance{}, fmt.Errorf("read agent money limits: %w", err)
	}
	currency, err := s.functionalCurrency(ctx, tenantInfo)
	if err != nil {
		return agenttooltrust.MoneyAllowance{}, err
	}
	dayStart, err := s.moneyDayStart(ctx, tenantInfo)
	if err != nil {
		return agenttooltrust.MoneyAllowance{}, err
	}

	day, err := s.money.SumAutomaticMoney(ctx, &agentproposalports.SumAutomaticMoneyRequest{
		TenantInfo: tenantInfo,
		Currency:   currency,
		Since:      dayStart,
	})
	if err != nil {
		return agenttooltrust.MoneyAllowance{}, err
	}

	simulated := req.Definition.SimulationMode
	run := agenttooltrust.MoneyUse{}
	if req.RunID.IsNotNil() {
		if run, err = s.money.SumAutomaticMoney(ctx, &agentproposalports.SumAutomaticMoneyRequest{
			TenantInfo: tenantInfo,
			Currency:   currency,
			RunID:      req.RunID,
			Simulated:  simulated,
		}); err != nil {
			return agenttooltrust.MoneyAllowance{}, err
		}
	}
	if simulated {
		day = day.Plus(run)
	}

	return agenttooltrust.MoneyAllowance{
		Limits: agenttooltrust.MoneyLimits{
			CallCeilingMinor: control.AgentMoneyCallCeilingMinor,
			RunCalls:         control.AgentMoneyRunCalls,
			RunAmountMinor:   control.AgentMoneyRunAmountMinor,
			DayCalls:         control.AgentMoneyDayCalls,
			DayAmountMinor:   control.AgentMoneyDayAmountMinor,
		},
		Currency: currency,
		Run:      run.Plus(req.Unrecorded),
		Day:      day.Plus(req.Unrecorded),
	}, nil
}

func (s *Service) functionalCurrency(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (string, error) {
	if s.books == nil {
		return "", errNoCurrency
	}
	books, err := s.books.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil {
		return "", fmt.Errorf("read the organization's functional currency: %w", err)
	}
	if books == nil {
		return "", errNoCurrency
	}
	currency := money.ISOCode(books.FunctionalCurrencyCode)
	if currency == "" {
		return "", errNoCurrency
	}

	return currency, nil
}

func (s *Service) moneyDayStart(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int64, error) {
	zone, err := s.zones.Timezone(ctx, tenantInfo)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errNoTimezone, err)
	}
	if strings.TrimSpace(zone) == "" {
		return 0, errNoTimezone
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errNoTimezone, err)
	}
	now := time.Unix(s.now(), 0).In(loc)

	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).Unix(), nil
}

func (s *Service) Status(
	ctx context.Context,
	definition *agentdefinition.Definition,
) (*agentbudgetports.AgentBudgetStatus, error) {
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
	started, err := s.runs.CountSince(ctx, agentruntimeports.CountAgentRunsSinceRequest{
		TenantInfo:   tenant,
		DefinitionID: definition.ID,
		Since:        windows.dayStart,
	})
	if err != nil {
		return nil, err
	}

	status := &agentbudgetports.AgentBudgetStatus{
		MonthStart:     windows.monthStart,
		DayStart:       windows.dayStart,
		SpentUSD:       cost.CostUSD.StringFixed(2),
		MonthCalls:     cost.Calls,
		UnpricedCalls:  cost.UnpricedCalls,
		RunsToday:      started,
		DailyRunLimit:  definition.DailyRunLimit,
		SimulationMode: definition.SimulationMode,
		Tools:          make([]agentbudgetports.ToolBudgetUse, 0, len(definition.ToolDailyLimits)),
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
		used, cErr := s.proposals.CountExecutedTool(
			ctx,
			&agentproposalports.CountExecutedToolRequest{
				TenantInfo:   tenant,
				DefinitionID: definition.ID,
				ToolName:     tool,
				Since:        windows.dayStart,
			},
		)
		if cErr != nil {
			return nil, cErr
		}
		status.Tools = append(
			status.Tools,
			agentbudgetports.ToolBudgetUse{Tool: tool, Used: used, Limit: limit},
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
	zone, _ := s.zones.Timezone(ctx, tenantOf(definition))

	return zone
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
	zone, _ := s.zones.Timezone(ctx, tenant)
	loc := timeutils.LoadLocation(zone)
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

// organizationZones reads the zone from the organization. The run, cost and
// tool caps fall back to UTC when it cannot be read: a cap checked in the
// wrong zone is still a cap, and a run refused because the organization could
// not be read is not. The money caps do not: a write whose day cannot be
// placed waits for a person.
type organizationZones struct {
	organizations tenantports.OrganizationRepository
	logger        *zap.Logger
}

func (z organizationZones) Timezone(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (string, error) {
	org, err := z.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		z.logger.Warn("agent budget: organization lookup failed",
			zap.String("organization", tenantInfo.OrgID.String()), zap.Error(err))

		return "", err
	}

	return org.Timezone, nil
}
