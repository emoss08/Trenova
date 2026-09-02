package effectiveversioncache_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/services/formula/effectiveversioncache"
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

func TestGetLatestActive_IsSeparateFromNumberedVersions(t *testing.T) {
	t.Parallel()

	ctx := effectiveversioncache.With(t.Context())
	templateID := pulid.MustNew("ft_")
	active := &formulatemplate.FormulaTemplateVersion{VersionNumber: 2}
	calls := 0

	got, err := effectiveversioncache.GetLatestActive(ctx, templateID, countingLoad(&calls, active))
	require.NoError(t, err)
	assert.Same(t, active, got)

	got, err = effectiveversioncache.GetVersion(ctx, templateID, 2, countingLoad(&calls, active))
	require.NoError(t, err)
	assert.Same(t, active, got)
	assert.Equal(t, 2, calls, "latest-active and version 2 are distinct slots")
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
