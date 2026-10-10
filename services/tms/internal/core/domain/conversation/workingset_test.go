package conversation_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func working(t *testing.T, prefix string, source conversation.WorkingSource, at int64) conversation.WorkingRecord {
	t.Helper()

	record, ok := conversation.NewWorkingRecord(pulid.MustNew(prefix).String(), "", source, at)
	require.True(t, ok)

	return record
}

func TestNewWorkingRecord_KnowsTheKindByItsPrefix(t *testing.T) {
	t.Parallel()

	record, ok := conversation.NewWorkingRecord(
		pulid.MustNew("shp_").String(), "SEED-DET-001", conversation.WorkingSourceRead, 10,
	)
	require.True(t, ok)
	assert.Equal(t, permission.RecordKind(permission.ResourceShipment), record.Kind)
	assert.Equal(t, "SEED-DET-001", record.Label)

	_, ok = conversation.NewWorkingRecord("not-an-id", "", conversation.WorkingSourceRead, 10)
	assert.False(t, ok)
	_, ok = conversation.NewWorkingRecord("zzz_01M49Z0BXETE12D1CPXJQ15NWE", "", conversation.WorkingSourceRead, 10)
	assert.False(t, ok, "a prefix no record kind carries")
}

func TestTouch_KeepsTheNewestFirstAndTheSubjectAhead(t *testing.T) {
	t.Parallel()

	subject := working(t, "shp_", conversation.WorkingSourceSubject, 1)
	old := working(t, "inv_", conversation.WorkingSourceRead, 2)
	recent := working(t, "wrk_", conversation.WorkingSourceMention, 5)

	set := conversation.Touch(nil, subject, old, recent)
	require.Len(t, set, 3)
	assert.Equal(t, subject.ID, set[0].ID)
	assert.Equal(t, recent.ID, set[1].ID)
	assert.Equal(t, old.ID, set[2].ID)

	again := old
	again.TouchedAt = 9
	again.Source = conversation.WorkingSourceWrote
	set = conversation.Touch(set, again)
	require.Len(t, set, 3)
	assert.Equal(t, old.ID, set[1].ID, "touched again, it moves up")
	assert.Equal(t, conversation.WorkingSourceWrote, set[1].Source)
}

func TestTouch_HoldsAtMostTheCap(t *testing.T) {
	t.Parallel()

	set := []conversation.WorkingRecord{}
	for idx := range conversation.MaxWorkingSet + 5 {
		set = conversation.Touch(set, working(t, "shp_", conversation.WorkingSourceRead, int64(idx)))
	}

	require.Len(t, set, conversation.MaxWorkingSet)
	assert.EqualValues(t, conversation.MaxWorkingSet+4, set[0].TouchedAt, "the oldest fall off")
}

func TestForget_DropsOneRecord(t *testing.T) {
	t.Parallel()

	first := working(t, "shp_", conversation.WorkingSourceRead, 1)
	second := working(t, "inv_", conversation.WorkingSourceRead, 2)
	set := conversation.Touch(nil, first, second)

	assert.Len(t, conversation.Forget(set, first.ID), 1)
	assert.Len(t, set, 2, "the set given is left alone")
}
