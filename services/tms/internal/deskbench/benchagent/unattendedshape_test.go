package benchagent

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/stretchr/testify/assert"
)

func TestUnattendedTemplates_CoverEveryEventAndScheduledTemplate(t *testing.T) {
	t.Parallel()

	templates := UnattendedTemplates()
	assert.Contains(t, templates, agenttemplate.TemplateLoadMonitor)
	assert.Contains(t, templates, agenttemplate.TemplateDispatchAssignment)
	assert.Contains(t, templates, agenttemplate.TemplateEDIDesk)
	for _, template := range templates {
		trigger := agentpreset.StarterTrigger(template)
		assert.True(t,
			trigger == agentdefinition.TriggerEvent || trigger == agentdefinition.TriggerScheduled,
			template)
		shape := UnattendedShape(template)
		assert.True(t, shape.LearningOff)
		assert.False(t, shape.Enabled, "kept off between cases, so nothing real starts it")
		assert.Equal(t, agentdefinition.OutputReport, shape.OutputMode)
		if trigger == agentdefinition.TriggerScheduled {
			assert.Equal(t, unattendedCron, shape.CronExpression)
		}
	}
}

func TestUnattendedMatches_ResetsToolTiersALeftRunSet(t *testing.T) {
	t.Parallel()

	template := agenttemplate.TemplateBillingException
	existing := UnattendedShape(template)
	assert.True(t, UnattendedMatches(existing, UnattendedShape(template)))

	existing.ToolTiers = map[string]agent.AutonomyTier{
		agentdefinition.CoreToolRaiseException: agent.TierPropose,
	}
	assert.False(t, UnattendedMatches(existing, UnattendedShape(template)),
		"tiers a crashed case left on the agent are cleared by the next sync")
}
