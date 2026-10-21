package readscope_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncode_IsEmptyWithoutAHolder(t *testing.T) {
	t.Parallel()

	encoded, err := readscope.Encode(t.Context())

	require.NoError(t, err)
	assert.Empty(t, encoded)
}

func TestEncode_NamesOnlyTheNarrowKindsInOrder(t *testing.T) {
	t.Parallel()

	ctx := readscope.With(t.Context(), func(context.Context) (readscope.Grants, error) {
		return readscope.Grants{
			"shipment": {Scope: readscope.ScopeOwn},
			"invoice":  {Scope: readscope.ScopeOwn},
			"order":    {Scope: "organization"},
		}, nil
	})

	encoded, err := readscope.Encode(ctx)

	require.NoError(t, err)
	assert.Equal(t, "invoice:own;shipment:own", encoded)
}

func TestEncode_ListsTheTeamSortedAndOnce(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("usr_"), pulid.MustNew("usr_")
	ids := []string{first.String(), second.String()}
	if ids[0] > ids[1] {
		ids[0], ids[1] = ids[1], ids[0]
	}
	ctx := readscope.With(t.Context(), func(context.Context) (readscope.Grants, error) {
		return readscope.Grants{
			"worker": {Scope: readscope.ScopeTeam, Team: []pulid.ID{second, first, second}},
		}, nil
	})

	encoded, err := readscope.Encode(ctx)

	require.NoError(t, err)
	assert.Equal(t, "worker:team:"+ids[0]+","+ids[1], encoded)
}

func TestEncode_ResolvesOnceAndOutsideItself(t *testing.T) {
	t.Parallel()

	calls := 0
	ctx := readscope.With(t.Context(), func(ctx context.Context) (readscope.Grants, error) {
		calls++
		nested, err := readscope.Encode(ctx)
		require.NoError(t, err)
		assert.Empty(t, nested, "the resolver's own queries are not narrowed")
		return readscope.Grants{"shipment": {Scope: readscope.ScopeOwn}}, nil
	})

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			encoded, err := readscope.Encode(ctx)
			assert.NoError(t, err)
			assert.Equal(t, "shipment:own", encoded)
		})
	}
	wg.Wait()

	assert.Equal(t, 1, calls)
}

func TestEncode_FailsClosedWhenTheScopeCannotBeResolved(t *testing.T) {
	t.Parallel()

	ctx := readscope.With(t.Context(), func(context.Context) (readscope.Grants, error) {
		return nil, errors.New("cache unavailable")
	})

	_, err := readscope.Encode(ctx)

	require.Error(t, err)
}

func TestEncode_RefusesAKindThatIsNotAPlainName(t *testing.T) {
	t.Parallel()

	ctx := readscope.With(t.Context(), func(context.Context) (readscope.Grants, error) {
		return readscope.Grants{"shipment'; RESET ROLE; --": {Scope: readscope.ScopeOwn}}, nil
	})

	_, err := readscope.Encode(ctx)

	require.ErrorIs(t, err, readscope.ErrUnsafeValue)
}

func TestWithout_HidesTheHolder(t *testing.T) {
	t.Parallel()

	ctx := readscope.With(t.Context(), func(context.Context) (readscope.Grants, error) {
		return readscope.Grants{"shipment": {Scope: readscope.ScopeOwn}}, nil
	})

	encoded, err := readscope.Encode(readscope.Without(ctx))

	require.NoError(t, err)
	assert.Empty(t, encoded)
}

func TestNarrows_IsFalseWithoutAHolder(t *testing.T) {
	t.Parallel()

	narrowed, err := readscope.Narrows(t.Context(), "worker")

	require.NoError(t, err)
	assert.False(t, narrowed)
}

func TestNarrows_NamesOnlyTheKindsAnOwnOrTeamGrantNarrows(t *testing.T) {
	t.Parallel()

	ctx := readscope.With(t.Context(), func(context.Context) (readscope.Grants, error) {
		return readscope.Grants{
			"worker":   {Scope: readscope.ScopeTeam, Team: []pulid.ID{pulid.MustNew("usr_")}},
			"shipment": {Scope: readscope.ScopeOwn},
			"order":    {Scope: "organization"},
		}, nil
	})

	for kind, want := range map[string]bool{
		"worker":   true,
		"shipment": true,
		"order":    false,
		"customer": false,
	} {
		narrowed, err := readscope.Narrows(ctx, kind)
		require.NoError(t, err, kind)
		assert.Equal(t, want, narrowed, kind)
	}
}

func TestNarrows_TreatsAScopeThatCannotBeResolvedAsNarrowed(t *testing.T) {
	t.Parallel()

	ctx := readscope.With(t.Context(), func(context.Context) (readscope.Grants, error) {
		return nil, errors.New("team lookup failed")
	})

	narrowed, err := readscope.Narrows(ctx, "customer")

	require.Error(t, err)
	assert.True(t, narrowed)
}
