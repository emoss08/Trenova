package hazmatsegreationrulevalidator

import (
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
)

// SegregationViolation represents a violation of a hazmat segregation rule.
// It contains the rule that was violated, the commodities involved, and the hazardous materials involved.
// It also contains a message that describes the violation.
type SegregationViolation struct {
	Rule       *hazmatsegregationrule.HazmatSegregationRule
	CommodityA *commodity.Commodity
	CommodityB *commodity.Commodity
	HazmatA    *hazardousmaterial.HazardousMaterial
	HazmatB    *hazardousmaterial.HazardousMaterial
	Message    string
}

type hazmatPair struct {
	classA    hazardousmaterial.HazardousClass
	classB    hazardousmaterial.HazardousClass
	hazmatAID string
	hazmatBID string
}
