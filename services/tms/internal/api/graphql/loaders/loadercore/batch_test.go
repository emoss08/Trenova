package loadercore

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchCountFunc_MapsCountsAndDefaultsMissingToZero(t *testing.T) {
	t.Parallel()

	firstID := pulid.MustNew("pp_")
	secondID := pulid.MustNew("pp_")
	batch := BatchCountFunc(func(_ context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
		assert.Equal(t, []pulid.ID{firstID, secondID}, ids)
		return map[pulid.ID]int{firstID: 3}, nil
	})

	values, errs := batch(t.Context(), []string{
		firstID.String(),
		"bad",
		secondID.String(),
		firstID.String(),
	})

	require.Len(t, values, 4)
	require.NoError(t, errs[0])
	assert.Equal(t, 3, values[0])
	require.Error(t, errs[1])
	require.NoError(t, errs[2])
	assert.Zero(t, values[2])
	require.NoError(t, errs[3])
	assert.Equal(t, 3, values[3])
}

func TestBatchCountFunc_RepositoryErrorFillsValidResults(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("pp_")
	repoErr := errors.New("count failed")
	batch := BatchCountFunc(func(context.Context, []pulid.ID) (map[pulid.ID]int, error) {
		return nil, repoErr
	})

	values, errs := batch(t.Context(), []string{"bad", id.String()})

	require.Len(t, values, 2)
	require.Error(t, errs[0])
	require.NotErrorIs(t, errs[0], repoErr)
	require.ErrorIs(t, errs[1], repoErr)
}

func TestBatchCountFunc_NoValidKeysSkipsFetch(t *testing.T) {
	t.Parallel()

	batch := BatchCountFunc(func(context.Context, []pulid.ID) (map[pulid.ID]int, error) {
		t.Fatal("fetch must not run without valid keys")
		return nil, nil
	})

	values, errs := batch(t.Context(), []string{"bad"})

	require.Len(t, values, 1)
	require.Error(t, errs[0])
}

func TestBatchGroupFunc_MapsGroupsAndDefaultsMissingToEmpty(t *testing.T) {
	t.Parallel()

	firstID := pulid.MustNew("wk_")
	secondID := pulid.MustNew("wk_")
	batch := BatchGroupFunc(func(_ context.Context, ids []pulid.ID) (map[pulid.ID][]string, error) {
		assert.Equal(t, []pulid.ID{firstID, secondID}, ids)
		return map[pulid.ID][]string{firstID: {"a", "b"}}, nil
	})

	values, errs := batch(t.Context(), []string{
		firstID.String(),
		secondID.String(),
		"bad",
	})

	require.Len(t, values, 3)
	require.NoError(t, errs[0])
	assert.Equal(t, []string{"a", "b"}, values[0])
	require.NoError(t, errs[1])
	assert.NotNil(t, values[1])
	assert.Empty(t, values[1])
	require.Error(t, errs[2])
	assert.Nil(t, values[2])
}

func TestBatchGroupFunc_RepositoryErrorFillsValidResults(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("wk_")
	repoErr := errors.New("list failed")
	batch := BatchGroupFunc(func(context.Context, []pulid.ID) (map[pulid.ID][]string, error) {
		return nil, repoErr
	})

	_, errs := batch(t.Context(), []string{id.String()})

	require.Len(t, errs, 1)
	require.ErrorIs(t, errs[0], repoErr)
}

func TestBatchFirstFunc_ReadsEachWindowOnceForEveryParentInIt(t *testing.T) {
	t.Parallel()

	a := pulid.MustNew("ragr_")
	b := pulid.MustNew("ragr_")
	type call struct {
		ids   []pulid.ID
		first int
	}
	var calls []call
	batch := BatchFirstFunc[string](
		func(_ context.Context, ids []pulid.ID, first int) (map[pulid.ID][]string, error) {
			calls = append(calls, call{ids: ids, first: first})
			rows := map[pulid.ID][]string{a: {"a1", "a2"}}
			if first == 1 {
				rows = map[pulid.ID][]string{a: {"a1"}}
			}
			return rows, nil
		},
	)

	values, errs := batch(t.Context(), []string{
		FirstKey(a.String(), 2),
		FirstKey(b.String(), 2),
		FirstKey(a.String(), 1),
		FirstKey(a.String(), 2),
		"2:bad",
		"x:" + a.String(),
	})

	assert.Equal(
		t,
		[]call{{ids: []pulid.ID{a, b}, first: 2}, {ids: []pulid.ID{a}, first: 1}},
		calls,
		"one read per first, every parent of that first in it, and different firsts never mixed",
	)
	require.NoError(t, errs[0])
	assert.Equal(t, []string{"a1", "a2"}, values[0])
	require.NoError(t, errs[1])
	assert.Equal(t, []string{}, values[1], "a parent with no rows reads as an empty list")
	assert.Equal(t, []string{"a1"}, values[2])
	assert.Equal(t, values[0], values[3])
	require.Error(t, errs[4])
	require.Error(t, errs[5])
}

func TestBatchFirstFunc_AFailedWindowFailsOnlyItsKeys(t *testing.T) {
	t.Parallel()

	a := pulid.MustNew("ragr_")
	readErr := errors.New("read failed")
	batch := BatchFirstFunc[string](
		func(_ context.Context, _ []pulid.ID, first int) (map[pulid.ID][]string, error) {
			if first == 5 {
				return nil, readErr
			}
			return map[pulid.ID][]string{a: {"a1"}}, nil
		},
	)

	values, errs := batch(t.Context(), []string{FirstKey(a.String(), 5), FirstKey(a.String(), 1)})

	require.ErrorIs(t, errs[0], readErr)
	require.NoError(t, errs[1])
	assert.Equal(t, []string{"a1"}, values[1])
}
