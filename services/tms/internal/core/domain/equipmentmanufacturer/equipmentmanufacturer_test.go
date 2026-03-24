package equipmentmanufacturer

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestEquipmentManufacturer_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entity  EquipmentManufacturer
		wantErr bool
	}{
		{
			name: "valid entity passes",
			entity: EquipmentManufacturer{
				ID:             pulid.MustNew("em_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Name:           "Volvo",
			},
			wantErr: false,
		},
		{
			name: "name empty fails",
			entity: EquipmentManufacturer{
				ID:             pulid.MustNew("em_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Name:           "",
			},
			wantErr: true,
		},
		{
			name: "name too long fails",
			entity: EquipmentManufacturer{
				ID:             pulid.MustNew("em_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Name:           strings.Repeat("a", 101),
			},
			wantErr: true,
		},
		{
			name: "name at max length passes",
			entity: EquipmentManufacturer{
				ID:             pulid.MustNew("em_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Name:           strings.Repeat("a", 100),
			},
			wantErr: false,
		},
		{
			name: "name single character passes",
			entity: EquipmentManufacturer{
				ID:             pulid.MustNew("em_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Name:           "A",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			multiErr := errortypes.NewMultiError()
			tt.entity.Validate(multiErr)

			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestEquipmentManufacturer_GetTableName(t *testing.T) {
	t.Parallel()

	em := &EquipmentManufacturer{}
	assert.Equal(t, "equipment_manufacturers", em.GetTableName())
}

func TestEquipmentManufacturer_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("em_")
	em := &EquipmentManufacturer{ID: id}
	assert.Equal(t, id, em.GetID())
}

func TestEquipmentManufacturer_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	em := &EquipmentManufacturer{OrganizationID: orgID}
	assert.Equal(t, orgID, em.GetOrganizationID())
}

func TestEquipmentManufacturer_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	em := &EquipmentManufacturer{BusinessUnitID: buID}
	assert.Equal(t, buID, em.GetBusinessUnitID())
}

func TestEquipmentManufacturer_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		em := &EquipmentManufacturer{}
		require.True(t, em.ID.IsNil())

		err := em.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, em.ID.IsNil())
		assert.NotZero(t, em.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("em_")
		em := &EquipmentManufacturer{ID: existingID}

		err := em.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, em.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		em := &EquipmentManufacturer{}

		err := em.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, em.UpdatedAt)
	})

	t.Run("update does not set CreatedAt", func(t *testing.T) {
		t.Parallel()

		em := &EquipmentManufacturer{}

		err := em.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Zero(t, em.CreatedAt)
		assert.NotZero(t, em.UpdatedAt)
	})

	t.Run("select query does nothing", func(t *testing.T) {
		t.Parallel()

		em := &EquipmentManufacturer{}

		err := em.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, em.ID.IsNil())
		assert.Zero(t, em.CreatedAt)
		assert.Zero(t, em.UpdatedAt)
	})
}

func TestEquipmentManufacturer_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	em := &EquipmentManufacturer{}
	config := em.GetPostgresSearchConfig()

	assert.Equal(t, "em", config.TableAlias)
	assert.False(t, config.UseSearchVector)
	assert.Len(t, config.SearchableFields, 1)
	assert.Equal(t, "name", config.SearchableFields[0].Name)
}
