package benchagent

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentinstructions"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
)

const (
	unattendedSuffix = " (unattended)"
	unattendedCron   = "0 3 1 1 *"
)

var unattendedContext = []agentdefinition.ContextProvider{
	agentdefinition.ContextOrganization,
	agentdefinition.ContextClock,
	agentdefinition.ContextTools,
	agentdefinition.ContextMemory,
}

func UnattendedTemplates() []agenttemplate.Template {
	all := []agenttemplate.Template{
		agenttemplate.TemplateBillingException,
		agenttemplate.TemplateShipmentIntake,
		agenttemplate.TemplateCashApplication,
		agenttemplate.TemplateDetentionDesk,
		agenttemplate.TemplateCredentialDesk,
		agenttemplate.TemplateCustomerUpdateDesk,
		agenttemplate.TemplateCarrierRiskDesk,
		agenttemplate.TemplateIntakeDesk,
		agenttemplate.TemplateLoadEntryCheck,
		agenttemplate.TemplateServiceFailureDesk,
		agenttemplate.TemplateInsightAnalyst,
		agenttemplate.TemplateEDIDesk,
		agenttemplate.TemplateBooksKeeper,
		agenttemplate.TemplateFactoringClerk,
		agenttemplate.TemplateDispatchAssignment,
		agenttemplate.TemplateLoadMonitor,
	}

	return slices.DeleteFunc(all, func(template agenttemplate.Template) bool {
		trigger := agentpreset.StarterTrigger(template)
		return trigger != agentdefinition.TriggerEvent &&
			trigger != agentdefinition.TriggerScheduled
	})
}

func UnattendedAgentName(template agenttemplate.Template) string {
	return BenchAgentPrefix + template.Label() + unattendedSuffix
}

func UnattendedShape(template agenttemplate.Template) *agentdefinition.Definition {
	definition := &agentdefinition.Definition{
		Name:              UnattendedAgentName(template),
		Description:       template.Description(),
		Template:          template,
		Instructions:      agentinstructions.StarterInstructions(template),
		ToolNames:         agentdefinition.WithoutCoreTools(agentpreset.StarterTools(template)),
		AutonomyCeiling:   agentpreset.StarterCeiling(template),
		DataAccessCeiling: agentpreset.StarterDataAccess(template),
		TriggerMode:       agentpreset.StarterTrigger(template),
		EventKinds:        agentpreset.StarterEvents(template),
		OutputMode:        agentdefinition.OutputReport,
		ContextProviders:  slices.Clone(unattendedContext),
		LearningOff:       true,
	}
	if definition.TriggerMode == agentdefinition.TriggerScheduled {
		definition.CronExpression = unattendedCron
	}

	return definition
}

func UnattendedMatches(existing, want *agentdefinition.Definition) bool {
	held := slices.Clone(existing.ToolNames)
	wanted := slices.Clone(want.ToolNames)
	slices.Sort(held)
	slices.Sort(wanted)

	return !existing.Enabled &&
		!existing.ShadowMode &&
		existing.LearningOff &&
		existing.Template == want.Template &&
		existing.Description == want.Description &&
		existing.Instructions == want.Instructions &&
		existing.AutonomyCeiling == want.AutonomyCeiling &&
		existing.DataAccessCeiling == want.DataAccessCeiling &&
		existing.TriggerMode == want.TriggerMode &&
		existing.CronExpression == want.CronExpression &&
		slices.Equal(existing.EventKinds, want.EventKinds) &&
		existing.OutputMode == want.OutputMode &&
		len(existing.ToolTiers) == 0 &&
		slices.Equal(held, wanted)
}
