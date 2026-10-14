package effectiveversioncache_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/effectiveversioncache"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func countingLoad(
	calls *int,
	version *formulatemplate.FormulaTemplateVersion,
) effectiveversioncache.LoadOne {
	return func(context.Context) (*formulatemplate.FormulaTemplateVersion, error) {
		*calls++
		return version, nil
	}
}

func TestGetVersion_LoadsOncePerVersionWithinContext(t *testing.T) {
	t.Parallel()

	ctx := effectiveversioncache.With(t.Context())
	templateID := pulid.MustNew("ft_")
	snapshot := &formulatemplate.FormulaTemplateVersion{VersionNumber: 3}
	calls := 0

	for range 3 {
		got, err := effectiveversioncache.GetVersion(
			ctx,
			templateID,
			3,
			countingLoad(&calls, snapshot),
		)
		require.NoError(t, err)
		assert.Same(t, snapshot, got)
	}
	assert.Equal(t, 1, calls)

	_, err := effectiveversioncache.GetVersion(ctx, templateID, 4, countingLoad(&calls, snapshot))
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "a different version number is its own entry")
}

func TestGetVersion_MemoizesAbsence(t *testing.T) {
	t.Parallel()

	ctx := effectiveversioncache.With(t.Context())
	templateID := pulid.MustNew("ft_")
	calls := 0

	for range 2 {
		got, err := effectiveversioncache.GetVersion(ctx, templateID, 9, countingLoad(&calls, nil))
		require.NoError(t, err)
		assert.Nil(t, got)
	}
	assert.Equal(t, 1, calls)
}

func TestGetVersion_WithoutMemoLoadsEveryTime(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")
	calls := 0

	for range 2 {
		_, err := effectiveversioncache.GetVersion(
			t.Context(),
			templateID,
			1,
			countingLoad(&calls, nil),
		)
		require.NoError(t, err)
	}
	assert.Equal(t, 2, calls)
}

func TestEffectiveAsOf_PicksTheWindowInForceElseTheEarliest(t *testing.T) {
	t.Parallel()

	scheduled := int64(900)
	approved := []*formulatemplate.FormulaTemplateVersion{
		{VersionNumber: 2, CreatedAt: 500},
		{VersionNumber: 1, CreatedAt: 100},
		{VersionNumber: 3, CreatedAt: 600, EffectiveFrom: &scheduled},
		nil,
	}

	versionAt := func(asOf int64) int64 {
		return effectiveversioncache.EffectiveAsOf(approved, asOf).VersionNumber
	}
	assert.EqualValues(t, 1, versionAt(50), "before every window, the earliest approved snapshot")
	assert.EqualValues(t, 1, versionAt(499))
	assert.EqualValues(t, 2, versionAt(500))
	assert.EqualValues(t, 2, versionAt(899), "a scheduled start is not in force before its date")
	assert.EqualValues(t, 3, versionAt(900))
	assert.Nil(t, effectiveversioncache.EffectiveAsOf(nil, 900))
}
