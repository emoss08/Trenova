package documentanalysis

const (
	ClassificationSettledByRule  = "rule"
	ClassificationSettledByModel = "model"

	ClassificationConfidentThreshold = reviewRequiredConfidenceFloor
)

type ClassificationSettlement struct {
	SettledBy       string
	RuleKind        string
	RuleConfidence  float64
	ModelKind       string
	ModelConfidence float64
}

func (s *ClassificationSettlement) ToMap() map[string]any {
	data := map[string]any{
		"settledBy":      s.SettledBy,
		"ruleKind":       s.RuleKind,
		"ruleConfidence": s.RuleConfidence,
	}
	if s.ModelKind != "" {
		data["modelKind"] = s.ModelKind
		data["modelConfidence"] = s.ModelConfidence
	}
	return data
}

func ConfidentClassification(confidence float64) bool {
	return confidence >= ClassificationConfidentThreshold
}

func settledByRule(rule *ClassificationResult) *ClassificationResult {
	rule.Settlement = &ClassificationSettlement{
		SettledBy:      ClassificationSettledByRule,
		RuleKind:       rule.Kind,
		RuleConfidence: rule.Confidence,
	}
	return rule
}

func SettleClassification(rule, model *ClassificationResult) *ClassificationResult {
	settlement := &ClassificationSettlement{
		SettledBy:      ClassificationSettledByRule,
		RuleKind:       rule.Kind,
		RuleConfidence: rule.Confidence,
	}
	settled := *rule
	if model != nil {
		settlement.ModelKind = model.Kind
		settlement.ModelConfidence = model.Confidence
		if !ConfidentClassification(rule.Confidence) &&
			ConfidentClassification(model.Confidence) {
			settled = *model
			settlement.SettledBy = ClassificationSettledByModel
		}
	}
	settled.Settlement = settlement

	return &settled
}
