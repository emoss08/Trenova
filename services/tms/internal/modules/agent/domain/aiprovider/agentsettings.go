package aiprovider

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// DefaultAdvisorModel is the advisor a provider consults when its advisor is
// turned on without a model named.
const DefaultAdvisorModel = "claude-opus-5-5"

var (
	ErrAdvisorNeedsAnthropic = errors.New(
		"the advisor needs a provider that speaks the Anthropic Messages API",
	)
	ErrAdvisorNotServed = errors.New("the platform does not serve the advisor tool")
)

// AdvisorRefusal is the API refusing the advisor a provider declared, kept
// so AI control can show it. Whether an advisor is strong enough for the
// provider's model is the API's to say, and it says so with a 400 on the
// first call; until the advisor or the model changes, calls go out without it.
type AdvisorRefusal struct {
	Advisor   string `json:"advisor"`
	Model     string `json:"model"`
	Message   string `json:"message"`
	RefusedAt int64  `json:"refusedAt"`
}

// ScriptSettings say whether a provider's model may answer with code mode: a
// read-only script (Scripts), and a script that prepares writes proposed as
// one batch for a person (Writes, which needs Scripts).
type ScriptSettings struct {
	Scripts bool `json:"scripts,omitempty"`
	Writes  bool `json:"writes,omitempty"`
}

// ScriptSettings is the provider's code-mode setting, with Writes only where
// Scripts is on.
func (p *Provider) ScriptSettings() ScriptSettings {
	if p == nil || !p.Scripts {
		return ScriptSettings{}
	}

	return ScriptSettings{Scripts: true, Writes: p.ScriptWrites}
}

// ResolvedAdvisorModel is the advisor every chat call routed to this provider
// declares, empty when the advisor is off, the platform does not serve it, or
// the API refused this advisor for this model.
func (p *Provider) ResolvedAdvisorModel() string {
	if p == nil || !p.AdvisorEnabled || p.Kind != KindAnthropicMessages ||
		!p.ResolvedPlatform().ServesAdvisor() {
		return ""
	}
	advisor := p.namedAdvisorModel()
	if p.AdvisorRefused(advisor) {
		return ""
	}

	return advisor
}

// AdvisorRefused reports a refusal the API gave this advisor for the
// provider's current model.
func (p *Provider) AdvisorRefused(advisor string) bool {
	return p.AdvisorRefusal != nil &&
		strings.EqualFold(p.AdvisorRefusal.Advisor, advisor) &&
		strings.EqualFold(p.AdvisorRefusal.Model, p.Model)
}

// CheckAdvisor says whether this provider can declare an advisor at all: it
// speaks the Anthropic Messages API on a platform that serves the advisor
// tool. That the advisor is a model the provider serves is checked against
// its model list when it is saved.
func (p *Provider) CheckAdvisor() error {
	if p == nil || p.Kind != KindAnthropicMessages {
		return ErrAdvisorNeedsAnthropic
	}
	if platform := p.ResolvedPlatform(); !platform.ServesAdvisor() {
		return fmt.Errorf("%w: %s; use the Claude API or Claude Platform on AWS",
			ErrAdvisorNotServed, platform)
	}

	return nil
}

func (p *Provider) validateAgentSettings(multiErr *errortypes.MultiError) {
	if len(p.AdvisorModel) > maxModelLength {
		multiErr.Add("advisorModel", errortypes.ErrInvalid,
			"Advisor model cannot be longer than 200 characters")
	}
	if p.AdvisorEnabled {
		p.validateAdvisor(multiErr)
	}
	if p.ScriptWrites && !p.Scripts {
		multiErr.Add("scriptWrites", errortypes.ErrInvalid,
			"Scripts may propose writes only when scripts are turned on")
	}
	if p.Scripts && !p.ServesTask(aitask.TaskAssistantChat) {
		multiErr.Add("scripts", errortypes.ErrInvalid,
			"Scripts apply only to a provider that handles assistant chat")
	}
}

func (p *Provider) validateAdvisor(multiErr *errortypes.MultiError) {
	if !p.ServesTask(aitask.TaskAssistantChat) {
		multiErr.Add("advisorEnabled", errortypes.ErrInvalid,
			"The advisor applies only to a provider that handles assistant chat")
		return
	}
	if err := p.CheckAdvisor(); err != nil {
		multiErr.Add("advisorEnabled", errortypes.ErrInvalid,
			"The advisor is served only on Anthropic Messages providers that call the "+
				"Claude API or Claude Platform on AWS")
	}
}

// AdvisorModelName is the advisor the setting names, the default when none
// is named, whatever the platform.
func (p *Provider) AdvisorModelName() string {
	return p.namedAdvisorModel()
}

func (p *Provider) namedAdvisorModel() string {
	if model := strings.TrimSpace(p.AdvisorModel); model != "" {
		return model
	}

	return DefaultAdvisorModel
}
