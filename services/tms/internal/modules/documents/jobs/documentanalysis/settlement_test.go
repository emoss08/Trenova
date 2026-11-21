package documentanalysis

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ruleResult(kind string, confidence float64) *ClassificationResult {
	return &ClassificationResult{
		Kind:       kind,
		Confidence: confidence,
		Source:     "deterministic",
		Reason:     "rule reason",
	}
}

func modelResult(kind string, confidence float64) *ClassificationResult {
	return &ClassificationResult{
		Kind:       kind,
		Confidence: confidence,
		Source:     "ai",
		Reason:     "model reason",
	}
}

func TestSettleClassification_ConfidentRuleStandsWhenTheModelDisagrees(t *testing.T) {
	t.Parallel()

	rule := ruleResult(KindRateConfirmation, ClassificationConfidentThreshold)

	settled := SettleClassification(rule, modelResult(KindBillOfLading, 0.99))

	assert.Equal(t, KindRateConfirmation, settled.Kind)
	assert.InDelta(t, 0.8, settled.Confidence, 1e-9)
	assert.Equal(t, "deterministic", settled.Source)
	assert.Equal(t, "rule reason", settled.Reason)
	assert.Equal(t, &ClassificationSettlement{
		SettledBy:       ClassificationSettledByRule,
		RuleKind:        KindRateConfirmation,
		RuleConfidence:  0.8,
		ModelKind:       KindBillOfLading,
		ModelConfidence: 0.99,
	}, settled.Settlement)
}

func TestSettleClassification_ConfidentModelSettlesAnUnsureRule(t *testing.T) {
	t.Parallel()

	rule := ruleResult(KindRateConfirmation, 0.7)

	settled := SettleClassification(rule, modelResult(KindBillOfLading, 0.9))

	assert.Equal(t, KindBillOfLading, settled.Kind)
	assert.InDelta(t, 0.9, settled.Confidence, 1e-9)
	assert.Equal(t, "ai", settled.Source)
	assert.Equal(t, "model reason", settled.Reason)
	assert.Equal(t, &ClassificationSettlement{
		SettledBy:       ClassificationSettledByModel,
		RuleKind:        KindRateConfirmation,
		RuleConfidence:  0.7,
		ModelKind:       KindBillOfLading,
		ModelConfidence: 0.9,
	}, settled.Settlement)
	assert.Equal(t, KindRateConfirmation, rule.Kind)
	assert.Nil(t, rule.Settlement)
}

func TestSettleClassification_UnsureRuleStandsWhenTheModelIsUnsure(t *testing.T) {
	t.Parallel()

	settled := SettleClassification(
		ruleResult(KindOther, 0.3),
		modelResult(KindRateConfirmation, 0.79),
	)

	assert.Equal(t, KindOther, settled.Kind)
	assert.InDelta(t, 0.3, settled.Confidence, 1e-9)
	assert.Equal(t, &ClassificationSettlement{
		SettledBy:       ClassificationSettledByRule,
		RuleKind:        KindOther,
		RuleConfidence:  0.3,
		ModelKind:       KindRateConfirmation,
		ModelConfidence: 0.79,
	}, settled.Settlement)
}

func TestSettleClassification_RuleStandsWithoutAModelAnswer(t *testing.T) {
	t.Parallel()

	settled := SettleClassification(ruleResult(KindOther, 0.3), nil)

	assert.Equal(t, KindOther, settled.Kind)
	assert.Equal(t, &ClassificationSettlement{
		SettledBy:      ClassificationSettledByRule,
		RuleKind:       KindOther,
		RuleConfidence: 0.3,
	}, settled.Settlement)
}

func TestClassifyDocumentWithControl_RecordsTheRuleAsWhatSettledTheKind(t *testing.T) {
	t.Parallel()

	control := documentcontrol.NewDefaultDocumentControl(
		pulid.MustNew("org_"),
		pulid.MustNew("bu_"),
	)
	text := "Bill of Lading\nShipper: ACME Foods\nConsignee: Blue Market\nBOL #: B12345"
	features := ExtractDocumentFeatures("bol.txt", nil, text)

	result := ClassifyDocumentWithControl("bol.txt", text, control, features, nil)

	require.NotNil(t, result.Settlement)
	assert.Equal(t, ClassificationSettledByRule, result.Settlement.SettledBy)
	assert.Equal(t, result.Kind, result.Settlement.RuleKind)
	assert.InDelta(t, result.Confidence, result.Settlement.RuleConfidence, 1e-9)
	assert.Empty(t, result.Settlement.ModelKind)
}

func TestAnalyzeDocument_RecordsTheSettlementForAudit(t *testing.T) {
	t.Parallel()

	settled := SettleClassification(
		ruleResult(KindOther, 0.3),
		modelResult(KindBillOfLading, 0.92),
	)

	analysis := AnalyzeDocument(settled, &ExtractionResult{Text: "Bill of Lading"})

	assert.Equal(t, KindBillOfLading, analysis.Kind)
	assert.Equal(t, "ai", analysis.ClassifierSource)
	assert.Equal(t, map[string]any{
		"settledBy":       ClassificationSettledByModel,
		"ruleKind":        KindOther,
		"ruleConfidence":  0.3,
		"modelKind":       KindBillOfLading,
		"modelConfidence": 0.92,
	}, analysis.ToMap()["classificationSettlement"])
}
