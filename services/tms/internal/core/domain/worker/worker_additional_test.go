package worker

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestWorker_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("wrk_")
	w := &Worker{ID: id}
	assert.Equal(t, id, w.GetID())
}

func TestWorker_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	w := &Worker{OrganizationID: orgID}
	assert.Equal(t, orgID, w.GetOrganizationID())
}

func TestWorker_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	w := &Worker{BusinessUnitID: buID}
	assert.Equal(t, buID, w.GetBusinessUnitID())
}

func TestWorker_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	w := &Worker{}
	config := w.GetPostgresSearchConfig()

	assert.Equal(t, "wrk", config.TableAlias)
	assert.True(t, config.UseSearchVector)
	assert.Len(t, config.SearchableFields, 3)

	assert.Equal(t, "first_name", config.SearchableFields[0].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[0].Type)
	assert.Equal(t, domaintypes.SearchWeightA, config.SearchableFields[0].Weight)

	assert.Equal(t, "last_name", config.SearchableFields[1].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[1].Type)
	assert.Equal(t, domaintypes.SearchWeightA, config.SearchableFields[1].Weight)

	assert.Equal(t, "type", config.SearchableFields[2].Name)
	assert.Equal(t, domaintypes.FieldTypeEnum, config.SearchableFields[2].Type)
	assert.Equal(t, domaintypes.SearchWeightB, config.SearchableFields[2].Weight)
}
