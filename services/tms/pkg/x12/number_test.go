package x12_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/x12"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		dataType string
		want     string
	}{
		{name: "implied cents", value: "125000", dataType: "N2", want: "1250"},
		{name: "negative implied cents", value: "-1999", dataType: "N2", want: "-19.99"},
		{name: "explicit point keeps its value", value: "1250.00", dataType: "N2", want: "1250"},
		{name: "whole number", value: "42", dataType: "N0", want: "42"},
		{name: "decimal type is literal", value: "2.15", dataType: "R", want: "2.15"},
		{name: "unknown type is literal", value: "7", dataType: "", want: "7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := x12.ParseNumber(tt.value, tt.dataType)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestParseNumberRejectsText(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "  ", "12A"} {
		_, err := x12.ParseNumber(value, "N2")
		require.ErrorIs(t, err, x12.ErrNotNumber)
	}
}
