package tablechangealert

type ConditionOperator string

const (
	OpEq          = ConditionOperator("eq")
	OpNeq         = ConditionOperator("neq")
	OpGt          = ConditionOperator("gt")
	OpGte         = ConditionOperator("gte")
	OpLt          = ConditionOperator("lt")
	OpLte         = ConditionOperator("lte")
	OpIsNull      = ConditionOperator("is_null")
	OpIsNotNull   = ConditionOperator("is_not_null")
	OpContains    = ConditionOperator("contains")
	OpNotContains = ConditionOperator("not_contains")
	OpChangedTo   = ConditionOperator("changed_to")
	OpChangedFrom = ConditionOperator("changed_from")
	OpChanged     = ConditionOperator("changed")
)

type Condition struct {
	Field    string            `json:"field"`
	Operator ConditionOperator `json:"operator"`
	Value    any               `json:"value"`
}

func ValidConditionOperator(s string) bool {
	switch ConditionOperator(s) {
	case OpEq, OpNeq, OpGt, OpGte, OpLt, OpLte,
		OpIsNull, OpIsNotNull, OpContains, OpNotContains,
		OpChangedTo, OpChangedFrom, OpChanged:
		return true
	default:
		return false
	}
}

func IsUnaryOperator(op ConditionOperator) bool {
	switch op {
	case OpIsNull, OpIsNotNull, OpChanged:
		return true
	default:
		return false
	}
}
