package aiprovider

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
)

// DefaultAdvisorModel is the advisor a provider consults when its advisor is
// turned on without a model named.
const DefaultAdvisorModel = "claude-opus-5-5"

var (
	ErrAdvisorNeedsAnthropic = errors.New(
		"the advisor needs a provider that speaks the Anthropic Messages API",
	)
	ErrAdvisorNotServed   = errors.New("the platform does not serve the advisor tool")
	ErrAdvisorUnreadable  = errors.New("the advisor is not a Claude model this system can read")
	ErrAdvisorUnpriced    = errors.New("the advisor has no list price")
	ErrExecutorUnreadable = errors.New(
		"the provider's model is not a Claude model this system can read",
	)
	ErrAdvisorLessCapable = errors.New("the advisor is less capable than the provider's model")
)

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
// declares, empty when the advisor is off or the platform does not serve it.
func (p *Provider) ResolvedAdvisorModel() string {
	if p == nil || !p.AdvisorEnabled || p.Kind != KindAnthropicMessages ||
		!p.ResolvedPlatform().ServesAdvisor() {
		return ""
	}
	return p.namedAdvisorModel()
}

// CheckAdvisor says whether this provider may consult advisor: it speaks the
// Anthropic Messages API on a platform that serves the advisor tool, the
// advisor is a Claude model with a list price, so its calls are priced, and it
// is at least as capable as the provider's own model.
func (p *Provider) CheckAdvisor(advisor string) error {
	if p == nil || p.Kind != KindAnthropicMessages {
		return ErrAdvisorNeedsAnthropic
	}
	if platform := p.ResolvedPlatform(); !platform.ServesAdvisor() {
		return fmt.Errorf("%w: %s; use the Claude API or Claude Platform on AWS",
			ErrAdvisorNotServed, platform)
	}
	advised, ok := ParseClaudeModel(advisor)
	if !ok {
		return fmt.Errorf("%w: %q", ErrAdvisorUnreadable, advisor)
	}
	if _, priced := ListPrice(KindAnthropicMessages, advisor); !priced {
		return fmt.Errorf("%w: %q", ErrAdvisorUnpriced, advisor)
	}
	executor, ok := ParseClaudeModel(p.Model)
	if !ok {
		return fmt.Errorf("%w: %q", ErrExecutorUnreadable, p.Model)
	}
	if !advised.AtLeastAsCapableAs(executor) {
		return fmt.Errorf("%w: %s advising %s", ErrAdvisorLessCapable, advisor, p.Model)
	}

	return nil
}

func (p *Provider) validateAgentSettings(multiErr *errortypes.MultiError) {
	if len(strings.TrimSpace(p.AdvisorModel)) > maxModelLength {
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
	if p.Scripts && !p.ServesTask(TaskAssistantChat) {
		multiErr.Add("scripts", errortypes.ErrInvalid,
			"Scripts apply only to a provider that handles assistant chat")
	}
}

func (p *Provider) validateAdvisor(multiErr *errortypes.MultiError) {
	if !p.ServesTask(TaskAssistantChat) {
		multiErr.Add("advisorEnabled", errortypes.ErrInvalid,
			"The advisor applies only to a provider that handles assistant chat")
		return
	}
	err := p.CheckAdvisor(p.namedAdvisorModel())
	switch {
	case err == nil:
	case errors.Is(err, ErrAdvisorNeedsAnthropic), errors.Is(err, ErrAdvisorNotServed):
		multiErr.Add("advisorEnabled", errortypes.ErrInvalid,
			"The advisor is served only on Anthropic Messages providers that call the "+
				"Claude API or Claude Platform on AWS")
	case errors.Is(err, ErrExecutorUnreadable):
		multiErr.Add("advisorEnabled", errortypes.ErrInvalid,
			"The advisor needs a Claude model id this system can read, to check the advisor "+
				"is at least as capable")
	case errors.Is(err, ErrAdvisorLessCapable):
		multiErr.Add("advisorModel", errortypes.ErrInvalid,
			"The advisor must be at least as capable as the provider's model")
	default:
		multiErr.Add("advisorModel", errortypes.ErrInvalid,
			"The advisor must be a Claude model with a list price")
	}
}

// namedAdvisorModel is the advisor the setting names, the default when none
// is named, whatever the platform.
func (p *Provider) namedAdvisorModel() string {
	if model := strings.TrimSpace(p.AdvisorModel); model != "" {
		return model
	}

	return DefaultAdvisorModel
}

// claudeFamilyRank orders the families by capability. A family this does not
// know ranks lowest, so it is never taken for a stronger advisor.
func claudeFamilyRank(family ClaudeFamily) int {
	switch family {
	case ClaudeFamilyHaiku:
		return 1
	case ClaudeFamilySonnet:
		return 2
	case ClaudeFamilyOpus:
		return 3
	case ClaudeFamilyFable, ClaudeFamilyMythos:
		return 4
	default:
		return 0
	}
}

// AtLeastAsCapableAs reports a model of a higher family, or of the same
// family and at least the same version.
func (m ClaudeModel) AtLeastAsCapableAs(other ClaudeModel) bool {
	mine, theirs := claudeFamilyRank(m.Family), claudeFamilyRank(other.Family)
	if mine == 0 {
		return false
	}
	if mine != theirs {
		return mine > theirs
	}

	return m.Version >= other.Version
}
