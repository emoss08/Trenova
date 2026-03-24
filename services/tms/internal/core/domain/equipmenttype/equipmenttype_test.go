package equipmenttype

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestEquipmentType_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entity  EquipmentType
		wantErr bool
	}{
		{
			name: "valid entity passes",
			entity: EquipmentType{
				ID:             pulid.MustNew("et_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABC",
				Class:          ClassTrailer,
			},
			wantErr: false,
		},
		{
			name: "code empty fails",
			entity: EquipmentType{
				ID:             pulid.MustNew("et_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "",
				Class:          ClassTrailer,
			},
			wantErr: true,
		},
		{
			name: "code too long fails",
			entity: EquipmentType{
				ID:             pulid.MustNew("et_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABCDEFGHIJK",
				Class:          ClassTrailer,
			},
			wantErr: true,
		},
		{
			name: "invalid class fails",
			entity: EquipmentType{
				ID:             pulid.MustNew("et_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABC",
				Class:          Class("Invalid"),
			},
			wantErr: true,
		},
		{
			name: "class empty fails",
			entity: EquipmentType{
				ID:             pulid.MustNew("et_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABC",
				Class:          Class(""),
			},
			wantErr: true,
		},
		{
			name: "code at max length passes",
			entity: EquipmentType{
				ID:             pulid.MustNew("et_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABCDEFGHIJ",
				Class:          ClassTractor,
			},
			wantErr: false,
		},
		{
			name: "code single character passes",
			entity: EquipmentType{
				ID:             pulid.MustNew("et_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "X",
				Class:          ClassContainer,
			},
			wantErr: false,
		},
		{
			name: "class other passes",
			entity: EquipmentType{
				ID:             pulid.MustNew("et_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "OTH",
				Class:          ClassOther,
			},
			wantErr: false,
		},
		{
			name: "both code empty and class invalid fails",
			entity: EquipmentType{
				ID:             pulid.MustNew("et_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "",
				Class:          Class("BadClass"),
			},
			wantErr: true,
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

func TestEquipmentType_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		et := &EquipmentType{}
		require.True(t, et.ID.IsNil())

		err := et.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, et.ID.IsNil())
		assert.NotZero(t, et.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("et_")
		et := &EquipmentType{ID: existingID}

		err := et.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, et.ID)
		assert.NotZero(t, et.CreatedAt)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		et := &EquipmentType{}

		err := et.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, et.UpdatedAt)
	})

	t.Run("update does not set CreatedAt", func(t *testing.T) {
		t.Parallel()

		et := &EquipmentType{}

		err := et.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Zero(t, et.CreatedAt)
		assert.NotZero(t, et.UpdatedAt)
	})

	t.Run("select query does nothing", func(t *testing.T) {
		t.Parallel()

		et := &EquipmentType{}

		err := et.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, et.ID.IsNil())
		assert.Zero(t, et.CreatedAt)
		assert.Zero(t, et.UpdatedAt)
	})
}

func TestEquipmentType_GetTableName(t *testing.T) {
	t.Parallel()

	et := &EquipmentType{}
	assert.Equal(t, "equipment_types", et.GetTableName())
}

func TestEquipmentType_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("et_")
	et := &EquipmentType{ID: id}
	assert.Equal(t, id, et.GetID())
}

func TestEquipmentType_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	et := &EquipmentType{OrganizationID: orgID}
	assert.Equal(t, orgID, et.GetOrganizationID())
}

func TestEquipmentType_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	et := &EquipmentType{BusinessUnitID: buID}
	assert.Equal(t, buID, et.GetBusinessUnitID())
}

func TestEquipmentType_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	et := &EquipmentType{}
	config := et.GetPostgresSearchConfig()

	assert.Equal(t, "et", config.TableAlias)
	assert.True(t, config.UseSearchVector)
	assert.Len(t, config.SearchableFields, 2)
	assert.Equal(t, "code", config.SearchableFields[0].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[0].Type)
	assert.Equal(t, "description", config.SearchableFields[1].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[1].Type)
}

func TestClass_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		class Class
		want  bool
	}{
		{name: "tractor is valid", class: ClassTractor, want: true},
		{name: "trailer is valid", class: ClassTrailer, want: true},
		{name: "container is valid", class: ClassContainer, want: true},
		{name: "other is valid", class: ClassOther, want: true},
		{name: "empty is invalid", class: Class(""), want: false},
		{name: "unknown is invalid", class: Class("Unknown"), want: false},
		{name: "lowercase is invalid", class: Class("tractor"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.class.IsValid())
		})
	}
}

func TestClass_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		class Class
		want  string
	}{
		{name: "tractor", class: ClassTractor, want: "Tractor"},
		{name: "trailer", class: ClassTrailer, want: "Trailer"},
		{name: "container", class: ClassContainer, want: "Container"},
		{name: "other", class: ClassOther, want: "Other"},
		{name: "empty", class: Class(""), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.class.String())
		})
	}
}
