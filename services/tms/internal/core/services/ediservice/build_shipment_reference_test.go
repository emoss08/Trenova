package ediservice

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExternalShipmentReference(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "LOAD-42", externalShipmentReference(map[string]any{"externalShipmentId": " LOAD-42 "}))
	assert.Empty(t, externalShipmentReference(map[string]any{}))
	assert.Empty(t, externalShipmentReference(nil))
	assert.Empty(t, externalShipmentReference(map[string]any{"externalShipmentId": 42}))
	assert.Len(t, externalShipmentReference(map[string]any{
		"externalShipmentId": strings.Repeat("x", 150),
	}), 100)
}
