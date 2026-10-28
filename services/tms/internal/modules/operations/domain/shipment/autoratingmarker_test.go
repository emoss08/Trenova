package shipment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClearAutoRating(t *testing.T) {
	t.Parallel()

	ratedAt := int64(7300)
	entity := &Shipment{AutoRated: true, AutoRatedAt: &ratedAt}

	entity.ClearAutoRating()

	assert.False(t, entity.AutoRated)
	assert.Nil(t, entity.AutoRatedAt)
}

func TestMarkAutoRated(t *testing.T) {
	t.Parallel()

	entity := &Shipment{}
	entity.MarkAutoRated(7300)

	assert.True(t, entity.AutoRated)
	assert.Equal(t, int64(7300), *entity.AutoRatedAt)
}
