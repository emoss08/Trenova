package benchscenario

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
)

type Fire struct {
	Agent            string                        `yaml:"agent"            json:"agent"`
	Event            agenttrigger.EventKind        `yaml:"event"            json:"event,omitempty"`
	Subject          string                        `yaml:"subject"          json:"subject,omitempty"`
	Schedule         bool                          `yaml:"schedule"         json:"schedule,omitempty"`
	Ceiling          agent.AutonomyTier            `yaml:"ceiling"          json:"ceiling,omitempty"`
	ToolTiers        map[string]agent.AutonomyTier `yaml:"toolTiers"        json:"toolTiers,omitempty"`
	StopWhileWaiting bool                          `yaml:"stopWhileWaiting" json:"stopWhileWaiting,omitempty"`
}

func (f *Fire) Template() agenttemplate.Template {
	return agenttemplate.Template(strings.TrimSpace(f.Agent))
}

func (f *Fire) Label() string {
	if f.Schedule {
		return "scheduled run of " + f.Agent
	}

	return string(f.Event) + " for " + f.Agent
}

func (f *Fire) validate(where string) error {
	template := f.Template()
	if !template.IsValid() {
		return fmt.Errorf("%s: fire.agent %q is not an agent template", where, f.Agent)
	}

	switch agentpreset.StarterTrigger(template) {
	case agentdefinition.TriggerEvent:
		if f.Schedule {
			return fmt.Errorf("%s: %s is started by events, not a schedule", where, template)
		}
		if !slices.Contains(agentpreset.StarterEvents(template), f.Event) {
			return fmt.Errorf("%s: %s does not answer the event %q", where, template, f.Event)
		}
		if strings.TrimSpace(f.Subject) == "" {
			return fmt.Errorf("%s: an event run names its subject with a SELECT", where)
		}
		fact := Fact{SQL: f.Subject}
		if err := fact.validate(); err != nil {
			return fmt.Errorf("%s: subject: %w", where, err)
		}
	case agentdefinition.TriggerScheduled, agentdefinition.TriggerContinuous:
		if !f.Schedule || f.Event != "" || f.Subject != "" {
			return fmt.Errorf("%s: %s runs on a schedule; give schedule: true and no event", where,
				template)
		}
	case agentdefinition.TriggerChat:
		return fmt.Errorf("%s: %s is a chat agent; fire starts event and scheduled runs", where,
			template)
	default:
		return fmt.Errorf("%s: %s has no trigger the bench can fire", where, template)
	}

	return f.validateShape(where, template)
}

func (f *Fire) validateShape(where string, template agenttemplate.Template) error {
	if f.Ceiling != "" && !f.Ceiling.IsValid() {
		return fmt.Errorf("%s: fire.ceiling %q is not an autonomy tier", where, f.Ceiling)
	}
	held := agentpreset.StarterTools(template)
	for tool, tier := range f.ToolTiers {
		if !tier.IsValid() {
			return fmt.Errorf("%s: fire.toolTiers.%s %q is not an autonomy tier", where, tool, tier)
		}
		if !agentdefinition.IsCoreTool(tool) && !slices.Contains(held, tool) {
			return fmt.Errorf("%s: fire.toolTiers names %s, which %s does not hold", where, tool,
				template)
		}
	}

	return nil
}

func (f *Fire) AutonomyCeiling(template agenttemplate.Template) agent.AutonomyTier {
	if f.Ceiling != "" {
		return f.Ceiling
	}

	return agentpreset.StarterCeiling(template)
}
