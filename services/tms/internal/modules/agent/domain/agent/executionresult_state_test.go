package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDescribe_SaysWhereTheRecordStandsAfter(t *testing.T) {
	t.Parallel()

	result := (&ToolExecutionResult{
		Action: "posted",
		Kind:   "invoice",
		Name:   "INV-1001",
		State:  "Posted, 2300.00 USD to FreshHaul Foods, unpaid",
	}).Bounded()

	assert.Equal(t,
		`It posted the invoice "INV-1001". Now: Posted, 2300.00 USD to FreshHaul Foods, unpaid.`,
		result.Describe(),
	)
	assert.NotContains(t, (&ToolExecutionResult{Action: "posted", Kind: "invoice"}).Describe(), "Now:")
}
