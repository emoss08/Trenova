package fuelsurcharge_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/stretchr/testify/assert"
)

func TestNormalizePriceDate(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"2026-08-31":                "2026-08-31",
		"2026-08-31T00:00:00Z":      "2026-08-31",
		"2026-08-31 00:00:00+00:00": "2026-08-31",
		"":                          "",
		"08/31/2026":                "08/31/2026",
		"not-a-date-at-all":         "not-a-date-at-all",
	}
	for input, want := range cases {
		assert.Equal(t, want, fuelsurcharge.NormalizePriceDate(input), input)
	}
}
