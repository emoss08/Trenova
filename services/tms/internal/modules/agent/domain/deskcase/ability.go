package deskcase

import (
	"slices"

	"github.com/emoss08/trenova/shared/pulid"
)

// stepTools are the tools that can take each next step, any one of which
// is enough. A step's request to the agent is only worth sending to an
// agent that holds one of them and whose person may use it; otherwise the
// agent can only say it cannot, as Shipment Assistant did when asked to mark
// a shipment ready to invoice. A step the organization added names no tool:
// what it asks is the organization's own words, and any agent may try.
var stepTools = map[StepKey][]string{
	StepTrackDelivery:       {"get_shipment_tracking"},
	StepRequestPOD:          {"request_missing_docs"},
	StepRequestPaperwork:    {"request_missing_docs"},
	StepReviewRate:          {"get_billing_queue_item", "explain_rate"},
	StepConfirmRate:         {"send_rate_confirmation", "record_rate_confirmation_confirmed"},
	StepApproveAccessorials: {"approve_detention", "get_detention_occurrence"},
	StepNotifyCustomer:      {"email_customer"},
	StepClearHolds:          {"release_shipment_hold", "resolve_service_failure"},
	StepMarkReady:           {"transfer_to_billing"},
	StepSendInvoice:         {"send_invoice"},
	StepPostInvoice:         {"post_invoice"},
	StepWorkDispute:         {"resolve_invoice_dispute", "list_invoice_disputes"},
	StepFollowUpPayment:     {"email_customer"},
}

// StepTools are the tools that can take a step, nil for a step the
// organization added.
func StepTools(step StepKey) []string {
	return stepTools[step]
}

// AllStepTools is every tool any step names, for checking each is still
// registered.
func AllStepTools() []string {
	out := make([]string, 0, len(stepTools)*2)
	for _, tools := range stepTools {
		for _, tool := range tools {
			if !slices.Contains(out, tool) {
				out = append(out, tool)
			}
		}
	}
	slices.Sort(out)

	return out
}

// AbilityVia is who can take a step: the conversation's own agent; another
// agent the person may use, asked from this conversation (the person hands
// it the step, it runs as them inside the conversation's turn, and its
// answer stays in the thread for the conversation's agent to read); or
// nobody the person may ask, so the person takes it on the page where it is
// done.
type AbilityVia string

const (
	ViaAgent  = AbilityVia("Agent")
	ViaAsk    = AbilityVia("Ask")
	ViaPerson = AbilityVia("Person")
)

// StepAbility is who can take one step; for another agent, which.
type StepAbility struct {
	Via       AbilityVia `json:"via"`
	AgentID   pulid.ID   `json:"agentId,omitempty"`
	AgentName string     `json:"agentName,omitempty"`
}

// Steps are the steps a checklist offers: each blocked or pending item's
// step and the next one, once each, in the order the checklist shows them.
func (c *Checklist) Steps() []StepKey {
	if c == nil {
		return nil
	}
	out := make([]StepKey, 0, len(c.Items)+1)
	add := func(step StepKey) {
		if step != "" && !slices.Contains(out, step) {
			out = append(out, step)
		}
	}
	add(c.Next)
	for _, item := range c.Items {
		add(item.Step)
	}

	return out
}

// Holder is one agent the person may use, with the tools it holds.
type Holder struct {
	ID    pulid.ID
	Name  string
	Tools map[string]struct{}
}

func (h *Holder) holdsAny(tools []string, permitted map[string]struct{}) bool {
	for _, tool := range tools {
		_, held := h.Tools[tool]
		_, allowed := permitted[tool]
		if held && allowed {
			return true
		}
	}

	return false
}

// AbilityInputs is what deciding who can take each step reads: the
// conversation's agent, the other agents the person may use in the order
// they are offered, and the tools the person may use at all.
type AbilityInputs struct {
	Steps     []StepKey
	Own       *Holder
	Others    []*Holder
	Permitted map[string]struct{}
}

// Abilities says who can take each step. The conversation's agent is
// preferred; then the first other agent the person may use that can, asked
// from the same conversation; then the person.
func Abilities(in *AbilityInputs) map[StepKey]StepAbility {
	out := make(map[StepKey]StepAbility, len(in.Steps))
	for _, step := range in.Steps {
		tools := StepTools(step)
		if tools == nil || (in.Own != nil && in.Own.holdsAny(tools, in.Permitted)) {
			out[step] = StepAbility{Via: ViaAgent}
			continue
		}
		ability := StepAbility{Via: ViaPerson}
		for _, other := range in.Others {
			if other.holdsAny(tools, in.Permitted) {
				ability = StepAbility{Via: ViaAsk, AgentID: other.ID, AgentName: other.Name}
				break
			}
		}
		out[step] = ability
	}

	return out
}

// NeedsOthers reports a step the conversation's agent cannot take, which is
// when the other agents are worth reading at all.
func NeedsOthers(steps []StepKey, own *Holder, permitted map[string]struct{}) bool {
	for _, step := range steps {
		tools := StepTools(step)
		if tools != nil && (own == nil || !own.holdsAny(tools, permitted)) {
			return true
		}
	}

	return false
}

// ToolsFor is every tool the steps name, to ask once which the person may
// use.
func ToolsFor(steps []StepKey) []string {
	out := make([]string, 0, len(steps)*2)
	for _, step := range steps {
		for _, tool := range StepTools(step) {
			if !slices.Contains(out, tool) {
				out = append(out, tool)
			}
		}
	}

	return out
}
