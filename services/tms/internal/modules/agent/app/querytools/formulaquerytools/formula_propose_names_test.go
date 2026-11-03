package formulaquerytools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Each propose tool makes the page edit of the same name.
func TestProposeTools_AreNamedForTheEditTheyMake(t *testing.T) {
	t.Parallel()

	for _, spec := range []proposeSourceSpec{proposeFormulaSpec, proposeLibrarySpec} {
		assert.Equal(t, string(spec.action), spec.name)
		assert.Equal(t, spec.name, (&proposeSourceTool{spec: spec}).Name())
	}
}
