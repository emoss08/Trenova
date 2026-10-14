package formulatypes_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/stretchr/testify/assert"
)

func TestLanguageVersion_AnUnsetVersionIsTheCurrentOne(t *testing.T) {
	t.Parallel()

	assert.Equal(t, formulatypes.CurrentLanguageVersion, formulatypes.LanguageVersion(0).OrCurrent())
	assert.Equal(t, formulatypes.LanguageVersion1, formulatypes.LanguageVersion1.OrCurrent())
}

func TestValidateLanguageVersion_RefusesAVersionThisServerCannotRead(t *testing.T) {
	t.Parallel()

	for _, accepted := range []formulatypes.LanguageVersion{0, formulatypes.LanguageVersion1} {
		multiErr := errortypes.NewMultiError()
		formulatypes.ValidateLanguageVersion("languageVersion", accepted, multiErr)
		assert.False(t, multiErr.HasErrors(), "version %d", accepted)
	}

	for _, refused := range []formulatypes.LanguageVersion{-1, 2} {
		multiErr := errortypes.NewMultiError()
		formulatypes.ValidateLanguageVersion("languageVersion", refused, multiErr)
		assert.True(t, multiErr.HasErrors(), "version %d", refused)
	}
}
