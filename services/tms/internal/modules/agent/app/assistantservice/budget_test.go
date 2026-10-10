package assistantservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBudgetShare(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 0.964, budgetShare("48.20", "50.00"), 1e-9)
	assert.InDelta(t, 1.1, budgetShare("55", "50"), 1e-9)
	assert.Zero(t, budgetShare("10", "0"), "a nought budget has no share")
	assert.Zero(t, budgetShare("10", ""), "no budget has no share")
	assert.Zero(t, budgetShare("ten", "50"))
}
