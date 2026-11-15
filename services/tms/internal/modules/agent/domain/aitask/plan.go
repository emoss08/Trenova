package aitask

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

// trenovaAI is every Trenova AI feature whose work calls a model. A task
// shared by several of them (the Desk's chat answers people and drives agent
// runs) runs when any of them is on; the entry point of each piece of work
// checks its own feature.
var trenovaAI = []platformcatalog.FeatureKey{
	platformcatalog.FeatureAIDesk,
	platformcatalog.FeatureAgentAutomation,
	platformcatalog.FeatureAIWatchtower,
	platformcatalog.FeatureAITriage,
}

// Features are the plan features that pay for the task: the router serves
// it only when one of them is on, so a model is never called for work the
// organization's plan does not include. Reading a document into fields is
// Document Intelligence and runs without Trenova AI; everything else is
// Trenova AI.
func (t Task) Features() []platformcatalog.FeatureKey {
	switch t {
	case TaskDocumentClassification,
		TaskDocumentExtraction,
		TaskRemittanceExtraction,
		TaskPaperworkReading:
		return []platformcatalog.FeatureKey{platformcatalog.FeatureDocumentIntelligence}
	case TaskFormulaAssistant,
		TaskOperationalInsights,
		TaskDailyBriefing,
		TaskQueryCompose,
		TaskChapterTitling,
		TaskPaletteAnswer,
		TaskCaseReading,
		TaskEmbedding:
		return []platformcatalog.FeatureKey{platformcatalog.FeatureAIDesk}
	case TaskBillingDiagnosis, TaskAccountingMapping, TaskRequestWording:
		return []platformcatalog.FeatureKey{platformcatalog.FeatureAgentAutomation}
	case TaskInboundClassification:
		return []platformcatalog.FeatureKey{platformcatalog.FeatureAITriage}
	case TaskScopeClassification, TaskAssistantChat, TaskEvaluationJudge, TaskGeneral:
		return slices.Clone(trenovaAI)
	default:
		return nil
	}
}
