package validationframework

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReferenceRequest(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	refID := pulid.MustNew("ref_")

	req := &ReferenceRequest{
		TableName:      "parents",
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ID:             refID,
	}

	assert.Equal(t, "parents", req.TableName)
	assert.Equal(t, orgID, req.OrganizationID)
	assert.Equal(t, buID, req.BusinessUnitID)
	assert.Equal(t, refID, req.ID)
}

func TestNewBunReferenceChecker(t *testing.T) {
	t.Parallel()

	checker := NewBunReferenceChecker(nil)

	require.NotNil(t, checker)
	assert.Nil(t, checker.db)
}

func TestReferenceFieldConfig(t *testing.T) {
	t.Parallel()

	getter := func(e *mockTenantedEntity) pulid.ID {
		return e.ParentID
	}

	config := ReferenceFieldConfig[*mockTenantedEntity]{
		FieldName: "parentId",
		TableName: "parents",
		Message:   "Parent not found",
		Optional:  true,
		GetID:     getter,
	}

	assert.Equal(t, "parentId", config.FieldName)
	assert.Equal(t, "parents", config.TableName)
	assert.Equal(t, "Parent not found", config.Message)
	assert.True(t, config.Optional)
	assert.NotNil(t, config.GetID)

	entity := &mockTenantedEntity{ParentID: pulid.MustNew("par_")}
	assert.Equal(t, entity.ParentID, config.GetID(entity))
}
