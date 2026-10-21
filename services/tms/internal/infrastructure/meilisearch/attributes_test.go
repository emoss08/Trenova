package meilisearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAttributesToRetrieve_DefaultsToTheIDAlone(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"id"}, attributesToRetrieve(nil))
	assert.Equal(t, []string{"id", "name"}, attributesToRetrieve([]string{"id", "name"}))
}
