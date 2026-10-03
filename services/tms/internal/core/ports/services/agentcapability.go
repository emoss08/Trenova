package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// AgentCapabilityMode is how an agent may use one tool, in the words of its
// capabilities page. Allowed runs on its own (AutoExecute), AskFirst waits
// for a person (ActWithApproval, and Propose, which a person meets the same
// way), and Off means the agent does not hold the tool.
type AgentCapabilityMode string

const (
	AgentCapabilityAllowed  = AgentCapabilityMode("Allowed")
	AgentCapabilityAskFirst = AgentCapabilityMode("AskFirst")
	AgentCapabilityOff      = AgentCapabilityMode("Off")
)

func (m AgentCapabilityMode) IsValid() bool {
	switch m {
	case AgentCapabilityAllowed, AgentCapabilityAskFirst, AgentCapabilityOff:
		return true
	default:
		return false
	}
}

// AgentCapabilityTool is one row of the page: a tool the agent holds or had
// switched off there.
type AgentCapabilityTool struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	// Write separates the tools that change things from the ones that read.
	Write        bool                  `json:"write"`
	Mode         AgentCapabilityMode   `json:"mode"`
	AllowedModes []AgentCapabilityMode `json:"allowedModes"`
	// LockReason says why some mode cannot be chosen, such as "Always asks ·
	// this can't be undone". Empty when every mode can.
	LockReason string `json:"lockReason,omitempty"`
}

// AgentCapabilityHandoff is an agent this one hands work to, and which work.
type AgentCapabilityHandoff struct {
	AgentID     pulid.ID `json:"agentId"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Accent      string   `json:"accent"`
	Template    string   `json:"template"`
	// Topic is what goes to it; the agent's description when none was set.
	Topic string `json:"topic"`
}

// AgentCapabilityLimits are the agent's caps and where it stands against them.
type AgentCapabilityLimits struct {
	RequestsToday     int   `json:"requestsToday"`
	DailyRequestLimit int   `json:"dailyRequestLimit"`
	DayResetsAt       int64 `json:"dayResetsAt"`
	// MonthlySpentUSD and MonthlyBudgetUSD are fixed to cents; the budget is
	// nil when there is none.
	MonthlySpentUSD  string  `json:"monthlySpentUsd"`
	MonthlyBudgetUSD *string `json:"monthlyBudgetUsd"`
	MonthStart       int64   `json:"monthStart"`
	MonthResetsAt    int64   `json:"monthResetsAt"`
	MaxChangeItems   int     `json:"maxChangeItems"`

	BusinessHoursOnly  bool `json:"businessHoursOnly"`
	BusinessHoursStart int  `json:"businessHoursStart"`
	BusinessHoursEnd   int  `json:"businessHoursEnd"`
	// BusinessHoursTimezone is the zone the window is read in: the agent's
	// own, or the organization's.
	BusinessHoursTimezone string `json:"businessHoursTimezone"`
}

// AgentCapabilities is what an agent can do, as anyone who may use it reads
// it: its tools and how far each may go, who it hands work to, and its limits.
type AgentCapabilities struct {
	AgentID     pulid.ID `json:"agentId"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Template    string   `json:"template"`
	Icon        string   `json:"icon"`
	Accent      string   `json:"accent"`
	SystemKey   string   `json:"systemKey"`
	// Model is the model the agent answers with, by the name an
	// administrator gave it; empty when none is set up.
	Model string `json:"model"`
	// SetUpBy names the person who set the agent up; empty when unknown.
	SetUpBy string `json:"setUpBy"`
	Enabled bool   `json:"enabled"`
	// CanEdit says the reader may change what is here.
	CanEdit    bool                     `json:"canEdit"`
	Version    int64                    `json:"version"`
	ReadTools  []AgentCapabilityTool    `json:"readTools"`
	WriteTools []AgentCapabilityTool    `json:"writeTools"`
	Handoffs   []AgentCapabilityHandoff `json:"handoffs"`
	Limits     AgentCapabilityLimits    `json:"limits"`
}

type GetAgentCapabilitiesRequest struct {
	TenantInfo pagination.TenantInfo
	AgentID    pulid.ID
}

// AgentCapabilityToolChange sets one tool's mode.
type AgentCapabilityToolChange struct {
	Key  string
	Mode AgentCapabilityMode
}

// UpdateAgentCapabilitiesRequest changes what the page shows. Nil fields are
// left as they are.
type UpdateAgentCapabilitiesRequest struct {
	TenantInfo pagination.TenantInfo
	AgentID    pulid.ID
	// Version is the version the page was read at.
	Version           int64
	Enabled           *bool
	Tools             []AgentCapabilityToolChange
	DailyRequestLimit *int
	// MonthlyBudgetUSD sets the budget; ClearMonthlyBudget removes it.
	MonthlyBudgetUSD      *decimal.Decimal
	ClearMonthlyBudget    bool
	MaxChangeItems        *int
	BusinessHoursOnly     *bool
	BusinessHoursStart    *int
	BusinessHoursEnd      *int
	BusinessHoursTimezone *string
	// DelegateTopics replaces the topics of the agents named; an empty
	// topic clears one.
	DelegateTopics map[pulid.ID]string
}

// AgentCapabilityService serves an agent's capabilities page. Reading needs
// only that the reader may use the agent; changing needs permission to
// update agents.
type AgentCapabilityService interface {
	Get(
		ctx context.Context,
		req *GetAgentCapabilitiesRequest,
		actor *RequestActor,
	) (*AgentCapabilities, error)
	Update(
		ctx context.Context,
		req *UpdateAgentCapabilitiesRequest,
		actor *RequestActor,
	) (*AgentCapabilities, error)
}
