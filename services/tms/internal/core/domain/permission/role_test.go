package permission

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestRole_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt and UpdatedAt", func(t *testing.T) {
		t.Parallel()

		r := &Role{}
		require.True(t, r.ID.IsNil())

		err := r.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, r.ID.IsNil())
		assert.True(t, strings.HasPrefix(string(r.ID), "rol_"))
		assert.NotZero(t, r.CreatedAt)
		assert.NotZero(t, r.UpdatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("rol_")
		r := &Role{ID: existingID}

		err := r.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, r.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		r := &Role{}

		err := r.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, r.UpdatedAt)
	})

	t.Run("update does not set CreatedAt", func(t *testing.T) {
		t.Parallel()

		r := &Role{}

		err := r.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Zero(t, r.CreatedAt)
		assert.NotZero(t, r.UpdatedAt)
	})
}

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
