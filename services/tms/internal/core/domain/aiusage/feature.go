package aiusage

import "unicode/utf8"

type Feature string

const (
	FeatureAgentTurn                   = Feature("AgentTurn")
	FeatureAgentEvaluation             = Feature("AgentEvaluation")
	FeatureTableQuery                  = Feature("TableQuery")
	FeatureFormulaGenerate             = Feature("FormulaGenerate")
	FeatureFormulaExplain              = Feature("FormulaExplain")
	FeatureShipmentImportChat          = Feature("ShipmentImportChat")
	FeatureDocumentIntelligenceRoute   = Feature("DocumentIntelligenceRoute")
	FeatureDocumentIntelligenceExtract = Feature("DocumentIntelligenceExtract")
)

func AllFeatures() []Feature {
	return []Feature{
		FeatureAgentTurn,
		FeatureAgentEvaluation,
		FeatureTableQuery,
		FeatureFormulaGenerate,
		FeatureFormulaExplain,
		FeatureShipmentImportChat,
		FeatureDocumentIntelligenceRoute,
		FeatureDocumentIntelligenceExtract,
	}
}

type SubjectType string

const (
	SubjectTypeDocument      = SubjectType("Document")
	SubjectTypeFormulaSchema = SubjectType("FormulaSchema")
)

func AllSubjectTypes() []SubjectType {
	return []SubjectType{
		SubjectTypeDocument,
		SubjectTypeFormulaSchema,
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
