package briefingjobs

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDueSkipsOrganizationsWithoutTheDesk(t *testing.T) {
	t.Parallel()

	a := &Activities{features: plantest.Without(platformcatalog.FeatureAIDesk)}

	due, err := a.isDue(t.Context(), plantest.Tenant(), time.Now().Unix())

	require.NoError(t, err)
	assert.False(t, due)
}
