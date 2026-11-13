package aiusage

import "unicode/utf8"

type Feature string

const (
	FeatureAgentTurn                   = Feature("AgentTurn")
	FeatureAgentWakeUp                 = Feature("AgentWakeUp")
	FeatureAgentEvaluation             = Feature("AgentEvaluation")
	FeatureTableQuery                  = Feature("TableQuery")
	FeatureFormulaGenerate             = Feature("FormulaGenerate")
	FeatureFormulaExplain              = Feature("FormulaExplain")
	FeatureShipmentImportChat          = Feature("ShipmentImportChat")
	FeatureDocumentIntelligenceRoute   = Feature("DocumentIntelligenceRoute")
	FeatureDocumentIntelligenceExtract = Feature("DocumentIntelligenceExtract")
	FeatureAccountingMapping           = Feature("AccountingMapping")
	FeatureFactoringRemittanceExtract  = Feature("FactoringRemittanceExtract")
	FeatureAgentReflection             = Feature("AgentReflection")
	FeatureAgentMemoryConsolidation    = Feature("AgentMemoryConsolidation")
	FeatureAgentDrafting               = Feature("AgentDrafting")
	FeatureAgentProfile                = Feature("AgentProfile")
	FeatureAgentRecommendation         = Feature("AgentRecommendation")
	FeatureConversationChapters        = Feature("ConversationChapters")
	FeatureBillingQueueReview          = Feature("BillingQueueReview")
	FeatureBillingQueueBrief           = Feature("BillingQueueBrief")
	FeatureBillingQueueSearch          = Feature("BillingQueueSearch")
	FeaturePaperworkReading            = Feature("PaperworkReading")
	FeatureInboundMail                 = Feature("InboundMail")
	FeaturePaletteAnswer               = Feature("PaletteAnswer")
	FeatureInsightNarration            = Feature("InsightNarration")
	FeatureCaseReading                 = Feature("CaseReading")
	FeatureRequestWording              = Feature("RequestWording")
)

func AllFeatures() []Feature {
	return []Feature{
		FeatureAgentTurn,
		FeatureAgentWakeUp,
		FeatureAgentEvaluation,
		FeatureTableQuery,
		FeatureFormulaGenerate,
		FeatureFormulaExplain,
		FeatureShipmentImportChat,
		FeatureDocumentIntelligenceRoute,
		FeatureDocumentIntelligenceExtract,
		FeatureAccountingMapping,
		FeatureFactoringRemittanceExtract,
		FeatureAgentReflection,
		FeatureAgentMemoryConsolidation,
		FeatureAgentDrafting,
		FeatureAgentProfile,
		FeatureAgentRecommendation,
		FeatureConversationChapters,
		FeatureBillingQueueReview,
		FeatureBillingQueueBrief,
		FeatureBillingQueueSearch,
		FeaturePaperworkReading,
		FeatureInboundMail,
		FeaturePaletteAnswer,
		FeatureInsightNarration,
		FeatureCaseReading,
		FeatureRequestWording,
	}
}

type SubjectType string

const (
	SubjectTypeDocument             = SubjectType("Document")
	SubjectTypeFormulaSchema        = SubjectType("FormulaSchema")
	SubjectTypeAccountingConnection = SubjectType("AccountingConnection")
)

func AllSubjectTypes() []SubjectType {
	return []SubjectType{
		SubjectTypeDocument,
		SubjectTypeFormulaSchema,
		SubjectTypeAccountingConnection,
	}
}

const MaxSubjectIDLength = 100

type Subject struct {
	Type SubjectType `json:"type"`
	ID   string      `json:"id"`
}

func (s Subject) Recordable() bool {
	return s.Type != "" && s.ID != "" && utf8.RuneCountInString(s.ID) <= MaxSubjectIDLength
}
