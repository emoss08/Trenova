package agentbudgetports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/shared/pulid"
)

// BudgetRefusal says which cap a run or a write ran into. A zero value is
// no refusal.
type BudgetRefusal struct {
	// Cap names the limit: monthly_budget, daily_runs or tool_daily_limit.
	Cap string `json:"cap"`
	// Tool is set for a tool cap.
	Tool  string `json:"tool,omitempty"`
	Spent string `json:"spent"`
	Limit string `json:"limit"`
	// ResetsAt is when the window rolls over and the cap clears.
	ResetsAt int64 `json:"resetsAt"`
}

// Message says the refusal in words a person or a model can act on.
func (r BudgetRefusal) Message(agentName string) string {
	switch r.Cap {
	case BudgetCapMonthly:
		return "Agent " + agentName + " has spent its monthly budget (" + r.Spent + " of " +
			r.Limit + " USD). It runs again when the month rolls over, or when the budget is raised in AI Control."
	case BudgetCapDailyRuns:
		return "Agent " + agentName + " has started its " + r.Limit +
			" runs for today. It runs again tomorrow, or when the daily limit is raised in AI Control."
	case BudgetCapTool:
		return "Tool " + r.Tool + " has reached its daily limit of " + r.Limit +
			" for agent " + agentName + ". It can run again tomorrow, or when the limit is raised in AI Control."
	default:
		return ""
	}
}

func (r BudgetRefusal) Refused() bool { return r.Cap != "" }

const BudgetCapMonthly = "monthly_budget"

const BudgetCapDailyRuns = "daily_runs"

const BudgetCapTool = "tool_daily_limit"

// ToolBudgetUse is one tool's executions today against its cap.
type ToolBudgetUse struct {
	Tool  string `json:"tool"`
	Used  int    `json:"used"`
	Limit int    `json:"limit"`
}

// AgentBudgetStatus is where an agent stands against its caps, for the page
// that sets them.
type AgentBudgetStatus struct {
	MonthStart int64 `json:"monthStart"`
	DayStart   int64 `json:"dayStart"`
	// SpentUSD is the priced cost of the month's calls; UnpricedCalls says how
	// many carried no price, so the figure can be labelled partial.
	SpentUSD       string          `json:"spentUsd"`
	MonthlyBudget  *string         `json:"monthlyBudgetUsd"`
	MonthCalls     int             `json:"monthCalls"`
	UnpricedCalls  int             `json:"unpricedCalls"`
	RunsToday      int             `json:"runsToday"`
	DailyRunLimit  int             `json:"dailyRunLimit"`
	Tools          []ToolBudgetUse `json:"tools"`
	SimulationMode bool            `json:"simulationMode"`
}

// CheckToolBudgetRequest asks whether one more write by a tool fits its cap.
type CheckToolBudgetRequest struct {
	Definition *agentdefinition.Definition
	ToolName   string
	// Unrecorded is how many times the tool already ran in the caller's run
	// or turn without a recorded proposal yet. A turn records its writes when
	// it ends, so without them a single turn could run past the cap.
	Unrecorded int
}

type MoneyAllowanceRequest struct {
	Definition *agentdefinition.Definition
	RunID      pulid.ID
	Unrecorded agenttooltrust.MoneyUse
}

// AgentBudgetService enforces an agent's caps and reports where it stands.
type AgentBudgetService interface {
	// CheckRun says whether the agent may start another run or turn now.
	CheckRun(ctx context.Context, definition *agentdefinition.Definition) (BudgetRefusal, error)
	// CheckTool says whether the agent may execute the tool once more today.
	CheckTool(ctx context.Context, req CheckToolBudgetRequest) (BudgetRefusal, error)
	MoneyAllowance(
		ctx context.Context,
		req MoneyAllowanceRequest,
	) (agenttooltrust.MoneyAllowance, error)
	Status(ctx context.Context, definition *agentdefinition.Definition) (*AgentBudgetStatus, error)
	// WithinBusinessHours says whether the agent may make a change on its
	// own now, reading its business-hours window in its own zone or the
	// organization's. Always true while the rule is off.
	WithinBusinessHours(ctx context.Context, definition *agentdefinition.Definition) bool
	// Timezone is the organization's zone, empty when it cannot be read.
	Timezone(ctx context.Context, definition *agentdefinition.Definition) string
}
