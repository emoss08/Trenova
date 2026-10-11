package hazardousmaterial

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func validEntity() HazardousMaterial {
	return HazardousMaterial{
		ID:             pulid.MustNew("hm_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Name:           "Hydrochloric Acid",
		Description:    "Corrosive liquid",
		Class:          HazardousClass8,
		PackingGroup:   PackingGroupII,
	}
}

func TestHazardousMaterial_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(*HazardousMaterial)
		wantErr bool
	}{
		{
			name:    "valid entity passes",
			modify:  func(_ *HazardousMaterial) {},
			wantErr: false,
		},
		{
			name:    "name empty fails",
			modify:  func(hm *HazardousMaterial) { hm.Name = "" },
			wantErr: true,
		},
		{
			name:    "description empty fails",
			modify:  func(hm *HazardousMaterial) { hm.Description = "" },
			wantErr: true,
		},
		{
			name:    "class empty fails",
			modify:  func(hm *HazardousMaterial) { hm.Class = "" },
			wantErr: true,
		},
		{
			name:    "invalid class fails",
			modify:  func(hm *HazardousMaterial) { hm.Class = HazardousClass("InvalidClass") },
			wantErr: true,
		},
		{
			name:    "packing group empty fails",
			modify:  func(hm *HazardousMaterial) { hm.PackingGroup = "" },
			wantErr: true,
		},
		{
			name:    "invalid packing group fails",
			modify:  func(hm *HazardousMaterial) { hm.PackingGroup = PackingGroup("IV") },
			wantErr: true,
		},
		{
			name: "valid special provisions passes",
			modify: func(hm *HazardousMaterial) {
				hm.SpecialProvisions = "A1, B2, C3"
			},
			wantErr: false,
		},
		{
			name: "invalid special provisions fails",
			modify: func(hm *HazardousMaterial) {
				hm.SpecialProvisions = "A1,,B2"
			},
			wantErr: true,
		},
		{
			name: "empty special provisions passes",
			modify: func(hm *HazardousMaterial) {
				hm.SpecialProvisions = ""
			},
			wantErr: false,
		},
		{
			name: "name too long fails",
			modify: func(hm *HazardousMaterial) {
				hm.Name = string(make([]byte, 101))
			},
			wantErr: true,
		},
		{
			name:    "all hazard classes are valid",
			modify:  func(hm *HazardousMaterial) { hm.Class = HazardousClass1And6 },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entity := validEntity()
			tt.modify(&entity)

			multiErr := errortypes.NewMultiError()
			entity.Validate(multiErr)

			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestHazardousMaterial_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		hm := &HazardousMaterial{}
		require.True(t, hm.ID.IsNil())

		err := hm.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, hm.ID.IsNil())
		assert.NotZero(t, hm.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("hm_")
		hm := &HazardousMaterial{ID: existingID}

		err := hm.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, hm.ID)
		assert.NotZero(t, hm.CreatedAt)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		hm := &HazardousMaterial{}

		err := hm.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, hm.UpdatedAt)
	})

	t.Run("update does not set CreatedAt", func(t *testing.T) {
		t.Parallel()

		hm := &HazardousMaterial{}

		err := hm.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Zero(t, hm.CreatedAt)
		assert.NotZero(t, hm.UpdatedAt)
	})

	t.Run("select query does nothing", func(t *testing.T) {
		t.Parallel()

		hm := &HazardousMaterial{}

		err := hm.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, hm.ID.IsNil())
		assert.Zero(t, hm.CreatedAt)
		assert.Zero(t, hm.UpdatedAt)
	})
}

func TestHazardousMaterial_GetTableName(t *testing.T) {
	t.Parallel()

	hm := &HazardousMaterial{}
	assert.Equal(t, "hazardous_materials", hm.GetTableName())
}

func TestHazardousMaterial_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("hm_")
	hm := &HazardousMaterial{ID: id}
	assert.Equal(t, id, hm.GetID())
}

func TestHazardousMaterial_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	hm := &HazardousMaterial{OrganizationID: orgID}
	assert.Equal(t, orgID, hm.GetOrganizationID())
}

func TestHazardousMaterial_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	hm := &HazardousMaterial{BusinessUnitID: buID}
	assert.Equal(t, buID, hm.GetBusinessUnitID())
}

func TestHazardousMaterial_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	hm := &HazardousMaterial{}
	config := hm.GetPostgresSearchConfig()

	assert.Equal(t, "hm", config.TableAlias)
	assert.Len(t, config.SearchableFields, 4)
	assert.Equal(t, "name", config.SearchableFields[0].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[0].Type)
	assert.Equal(t, domaintypes.SearchWeightA, config.SearchableFields[0].Weight)
	assert.Equal(t, "description", config.SearchableFields[1].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[1].Type)
	assert.Equal(t, "class", config.SearchableFields[2].Name)
	assert.Equal(t, domaintypes.FieldTypeEnum, config.SearchableFields[2].Type)
	assert.Equal(t, "packing_group", config.SearchableFields[3].Name)
	assert.Equal(t, domaintypes.FieldTypeEnum, config.SearchableFields[3].Type)
}
