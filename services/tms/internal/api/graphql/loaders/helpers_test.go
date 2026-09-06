package loaders

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
	batch := batchCountFunc(func(_ context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
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
	batch := batchCountFunc(func(context.Context, []pulid.ID) (map[pulid.ID]int, error) {
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

	batch := batchCountFunc(func(context.Context, []pulid.ID) (map[pulid.ID]int, error) {
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
	batch := batchGroupFunc(func(_ context.Context, ids []pulid.ID) (map[pulid.ID][]string, error) {
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
	batch := batchGroupFunc(func(context.Context, []pulid.ID) (map[pulid.ID][]string, error) {
		return nil, repoErr
	})

	_, errs := batch(t.Context(), []string{id.String()})

	require.Len(t, errs, 1)
	require.ErrorIs(t, errs[0], repoErr)
}
