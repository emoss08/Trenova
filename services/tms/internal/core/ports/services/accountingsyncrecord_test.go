package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccountingBackfillActionValues_AreEveryAction(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []AccountingBackfillAction{
		AccountingBackfillPause,
		AccountingBackfillResume,
		AccountingBackfillCancel,
	}, AccountingBackfillActionValues())
}
