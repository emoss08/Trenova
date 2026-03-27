package tablechangealertservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/stretchr/testify/assert"
)

func TestEvaluateConditions_EmptyConditions(t *testing.T) {
	result := EvaluateConditions(nil, "all", map[string]any{}, map[string]any{}, nil)
	assert.True(t, result)
}

func TestEvaluateConditions_EqMatch(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "status", Operator: tablechangealert.OpEq, Value: "active"},
	}
	result := EvaluateConditions(conds, "all", map[string]any{"status": "active"}, nil, nil)
	assert.True(t, result)
}

func TestEvaluateConditions_EqNoMatch(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "status", Operator: tablechangealert.OpEq, Value: "active"},
	}
	result := EvaluateConditions(conds, "all", map[string]any{"status": "inactive"}, nil, nil)
	assert.False(t, result)
}

func TestEvaluateConditions_Neq(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "status", Operator: tablechangealert.OpNeq, Value: "deleted"},
	}
	assert.True(t, EvaluateConditions(conds, "all", map[string]any{"status": "active"}, nil, nil))
	assert.False(t, EvaluateConditions(conds, "all", map[string]any{"status": "deleted"}, nil, nil))
}

func TestEvaluateConditions_NumericGt(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "amount", Operator: tablechangealert.OpGt, Value: float64(100)},
	}
	assert.True(t, EvaluateConditions(conds, "all", map[string]any{"amount": float64(150)}, nil, nil))
	assert.False(t, EvaluateConditions(conds, "all", map[string]any{"amount": float64(50)}, nil, nil))
}

func TestEvaluateConditions_NumericGte(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "amount", Operator: tablechangealert.OpGte, Value: float64(100)},
	}
	assert.True(t, EvaluateConditions(conds, "all", map[string]any{"amount": float64(100)}, nil, nil))
	assert.False(t, EvaluateConditions(conds, "all", map[string]any{"amount": float64(99)}, nil, nil))
}

func TestEvaluateConditions_NumericLt(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "amount", Operator: tablechangealert.OpLt, Value: float64(100)},
	}
	assert.True(t, EvaluateConditions(conds, "all", map[string]any{"amount": float64(50)}, nil, nil))
	assert.False(t, EvaluateConditions(conds, "all", map[string]any{"amount": float64(150)}, nil, nil))
}

func TestEvaluateConditions_NumericLte(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "amount", Operator: tablechangealert.OpLte, Value: float64(100)},
	}
	assert.True(t, EvaluateConditions(conds, "all", map[string]any{"amount": float64(100)}, nil, nil))
	assert.False(t, EvaluateConditions(conds, "all", map[string]any{"amount": float64(101)}, nil, nil))
}

func TestEvaluateConditions_IsNull(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "notes", Operator: tablechangealert.OpIsNull},
	}
	assert.True(t, EvaluateConditions(conds, "all", map[string]any{}, nil, nil))
	assert.True(t, EvaluateConditions(conds, "all", map[string]any{"notes": nil}, nil, nil))
	assert.False(t, EvaluateConditions(conds, "all", map[string]any{"notes": "something"}, nil, nil))
}

func TestEvaluateConditions_IsNotNull(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "notes", Operator: tablechangealert.OpIsNotNull},
	}
	assert.True(t, EvaluateConditions(conds, "all", map[string]any{"notes": "something"}, nil, nil))
	assert.False(t, EvaluateConditions(conds, "all", map[string]any{}, nil, nil))
	assert.False(t, EvaluateConditions(conds, "all", map[string]any{"notes": nil}, nil, nil))
}

func TestEvaluateConditions_Contains(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "name", Operator: tablechangealert.OpContains, Value: "foo"},
	}
	assert.True(t, EvaluateConditions(conds, "all", map[string]any{"name": "foobar"}, nil, nil))
	assert.False(t, EvaluateConditions(conds, "all", map[string]any{"name": "baz"}, nil, nil))
}

