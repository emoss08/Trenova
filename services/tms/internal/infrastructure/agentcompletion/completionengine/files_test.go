package completionengine

import (
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
)

func TestFileSource_IsAbsentWithoutAReader(t *testing.T) {
	t.Parallel()

	assert.Nil(t, (&Engine{}).FileSource(pagination.TenantInfo{}))
}
