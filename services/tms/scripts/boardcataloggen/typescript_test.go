package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheClientsBoardCatalogIsCurrent(t *testing.T) {
	t.Parallel()

	written, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(target)))
	require.NoError(t, err, "run go run ./scripts/boardcataloggen from services/tms")
	assert.Equal(t, typeScript(), string(written),
		"the catalog changed: run go run ./scripts/boardcataloggen from services/tms")
}
