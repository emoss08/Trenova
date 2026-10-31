package documentextractionjobs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOCRIsSkippedOnlyWhenEveryPageCarriesText(t *testing.T) {
	t.Parallel()

	assert.True(t, everyPageNative(map[int]bool{1: true, 2: true}, 2))
	assert.False(t, everyPageNative(map[int]bool{1: true, 2: false}, 2))
	assert.False(
		t,
		everyPageNative(map[int]bool{1: true}, 2),
		"a page the probe did not report is read with OCR",
	)
	assert.False(t, everyPageNative(nil, 0))
}
