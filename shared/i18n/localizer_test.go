package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type phrase string

func (p phrase) Localize(locale Locale) string {
	return Translate(locale, string(p))
}

func TestTranslate_LocalizesAnArgumentInTheTargetLocale(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"Row 2: El cliente facturable es obligatorio",
		Translate(ES, "Row {0}: {1}", 2, phrase("Bill-to customer is required")),
	)
	assert.Equal(
		t,
		"Row 2: Bill-to customer is required",
		Format("Row {0}: {1}", 2, phrase("Bill-to customer is required")),
	)
}
