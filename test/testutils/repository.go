package testutils

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"
)

// TestRepoList is a helper function to test repository List operations
func TestRepoList[T any, K any](
	ctx context.Context,
	t *testing.T,
	repo interface {
		List(context.Context, *K) (*ports.ListResult[T], error)
	},
	opts *K,
	expectedCount int,
) {
	result, err := repo.List(ctx, opts)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Items, expectedCount)
}

func TestRepoGetByID[T any, K any](
	ctx context.Context,
	t *testing.T,
	repo interface {
		GetByID(context.Context, K) (*T, error)
	},
	opts K,
) {
	result, err := repo.GetByID(ctx, opts)
	require.NoError(t, err)
	require.NotNil(t, result)
}

// TestRepoCreate is a helper function to test repository Create operations
func TestRepoCreate[T any](
	ctx context.Context,
	t *testing.T,
	repo interface {
		Create(context.Context, *T) (*T, error)
	},
	entity *T,
) {
	result, err := repo.Create(ctx, entity)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestRepoUpdate[T any](
	ctx context.Context,
	t *testing.T,
	repo interface {
		Update(context.Context, *T) (*T, error)
	},
	entity *T,
) {
	result, err := repo.Update(ctx, entity)
	require.NoError(t, err)
	require.NotNil(t, result)
}
