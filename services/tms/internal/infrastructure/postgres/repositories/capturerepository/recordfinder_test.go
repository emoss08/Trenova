package capturerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/stretchr/testify/assert"
)

// A kind the domain says is fileable but the finder cannot look up would pass
// validation and then fail every filing onto it.
func TestEveryFileableResourceHasALookup(t *testing.T) {
	t.Parallel()

	for _, resource := range capture.FileableResources() {
		_, ok := recordKinds[resource.String()]
		assert.True(t, ok, "no lookup for fileable resource %s", resource)
	}
	for kind := range recordKinds {
		assert.True(t, capture.IsCaptureTarget(kind), "lookup for %s, which is not a capture target", kind)
	}
}
