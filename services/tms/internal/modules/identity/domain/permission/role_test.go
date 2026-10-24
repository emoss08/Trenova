package permission

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRole_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("rol_")
	r := &Role{ID: id}
	assert.Equal(t, id, r.GetID())
}

func TestRole_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	r := &Role{OrganizationID: orgID}
	assert.Equal(t, orgID, r.GetOrganizationID())
}

func TestRole_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	r := &Role{BusinessUnitID: pulid.MustNew("bu_")}
	assert.Equal(t, pulid.ID(""), r.GetBusinessUnitID())
}

func TestRole_GetTableName(t *testing.T) {
	t.Parallel()

	r := &Role{}
	assert.Equal(t, "roles", r.GetTableName())
}

func TestRole_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		role    Role
		wantErr bool
	}{
		{
			name: "valid role with name only",
			role: Role{
				Name: "Admin",
			},
			wantErr: false,
		},
		{
			name: "valid role with name and description",
			role: Role{
				Name:        "Admin",
				Description: "Administrator role",
			},
			wantErr: false,
		},
		{
			name: "empty name fails",
			role: Role{
				Name: "",
			},
			wantErr: true,
		},
		{
			name: "name too long fails",
			role: Role{
				Name: strings.Repeat("a", 256),
			},
			wantErr: true,
		},
		{
			name: "name at max length passes",
			role: Role{
				Name: strings.Repeat("a", 255),
			},
			wantErr: false,
		},
		{
			name: "description is optional",
			role: Role{
				Name:        "Viewer",
				Description: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			multiErr := errortypes.NewMultiError()
			tt.role.Validate(multiErr)

			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestRole_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	r := &Role{}
	config := r.GetPostgresSearchConfig()

	assert.Equal(t, "r", config.TableAlias)
	require.Len(t, config.SearchableFields, 2)
	assert.Equal(t, "name", config.SearchableFields[0].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[0].Type)
	assert.Equal(t, "description", config.SearchableFields[1].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[1].Type)
}
