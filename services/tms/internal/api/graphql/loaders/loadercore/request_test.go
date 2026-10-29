package loadercore

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vikstrous/dataloadgen"
)

type echoSource struct {
	prefix string
	builds atomic.Int32
}

func (s *echoSource) Attach(ctx context.Context, tenantInfo pagination.TenantInfo) context.Context {
	return Attach(ctx, tenantInfo, s)
}

func newEchoLoader(
	s *echoSource,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, string] {
	s.builds.Add(1)
	return NewLoader(func(_ context.Context, keys []string) ([]string, []error) {
		values := make([]string, len(keys))
		for idx, key := range keys {
			values[idx] = s.prefix + tenantInfo.OrgID.String() + ":" + key
		}
		return values, make([]error, len(keys))
	})
}

type otherSource struct {
	builds atomic.Int32
}

func (s *otherSource) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return Attach(ctx, tenantInfo, s)
}

func newOtherLoader(
	s *otherSource,
	_ pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	s.builds.Add(1)
	return NewLoader(func(_ context.Context, keys []string) ([]int, []error) {
		return make([]int, len(keys)), make([]error, len(keys))
	})
}

var (
	echoKey  = NewKey(newEchoLoader)
	otherKey = NewKey(newOtherLoader)
)

func requestTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestLoad_WithoutARequestIsNotAttached(t *testing.T) {
	t.Parallel()

	assert.False(t, Available(t.Context()))

	_, err := Load(t.Context(), echoKey, "a")
	require.ErrorIs(t, err, ErrNotAttached)

	_, err = LoadAll(t.Context(), echoKey, []string{"a"})
	require.ErrorIs(t, err, ErrNotAttached)

	_, err = LoadThunk(t.Context(), echoKey, "a")()
	require.ErrorIs(t, err, ErrNotAttached)
}

func TestRegistry_BuildsOnlyTheLoadersARequestUsesOncePerRequest(t *testing.T) {
	t.Parallel()

	echo := &echoSource{prefix: "echo:"}
	other := &otherSource{}
	registry := NewRegistry(RegistryParams{Sources: []Source{echo, other}})
	tenant := requestTenant()
	ctx := registry.Attach(t.Context(), tenant)

	assert.True(t, Available(ctx))
	assert.Zero(t, echo.builds.Load())
	assert.Zero(t, other.builds.Load())

	first, err := Load(ctx, echoKey, "a")
	require.NoError(t, err)
	assert.Equal(t, "echo:"+tenant.OrgID.String()+":a", first)

	all, err := LoadAll(ctx, echoKey, []string{"b", "c"})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"echo:" + tenant.OrgID.String() + ":b",
		"echo:" + tenant.OrgID.String() + ":c",
	}, all)

	assert.Equal(t, int32(1), echo.builds.Load())
	assert.Zero(t, other.builds.Load())

	next := registry.Attach(t.Context(), tenant)
	_, err = Load(next, echoKey, "a")
	require.NoError(t, err)
	assert.Equal(t, int32(2), echo.builds.Load())
}

func TestAttach_AddsSourcesToTheCurrentRequest(t *testing.T) {
	t.Parallel()

	echo := &echoSource{prefix: "echo:"}
	other := &otherSource{}
	tenant := requestTenant()

	ctx := echo.Attach(t.Context(), tenant)
	_, err := Load(ctx, otherKey, "a")
	require.ErrorIs(t, err, ErrNotAttached)

	_, err = Load(ctx, echoKey, "a")
	require.NoError(t, err)

	ctx = other.Attach(ctx, tenant)
	_, err = Load(ctx, otherKey, "a")
	require.NoError(t, err)
	_, err = Load(ctx, echoKey, "b")
	require.NoError(t, err)

	assert.Equal(t, int32(1), echo.builds.Load())
	assert.Equal(t, int32(1), other.builds.Load())
}

func TestGet_MissingSourceIsNotBuilt(t *testing.T) {
	t.Parallel()

	ctx := NewRegistry(RegistryParams{}).Attach(t.Context(), requestTenant())

	loader, ok := Get(ctx, echoKey)
	assert.False(t, ok)
	assert.Nil(t, loader)
}