func TestEvaluateConditions_NotContains(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "name", Operator: tablechangealert.OpNotContains, Value: "foo"},
	}
	assert.True(t, EvaluateConditions(conds, "all", map[string]any{"name": "baz"}, nil, nil))
	assert.False(t, EvaluateConditions(conds, "all", map[string]any{"name": "foobar"}, nil, nil))
}

func TestEvaluateConditions_ChangedTo(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "status", Operator: tablechangealert.OpChangedTo, Value: "active"},
	}
	assert.True(t, EvaluateConditions(conds, "all",
		map[string]any{"status": "active"},
		map[string]any{"status": "inactive"},
		nil,
	))
	assert.False(t, EvaluateConditions(conds, "all",
		map[string]any{"status": "active"},
		map[string]any{"status": "active"},
		nil,
	))
	assert.True(t, EvaluateConditions(conds, "all",
		map[string]any{"status": "active"},
		map[string]any{},
		nil,
	))
}

func TestEvaluateConditions_ChangedFrom(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "status", Operator: tablechangealert.OpChangedFrom, Value: "draft"},
	}
	assert.True(t, EvaluateConditions(conds, "all",
		map[string]any{"status": "active"},
		map[string]any{"status": "draft"},
		nil,
	))
	assert.False(t, EvaluateConditions(conds, "all",
		map[string]any{"status": "draft"},
		map[string]any{"status": "draft"},
		nil,
	))
}

func TestEvaluateConditions_Changed(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "status", Operator: tablechangealert.OpChanged},
	}
	assert.True(t, EvaluateConditions(conds, "all", nil, nil, []string{"status", "name"}))
	assert.False(t, EvaluateConditions(conds, "all", nil, nil, []string{"name"}))
}

func TestEvaluateConditions_AllRequiresAll(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "status", Operator: tablechangealert.OpEq, Value: "active"},
		{Field: "amount", Operator: tablechangealert.OpGt, Value: float64(100)},
	}
	assert.True(t, EvaluateConditions(conds, "all",
		map[string]any{"status": "active", "amount": float64(200)}, nil, nil,
	))
	assert.False(t, EvaluateConditions(conds, "all",
		map[string]any{"status": "active", "amount": float64(50)}, nil, nil,
	))
}

func TestEvaluateConditions_AnyRequiresOne(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "status", Operator: tablechangealert.OpEq, Value: "active"},
		{Field: "amount", Operator: tablechangealert.OpGt, Value: float64(100)},
	}
	assert.True(t, EvaluateConditions(conds, "any",
		map[string]any{"status": "inactive", "amount": float64(200)}, nil, nil,
	))
	assert.False(t, EvaluateConditions(conds, "any",
		map[string]any{"status": "inactive", "amount": float64(50)}, nil, nil,
	))
}

func TestEvaluateConditions_TypeCoercion(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "count", Operator: tablechangealert.OpGt, Value: "10"},
	}
	assert.True(t, EvaluateConditions(conds, "all", map[string]any{"count": float64(20)}, nil, nil))

	conds2 := []tablechangealert.Condition{
		{Field: "count", Operator: tablechangealert.OpEq, Value: float64(42)},
	}
	assert.True(t, EvaluateConditions(conds2, "all", map[string]any{"count": "42"}, nil, nil))
}

func TestEvaluateConditions_MissingFieldTreatedAsNull(t *testing.T) {
	conds := []tablechangealert.Condition{
		{Field: "missing", Operator: tablechangealert.OpIsNull},
	}
	assert.True(t, EvaluateConditions(conds, "all", map[string]any{}, nil, nil))

	conds2 := []tablechangealert.Condition{
		{Field: "missing", Operator: tablechangealert.OpEq, Value: "something"},
	}
	assert.False(t, EvaluateConditions(conds2, "all", map[string]any{}, nil, nil))
}
