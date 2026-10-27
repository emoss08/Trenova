package lading

import (
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/shared/jsonutils"
)

const (
	EventReceived = "received"
	EventTrenova  = "trenova"
	EventPlanner  = "planner"
	EventSent     = "sent"
	EventAck      = "ack"
	EventTimer    = "timer"
	EventLoad     = "load"
)

const (
	ActionAcknowledge = "acknowledge"
	ActionCreate      = "create"
	ActionUpdate      = "update"
	ActionCancel      = "cancel"
	ActionApply       = "apply"
	ActionHold        = "hold"
	ActionIgnore      = "ignore"
	ActionAskPlanner  = "askPlanner"
	ActionSend        = "send"
	ActionResend      = "resend"
	ActionEscalate    = "escalate"
)

const (
	DecisionAccept  = "accept"
	DecisionDecline = "decline"
)

const (
	LoadOpen       = "open"
	LoadDispatched = "dispatched"
	LoadDelivered  = "delivered"
	LoadInvoiced   = "invoiced"
	LoadCancelled  = "cancelled"
)

type LoadState struct {
	Exists bool   `json:"exists"`
	State  string `json:"state"`
}

type OutcomeInput struct {
	Kind    string `json:"kind"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// FlowEvent is one event a flow handles; Kind says which of its fields are set.
type FlowEvent struct {
	Kind     string            `json:"kind"`
	Set      string            `json:"set,omitempty"`
	Document jsonutils.RawJSON `json:"document,omitempty"`
	Outcomes []OutcomeInput    `json:"outcomes,omitempty"`
	Current  jsonutils.RawJSON `json:"current,omitempty"`
	Load     *LoadState        `json:"load,omitempty"`
	Event    string            `json:"event,omitempty"`
	Decision string            `json:"decision,omitempty"`
	Reason   string            `json:"reason,omitempty"`
	ID       string            `json:"id,omitempty"`
	Control  string            `json:"control,omitempty"`
	Accepted bool              `json:"accepted,omitempty"`
}

type FlowRequest struct {
	Flow     string                       `json:"flow"`
	State    jsonutils.RawJSON            `json:"state,omitempty"`
	Event    FlowEvent                    `json:"event"`
	Now      string                       `json:"now,omitempty"`
	Settings map[string]jsonutils.RawJSON `json:"settings,omitempty"`
}

type FlowAction struct {
	Kind     string              `json:"kind"`
	Set      string              `json:"set,omitempty"`
	Rejected bool                `json:"rejected,omitempty"`
	Document jsonutils.RawJSON   `json:"document,omitempty"`
	Fields   []string            `json:"fields,omitempty"`
	Message  *wasmmodule.Message `json:"message,omitempty"`
	Due      *int64              `json:"due,omitempty"`
	Proposal string              `json:"proposal,omitempty"`
	ID       string              `json:"id,omitempty"`
	Map      string              `json:"map,omitempty"`
}

type FlowStep struct {
	State   jsonutils.RawJSON   `json:"state"`
	Actions []FlowAction        `json:"actions"`
	Wake    *int64              `json:"wake"`
	Error   *wasmmodule.Message `json:"error"`
}

// FlowSummary is what the host reads of a flow instance's state to list and route it.
type FlowSummary struct {
	Key              string    `json:"key"`
	Load             LoadState `json:"load"`
	AwaitingResponse *int64    `json:"awaitingResponse"`
	ResponseDue      *int64    `json:"responseDue"`
	Escalated        bool      `json:"escalated"`
	Declined         bool      `json:"declined"`
	Held             bool      `json:"held"`
	DeclineReason    string    `json:"declineReason"`
	Pending          []Pending `json:"pending"`
}

type Pending struct {
	ID      string `json:"id"`
	Set     string `json:"set"`
	Control string `json:"control"`
	Due     int64  `json:"due"`
	Resends int    `json:"resends"`
}
