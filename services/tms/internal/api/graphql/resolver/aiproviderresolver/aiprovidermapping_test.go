package aiproviderresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelOptionsToModel_CarriesWhenEachModelWasCreated(t *testing.T) {
	t.Parallel()

	created := int64(1754352000)
	out := modelOptionsToModel([]services.AIProviderModelOption{
		{ID: "claude-opus-4-1", CreatedAt: &created},
		{ID: "local"},
	})

	require.Len(t, out, 2)
	require.NotNil(t, out[0].CreatedAt)
	assert.Equal(t, 1754352000, *out[0].CreatedAt)
	assert.Nil(t, out[1].CreatedAt)
}
