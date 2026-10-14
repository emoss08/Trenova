package commodity_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/stretchr/testify/assert"
)

func TestFreightClass_Number(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "85", commodity.FreightClass85.Number())
	assert.Equal(t, "92.5", commodity.FreightClass92_5.Number())
	assert.Equal(t, "77.5", commodity.FreightClass77_5.Number())
	assert.Equal(t, "500", commodity.FreightClass500.Number())
	assert.Empty(t, commodity.FreightClass("Class86").Number())
	assert.Empty(t, commodity.FreightClass("").Number())

	for _, class := range commodity.FreightClasses {
		parsed, ok := commodity.ParseFreightClass(class.Number())
		assert.True(t, ok, class)
		assert.Equal(t, class, parsed)
	}
}

func TestParseFreightClass(t *testing.T) {
	t.Parallel()

	cases := map[string]commodity.FreightClass{
		"85":        commodity.FreightClass85,
		"85.0":      commodity.FreightClass85,
		" 92.5 ":    commodity.FreightClass92_5,
		"92_5":      commodity.FreightClass92_5,
		"Class92_5": commodity.FreightClass92_5,
		"class 100": commodity.FreightClass100,
		"CLASS500":  commodity.FreightClass500,
	}
	for input, want := range cases {
		got, ok := commodity.ParseFreightClass(input)
		assert.True(t, ok, input)
		assert.Equal(t, want, got, input)
	}

	for _, input := range []string{"", "86", "Class", "abc", "Class86"} {
		_, ok := commodity.ParseFreightClass(input)
		assert.False(t, ok, input)
	}
}
