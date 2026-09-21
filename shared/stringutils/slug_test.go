package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t, "revenue-by-service-shipment-type", Slugify("Revenue by Service & Shipment Type", 0),
	)
	assert.Equal(t, "run-the-report", Slugify("  Run the \"report\"!  ", 0))
	assert.Equal(t, "revenue-by", Slugify("Revenue by service", 11))
	assert.Equal(t, "", Slugify("???", 20))
	assert.Equal(t, "café-au-lait", Slugify("Café au lait", 0))
}
