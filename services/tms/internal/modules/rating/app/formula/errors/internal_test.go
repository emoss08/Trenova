package errors_test

import (
	"fmt"
	"testing"
	"time"

	formulaerrors "github.com/emoss08/trenova/internal/modules/rating/app/formula/errors"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestDescribeValue_NamesNoGoTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value any
		want  string
	}{
		{nil, "an empty value"},
		{true, "true/false"},
		{"85", "text"},
		{int64(3), "a number"},
		{2.5, "a number"},
		{decimal.NewFromInt(1), "a number"},
		{time.Time{}, "a date"},
		{time.Minute, "a duration"},
		{[]any{1}, "a list"},
		{map[string]any{"code": "A"}, "a group of fields"},
		{&struct{}{}, "a record"},
		{func() {}, "a function"},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.want, formulaerrors.DescribeValue(tc.value), "%T", tc.value)
	}
}

func TestLogFields_KeepWhatTheMessageLeavesOut(t *testing.T) {
	t.Parallel()

	panicked := formulaerrors.NewPanicError("index out of range", []byte("stack"))
	err := fmt.Errorf("rating: %w", formulaerrors.NewResolveError("Customer.Code", "*shipment.Shipment", panicked))

	assert.NotContains(t, err.Error(), "*shipment.Shipment")
	assert.NotContains(t, err.Error(), "index out of range")

	encoder := zapcore.NewMapObjectEncoder()
	for _, field := range formulaerrors.LogFields(err) {
		field.AddTo(encoder)
	}
	assert.Equal(t, "*shipment.Shipment", encoder.Fields["entityType"])
	assert.Equal(t, "index out of range", encoder.Fields["panic"])
	assert.Equal(t, "stack", encoder.Fields["stack"])
}
