package insightrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/stretchr/testify/assert"
)

// A finding whose key was just superseded is the same finding measured again;
// only the ones with no active row before the run are new.
func TestNewlyDetected_KeepsOnlyTheFindingsWithNoPriorActiveRow(t *testing.T) {
	t.Parallel()

	inserted := []*insight.Insight{
		{DedupeKey: "ontime:cus_1"},
		{DedupeKey: "ontime:cus_2"},
		{DedupeKey: "ontime:cus_3"},
	}

	detected := newlyDetected(inserted, []string{"ontime:cus_1", "ontime:cus_3"})

	assert.Len(t, detected, 1)
	assert.Equal(t, "ontime:cus_2", detected[0].DedupeKey)
}

func TestNewlyDetected_IsEveryInsertedFindingOnAFirstRun(t *testing.T) {
	t.Parallel()

	inserted := []*insight.Insight{{DedupeKey: "a"}, {DedupeKey: "b"}}

	assert.Equal(t, inserted, newlyDetected(inserted, nil))
	assert.Empty(t, newlyDetected(nil, []string{"a"}))
}
