package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestATermReadsInTheReadersLanguage(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Freight", Term("Freight").Localize(EN))
	assert.Equal(t, "Flete", Term("Freight").Localize(ES))
	assert.Equal(t, "Teleport Suite", Term("Teleport Suite").Localize(ES), "an unknown term stays as written")
	assert.Equal(t, "", Term("").Localize(ES))
}

func TestATermArgumentIsTranslatedWithTheMessage(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"Flete",
		Translate(ES, "{0}", Term("Freight")),
	)
	assert.Equal(t, "Freight", Term("Freight").String())
}
