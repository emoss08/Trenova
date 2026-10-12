package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithArticle(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "a shipment", WithArticle("shipment"))
	assert.Equal(t, "an invoice", WithArticle("invoice"))
	assert.Equal(t, "an Order", WithArticle(" Order "))
	assert.Empty(t, WithArticle(" "))
}
