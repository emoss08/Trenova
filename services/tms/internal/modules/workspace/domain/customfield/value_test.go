package customfield

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestCustomFieldValue_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("cfv_")
	v := &CustomFieldValue{ID: id}
	assert.Equal(t, id, v.GetID())
}

func TestCustomFieldValue_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	v := &CustomFieldValue{OrganizationID: orgID}
	assert.Equal(t, orgID, v.GetOrganizationID())
}

func TestCustomFieldValue_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	v := &CustomFieldValue{BusinessUnitID: buID}
	assert.Equal(t, buID, v.GetBusinessUnitID())
}
