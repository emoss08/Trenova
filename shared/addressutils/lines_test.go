package addressutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCityStatePostal(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Dallas, TX 75201", CityStatePostal("Dallas", "TX", "75201"))
	assert.Equal(t, "Dallas", CityStatePostal(" Dallas ", "", ""))
	assert.Equal(t, "TX 75201", CityStatePostal("", "TX", "75201"))
	assert.Equal(t, "Dallas, 75201", CityStatePostal("Dallas", "", "75201"))
	assert.Empty(t, CityStatePostal("", "", ""))
}

func TestMailingLines(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]string{"100 Main St", "Dallas, TX 75201", "United States"},
		Mailing{
			Line1:      "100 Main St",
			Line2:      "  ",
			City:       "Dallas",
			State:      "TX",
			PostalCode: "75201",
			Country:    "United States",
		}.Lines(),
	)
	assert.Empty(t, Mailing{}.Lines())
}
