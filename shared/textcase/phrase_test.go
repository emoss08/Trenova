package textcase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHumanizeSnakeCasePhrase(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"formula_template":   "formula template",
		"ifta_report":        "IFTA report",
		"edi_partner":        "EDI partner",
		"dot_random_pool":    "DOT random pool",
		"worker-pto":         "worker PTO",
		"Billing_Queue_Item": "billing queue item",
		"":                   "",
	}
	for input, want := range cases {
		assert.Equal(t, want, HumanizeSnakeCasePhrase(input), input)
	}
}
