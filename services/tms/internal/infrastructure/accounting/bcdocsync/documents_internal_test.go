package bcdocsync

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/stretchr/testify/assert"
)

func TestARevisedBillNumberFitsAndDiffersFromTheOriginal(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("9", businesscentral.MaxExternalDocumentNumberLength)
	revised := RevisedNumber(long, "req-1")
	assert.Len(t, revised, businesscentral.MaxExternalDocumentNumberLength)
	assert.True(
		t,
		strings.HasSuffix(revised, revisionSeparator+revised[len(revised)-revisionHashLength:]),
	)
	assert.NotEqual(t, RevisedNumber("OO-78", "req-1"), RevisedNumber("OO-78", "req-2"))
	assert.Equal(t, RevisedNumber("OO-78", "req-1"), RevisedNumber("OO-78", "req-1"))
	assert.Empty(t, RevisedNumber(" ", "req-1"))
}
