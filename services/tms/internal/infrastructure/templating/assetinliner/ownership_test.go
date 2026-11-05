package assetinliner

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestOwnsStorageKey(t *testing.T) {
	t.Parallel()

	organizationID := pulid.MustNew("org_")
	other := pulid.MustNew("org_")

	assert.True(t, ownsStorageKey(organizationID, organizationID.String()+"/organization/logo/a.png"))
	assert.False(t, ownsStorageKey(organizationID, other.String()+"/organization/logo/a.png"))
	assert.False(t, ownsStorageKey(organizationID, organizationID.String()+"/../"+other.String()+"/a.png"))
	assert.False(t, ownsStorageKey(pulid.Nil, "/a.png"))
	assert.False(t, ownsStorageKey(organizationID, "a.png"))
}
